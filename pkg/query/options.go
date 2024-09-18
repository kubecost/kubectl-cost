package query

import (
	"context"
	"fmt"
	"time"

	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/opencost/opencost/core/pkg/log"

	"k8s.io/client-go/rest"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// OpenCost specification parameter values
const (
	OpenCostServiceName      = "opencost"
	OpenCostServiceNamespace = "opencost"
	OpenCostServicePort      = 9003
	OpenCostAllocationPath   = "/allocation/compute"
)

// QueryBackendOptions holds common options for managing the query backend used
// by kubectl-cost, like service name, namespace, etc.
type QueryBackendOptions struct {
	// If set, will proxy a request through the K8s API server
	// instead of port forwarding.
	UseProxy bool

	// HelmReleaseName is used to template into service name/etc. to require
	// less flags if Kubecost is installed in a non-"kubecost" namespace.
	//
	// Defaults to "kubecost".
	HelmReleaseName string

	// The name of the K8s service for Kubecost. By default, this is templated
	// from HelmReleaseName.
	ServiceName string

	// The namespace in which Kubecost is running. By default, this is templated
	// from HelmReleaseName.
	KubecostNamespace string

	// The port at which the Service should be queried
	ServicePort int

	// A path which can serve Allocation queries, e.g. "/model/allocation"
	AllocationPath string

	// A path which can serve Spec Cost Prediction queries.
	// e.g. "/prediction/speccost"
	PredictSpecCostPath string

	// A boolean value  to automatically set parameters according to OpenCost specification.
	OpenCost bool

	restConfig *rest.Config
	pfQuerier  *PortForwardQuerier
}

func (o *QueryBackendOptions) Complete(restConfig *rest.Config) error {
	if o.OpenCost {
		o.ServiceName = OpenCostServiceName
		o.KubecostNamespace = OpenCostServiceNamespace
		o.ServicePort = OpenCostServicePort
		o.AllocationPath = OpenCostAllocationPath
	}
	if o.ServiceName == "" {
		o.ServiceName = fmt.Sprintf("%s-cost-analyzer", o.HelmReleaseName)
		log.Debugf("ServiceName set to: %s", o.ServiceName)
	}
	if o.KubecostNamespace == "" {
		o.KubecostNamespace = o.HelmReleaseName
		log.Debugf("KubecostNamespace set to: %s", o.KubecostNamespace)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if !o.UseProxy {
		pfQ, err := CreatePortForwardForService(restConfig, o.KubecostNamespace, o.ServiceName, o.ServicePort, ctx)
		if err != nil {
			return fmt.Errorf("port-forwarding requested service '%s' (port %d) in namespace '%s': %s", o.ServiceName, o.ServicePort, o.KubecostNamespace, err)
		}
		o.pfQuerier = pfQ
	} else {
		o.restConfig = restConfig
	}
	return nil
}

func (o *QueryBackendOptions) Validate() error {
	if o.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if o.KubecostNamespace == "" {
		return fmt.Errorf("namespace for Kubecost cannot be empty")
	}
	return nil
}

func AddQueryBackendOptionsFlags(cmd *cobra.Command, options *QueryBackendOptions) {
	cmd.Flags().StringVarP(&options.HelmReleaseName, "release-name", "r", "kubecost", "The name of the Helm release, used to template service names if they are unset. For example, if Kubecost is installed with 'helm install kubecost2 kubecost/cost-analyzer', then this should be set to 'kubecost2'.")
	cmd.Flags().StringVarP(&options.KubecostNamespace, "kubecost-namespace", "N", "", "The namespace that Kubecost is deployed in. Requests to the API will be directed to this namespace. Defaults to the Helm release name.")

	cmd.Flags().IntVar(&options.ServicePort, "service-port", 9090, "The port of the service at which the APIs are running. If using OpenCost, you may want to set this to 9003.")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "", "The name of the Kubecost cost analyzer service. By default, it is derived from the Helm release name and should not need to be overridden.")
	cmd.Flags().BoolVar(&options.UseProxy, "use-proxy", false, "Instead of temporarily port-forwarding, proxy a request to Kubecost through the Kubernetes API server.")
	cmd.Flags().StringVar(&options.AllocationPath, "allocation-path", "/model/allocation", "URL path at which Allocation queries can be served from the configured service. If using OpenCost, you may want to set this to '/allocation/compute'")
	cmd.Flags().StringVar(&options.PredictSpecCostPath, "predict-speccost-path", "/model/prediction/speccost", "URL path at which Prediction queries can be served from the configured service.")
	cmd.Flags().BoolVar(&options.OpenCost, "opencost", false, " Set true to configure Kubecost parameters according to the OpenCost default specification. It is equivalent to providing the options '--service-port 9003 --service-name opencost --kubecost-namespace opencost --allocation-path /allocation/compute'.")

	//Check if environment variable KUBECTL_COST_USE_PROXY is set, it defaults to false
	v := viper.New()
	v.SetEnvPrefix(utilities.EnvPrefix)
	v.AutomaticEnv()
	utilities.BindAFlagToViperEnv(cmd, v, "use-proxy")
}

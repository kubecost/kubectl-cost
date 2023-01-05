package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/kubecost"
)

// CostOptions holds common options for querying and displaying
// data from the kubecost API
type CostOptions struct {
	window          string
	filterNamespace string

	isHistorical bool
	showAll      bool

	displayOptions
	query.QueryBackendOptions
}

const (
	envPrefix = "KUBECTL_COST"
)

// With the addition of commands which query the assets API,
// some of these don't apply to all commands. However, as they
// are applied during the output, this shouldn't cause issues.
type displayOptions struct {
	showCPUCost          bool
	showMemoryCost       bool
	showGPUCost          bool
	showPVCost           bool
	showNetworkCost      bool
	showEfficiency       bool
	showSharedCost       bool
	showLoadBalancerCost bool
	showAssetType        bool
}

func addCostOptionsFlags(cmd *cobra.Command, options *CostOptions) {
	cmd.Flags().StringVar(&options.window, "window", "1d", "The window of data to query. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")
	cmd.Flags().BoolVar(&options.isHistorical, "historical", false, "show the total cost during the window instead of the projected monthly rate based on the data in the window")
	cmd.Flags().BoolVar(&options.showCPUCost, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&options.showMemoryCost, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&options.showGPUCost, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&options.showPVCost, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&options.showNetworkCost, "show-network", false, "show data for network cost")
	cmd.Flags().BoolVar(&options.showSharedCost, "show-shared", false, "show shared cost data")
	cmd.Flags().BoolVar(&options.showLoadBalancerCost, "show-lb", false, "show load balancer cost data")
	cmd.Flags().BoolVar(&options.showEfficiency, "show-efficiency", true, "show efficiency of cost alongside CPU and memory cost")
	cmd.Flags().BoolVar(&options.showAssetType, "show-asset-type", false, "show type of assets displayed.")
	cmd.Flags().BoolVarP(&options.showAll, "show-all-resources", "A", false, "Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network --show-efficiency for namespace, deployment, controller, lable and pod OR --show-type --show-cpu --show-memory for node.")

	addQueryBackendOptionsFlags(cmd, &options.QueryBackendOptions)
}

func addQueryBackendOptionsFlags(cmd *cobra.Command, options *query.QueryBackendOptions) {
	cmd.Flags().StringVarP(&options.HelmReleaseName, "release-name", "r", "kubecost", "The name of the Helm release, used to template service names if they are unset. For example, if Kubecost is installed with 'helm install kubecost2 kubecost/cost-analyzer', then this should be set to 'kubecost2'.")
	cmd.Flags().StringVarP(&options.KubecostNamespace, "kubecost-namespace", "N", "", "The namespace that Kubecost is deployed in. Requests to the API will be directed to this namespace. Defaults to the Helm release name.")

	cmd.Flags().IntVar(&options.ServicePort, "service-port", 9090, "The port of the service at which the APIs are running. If using OpenCost, you may want to set this to 9003.")
	cmd.Flags().StringVar(&options.ServiceName, "service-name", "", "The name of the Kubecost cost analyzer service. By default, it is derived from the Helm release name and should not need to be overridden.")
	cmd.Flags().BoolVar(&options.UseProxy, "use-proxy", false, "Instead of temporarily port-forwarding, proxy a request to Kubecost through the Kubernetes API server.")
	cmd.Flags().StringVar(&options.AllocationPath, "allocation-path", "/model/allocation", "URL path at which Allocation queries can be served from the configured service. If using OpenCost, you may want to set this to '/allocation/compute'")
	cmd.Flags().StringVar(&options.PredictResourceCostPath, "predict-resource-cost-path", "/model/prediction/resourcecost", "URL path at which Resource Cost Prediction queries can be served from the configured service.")

	//Check if environment variable KUBECTL_COST_USE_PROXY is set, it defaults to false
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	bindAFlagToViperEnv(cmd, v, "use-proxy")
}

// addKubeOptionsFlags sets up the cobra command with the flags from
// KubeOptions' configFlags so that a kube client can be built to a
// user's specification. Its one modification is to change the name
// of the namespace flag to kubecost-namespace because we want to
// "behave as expected", i.e. --namespace affects the request to the
// kubecost API, not the request to the k8s API.
func addKubeOptionsFlags(cmd *cobra.Command, ko *KubeOptions) {
	// By setting Namespace to nil, AddFlags won't create
	// the --namespace flag, which we want to use for scoping
	// kubecost requests (for some subcommands). We can then
	// create a differently-named flag for the same variable.
	ko.configFlags.Namespace = nil
	ko.configFlags.AddFlags(cmd.Flags())

	// Reset Namespace to a valid string to avoid a nil pointer
	// deref.
	// emptyStr := ""
	// ko.configFlags.Namespace = &emptyStr
}

func (co *CostOptions) Complete(restConfig *rest.Config) error {
	if co.showAll {
		co.showCPUCost = true
		co.showMemoryCost = true
		co.showGPUCost = true
		co.showPVCost = true
		co.showNetworkCost = true
		co.showSharedCost = true
		co.showLoadBalancerCost = true
		co.showAssetType = true
	}
	if err := co.QueryBackendOptions.Complete(restConfig); err != nil {
		return fmt.Errorf("complete backend opts: %s", err)
	}
	return nil
}

func (co *CostOptions) Validate() error {
	// make sure window parses client-side, may not be necessary but allows
	// for a nicer error message for the user
	if _, err := kubecost.ParseWindowWithOffset(co.window, 0); err != nil {
		return fmt.Errorf("failed to parse window: %s", err)
	}

	if err := co.QueryBackendOptions.Validate(); err != nil {
		return fmt.Errorf("validating query options: %s", err)
	}

	return nil
}

// Binds the flag with viper environment variable and ensures the order of precendence
// command line > environment variable > default value
func bindAFlagToViperEnv(cmd *cobra.Command, v *viper.Viper, flag string) {
	flagPtr := cmd.Flags().Lookup(flag)
	envVarSuffix := strings.ToUpper(strings.ReplaceAll(flagPtr.Name, "-", "_"))
	v.BindEnv(flagPtr.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
	if !flagPtr.Changed && v.IsSet(flagPtr.Name) {
		val := v.Get(flagPtr.Name)
		cmd.Flags().Set(flagPtr.Name, fmt.Sprintf("%v", val))
	}
}

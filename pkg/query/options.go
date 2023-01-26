package query

import (
	"context"
	"fmt"
	"time"

	"github.com/opencost/opencost/pkg/log"

	"k8s.io/client-go/rest"
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

	// A path which can serve Resource Cost Prediction queries,
	// e.g. "/prediction/resourcecost"
	PredictResourceCostPath string

	restConfig *rest.Config
	pfQuerier  *PortForwardQuerier
}

func (o *QueryBackendOptions) Complete(restConfig *rest.Config) error {
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

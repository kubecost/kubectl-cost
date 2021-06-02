package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

// CostOptions holds common options for querying and displaying
// data from the kubecost API
type CostOptions struct {
	// If set, will proxy a request through the K8s API server
	// instead of port forwarding.
	useProxy bool

	window string

	isHistorical bool
	showAll      bool

	// The name of the cost-analyzer service in the cluster,
	// in case user is running a non-standard name (like the
	// staging helm chart). Combines with
	// commonOptions.configFlags.Namespace to direct the API
	// request.
	serviceName string

	displayOptions
}

type displayOptions struct {
	showCPUCost          bool
	showMemoryCost       bool
	showGPUCost          bool
	showPVCost           bool
	showNetworkCost      bool
	showEfficiency       bool
	showSharedCost       bool
	showLoadBalancerCost bool
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
	cmd.Flags().BoolVar(&options.showLoadBalancerCost, "show-loadbalancer", false, "show load balancer cost data")
	cmd.Flags().BoolVar(&options.showEfficiency, "show-efficiency", true, "show efficiency of cost alongside CPU and memory cost")
	cmd.Flags().BoolVarP(&options.showAll, "show-all-resources", "A", false, "Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network --show-efficiency.")
	cmd.Flags().StringVar(&options.serviceName, "service-name", "kubecost-cost-analyzer", "The name of the kubecost cost analyzer service. Change if you're running a non-standard deployment, like the staging helm chart.")
	cmd.Flags().BoolVar(&options.useProxy, "use-proxy", false, "Instead of temporarily port-forwarding, proxy a request to Kubecost through the Kubernetes API server.")
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
	emptyStr := ""
	ko.configFlags.Namespace = &emptyStr

	cmd.Flags().StringVarP(ko.configFlags.Namespace, "kubecost-namespace", "N", "kubecost", "The namespace that kubecost is deployed in. Requests to the API will be directed to this namespace.")
}

func (co *CostOptions) Complete() {
	if co.showAll {
		co.showCPUCost = true
		co.showMemoryCost = true
		co.showGPUCost = true
		co.showPVCost = true
		co.showNetworkCost = true
		co.showSharedCost = true
		co.showLoadBalancerCost = true
	}
}

func (co *CostOptions) Validate() error {
	// make sure window parses client-side, may not be necessary but allows
	// for a nicer error message for the user
	if _, err := kubecost.ParseWindowWithOffset(co.window, 0); err != nil {
		return fmt.Errorf("failed to parse window: %s", err)
	}

	return nil
}

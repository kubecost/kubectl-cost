package cmd

import "github.com/spf13/cobra"

// CostOptions holds common options for querying and displaying
// data from the kubecost API
type CostOptions struct {
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
	showCPUCost     bool
	showMemoryCost  bool
	showGPUCost     bool
	showPVCost      bool
	showNetworkCost bool
	showEfficiency  bool
}

func addCostOptionsFlags(cmd *cobra.Command, options *CostOptions) {
	cmd.Flags().StringVar(&options.window, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&options.isHistorical, "historical", false, "show the total cost during the window instead of the projected monthly rate based on the data in the window")
	cmd.Flags().BoolVar(&options.showCPUCost, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&options.showMemoryCost, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&options.showGPUCost, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&options.showPVCost, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&options.showNetworkCost, "show-network", false, "show data for network cost")
	cmd.Flags().BoolVar(&options.showEfficiency, "show-efficiency", false, "Show efficiency of cost alongside CPU and memory cost. Only works with --historical.")
	cmd.Flags().BoolVarP(&options.showAll, "show-all-resources", "A", false, "Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network.")
	cmd.Flags().StringVar(&options.serviceName, "service-name", "kubecost-cost-analyzer", "The name of the kubecost cost analyzer service. Change if you're running a non-standard deployment, like the staging helm chart.")
}

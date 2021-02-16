package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsController contains the standard CostOptions and any
// options specific to controller queries.
type CostOptionsController struct {
	CostOptions
}

func newCmdCostController(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	controllerO := &CostOptionsController{}

	cmd := &cobra.Command{
		Use:   "controller",
		Short: "view cost information aggregated by controller",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			controllerO.Complete()

			return runCostController(kubeO, controllerO)
		},
	}

	addCostOptionsFlags(cmd, &controllerO.CostOptions)
	kubeO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (no *CostOptionsController) Complete() {
	if no.showAll {
		no.showCPUCost = true
		no.showMemoryCost = true
		no.showGPUCost = true
		no.showPVCost = true
		no.showNetworkCost = true
	}
}

func runCostController(ko *KubeOptions, no *CostOptionsController) error {

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "controller")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		// don't show unallocated controller data
		delete(aggs, "__unallocated__")

		err = writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"namespace", "controller"},
			controllerTitleExtractor,
			no.displayOptions,
		)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		allocations, err := query.QueryAllocation(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "controller")
		if err != nil {
			return fmt.Errorf("failed to query allocation API: %s", err)
		}

		// Use Data[0] because the query accumulates
		err = writeNamespaceTable(ko.Out, allocations[0], no.displayOptions)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}

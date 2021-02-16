package cmd

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsNamespace contains the standard CostOptions and any
// options specific to namespace queries.
type CostOptionsNamespace struct {
	CostOptions
}

func newCmdCostNamespace(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	namespaceO := &CostOptionsNamespace{}

	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "view cost information aggregated by namespace",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			namespaceO.CostOptions.Complete()

			if err := namespaceO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostNamespace(kubeO, namespaceO)
		},
	}

	addCostOptionsFlags(cmd, &namespaceO.CostOptions)
	kubeO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostNamespace(ko *KubeOptions, no *CostOptionsNamespace) error {

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace", "")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"namespace"},
			noopTitleExtractor,
			no.displayOptions,
		)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		allocations, err := query.QueryAllocation(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query allocation API: %s", err)
		}

		// Use allocations[0] because the query accumulates to a single result
		err = writeAllocationTable(ko.Out, "Namespace", allocations[0], no.displayOptions)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}

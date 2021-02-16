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
	filterNamespace string

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

			controllerO.CostOptions.Complete()

			if err := controllerO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostController(kubeO, controllerO)
		},
	}

	cmd.Flags().StringVarP(&controllerO.filterNamespace, "namespace-filter", "N", "", "Limit results to only one namespace. Defaults to all namespaces.")
	addCostOptionsFlags(cmd, &controllerO.CostOptions)
	kubeO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostController(ko *KubeOptions, no *CostOptionsController) error {

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "controller")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		// don't show unallocated controller data
		delete(aggs, "__unallocated__")

		applyNamespaceFilter(aggs, no.filterNamespace)

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
		// Not supported because the allocation API does not return the namespace
		// of controllers.
		return fmt.Errorf("kubectl cost controller does not yet support historical queries")
	}

	return nil
}

package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
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

	cmd.Flags().StringVarP(&controllerO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")
	addCostOptionsFlags(cmd, &controllerO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostController(ko *KubeOptions, no *CostOptionsController) error {

	currencyCode, err := query.QueryCurrencyCode(ko.clientset, *ko.configFlags.Namespace, no.serviceName, context.Background())
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	if !no.isHistorical {
		var aggs map[string]query.Aggregation
		var err error

		if no.useProxy {
			aggs, err = query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "controller", "", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query agg cost model: %s", err)
			}
		} else {
			aggs, err = query.QueryAggCostModelFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "controller", "", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query agg cost model: %s", err)
			}
		}

		// don't show unallocated controller data
		delete(aggs, "__unallocated__")

		applyNamespaceFilter(aggs, no.filterNamespace)

		writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"namespace", "controller"},
			controllerTitleExtractor,
			no.displayOptions,
			currencyCode,
		)
	} else {
		var allocations []map[string]kubecost.Allocation
		var err error
		if no.useProxy {
			allocations, err = query.QueryAllocation(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "controller", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		} else {
			allocations, err = query.QueryAllocationFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "controller", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		}

		// Use allocations[0] because the query accumulates to a single result
		applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

		writeAllocationTable(ko.Out, "Controller", allocations[0], no.displayOptions, currencyCode, true)
	}

	return nil
}

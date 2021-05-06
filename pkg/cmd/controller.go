package cmd

import (
	"context"
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

	cmd.Flags().StringVarP(&controllerO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")
	addCostOptionsFlags(cmd, &controllerO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostController(ko *KubeOptions, no *CostOptionsController) error {

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		RestConfig:        ko.restConfig,
		Ctx:               context.Background(),
		KubecostNamespace: *ko.configFlags.Namespace,
		ServiceName:       no.serviceName,
		UseProxy:          no.useProxy,
	})
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	if !no.isHistorical {
		aggs, err := query.QueryAggCostModel(query.AggCostModelParameters{
			RestConfig:        ko.restConfig,
			Ctx:               context.Background(),
			KubecostNamespace: *ko.configFlags.Namespace,
			ServiceName:       no.serviceName,
			Window:            no.window,
			Aggregate:         "controller",
			UseProxy:          no.useProxy,
		})
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
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
		allocations, err := query.QueryAllocation(query.AllocationParameters{
			RestConfig:        ko.restConfig,
			Ctx:               context.Background(),
			KubecostNamespace: *ko.configFlags.Namespace,
			ServiceName:       no.serviceName,
			Window:            no.window,
			Aggregate:         "controller",
			UseProxy:          no.useProxy,
		})
		if err != nil {
			return fmt.Errorf("failed to query allocation API: %s", err)
		}

		// Use allocations[0] because the query accumulates to a single result
		applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

		writeAllocationTable(ko.Out, "Controller", allocations[0], no.displayOptions, currencyCode, true)
	}

	return nil
}

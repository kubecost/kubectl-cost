package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsPod contains the standard CostOptions and any
// options specific to pod queries.
type CostOptionsPod struct {
	filterNamespace string

	CostOptions
}

func newCmdCostPod(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	podO := &CostOptionsPod{}

	cmd := &cobra.Command{
		Use:   "pod",
		Short: "view cost information aggregated by pod",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			podO.CostOptions.Complete()

			if err := podO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostPod(kubeO, podO)
		},
	}

	cmd.Flags().StringVarP(&podO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")
	addCostOptionsFlags(cmd, &podO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostPod(ko *KubeOptions, no *CostOptionsPod) error {

	currencyCode, err := query.QueryCurrencyCode(ko.clientset, *ko.configFlags.Namespace, no.serviceName, context.Background())
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	if !no.isHistorical {
		var aggs map[string]query.Aggregation
		var err error

		if no.useProxy {
			aggs, err = query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "pod", "", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query agg cost model: %s", err)
			}
		} else {
			aggs, err = query.QueryAggCostModelFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "pod", "", context.Background())
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
			[]string{"namespace", "pod"},
			podTitleExtractor,
			no.displayOptions,
			currencyCode,
		)
	} else {
		var allocations []map[string]kubecost.Allocation
		var err error
		if no.useProxy {
			allocations, err = query.QueryAllocation(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "pod", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		} else {
			allocations, err = query.QueryAllocationFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "pod", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		}

		// Use allocations[0] because the query accumulates to a single result
		applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

		writeAllocationTable(ko.Out, "Pod", allocations[0], no.displayOptions, currencyCode, true)
	}

	return nil
}

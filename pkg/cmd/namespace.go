package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
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
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostNamespace(ko *KubeOptions, no *CostOptionsNamespace) error {

	currencyCode, err := query.QueryCurrencyCode(ko.clientset, *ko.configFlags.Namespace, no.serviceName, context.Background())
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	if !no.isHistorical {
		var aggs map[string]query.Aggregation
		var err error

		if no.useProxy {
			aggs, err = query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace", "", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query agg cost model: %s", err)
			}
		} else {
			aggs, err = query.QueryAggCostModelFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace", "", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query agg cost model: %s", err)
			}
		}

		writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"namespace"},
			noopTitleExtractor,
			no.displayOptions,
			currencyCode,
		)
	} else {
		var allocations []map[string]kubecost.Allocation
		var err error
		if no.useProxy {
			allocations, err = query.QueryAllocation(ko.clientset, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		} else {
			allocations, err = query.QueryAllocationFwd(ko.restConfig, *ko.configFlags.Namespace, no.serviceName, no.window, "namespace", context.Background())
			if err != nil {
				return fmt.Errorf("failed to query allocation API: %s", err)
			}
		}

		// Use allocations[0] because the query accumulates to a single result
		writeAllocationTable(ko.Out, "Namespace", allocations[0], no.displayOptions, currencyCode, false)
	}

	return nil
}

package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsLabel contains the standard CostOptions and any
// options specific to label queries.
type CostOptionsLabel struct {
	// The label to perform the aggregation on, "app" is a common one
	queryLabel string

	CostOptions
}

func newCmdCostLabel(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	labelO := &CostOptionsLabel{}

	cmd := &cobra.Command{
		Use:   "label",
		Short: "view cost information aggregated by label",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			labelO.CostOptions.Complete()

			if err := labelO.CostOptions.Validate(); err != nil {
				return err
			}

			if err := labelO.Validate(); err != nil {
				return err
			}

			return runCostLabel(kubeO, labelO)
		},
	}

	cmd.Flags().StringVarP(&labelO.queryLabel, "label", "l", "", "The label to perform aggregation on, \"app\" is a common one.")
	cmd.MarkFlagRequired("label")

	addCostOptionsFlags(cmd, &labelO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostLabel(ko *KubeOptions, no *CostOptionsLabel) error {

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
			RestConfig:          ko.restConfig,
			Ctx:                 context.Background(),
			KubecostNamespace:   *ko.configFlags.Namespace,
			ServiceName:         no.serviceName,
			Window:              no.window,
			Aggregate:           "label",
			AggregationSubfield: no.queryLabel,
			UseProxy:            no.useProxy,
		})
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		// don't show unallocated controller data
		delete(aggs, "__unallocated__")

		writeAggregationRateTable(
			ko.Out,
			aggs,
			[]string{"label"},
			noopTitleExtractor,
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
			Aggregate:         fmt.Sprintf("label:%s", no.queryLabel),
			UseProxy:          no.useProxy,
		})
		if err != nil {
			return fmt.Errorf("failed to query allocation API: %s", err)
		}

		// Use allocations[0] because the query accumulates to a single result
		writeAllocationTable(ko.Out, "Label", allocations[0], no.displayOptions, currencyCode, false)
	}

	return nil
}

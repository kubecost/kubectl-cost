package cmd

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/cmd/display"
	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/log"
)

// CostOptionsLabel contains the standard CostOptions and any
// options specific to label queries.
type CostOptionsLabel struct {
	// The label to perform the aggregation on, "app" is a common one
	queryLabel string

	CostOptions
	display.AllocationDisplayOptions
}

func newCmdCostLabel(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
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

			if err := labelO.CostOptions.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("completing options: %s", err)
			}
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
	display.AddAllocationDisplayOptionsFlags(cmd, &labelO.AllocationDisplayOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostLabel(ko *utilities.KubeOptions, no *CostOptionsLabel) error {

	aggregation := []string{"cluster", fmt.Sprintf("label:%s", no.queryLabel)}

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		Ctx: context.Background(),
		QueryParams: map[string]string{
			"window":      no.window,
			"aggregate":   strings.Join(aggregation, ","),
			"accumulate":  "true",
			"includeIdle": fmt.Sprintf("%t", no.includeIdle),
			"idle":        fmt.Sprintf("%t", no.includeIdle),
		},
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use allocations[0] because the query accumulates to a single result
	display.WriteAllocationTable(ko.Out, aggregation, allocations[0], no.AllocationDisplayOptions, currencyCode, !no.isHistorical)

	return nil
}

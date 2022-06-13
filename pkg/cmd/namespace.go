package cmd

import (
	"context"
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
		Aliases: []string{"ns"},
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

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		RestConfig:          ko.restConfig,
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		RestConfig:          ko.restConfig,
		Ctx:                 context.Background(),
		Window:              no.window,
		Aggregate:           "namespace",
		Accumulate:          "true",
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use allocations[0] because the query accumulates to a single result
	writeAllocationTable(ko.Out, "Namespace", allocations[0], no.displayOptions, currencyCode, false, !no.isHistorical)

	return nil
}

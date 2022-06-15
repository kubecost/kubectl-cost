package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/kubecost/opencost/pkg/kubecost"
)

// CostOptionsDeployment contains the standard CostOptions and any
// options specific to deployment queries.
type CostOptionsDeployment struct {
	filterNamespace string

	CostOptions
}

func newCmdCostDeployment(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	deploymentO := &CostOptionsDeployment{}

	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "view cost information aggregated by deployment",
		Aliases: []string{"deploy"},
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			deploymentO.CostOptions.Complete()

			if err := deploymentO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostDeployment(kubeO, deploymentO)
		},
	}

	cmd.Flags().StringVarP(&deploymentO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")

	addCostOptionsFlags(cmd, &deploymentO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostDeployment(ko *KubeOptions, no *CostOptionsDeployment) error {

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
		Aggregate:           "deployment",
		Accumulate:          "true",
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use allocations[0] because the query accumulates to a single result
	applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

	writeAllocationTable(ko.Out, "Deployment", allocations[0], no.displayOptions, currencyCode, true, !no.isHistorical)

	return nil
}

func applyNamespaceFilterAllocation(allocData map[string]kubecost.Allocation, namespaceFilter string) {
	if namespaceFilter == "" {
		return
	}

	for allocName, alloc := range allocData {
		ns := alloc.Properties.Namespace
		if ns != namespaceFilter {
			delete(allocData, allocName)
		}
	}
}

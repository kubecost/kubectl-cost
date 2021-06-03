package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

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

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		RestConfig:        ko.restConfig,
		Ctx:               context.Background(),
		KubecostNamespace: *ko.configFlags.Namespace,
		ServiceName:       no.serviceName,
		Window:            no.window,
		Aggregate:         "pod",
		UseProxy:          no.useProxy,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use allocations[0] because the query accumulates to a single result
	applyNamespaceFilterAllocation(allocations[0], no.filterNamespace)

	writeAllocationTable(ko.Out, "Pod", allocations[0], no.displayOptions, currencyCode, true, !no.isHistorical)

	return nil
}

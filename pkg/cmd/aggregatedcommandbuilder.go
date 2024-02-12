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

type AggregatedAllocationOptions struct {
	CostOptions
	display.AllocationDisplayOptions
}

func buildStandardAggregatedAllocationCommand(streams genericclioptions.IOStreams, commandName string, commandAliases []string, aggregation []string, enableNamespaceFilter bool) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	o := AggregatedAllocationOptions{}

	cmd := &cobra.Command{
		Use:     commandName,
		Short:   fmt.Sprintf("view cost information aggregated by %s", aggregation),
		Aliases: commandAliases,
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			if err := o.CostOptions.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("completing options: %s", err)
			}
			if err := o.CostOptions.Validate(); err != nil {
				return err
			}

			return runAggregatedAllocationCommand(kubeO, o, aggregation)
		},
	}

	// TODO: Replace entirely when we have generic filter language (v2)
	if enableNamespaceFilter {
		cmd.Flags().StringVarP(&o.CostOptions.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")
	}

	addCostOptionsFlags(cmd, &o.CostOptions)
	display.AddAllocationDisplayOptionsFlags(cmd, &o.AllocationDisplayOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runAggregatedAllocationCommand(ko *utilities.KubeOptions, o AggregatedAllocationOptions, aggregation []string) error {

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: o.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		Ctx: context.Background(),
		QueryParams: map[string]string{
			"window":           o.window,
			"aggregate":        strings.Join(aggregation, ","),
			"accumulate":       "true",
			"includeIdle":      "true",
			"filterNamespaces": o.filterNamespace,
		},
		QueryBackendOptions: o.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	display.WriteAllocationTable(ko.Out, aggregation, allocations[0], o.AllocationDisplayOptions, currencyCode, !o.isHistorical)

	return nil
}

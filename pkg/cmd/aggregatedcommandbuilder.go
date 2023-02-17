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

func buildStandardAggregatedAllocationCommand(streams genericclioptions.IOStreams, commandName string, commandAliases []string, aggregation []string, enableNamespaceFilter bool) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	costO := CostOptions{}

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

			if err := costO.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("completing options: %s", err)
			}
			if err := costO.Validate(); err != nil {
				return err
			}

			return runAggregatedAllocationCommand(kubeO, costO, aggregation)
		},
	}

	// TODO: Replace entirely when we have generic filter language (v2)
	if enableNamespaceFilter {
		cmd.Flags().StringVarP(&costO.filterNamespace, "namespace", "n", "", "Limit results to only one namespace. Defaults to all namespaces.")
	}

	addCostOptionsFlags(cmd, &costO)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runAggregatedAllocationCommand(ko *utilities.KubeOptions, co CostOptions, aggregation []string) error {

	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: co.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	allocations, err := query.QueryAllocation(query.AllocationParameters{
		Ctx: context.Background(),
		QueryParams: map[string]string{
			"window":           co.window,
			"aggregate":        strings.Join(aggregation, ","),
			"accumulate":       "true",
			"filterNamespaces": co.filterNamespace,
		},
		QueryBackendOptions: co.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	display.WriteAllocationTable(ko.Out, aggregation, allocations[0], co.displayOptions, currencyCode, !co.isHistorical)

	return nil
}

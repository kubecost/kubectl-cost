package cmd

import (
	"context"
	"fmt"

	"github.com/kubecost/kubectl-cost/pkg/cmd/display"
	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"

	"github.com/opencost/opencost/core/pkg/log"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// SavingsOptions contains options specific to savings queries.
type SavingsOptions struct {
	window string

	query.QueryBackendOptions
}

func newCmdCostSavings(
	streams genericclioptions.IOStreams,
) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	savingsO := &SavingsOptions{}

	cmd := &cobra.Command{
		Use:   "savings",
		Short: "Show container request sizing recommendations and estimated monthly savings from right-sizing.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return fmt.Errorf("complete k8s options: %s", err)
			}
			if err := kubeO.Validate(); err != nil {
				return fmt.Errorf("validate k8s options: %s", err)
			}

			if err := savingsO.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("complete: %s", err)
			}
			if err := savingsO.Validate(); err != nil {
				return fmt.Errorf("validate: %s", err)
			}

			return runCostSavings(kubeO, savingsO)
		},
	}
	cmd.Flags().StringVarP(&savingsO.window, "window", "w", "2d", "The window of data to use for the savings recommendation. See https://github.com/kubecost/docs/blob/master/allocation.md#querying for a detailed explanation of what can be passed here.")

	query.AddQueryBackendOptionsFlags(cmd, &savingsO.QueryBackendOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	cmd.SilenceUsage = true

	return cmd
}

func (savingsO *SavingsOptions) Validate() error {
	if err := savingsO.QueryBackendOptions.Validate(); err != nil {
		return fmt.Errorf("validating query options: %s", err)
	}

	return nil
}

func (savingsO *SavingsOptions) Complete(restConfig *rest.Config) error {
	if err := savingsO.QueryBackendOptions.Complete(restConfig); err != nil {
		return fmt.Errorf("complete backend opts: %s", err)
	}
	return nil
}

func runCostSavings(ko *utilities.KubeOptions, so *SavingsOptions) error {
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: so.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	recs, err := query.QuerySavings(query.SavingsParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: so.QueryBackendOptions,
		QueryParams: map[string]string{
			"window": so.window,
		},
	})
	if err != nil {
		return fmt.Errorf("querying savings API: %s", err)
	}

	display.WriteSavingsTable(ko.Out, recs, currencyCode)
	return nil
}

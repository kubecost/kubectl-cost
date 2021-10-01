package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

// CostOptionsNetwork contains the standard CostOptions and any
// options specific to network queries.
type CostOptionsNetwork struct {
	CostOptions
}

func newCmdCostNetwork(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	assetsO := &CostOptionsNetwork{}

	cmd := &cobra.Command{
		Use:   "network",
		Short: "view cost information by network",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			assetsO.CostOptions.Complete()

			if err := assetsO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostNetwork(kubeO, assetsO)
		},
	}

	addCostOptionsFlags(cmd, &assetsO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostNetwork(ko *KubeOptions, no *CostOptionsNetwork) error {

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

	assets, err := query.QueryAssets(query.AssetParameters{
		RestConfig:        ko.restConfig,
		Ctx:               context.Background(),
		KubecostNamespace: *ko.configFlags.Namespace,
		ServiceName:       no.serviceName,
		Window:            no.window,
		Accumulate:        "true",
		UseProxy:          no.useProxy,
		FilterTypes:       "Network",
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use assets[0] because the query accumulates to a single result
	writeAssetTable(ko.Out, "Network", assets[0], no.displayOptions, currencyCode, !no.isHistorical)

	return nil
}

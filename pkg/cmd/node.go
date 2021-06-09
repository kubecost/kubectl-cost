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
type CostOptionsNode struct {
	CostOptions
}

func newCmdCostNode(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	assetsO := &CostOptionsNode{}

	cmd := &cobra.Command{
		Use:   "node",
		Short: "view cost information by nodes",
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

			return runCostNode(kubeO, assetsO)
		},
	}

	addCostOptionsFlags(cmd, &assetsO.CostOptions)
	addKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostNode(ko *KubeOptions, no *CostOptionsNode) error {

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
		RestConfig:         ko.restConfig,
		Ctx:                context.Background(),
		KubecostNamespace:  *ko.configFlags.Namespace,
		ServiceName:        no.serviceName,
		Window:             no.window,
		Aggregate:          "",
		DisableAdjustments: false,
		Accumulate:         false,
		UseProxy:           no.useProxy,
		FilterTypes:        "Node",
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	fmt.Println(currencyCode)
	fmt.Println(assets)
	atype := fmt.Sprintf("%T", assets)
	fmt.Println(atype)

	// Use allocations[0] because the query accumulates to a single result
	writeAssetTable(ko.Out, "Node", assets[0], no.displayOptions, currencyCode, !no.isHistorical)

	return nil
}

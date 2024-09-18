package cmd

import (
	"context"
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/cobra"

	"github.com/kubecost/kubectl-cost/pkg/cmd/display"
	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/core/pkg/log"
)

// CostOptionsNode contains the standard CostOptions and any
// options specific to node queries.
type CostOptionsNode struct {
	CostOptions
	display.AssetDisplayOptions
}

func newCmdCostNode(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
	assetsO := &CostOptionsNode{}

	cmd := &cobra.Command{
		Use:     "node",
		Short:   "view cost information by nodes",
		Aliases: []string{"no"},
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			if err := assetsO.CostOptions.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("completing options: %s", err)
			}

			if err := assetsO.CostOptions.Validate(); err != nil {
				return err
			}

			return runCostNode(kubeO, assetsO)
		},
	}

	addCostOptionsFlags(cmd, &assetsO.CostOptions)
	display.AddAssetDisplayOptionsFlags(cmd, &assetsO.AssetDisplayOptions)
	utilities.AddKubeOptionsFlags(cmd, kubeO)

	return cmd
}

func runCostNode(ko *utilities.KubeOptions, no *CostOptionsNode) error {
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 context.Background(),
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	assets, err := query.QueryAssets(query.AssetParameters{
		Ctx:                 context.Background(),
		Window:              no.window,
		Accumulate:          "true",
		FilterTypes:         "Node",
		QueryBackendOptions: no.QueryBackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to query allocation API: %s", err)
	}

	// Use assets[0] because the query accumulates to a single result
	display.WriteAssetTable(ko.Out, "Node", assets[0], no.AssetDisplayOptions, currencyCode, !no.isHistorical)

	return nil
}

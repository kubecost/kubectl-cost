package cmd

import (
	"fmt"
	"io"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

type CostOptionsNamespace struct {
	isRate bool
}

func newCmdCostNamespace(streams genericclioptions.IOStreams) *cobra.Command {
	commonO := NewCommonCostOptions(streams)
	namespaceO := &CostOptionsNamespace{}

	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "view cost information aggregated by namespace",
		RunE: func(c *cobra.Command, args []string) error {
			if err := commonO.Complete(c, args); err != nil {
				return err
			}
			if err := commonO.Validate(); err != nil {
				return err
			}

			return runCostNamespace(commonO, namespaceO)
		},
	}

	cmd.Flags().StringVar(&commonO.costWindow, "window", "yesterday", "the window of data to query")
	cmd.Flags().BoolVar(&namespaceO.isRate, "rate", false, "show the projected monthly rate based on data in the window instead of the total cost during the window")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostNamespace(co *CostOptionsCommon, no *CostOptionsNamespace) error {

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if no.isRate {
		// aggCMResp, err :=
	} else {
		allocR, err := queryAllocation(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query allocation API")
		}

		// Use Data[0] because the query accumulates
		err = writeNamespaceTable(co.Out, allocR.Data[0])
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}

func writeNamespaceTable(out io.Writer, allocations map[string]kubecost.Allocation) error {

	t := table.NewWriter()
	t.SetOutputMirror(out)

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:      "Cluster",
			AutoMerge: true,
		},
		{
			Name:      "Namespace",
			AutoMerge: true,
		},
		{
			Name:        "Total Cost",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
	})

	t.AppendHeader(table.Row{"Cluster", "Namespace", "Total Cost"})
	t.SortBy([]table.SortBy{
		{
			Name: "Cluster",
			Mode: table.Dsc,
		},
		{
			Name: "Total Cost",
			Mode: table.Dsc,
		},
	})

	var summedCost float64

	for _, alloc := range allocations {
		cluster, _ := alloc.Properties.GetCluster()
		namespace := alloc.Name
		totalCost := fmt.Sprintf("%.6f", alloc.TotalCost)

		t.AppendRow(table.Row{
			cluster, namespace, totalCost,
		})
		summedCost += alloc.TotalCost
	}
	t.AppendFooter(table.Row{"SUMMED", "", fmt.Sprintf("%.6f", summedCost)})
	t.Render()

	return nil
}

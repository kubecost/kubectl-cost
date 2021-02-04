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
	isRate      bool
	showCPU     bool
	showMemory  bool
	showGPU     bool
	showPV      bool
	showNetwork bool
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
	cmd.Flags().BoolVar(&namespaceO.showCPU, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&namespaceO.showMemory, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&namespaceO.showGPU, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&namespaceO.showPV, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&namespaceO.showNetwork, "show-network", false, "show data for network cost")
	commonO.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func runCostNamespace(co *CostOptionsCommon, no *CostOptionsNamespace) error {

	do := displayOptions{
		showCPUCost:     no.showCPU,
		showMemoryCost:  no.showMemory,
		showGPUCost:     no.showGPU,
		showPVCost:      no.showPV,
		showNetworkCost: no.showNetwork,
	}

	clientset, err := kubernetes.NewForConfig(co.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %s", err)
	}

	if no.isRate {
		aggCMResp, err := queryAggCostModel(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query agg cost model: %s", err)
		}

		err = writeAggregationRateTable(co.Out, aggCMResp.Data, "namespace", do)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	} else {
		allocR, err := queryAllocation(clientset, co.costWindow, "namespace")
		if err != nil {
			return fmt.Errorf("failed to query allocation API")
		}

		// Use Data[0] because the query accumulates
		err = writeNamespaceTable(co.Out, allocR.Data[0], do)
		if err != nil {
			return fmt.Errorf("failed to write table output: %s", err)
		}
	}

	return nil
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func writeNamespaceTable(out io.Writer, allocations map[string]kubecost.Allocation, opts displayOptions) error {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	columnConfigs := []table.ColumnConfig{}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      "cluster",
		AutoMerge: true,
	})
	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      "namespace",
		AutoMerge: true,
	})

	if opts.showCPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "CPU",
		})
	}

	if opts.showMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "Memory",
		})
	}

	if opts.showGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "GPU",
		})
	}

	if opts.showPVCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "PV",
		})
	}

	if opts.showNetworkCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "Network",
		})
	}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:        "Total Cost (All)",
		Align:       text.AlignRight,
		AlignFooter: text.AlignRight,
	})

	t.SetColumnConfigs(columnConfigs)

	headerRow := table.Row{}

	headerRow = append(headerRow, "Cluster")
	headerRow = append(headerRow, "Namespace")

	if opts.showCPUCost {
		headerRow = append(headerRow, "CPU")
	}

	if opts.showMemoryCost {
		headerRow = append(headerRow, "Memory")
	}

	if opts.showGPUCost {
		headerRow = append(headerRow, "GPU")
	}

	if opts.showPVCost {
		headerRow = append(headerRow, "PV")
	}

	if opts.showNetworkCost {
		headerRow = append(headerRow, "Network")
	}

	headerRow = append(headerRow, "Total Cost (All)")

	t.AppendHeader(headerRow)
	t.SortBy([]table.SortBy{
		{
			Name: "Total Cost (All)",
			Mode: table.Dsc,
		},
	})

	var summedCost float64
	var summedCPU float64
	var summedMemory float64
	var summedGPU float64
	var summedPV float64
	var summedNetwork float64

	for _, alloc := range allocations {
		cluster, _ := alloc.Properties.GetCluster()
		namespace := alloc.Name

		allocRow := table.Row{}

		allocRow = append(allocRow, cluster)
		allocRow = append(allocRow, namespace)

		if opts.showCPUCost {
			allocRow = append(allocRow, formatFloat(alloc.CPUCost))
			summedCPU += alloc.CPUCost
		}

		if opts.showMemoryCost {
			allocRow = append(allocRow, formatFloat(alloc.RAMCost))
			summedMemory += alloc.RAMCost
		}

		if opts.showGPUCost {
			allocRow = append(allocRow, formatFloat(alloc.GPUCost))
			summedGPU += alloc.GPUCost
		}

		if opts.showPVCost {
			allocRow = append(allocRow, formatFloat(alloc.PVCost))
			summedPV += alloc.PVCost
		}

		if opts.showNetworkCost {
			allocRow = append(allocRow, formatFloat(alloc.NetworkCost))
			summedNetwork += alloc.NetworkCost
		}

		cumulativeCost := formatFloat(alloc.TotalCost)
		allocRow = append(allocRow, cumulativeCost)

		t.AppendRow(allocRow)
		summedCost += alloc.TotalCost
	}

	footerRow := table.Row{}

	footerRow = append(footerRow, "SUMMED")
	footerRow = append(footerRow, "")

	if opts.showCPUCost {
		footerRow = append(footerRow, formatFloat(summedCPU))
	}

	if opts.showMemoryCost {
		footerRow = append(footerRow, formatFloat(summedMemory))
	}

	if opts.showGPUCost {
		footerRow = append(footerRow, formatFloat(summedGPU))
	}

	if opts.showPVCost {
		footerRow = append(footerRow, formatFloat(summedPV))
	}

	if opts.showNetworkCost {
		footerRow = append(footerRow, formatFloat(summedNetwork))
	}

	footerRow = append(footerRow, formatFloat(summedCost))

	t.AppendFooter(footerRow)
	t.Render()

	return nil
}

type displayOptions struct {
	showCPUCost     bool
	showMemoryCost  bool
	showGPUCost     bool
	showPVCost      bool
	showNetworkCost bool
}

func writeAggregationRateTable(out io.Writer, aggs map[string]aggregation, aggregatedBy string, opts displayOptions) error {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	columnConfigs := []table.ColumnConfig{}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      aggregatedBy,
		AutoMerge: true,
	})

	if opts.showCPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "CPU",
		})
	}

	if opts.showMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "Memory",
		})
	}

	if opts.showGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "GPU",
		})
	}

	if opts.showPVCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "PV",
		})
	}

	if opts.showNetworkCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: "Network",
		})
	}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:        "Monthly Cost (All)",
		Align:       text.AlignRight,
		AlignFooter: text.AlignRight,
	})

	t.SetColumnConfigs(columnConfigs)

	// t.SetColumnConfigs([]table.ColumnConfig{
	// 	{
	// 		Name:      "Cluster",
	// 		AutoMerge: true,
	// 	},
	// 	{
	// 		Name:      "Namespace",
	// 		AutoMerge: true,
	// 	},
	// 	{
	// 		Name:        "Monthly Cost",
	// 		Align:       text.AlignRight,
	// 		AlignFooter: text.AlignRight,
	// 	},
	// })

	headerRow := table.Row{}

	headerRow = append(headerRow, aggregatedBy)

	if opts.showCPUCost {
		headerRow = append(headerRow, "CPU")
	}

	if opts.showMemoryCost {
		headerRow = append(headerRow, "Memory")
	}

	if opts.showGPUCost {
		headerRow = append(headerRow, "GPU")
	}

	if opts.showPVCost {
		headerRow = append(headerRow, "PV")
	}

	if opts.showNetworkCost {
		headerRow = append(headerRow, "Network")
	}

	headerRow = append(headerRow, "Monthly Cost (All)")

	t.AppendHeader(headerRow)
	t.SortBy([]table.SortBy{
		{
			Name: "Monthly Cost (All)",
			Mode: table.Dsc,
		},
	})

	var summedCost float64
	var summedCPU float64
	var summedMemory float64
	var summedGPU float64
	var summedPV float64
	var summedNetwork float64

	for agBy, agg := range aggs {

		agRow := table.Row{}

		agRow = append(agRow, agBy)

		if opts.showCPUCost {
			agRow = append(agRow, formatFloat(agg.CPUCost))
			summedCPU += agg.CPUCost
		}

		if opts.showMemoryCost {
			agRow = append(agRow, formatFloat(agg.RAMCost))
			summedMemory += agg.RAMCost
		}

		if opts.showGPUCost {
			agRow = append(agRow, formatFloat(agg.GPUCost))
			summedGPU += agg.GPUCost
		}

		if opts.showPVCost {
			agRow = append(agRow, formatFloat(agg.PVCost))
			summedPV += agg.PVCost
		}

		if opts.showNetworkCost {
			agRow = append(agRow, formatFloat(agg.NetworkCost))
			summedNetwork += agg.NetworkCost
		}

		cumulativeCost := formatFloat(agg.TotalCost)
		agRow = append(agRow, cumulativeCost)

		t.AppendRow(agRow)
		summedCost += agg.TotalCost
	}

	footerRow := table.Row{}

	footerRow = append(footerRow, "SUMMED")

	if opts.showCPUCost {
		footerRow = append(footerRow, formatFloat(summedCPU))
	}

	if opts.showMemoryCost {
		footerRow = append(footerRow, formatFloat(summedMemory))
	}

	if opts.showGPUCost {
		footerRow = append(footerRow, formatFloat(summedGPU))
	}

	if opts.showPVCost {
		footerRow = append(footerRow, formatFloat(summedPV))
	}

	if opts.showNetworkCost {
		footerRow = append(footerRow, formatFloat(summedNetwork))
	}

	footerRow = append(footerRow, formatFloat(summedCost))

	t.AppendFooter(footerRow)
	t.Render()

	return nil
}

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/cost-model/pkg/kubecost"
)

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

func deploymentTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("deployment title should have 2 fields")
	}

	return sp, nil
}

func noopTitleExtractor(aggregationName string) ([]string, error) {
	return []string{aggregationName}, nil
}

func writeAggregationRateTable(out io.Writer, aggs map[string]aggregation, rowTitles []string, rowTitleExtractor func(string) ([]string, error), opts displayOptions) error {
	t := table.NewWriter()
	t.SetOutputMirror(out)

	columnConfigs := []table.ColumnConfig{}

	// "row titles" are, for example Namespace and Deployment
	// for a deployment aggregation
	for _, rowTitle := range rowTitles {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:      rowTitle,
			AutoMerge: true,
		})
	}

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
		Name:        "Monthly Rate (All)",
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

	for _, rowTitle := range rowTitles {
		headerRow = append(headerRow, rowTitle)
	}

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

	headerRow = append(headerRow, "Monthly Rate (All)")

	t.AppendHeader(headerRow)

	sortByConfig := []table.SortBy{}

	sortByConfig = append(sortByConfig, table.SortBy{
		Name: "Monthly Rate (All)",
		Mode: table.Dsc,
	})

	t.SortBy(sortByConfig)

	var summedCost float64
	var summedCPU float64
	var summedMemory float64
	var summedGPU float64
	var summedPV float64
	var summedNetwork float64

	for agBy, agg := range aggs {

		agRow := table.Row{}

		titles, err := rowTitleExtractor(agBy)
		if err != nil {
			for _, _ = range rowTitles {
				agRow = append(agRow, agBy)
			}
		} else {
			for _, title := range titles {
				agRow = append(agRow, title)
			}
		}

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

	for i := 0; i < len(rowTitles)-1; i++ {
		footerRow = append(footerRow, "")
	}

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

package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/cost-model/pkg/kubecost"
	"github.com/kubecost/kubectl-cost/pkg/query"
)

const (
	CPUCol              = "CPU"
	CPUEfficiencyCol    = "CPU Eff."
	MemoryCol           = "Memory"
	MemoryEfficiencyCol = "Memory Eff."
	GPUCol              = "GPU"
	PVCol               = "PV"
	NetworkCol          = "Network"
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
			Name: CPUCol,
		})
		if opts.showEfficiency {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Name: CPUEfficiencyCol,
			})
		}
	}

	if opts.showMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: MemoryCol,
		})
		if opts.showEfficiency {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Name: MemoryEfficiencyCol,
			})
		}
	}

	if opts.showGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: GPUCol,
		})
	}

	if opts.showPVCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: PVCol,
		})
	}

	if opts.showNetworkCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: NetworkCol,
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
		headerRow = append(headerRow, CPUCol)
		if opts.showEfficiency {
			headerRow = append(headerRow, CPUEfficiencyCol)
		}
	}

	if opts.showMemoryCost {
		headerRow = append(headerRow, MemoryCol)
		if opts.showEfficiency {
			headerRow = append(headerRow, MemoryEfficiencyCol)
		}
	}

	if opts.showGPUCost {
		headerRow = append(headerRow, GPUCol)
	}

	if opts.showPVCost {
		headerRow = append(headerRow, PVCol)
	}

	if opts.showNetworkCost {
		headerRow = append(headerRow, NetworkCol)
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
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.CPUEfficiency))
			}
		}

		if opts.showMemoryCost {
			allocRow = append(allocRow, formatFloat(alloc.RAMCost))
			summedMemory += alloc.RAMCost
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.RAMEfficiency))
			}
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
		if opts.showEfficiency {
			footerRow = append(footerRow, "")
		}
	}

	if opts.showMemoryCost {
		footerRow = append(footerRow, formatFloat(summedMemory))
		if opts.showEfficiency {
			footerRow = append(footerRow, "")
		}
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

func deploymentTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("deployment title should have 2 fields")
	}

	return sp, nil
}

// see the results of /model/aggregatedCostModel?window=1d&aggregation=controller
// format is namespace/controller (e.g. kubecost/deployment:kubecost-cost-analyzer)
func controllerTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("deployment title should have 2 fields")
	}

	return sp, nil
}

func noopTitleExtractor(aggregationName string) ([]string, error) {
	return []string{aggregationName}, nil
}

func writeAggregationRateTable(out io.Writer, aggs map[string]query.Aggregation, rowTitles []string, rowTitleExtractor func(string) ([]string, error), opts displayOptions) error {
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
			Name: CPUCol,
		})
	}

	if opts.showMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: MemoryCol,
		})
	}

	if opts.showGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: GPUCol,
		})
	}

	if opts.showPVCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: PVCol,
		})
	}

	if opts.showNetworkCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: NetworkCol,
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
		headerRow = append(headerRow, CPUCol)
	}

	if opts.showMemoryCost {
		headerRow = append(headerRow, MemoryCol)
	}

	if opts.showGPUCost {
		headerRow = append(headerRow, GPUCol)
	}

	if opts.showPVCost {
		headerRow = append(headerRow, PVCol)
	}

	if opts.showNetworkCost {
		headerRow = append(headerRow, NetworkCol)
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

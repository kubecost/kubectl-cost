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
	ClusterCol          = "Cluster"
	NamespaceCol        = "Namespace"
	CPUCol              = "CPU"
	CPUEfficiencyCol    = "CPU Eff."
	MemoryCol           = "Memory"
	MemoryEfficiencyCol = "Memory Eff."
	GPUCol              = "GPU"
	PVCol               = "PV"
	NetworkCol          = "Network"
	SharedCol           = "Shared Cost"
	LoadBalancerCol     = "Load Balancer Cost"
)

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func writeAllocationTable(out io.Writer, allocationType string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, showNamespace bool) {
	t := makeAllocationTable(allocationType, allocations, opts, currencyCode, showNamespace)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAllocationTable(allocationType string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, showNamespace bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      ClusterCol,
		AutoMerge: true,
	})
	if showNamespace {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:      NamespaceCol,
			AutoMerge: true,
		})
	}
	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      allocationType,
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

	if opts.showSharedCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: SharedCol,
		})
	}

	if opts.showLoadBalancerCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: LoadBalancerCol,
		})
	}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:        "Total Cost (All)",
		Align:       text.AlignRight,
		AlignFooter: text.AlignRight,
	})

	if opts.showEfficiency {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Cost Efficiency",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	t.SetColumnConfigs(columnConfigs)

	headerRow := table.Row{}

	headerRow = append(headerRow, ClusterCol)
	if showNamespace {
		headerRow = append(headerRow, NamespaceCol)
	}
	headerRow = append(headerRow, allocationType)

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

	if opts.showSharedCost {
		headerRow = append(headerRow, SharedCol)
	}

	if opts.showLoadBalancerCost {
		headerRow = append(headerRow, LoadBalancerCol)
	}

	headerRow = append(headerRow, "Total Cost (All)")

	if opts.showEfficiency {
		headerRow = append(headerRow, "Cost Efficiency")
	}

	t.AppendHeader(headerRow)
	t.SortBy([]table.SortBy{
		{
			Name: "Total Cost (All)",
			Mode: table.DscNumeric,
		},
	})

	var summedCost float64
	var summedCPU float64
	var summedMemory float64
	var summedGPU float64
	var summedPV float64
	var summedNetwork float64
	var summedShared float64
	var summedLoadBalancer float64

	for _, alloc := range allocations {
		cluster := alloc.Properties.Cluster
		allocName := alloc.Name

		allocRow := table.Row{}

		allocRow = append(allocRow, cluster)
		if showNamespace {
			ns := alloc.Properties.Namespace
			allocRow = append(allocRow, ns)
		}
		allocRow = append(allocRow, allocName)

		if opts.showCPUCost {
			allocRow = append(allocRow, formatFloat(alloc.CPUCost))
			summedCPU += alloc.CPUCost
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.CPUEfficiency()))
			}
		}

		if opts.showMemoryCost {
			allocRow = append(allocRow, formatFloat(alloc.RAMCost))
			summedMemory += alloc.RAMCost
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.RAMEfficiency()))
			}
		}

		if opts.showGPUCost {
			allocRow = append(allocRow, formatFloat(alloc.GPUCost))
			summedGPU += alloc.GPUCost
		}

		if opts.showPVCost {
			allocRow = append(allocRow, formatFloat(alloc.PVCost()))
			summedPV += alloc.PVCost()
		}

		if opts.showNetworkCost {
			allocRow = append(allocRow, formatFloat(alloc.NetworkCost))
			summedNetwork += alloc.NetworkCost
		}

		if opts.showSharedCost {
			allocRow = append(allocRow, formatFloat(alloc.SharedCost))
			summedShared += alloc.SharedCost
		}

		if opts.showLoadBalancerCost {
			allocRow = append(allocRow, formatFloat(alloc.LoadBalancerCost))
			summedLoadBalancer += alloc.LoadBalancerCost
		}

		cumulativeCost := formatFloat(alloc.TotalCost())
		allocRow = append(allocRow, cumulativeCost)

		if opts.showEfficiency {
			allocRow = append(allocRow, formatFloat(alloc.TotalEfficiency()))
		}

		t.AppendRow(allocRow)
		summedCost += alloc.TotalCost()
	}

	footerRow := table.Row{}

	footerRow = append(footerRow, "SUMMED")
	if showNamespace {
		footerRow = append(footerRow, "")
	}
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

	if opts.showSharedCost {
		footerRow = append(footerRow, formatFloat(summedShared))
	}

	if opts.showLoadBalancerCost {
		footerRow = append(footerRow, formatFloat(summedLoadBalancer))
	}

	footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCost)))

	if opts.showEfficiency {
		footerRow = append(footerRow, "")
	}

	t.AppendFooter(footerRow)

	return t
}

func deploymentTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("deployment title should have 2 fields")
	}

	return sp, nil
}

// see the results of /model/aggregatedCostModel?window=1d&aggregation=controller

func controllerTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("controller title should have 2 fields")
	}

	return sp, nil
}

func podTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("pod title should have 2 fields")
	}

	return sp, nil
}

func noopTitleExtractor(aggregationName string) ([]string, error) {
	return []string{aggregationName}, nil
}

func writeAggregationRateTable(out io.Writer, aggs map[string]query.Aggregation, rowTitles []string, rowTitleExtractor func(string) ([]string, error), opts displayOptions, currencyCode string) {
	t := makeAggregationRateTable(aggs, rowTitles, rowTitleExtractor, opts, currencyCode)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAggregationRateTable(aggs map[string]query.Aggregation, rowTitles []string, rowTitleExtractor func(string) ([]string, error), opts displayOptions, currencyCode string) table.Writer {
	t := table.NewWriter()

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

	if opts.showSharedCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: SharedCol,
		})
	}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:        "Monthly Rate (All)",
		Align:       text.AlignRight,
		AlignFooter: text.AlignRight,
	})

	if opts.showEfficiency {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Cost Efficiency",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

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

	if opts.showSharedCost {
		headerRow = append(headerRow, SharedCol)
	}

	headerRow = append(headerRow, "Monthly Rate (All)")

	if opts.showEfficiency {
		headerRow = append(headerRow, "Cost Efficiency")
	}

	t.AppendHeader(headerRow)

	sortByConfig := []table.SortBy{}
	sortByConfig = append(sortByConfig, table.SortBy{
		Name: "Monthly Rate (All)",
		Mode: table.DscNumeric,
	})

	t.SortBy(sortByConfig)

	var summedCost float64
	var summedCPU float64
	var summedMemory float64
	var summedGPU float64
	var summedPV float64
	var summedNetwork float64
	var summedShared float64

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
			if opts.showEfficiency {
				agRow = append(agRow, formatFloat(agg.CPUEfficiency))
			}
		}

		if opts.showMemoryCost {
			agRow = append(agRow, formatFloat(agg.RAMCost))
			summedMemory += agg.RAMCost
			if opts.showEfficiency {
				agRow = append(agRow, formatFloat(agg.RAMEfficiency))
			}
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

		if opts.showSharedCost {
			agRow = append(agRow, formatFloat(agg.SharedCost))
			summedShared += agg.SharedCost
		}

		cumulativeCost := formatFloat(agg.TotalCost)
		agRow = append(agRow, cumulativeCost)

		if opts.showEfficiency {
			agRow = append(agRow, formatFloat(agg.Efficiency))
		}

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

	if opts.showSharedCost {
		footerRow = append(footerRow, formatFloat(summedShared))
	}

	footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCost)))

	if opts.showEfficiency {
		footerRow = append(footerRow, "")
	}

	t.AppendFooter(footerRow)

	return t
}

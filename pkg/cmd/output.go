package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/cost-model/pkg/kubecost"
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
	NameCol             = "Name"
	AssetTypeCol        = "Asset Type"
	CPUCostCol          = "CPU Cost"
	GPUCostCol          = "GPU Cost"
	RAMCostCol          = "RAM Cost"
)

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func writeAllocationTable(out io.Writer, allocationType string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, showNamespace bool, projectToMonthlyRate bool) {
	t := makeAllocationTable(allocationType, allocations, opts, currencyCode, showNamespace, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAllocationTable(allocationType string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, showNamespace bool, projectToMonthlyRate bool) table.Writer {
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

	if projectToMonthlyRate {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Monthly Rate (All)",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	} else {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Total Cost (All)",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

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

	if projectToMonthlyRate {
		headerRow = append(headerRow, "Monthly Rate (All)")
	} else {
		headerRow = append(headerRow, "Total Cost (All)")
	}

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

		// This variable exists to scale costs by the active window
		var histScaleFactor float64 = 1

		if projectToMonthlyRate {

			// scale by minutes per month divided by duration
			// of window in minutes to get projected monthly cost.
			// Note that this approach assumes the window costs will apply
			// through the ENTIRE projected month, no matter the window size.
			histScaleFactor = 43200 / alloc.Minutes()

		}

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
			adjCPUCost := alloc.CPUCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjCPUCost))
			summedCPU += adjCPUCost
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.CPUEfficiency()))
			}
		}

		if opts.showMemoryCost {
			adjRAMCost := alloc.RAMCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjRAMCost))
			summedMemory += adjRAMCost
			if opts.showEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.RAMEfficiency()))
			}
		}

		if opts.showGPUCost {
			adjGPUCost := alloc.GPUCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjGPUCost))
			summedGPU += adjGPUCost
		}

		if opts.showPVCost {
			adjPVCost := alloc.PVCost() * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjPVCost))
			summedPV += adjPVCost
		}

		if opts.showNetworkCost {
			adjNetworkCost := alloc.NetworkCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjNetworkCost))
			summedNetwork += adjNetworkCost
		}

		if opts.showSharedCost {
			adjSharedCost := alloc.SharedCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjSharedCost))
			summedShared += adjSharedCost
		}

		if opts.showLoadBalancerCost {
			adjLoadBalancerCost := alloc.LoadBalancerCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjLoadBalancerCost))
			summedLoadBalancer += adjLoadBalancerCost
		}

		adjTotalCost := alloc.TotalCost() * histScaleFactor
		cumulativeCost := formatFloat(adjTotalCost)
		allocRow = append(allocRow, cumulativeCost)

		if opts.showEfficiency {
			allocRow = append(allocRow, formatFloat(alloc.TotalEfficiency()))
		}

		t.AppendRow(allocRow)
		summedCost += adjTotalCost
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

func writeAssetTable(out io.Writer, assetType string, assets map[string]kubecost.Asset, opts displayOptions, currencyCode string, projectToMonthlyRate bool) {
	t := makeAssetTable(assetType, assets, opts, currencyCode, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAssetTable(assetType string, assets map[string]kubecost.Asset, opts displayOptions, currencyCode string, projectToMonthlyRate bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      ClusterCol,
		AutoMerge: true,
	})

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name: NameCol,
	})

	if opts.showAssetType {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: AssetTypeCol,
		})
	}

	if opts.showCPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        CPUCostCol,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	if opts.showGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        GPUCostCol,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	if opts.showMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        RAMCostCol,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	if projectToMonthlyRate {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Monthly Cost",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	} else {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Total Cost",
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	t.SetColumnConfigs(columnConfigs)

	headerRow := table.Row{}

	headerRow = append(headerRow, ClusterCol)

	headerRow = append(headerRow, NameCol)

	if opts.showAssetType {
		headerRow = append(headerRow, AssetTypeCol)
	}

	if opts.showCPUCost {
		headerRow = append(headerRow, CPUCostCol)
	}

	if opts.showGPUCost {
		headerRow = append(headerRow, GPUCostCol)
	}

	if opts.showMemoryCost {
		headerRow = append(headerRow, RAMCostCol)
	}

	if projectToMonthlyRate {
		headerRow = append(headerRow, "Monthly Cost")
	} else {
		headerRow = append(headerRow, "Total Cost")
	}

	t.AppendHeader(headerRow)

	var summedCost float64
	var summedCPUCost float64
	var summedGPUCost float64
	var summedRAMCost float64

	for _, asset := range assets {

		// This variable exists to scale costs by the active window
		var histScaleFactor float64 = 1

		if projectToMonthlyRate {

			// scale by minutes per month divided by duration
			// of window in minutes to get projected monthly cost.
			// Note that this approach assumes the window costs will apply
			// through the ENTIRE projected month, no matter the window size.
			histScaleFactor = 43200 / asset.Minutes()

		}

		name := asset.Properties().Name
		cluster := asset.Properties().Cluster

		assetRow := table.Row{}

		assetRow = append(assetRow, cluster)

		assetRow = append(assetRow, name)

		switch a := asset.(type) {

		case *kubecost.Node:

			if opts.showAssetType {
				assetType := a.NodeType
				assetRow = append(assetRow, assetType)
			}

			if opts.showCPUCost {
				adjCPUCost := a.CPUCost * histScaleFactor
				assetRow = append(assetRow, formatFloat(adjCPUCost))
				summedCPUCost += adjCPUCost
			}

			if opts.showGPUCost {
				adjGPUCost := a.GPUCost * histScaleFactor
				assetRow = append(assetRow, formatFloat(adjGPUCost))
				summedGPUCost += adjGPUCost
			}

			if opts.showMemoryCost {
				adjRAMCost := a.RAMCost * histScaleFactor
				assetRow = append(assetRow, formatFloat(adjRAMCost))
				summedRAMCost += adjRAMCost
			}

			adjTotalCost := a.TotalCost() * histScaleFactor
			cumulativeCost := formatFloat(adjTotalCost)
			assetRow = append(assetRow, cumulativeCost)

			t.AppendRow(assetRow)
			summedCost += adjTotalCost
		}

	}

	footerRow := table.Row{}

	footerRow = append(footerRow, "SUMMED")

	footerRow = append(footerRow, "")

	if opts.showAssetType {
		footerRow = append(footerRow, "")
	}

	if opts.showCPUCost {
		footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCPUCost)))
	}

	if opts.showGPUCost {
		footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedGPUCost)))
	}

	if opts.showMemoryCost {
		footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedRAMCost)))
	}

	footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCost)))

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

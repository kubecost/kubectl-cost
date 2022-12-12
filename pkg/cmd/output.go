package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/kubecost/opencost/pkg/kubecost"
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
	RAMCostCol          = "RAM Cost"

	PredictColWorkload     = "Workload"
	PredictColReqCPU       = "CPU"
	PredictColReqMemory    = "Mem"
	PredictColMoCoreHours  = "Mo. core-hrs"
	PredictColMoGibHours   = "Mo. GiB-hrs"
	PredictColCostCoreHr   = "Cost/core-hr"
	PredictColCostGiBHr    = "Cost/GiB-hr"
	PredictColMoCostCPU    = "CPU/mo"
	PredictColMoCostMemory = "Mem/mo"
	PredictColMoCostTotal  = "Total/mo"
)

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

func writePredictionTable(out io.Writer, rowData []predictRowData, currencyCode string, showCostPerResourceHr bool) {
	t := makePredictionTable(rowData, currencyCode, showCostPerResourceHr)
	t.SetOutputMirror(out)
	t.Render()
}

func makePredictionTable(rowData []predictRowData, currencyCode string, showCostPerResourceHr bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{
		table.ColumnConfig{
			Name: PredictColWorkload,
		},
		table.ColumnConfig{
			Name: PredictColReqCPU,
		},
		table.ColumnConfig{
			Name: PredictColReqMemory,
		},
	}
	// table.ColumnConfig{
	// 	Name: PredictColMoCoreHours,
	// },
	// table.ColumnConfig{
	// 	Name: PredictColMoGibHours,
	// },

	if showCostPerResourceHr {
		columnConfigs = append(columnConfigs, []table.ColumnConfig{
			table.ColumnConfig{
				Name: PredictColCostCoreHr,
			},
			table.ColumnConfig{
				Name: PredictColCostGiBHr,
			},
		}...)
	}

	columnConfigs = append(columnConfigs, []table.ColumnConfig{
		table.ColumnConfig{
			Name:        PredictColMoCostCPU,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		table.ColumnConfig{
			Name:        PredictColMoCostMemory,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		table.ColumnConfig{
			Name:        PredictColMoCostTotal,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
	}...)
	t.SetColumnConfigs(columnConfigs)

	headerRow := table.Row{
		PredictColWorkload,
		PredictColReqCPU,
		PredictColReqMemory,
	}

	// PredictColMoCoreHours,
	// PredictColMoGibHours,

	if showCostPerResourceHr {
		headerRow = append(headerRow,
			PredictColCostCoreHr,
			PredictColCostGiBHr,
		)
	}

	headerRow = append(headerRow,
		PredictColMoCostCPU,
		PredictColMoCostMemory,
		PredictColMoCostTotal,
	)

	t.AppendHeader(headerRow)

	t.SortBy([]table.SortBy{
		{
			Name: PredictColMoCostTotal,
			Mode: table.DscNumeric,
		},
	})

	var summedMonthlyCPU float64
	var summedMonthlyMem float64
	var summedMonthlyTotal float64

	for _, rowDatum := range rowData {
		row := table.Row{}
		row = append(row, fmt.Sprintf("%s/%s", rowDatum.workloadType, rowDatum.workloadName))
		row = append(row, rowDatum.cpuStr)
		row = append(row, rowDatum.memStr)

		// row = append(row, prediction.MonthlyCoreHours)
		// row = append(row, fmt.Sprintf("%.2f", prediction.MonthlyByteHours/1024/1024/1024))

		if showCostPerResourceHr {
			row = append(row, fmt.Sprintf("%.4f %s", rowDatum.prediction.DerivedCostPerCoreHour, currencyCode))
			row = append(row, fmt.Sprintf("%.4f %s", rowDatum.prediction.DerivedCostPerByteHour*1024*1024*1024, currencyCode))
		}

		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.prediction.MonthlyCostCPU, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.prediction.MonthlyCostMemory, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.prediction.MonthlyCostTotal, currencyCode))

		summedMonthlyCPU += rowDatum.prediction.MonthlyCostCPU
		summedMonthlyMem += rowDatum.prediction.MonthlyCostMemory
		summedMonthlyTotal += rowDatum.prediction.MonthlyCostTotal

		t.AppendRow(row)
	}

	// A summary footer is redundant if there is only one row
	if len(rowData) > 1 {
		footerRow := table.Row{}
		blankRows := 3
		if showCostPerResourceHr {
			blankRows += 2
		}
		for i := 0; i < blankRows; i++ {
			footerRow = append(footerRow, "")
		}
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyCPU, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyMem, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyTotal, currencyCode))
		t.AppendFooter(footerRow)
	}

	return t
}

func writeAllocationTable(out io.Writer, aggregation []string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, projectToMonthlyRate bool) {
	t := makeAllocationTable(aggregation, allocations, opts, currencyCode, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAllocationTable(aggregation []string, allocations map[string]kubecost.Allocation, opts displayOptions, currencyCode string, projectToMonthlyRate bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{}

	for _, aggField := range aggregation {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:      strings.Title(aggField),
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

	for _, aggField := range aggregation {
		headerRow = append(headerRow, strings.Title(aggField))
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
		{
			Name: "Monthly Rate (All)",
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

		allocRow := table.Row{}

		if alloc.Name == "__idle__" {
			for range aggregation {
				allocRow = append(allocRow, "__idle__")
			}
		} else {
			splitName := strings.Split(alloc.Name, "/")
			if len(splitName) != len(aggregation) {
				panic(fmt.Sprintf("name '%s' split into '%+v' (len %d) should have the same number of fields as aggregation '%+v' (len %d)", alloc.Name, splitName, len(splitName), aggregation, len(aggregation)))
			}

			for _, fieldValue := range splitName {
				allocRow = append(allocRow, fieldValue)
			}
		}

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

	for i := 0; i < len(aggregation)-1; i++ {
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

func writeAssetTable(out io.Writer, assetType string, assets map[string]query.AssetNode, opts displayOptions, currencyCode string, projectToMonthlyRate bool) {
	t := makeAssetTable(assetType, assets, opts, currencyCode, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func makeAssetTable(assetType string, assets map[string]query.AssetNode, opts displayOptions, currencyCode string, projectToMonthlyRate bool) table.Writer {
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
	var summedRAMCost float64

	for _, asset := range assets {

		// This variable exists to scale costs by the active window
		var histScaleFactor float64 = 1

		if projectToMonthlyRate {

			// scale by minutes per month divided by duration
			// of window in minutes to get projected monthly cost.
			// Note that this approach assumes the window costs will apply
			// through the ENTIRE projected month, no matter the window size.
			histScaleFactor = 43200 / asset.Minutes

		}

		name := asset.Properties.Name
		cluster := asset.Properties.Cluster

		assetRow := table.Row{}

		assetRow = append(assetRow, cluster)

		assetRow = append(assetRow, name)

		if opts.showAssetType {
			assetType := asset.NodeType
			assetRow = append(assetRow, assetType)
		}

		if opts.showCPUCost {
			adjCPUCost := asset.CPUCost * histScaleFactor
			assetRow = append(assetRow, formatFloat(adjCPUCost))
			summedCPUCost += adjCPUCost
		}

		if opts.showMemoryCost {
			adjRAMCost := asset.RAMCost * histScaleFactor
			assetRow = append(assetRow, formatFloat(adjRAMCost))
			summedRAMCost += adjRAMCost
		}

		adjTotalCost := asset.TotalCost * histScaleFactor
		cumulativeCost := formatFloat(adjTotalCost)
		assetRow = append(assetRow, cumulativeCost)

		t.AppendRow(assetRow)
		summedCost += adjTotalCost
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

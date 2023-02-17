package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/kubecost"

	"github.com/spf13/cobra"
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
)

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}

// With the addition of commands which query the assets API,
// some of these don't apply to all commands. However, as they
// are applied during the output, this shouldn't cause issues.
type DisplayOptions struct {
	ShowCPUCost          bool
	ShowMemoryCost       bool
	ShowGPUCost          bool
	ShowPVCost           bool
	ShowNetworkCost      bool
	ShowEfficiency       bool
	ShowSharedCost       bool
	ShowLoadBalancerCost bool
	ShowAssetType        bool

	ShowAll bool
}

func AddDisplayOptionsFlags(cmd *cobra.Command, options *DisplayOptions) {
	cmd.Flags().BoolVar(&options.ShowCPUCost, "show-cpu", false, "show data for CPU cost")
	cmd.Flags().BoolVar(&options.ShowMemoryCost, "show-memory", false, "show data for memory cost")
	cmd.Flags().BoolVar(&options.ShowGPUCost, "show-gpu", false, "show data for GPU cost")
	cmd.Flags().BoolVar(&options.ShowPVCost, "show-pv", false, "show data for PV (physical volume) cost")
	cmd.Flags().BoolVar(&options.ShowNetworkCost, "show-network", false, "show data for network cost")
	cmd.Flags().BoolVar(&options.ShowSharedCost, "show-shared", false, "show shared cost data")
	cmd.Flags().BoolVar(&options.ShowLoadBalancerCost, "show-lb", false, "show load balancer cost data")
	cmd.Flags().BoolVar(&options.ShowEfficiency, "show-efficiency", true, "show efficiency of cost alongside CPU and memory cost")
	cmd.Flags().BoolVar(&options.ShowAssetType, "show-asset-type", false, "show type of assets displayed.")
	cmd.Flags().BoolVarP(&options.ShowAll, "show-all-resources", "A", false, "Equivalent to --show-cpu --show-memory --show-gpu --show-pv --show-network --show-efficiency for namespace, deployment, controller, lable and pod OR --show-type --show-cpu --show-memory for node.")
}

func WriteAllocationTable(out io.Writer, aggregation []string, allocations map[string]kubecost.Allocation, opts DisplayOptions, currencyCode string, projectToMonthlyRate bool) {
	t := MakeAllocationTable(aggregation, allocations, opts, currencyCode, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func (do *DisplayOptions) Complete() {
	if do.ShowAll {
		do.ShowCPUCost = true
		do.ShowMemoryCost = true
		do.ShowGPUCost = true
		do.ShowPVCost = true
		do.ShowNetworkCost = true
		do.ShowSharedCost = true
		do.ShowLoadBalancerCost = true
		do.ShowAssetType = true
	}
}

func MakeAllocationTable(aggregation []string, allocations map[string]kubecost.Allocation, opts DisplayOptions, currencyCode string, projectToMonthlyRate bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{}

	for _, aggField := range aggregation {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:      strings.Title(aggField),
			AutoMerge: true,
		})
	}

	if opts.ShowCPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: CPUCol,
		})
		if opts.ShowEfficiency {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Name: CPUEfficiencyCol,
			})
		}
	}

	if opts.ShowMemoryCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: MemoryCol,
		})
		if opts.ShowEfficiency {
			columnConfigs = append(columnConfigs, table.ColumnConfig{
				Name: MemoryEfficiencyCol,
			})
		}
	}

	if opts.ShowGPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: GPUCol,
		})
	}

	if opts.ShowPVCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: PVCol,
		})
	}

	if opts.ShowNetworkCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: NetworkCol,
		})
	}

	if opts.ShowSharedCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: SharedCol,
		})
	}

	if opts.ShowLoadBalancerCost {
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

	if opts.ShowEfficiency {
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

	if opts.ShowCPUCost {
		headerRow = append(headerRow, CPUCol)
		if opts.ShowEfficiency {
			headerRow = append(headerRow, CPUEfficiencyCol)
		}
	}

	if opts.ShowMemoryCost {
		headerRow = append(headerRow, MemoryCol)
		if opts.ShowEfficiency {
			headerRow = append(headerRow, MemoryEfficiencyCol)
		}
	}

	if opts.ShowGPUCost {
		headerRow = append(headerRow, GPUCol)
	}

	if opts.ShowPVCost {
		headerRow = append(headerRow, PVCol)
	}

	if opts.ShowNetworkCost {
		headerRow = append(headerRow, NetworkCol)
	}

	if opts.ShowSharedCost {
		headerRow = append(headerRow, SharedCol)
	}

	if opts.ShowLoadBalancerCost {
		headerRow = append(headerRow, LoadBalancerCol)
	}

	if projectToMonthlyRate {
		headerRow = append(headerRow, "Monthly Rate (All)")
	} else {
		headerRow = append(headerRow, "Total Cost (All)")
	}

	if opts.ShowEfficiency {
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

		if opts.ShowCPUCost {
			adjCPUCost := alloc.CPUCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjCPUCost))
			summedCPU += adjCPUCost
			if opts.ShowEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.CPUEfficiency()))
			}
		}

		if opts.ShowMemoryCost {
			adjRAMCost := alloc.RAMCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjRAMCost))
			summedMemory += adjRAMCost
			if opts.ShowEfficiency {
				allocRow = append(allocRow, formatFloat(alloc.RAMEfficiency()))
			}
		}

		if opts.ShowGPUCost {
			adjGPUCost := alloc.GPUCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjGPUCost))
			summedGPU += adjGPUCost
		}

		if opts.ShowPVCost {
			adjPVCost := alloc.PVCost() * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjPVCost))
			summedPV += adjPVCost
		}

		if opts.ShowNetworkCost {
			adjNetworkCost := alloc.NetworkCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjNetworkCost))
			summedNetwork += adjNetworkCost
		}

		if opts.ShowSharedCost {
			adjSharedCost := alloc.SharedCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjSharedCost))
			summedShared += adjSharedCost
		}

		if opts.ShowLoadBalancerCost {
			adjLoadBalancerCost := alloc.LoadBalancerCost * histScaleFactor
			allocRow = append(allocRow, formatFloat(adjLoadBalancerCost))
			summedLoadBalancer += adjLoadBalancerCost
		}

		adjTotalCost := alloc.TotalCost() * histScaleFactor
		cumulativeCost := formatFloat(adjTotalCost)
		allocRow = append(allocRow, cumulativeCost)

		if opts.ShowEfficiency {
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

	if opts.ShowCPUCost {
		footerRow = append(footerRow, formatFloat(summedCPU))
		if opts.ShowEfficiency {
			footerRow = append(footerRow, "")
		}
	}

	if opts.ShowMemoryCost {
		footerRow = append(footerRow, formatFloat(summedMemory))
		if opts.ShowEfficiency {
			footerRow = append(footerRow, "")
		}
	}

	if opts.ShowGPUCost {
		footerRow = append(footerRow, formatFloat(summedGPU))
	}

	if opts.ShowPVCost {
		footerRow = append(footerRow, formatFloat(summedPV))
	}

	if opts.ShowNetworkCost {
		footerRow = append(footerRow, formatFloat(summedNetwork))
	}

	if opts.ShowSharedCost {
		footerRow = append(footerRow, formatFloat(summedShared))
	}

	if opts.ShowLoadBalancerCost {
		footerRow = append(footerRow, formatFloat(summedLoadBalancer))
	}

	footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCost)))

	if opts.ShowEfficiency {
		footerRow = append(footerRow, "")
	}

	t.AppendFooter(footerRow)

	return t
}

func WriteAssetTable(out io.Writer, assetType string, assets map[string]query.AssetNode, opts DisplayOptions, currencyCode string, projectToMonthlyRate bool) {
	t := MakeAssetTable(assetType, assets, opts, currencyCode, projectToMonthlyRate)

	t.SetOutputMirror(out)
	t.Render()
}

func MakeAssetTable(assetType string, assets map[string]query.AssetNode, opts DisplayOptions, currencyCode string, projectToMonthlyRate bool) table.Writer {
	t := table.NewWriter()

	columnConfigs := []table.ColumnConfig{}

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name:      ClusterCol,
		AutoMerge: true,
	})

	columnConfigs = append(columnConfigs, table.ColumnConfig{
		Name: NameCol,
	})

	if opts.ShowAssetType {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name: AssetTypeCol,
		})
	}

	if opts.ShowCPUCost {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        CPUCostCol,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		})
	}

	if opts.ShowMemoryCost {
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

	if opts.ShowAssetType {
		headerRow = append(headerRow, AssetTypeCol)
	}

	if opts.ShowCPUCost {
		headerRow = append(headerRow, CPUCostCol)
	}

	if opts.ShowMemoryCost {
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

		if opts.ShowAssetType {
			assetType := asset.NodeType
			assetRow = append(assetRow, assetType)
		}

		if opts.ShowCPUCost {
			adjCPUCost := asset.CPUCost * histScaleFactor
			assetRow = append(assetRow, formatFloat(adjCPUCost))
			summedCPUCost += adjCPUCost
		}

		if opts.ShowMemoryCost {
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

	if opts.ShowAssetType {
		footerRow = append(footerRow, "")
	}

	if opts.ShowCPUCost {
		footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCPUCost)))
	}

	if opts.ShowMemoryCost {
		footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedRAMCost)))
	}

	footerRow = append(footerRow, fmt.Sprintf("%s %s", currencyCode, formatFloat(summedCost)))

	t.AppendFooter(footerRow)

	return t
}

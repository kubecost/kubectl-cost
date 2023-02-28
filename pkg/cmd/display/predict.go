package display

import (
	"fmt"
	"io"

	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/util/timeutil"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/spf13/cobra"
)

const (
	ColResourceUnit   = "resource"
	ColMoDiffResource = "Δ"
	ColMoDiffCost     = "Δ cost/mo"
	ColCostPerUnit    = "cost per unit"
)

type PredictDisplayOptions struct {
	OnlyDiff  bool
	OnlyAfter bool
}

func AddPredictDisplayOptionsFlags(cmd *cobra.Command, options *PredictDisplayOptions) {
	cmd.Flags().BoolVar(&options.OnlyDiff, "only-diff", true, "Set true to only show the cost difference (cost \"impact\") instead of the overall cost plus diff.")
	cmd.Flags().BoolVar(&options.OnlyAfter, "only-after", false, "Set true to only show the overall predicted cost of the workload.")
}

func (o *PredictDisplayOptions) Validate() error {
	if o.OnlyDiff && o.OnlyAfter {
		return fmt.Errorf("OnlyDiff and OnlyAfter cannot both be true.")
	}
	return nil
}

func WritePredictionTable(out io.Writer, rowData []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) {
	totalCostImpact := 0.0
	for _, specData := range rowData {
		totalCostImpact += specData.CostChange.TotalMonthlyRate
		t := MakePredictionTable(specData, currencyCode, opts)
		t.SetOutputMirror(out)
		t.Render()
	}
	out.Write([]byte(fmt.Sprintf("Total cost impact: %.2f %s\n", totalCostImpact, currencyCode)))
}

func MakePredictionTable(specData query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) table.Writer {
	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.SetTitle("%s/%s/%s", specData.Namespace, specData.ControllerKind, specData.ControllerName)

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:  ColResourceUnit,
			Align: text.AlignLeft,
		},
		{
			Name:  ColMoDiffResource,
			Align: text.AlignRight,
		},
		{
			Name:  ColCostPerUnit,
			Align: text.AlignRight,
		},
		{
			Name:  ColMoDiffCost,
			Align: text.AlignRight,
		},
	})

	t.AppendHeader(table.Row{
		ColResourceUnit,
		ColMoDiffResource,
		ColCostPerUnit,
		ColMoDiffCost,
	})

	// FIXME: Handle if speccost response doesn't have resource info

	avgCoreCount := specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth
	costPerCore := specData.CostChange.CPUMonthlyRate / avgCoreCount
	cpuRow := table.Row{
		"CPU Cores",
		fmt.Sprintf("%.2f", avgCoreCount),
		fmt.Sprintf("%.2f %s", costPerCore, currencyCode),
		fmt.Sprintf("%.2f %s", specData.CostChange.CPUMonthlyRate, currencyCode),
	}
	t.AppendRow(cpuRow)

	avgGiB := specData.CostChange.MonthlyRAMByteHours / (1024 * 1024 * 1024) / timeutil.HoursPerMonth
	costPerGiB := specData.CostChange.RAMMonthlyRate / avgGiB
	ramRow := table.Row{
		"RAM GiB",
		fmt.Sprintf("%.2f", avgGiB),
		fmt.Sprintf("%.2f %s", costPerGiB, currencyCode),
		fmt.Sprintf("%.2f %s", specData.CostChange.RAMMonthlyRate, currencyCode),
	}
	t.AppendRow(ramRow)

	if !(specData.CostBefore.GPUMonthlyRate == 0 && specData.CostAfter.GPUMonthlyRate == 0) {
		avgGPUs := specData.CostChange.MonthlyGPUHours / timeutil.HoursPerMonth
		costPerGPU := specData.CostChange.GPUMonthlyRate / avgGPUs
		gpuRow := table.Row{
			"GPUs",
			fmt.Sprintf("%.2f", avgGPUs),
			fmt.Sprintf("%.2f %s", costPerGPU, currencyCode),
			fmt.Sprintf("%.2f %s", specData.CostChange.GPUMonthlyRate, currencyCode),
		}
		t.AppendRow(gpuRow)
	}

	return t
}

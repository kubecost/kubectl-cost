package display

import (
	"fmt"
	"io"
	"strings"

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

type PredictDisplayOptions struct{}

func AddPredictDisplayOptionsFlags(cmd *cobra.Command, options *PredictDisplayOptions) {
}

func (o *PredictDisplayOptions) Validate() error {
	return nil
}

func WritePredictionTable(out io.Writer, rowData []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) {
	totalCostImpact := 0.0
	for _, specData := range rowData {
		totalCostImpact += specData.CostChange.TotalMonthlyRate
		t := MakePredictionTable(specData, currencyCode, opts)
		t.SetOutputMirror(out)
		t.Render()
		out.Write([]byte("\n"))
	}

	out.Write([]byte(
		text.Bold.Sprint(
			fmt.Sprintf("Total monthly cost change: %.2f %s\n", totalCostImpact, currencyCode),
		),
	))
}

// fmtResourceFloat formats with a precision of 3 and then trims trailing 0s in
// the decimal places.
func fmtResourceFloat(x float64) string {
	s := fmt.Sprintf("%.3f", x)

	// If formatted float ends in .000, remove
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")
	return s
}

// fmtCostFloat starts by formatting the given float at precision 2. If the
// precision is insufficient for showing useful information, the precision is
// increased.
//
// The bar for "useful information" is x < 1 (so we only have decimals) and all
// decimal places except the final place are '0'.
func fmtCostFloat(x float64) string {
	precision := 2
	precisionToFmt := func(precision int) string {
		return fmt.Sprintf("%%.%df", precision)
	}
	s := fmt.Sprintf(precisionToFmt(precision), x)
	if x < 1 && x > 0 {
		for {
			secondToLast := s[len(s)-2]
			if secondToLast != '0' {
				break
			}
			precision += 1
			s = fmt.Sprintf(precisionToFmt(precision), x)
		}
	}
	return s
}

func MakePredictionTable(specData query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) table.Writer {
	t := table.NewWriter()

	// start with this style, then we'll modify
	style := table.StyleLight
	style.Options.SeparateColumns = false
	style.Options.DrawBorder = false
	style.Options.SeparateHeader = true
	style.Title.Colors = append(style.Title.Colors, text.Bold)
	t.SetStyle(style)
	t.SetTitle("%s/%s/%s", specData.Namespace, specData.ControllerKind, specData.ControllerName)

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:  ColMoDiffResource,
			Align: text.AlignRight,
		},
		{
			Name:  ColResourceUnit,
			Align: text.AlignLeft,
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
		ColMoDiffResource,
		ColResourceUnit,
		ColCostPerUnit,
		ColMoDiffCost,
	})

	cpuUnits := "CPU cores"
	avgCPUInUnits := specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth
	if avgCPUInUnits < 1 {
		cpuUnits = "CPU millicores"
		avgCPUInUnits = specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth * 1000
	}
	costPerUnit := specData.CostChange.CPUMonthlyRate / avgCPUInUnits
	cpuRow := table.Row{
		fmtResourceFloat(avgCPUInUnits),
		cpuUnits,
		fmt.Sprintf("%s %s", fmtCostFloat(costPerUnit), currencyCode),
		fmt.Sprintf("%.2f %s", specData.CostChange.CPUMonthlyRate, currencyCode),
	}
	t.AppendRow(cpuRow)

	ramUnits := "RAM GiB"
	ramUnitDivisor := 1024 * 1024 * 1024.0
	avgRAMInUnits := specData.CostChange.MonthlyRAMByteHours / ramUnitDivisor / timeutil.HoursPerMonth
	if avgRAMInUnits < 1 {
		ramUnits = "RAM MiB"
		ramUnitDivisor = 1024 * 1024.0
		avgRAMInUnits = specData.CostChange.MonthlyRAMByteHours / ramUnitDivisor / timeutil.HoursPerMonth
	}
	costPerUnit = specData.CostChange.RAMMonthlyRate / avgRAMInUnits
	ramRow := table.Row{
		fmtResourceFloat(avgRAMInUnits),
		ramUnits,
		fmt.Sprintf("%s %s", fmtCostFloat(costPerUnit), currencyCode),
		fmt.Sprintf("%.2f %s", specData.CostChange.RAMMonthlyRate, currencyCode),
	}
	t.AppendRow(ramRow)

	if !(specData.CostBefore.GPUMonthlyRate == 0 && specData.CostAfter.GPUMonthlyRate == 0) {
		avgGPUs := specData.CostChange.MonthlyGPUHours / timeutil.HoursPerMonth
		costPerGPU := specData.CostChange.GPUMonthlyRate / avgGPUs
		gpuRow := table.Row{
			fmtResourceFloat(avgGPUs),
			"GPUs",
			fmt.Sprintf("%s %s", fmtCostFloat(costPerGPU), currencyCode),
			fmt.Sprintf("%.2f %s", specData.CostChange.GPUMonthlyRate, currencyCode),
		}
		t.AppendRow(gpuRow)
	}

	return t
}

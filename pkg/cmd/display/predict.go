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
	ColObject         = "object"
	ColResourceUnit   = "resource unit"
	ColMoDiffResource = "Δ qty"
	ColMoDiffCost     = "Δ cost/mo"
	ColCostPerUnit    = "cost per unit"
	ColPctChange      = "% change"
)

type PredictDisplayOptions struct{}

func AddPredictDisplayOptionsFlags(cmd *cobra.Command, options *PredictDisplayOptions) {
}

func (o *PredictDisplayOptions) Validate() error {
	return nil
}

func WritePredictionTable(out io.Writer, rowData []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) {
	t := MakePredictionTable(rowData, currencyCode, opts)
	t.SetOutputMirror(out)
	t.Render()
}

// fmtResourceFloat formats with a precision of 2 and then trims trailing 0s in
// the decimal places.
func fmtResourceFloat(x float64) string {
	s := fmt.Sprintf("%.2f", x)
	if x > 0 {
		s = fmt.Sprintf("+%s", s)
	}

	// If formatted float ends in .000, remove
	s = strings.TrimRight(s, "0")
	s = strings.TrimSuffix(s, ".")

	return s
}

// fmtResourceCostFloat starts by formatting the given float at precision 2. If
// the precision is insufficient for showing useful information, the precision
// is increased.
//
// The bar for "useful information" is x < 1 (so we only have decimals) and all
// decimal places except the final place are '0'.
func fmtResourceCostFloat(x float64) string {
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

func fmtOverallCostFloat(x float64) string {
	s := fmt.Sprintf("%.2f", x)
	if x > 0 {
		s = fmt.Sprintf("+%s", s)
	}
	return s
}

func MakePredictionTable(specDiffs []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) table.Writer {
	t := table.NewWriter()

	// start with this style, then we'll modify
	style := table.StyleLight
	style.Options.SeparateColumns = false
	style.Options.DrawBorder = false
	style.Options.SeparateHeader = true
	style.Title.Colors = append(style.Title.Colors, text.Bold)
	t.SetStyle(style)
	// t.SetTitle("%s/%s/%s", specData.Namespace, specData.ControllerKind, specData.ControllerName)

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:      ColObject,
			Align:     text.AlignLeft,
			AutoMerge: true,

			// Currently this wrapping can result in overly-tall rows if merging
			// wrapped text. While it is isn't perfect, I want to keep this on
			// to use as little horizontal space as possible.
			// When https://github.com/jedib0t/go-pretty/issues/261 is fixed, we
			// should be able to update go-pretty to fix this unnecessary
			// whitespace.
			WidthMax:         26,
			WidthMaxEnforcer: text.WrapSoft,
		},
		{
			Name:  ColMoDiffResource,
			Align: text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					return fmtResourceFloat(f)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
		},
		{
			Name:  ColResourceUnit,
			Align: text.AlignLeft,
		},
		{
			Name:  ColCostPerUnit,
			Align: text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					return fmt.Sprintf("%s %s", fmtResourceCostFloat(f), currencyCode)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
		},
		{
			Name:  ColMoDiffCost,
			Align: text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					return fmt.Sprintf("%s %s", fmtOverallCostFloat(f), currencyCode)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
			TransformerFooter: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					return fmt.Sprintf("%s %s", fmtOverallCostFloat(f), currencyCode)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
		},
		{
			Name:  ColPctChange,
			Align: text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					prefix := ""
					if f > 0 {
						prefix = "+"
					}
					return fmt.Sprintf("%s%.2f%%", prefix, f)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
		},
	})

	t.AppendHeader(table.Row{
		ColObject,
		ColMoDiffResource,
		ColResourceUnit,
		ColCostPerUnit,
		ColMoDiffCost,
		ColPctChange,
	})

	totalCostImpact := 0.0
	for _, specData := range specDiffs {
		totalCostImpact += specData.CostChange.TotalMonthlyRate

		workloadName := fmt.Sprintf("%s %s %s", specData.Namespace, specData.ControllerKind, specData.ControllerName)

		// Don't show resource if there is no cost data before or after
		if !(specData.CostBefore.CPUMonthlyRate == 0 && specData.CostAfter.CPUMonthlyRate == 0) {
			cpuUnits := "CPU cores"
			avgCPUInUnits := specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth
			if avgCPUInUnits < 1 {
				cpuUnits = "CPU millicores"
				avgCPUInUnits = specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth * 1000
			}
			costPerUnit := specData.CostChange.CPUMonthlyRate / avgCPUInUnits
			cpuRow := table.Row{
				workloadName,
				avgCPUInUnits,
				cpuUnits,
				costPerUnit,
				specData.CostChange.CPUMonthlyRate,
			}
			if specData.CostBefore.CPUMonthlyRate != 0 {
				cpuRow = append(cpuRow, specData.CostChange.CPUMonthlyRate/specData.CostBefore.CPUMonthlyRate*100)
			}
			t.AppendRow(cpuRow)
		}

		if !(specData.CostBefore.RAMMonthlyRate == 0 && specData.CostAfter.RAMMonthlyRate == 0) {

			ramUnits := "RAM GiB"
			ramUnitDivisor := 1024 * 1024 * 1024.0
			avgRAMInUnits := specData.CostChange.MonthlyRAMByteHours / ramUnitDivisor / timeutil.HoursPerMonth
			// If < 1 GiB, convert to MiB
			if avgRAMInUnits < 1 {
				ramUnits = "RAM MiB"
				ramUnitDivisor = 1024 * 1024.0
				avgRAMInUnits = specData.CostChange.MonthlyRAMByteHours / ramUnitDivisor / timeutil.HoursPerMonth
			}
			costPerUnit := specData.CostChange.RAMMonthlyRate / avgRAMInUnits
			ramRow := table.Row{
				workloadName,
				avgRAMInUnits,
				ramUnits,
				costPerUnit,
				specData.CostChange.RAMMonthlyRate,
			}
			if specData.CostBefore.RAMMonthlyRate != 0 {
				ramRow = append(ramRow, specData.CostChange.RAMMonthlyRate/specData.CostBefore.RAMMonthlyRate*100)
			}
			t.AppendRow(ramRow)
		}

		if !(specData.CostBefore.GPUMonthlyRate == 0 && specData.CostAfter.GPUMonthlyRate == 0) {
			avgGPUs := specData.CostChange.MonthlyGPUHours / timeutil.HoursPerMonth
			costPerGPU := specData.CostChange.GPUMonthlyRate / avgGPUs
			gpuRow := table.Row{
				workloadName,
				avgGPUs,
				"GPUs",
				costPerGPU,
				specData.CostChange.GPUMonthlyRate,
			}
			if specData.CostBefore.GPUMonthlyRate != 0 {
				gpuRow = append(gpuRow, specData.CostChange.GPUMonthlyRate/specData.CostBefore.GPUMonthlyRate*100)
			}
			t.AppendRow(gpuRow)
		}
		t.AppendSeparator()
	}

	t.AppendFooter(table.Row{
		"Total monthly cost change",
		"",
		"",
		"",
		totalCostImpact,
	})

	return t
}

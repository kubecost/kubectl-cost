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
	ColMoResource     = "qty"
	ColMoCost         = "cost/mo"
	ColCostPerUnit    = "cost per unit"
	ColPctChange      = "% change"
)

type PredictDisplayOptions struct {
	// ShowNew determines if "After" cost info will be shown alongside the
	// diff
	ShowNew bool

	// HideDiff will disable diff information if true
	HideDiff bool
}

func AddPredictDisplayOptionsFlags(cmd *cobra.Command, options *PredictDisplayOptions) {
}

func (o *PredictDisplayOptions) Validate() error {
	if !o.ShowNew && o.HideDiff {
		return fmt.Errorf("ShowNew and HideDiff cannot be set such that no data will be shown")
	}
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
			Name:   ColMoResource,
			Hidden: !opts.ShowNew,
			Align:  text.AlignRight,
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
			Name:   ColMoDiffResource,
			Hidden: opts.HideDiff,
			Align:  text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					s := fmtResourceFloat(f)
					if f > 0 {
						s = fmt.Sprintf("+%s", s)
					}
					return s
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
			Name:   ColMoCost,
			Hidden: !opts.ShowNew,
			Align:  text.AlignRight,
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
			Name:   ColMoDiffCost,
			Hidden: opts.HideDiff,
			Align:  text.AlignRight,
			Transformer: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					s := fmt.Sprintf("%s %s", fmtOverallCostFloat(f), currencyCode)
					if f > 0 {
						s = fmt.Sprintf("+%s", s)
					}
					return s
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
			TransformerFooter: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					s := fmt.Sprintf("%s %s", fmtOverallCostFloat(f), currencyCode)
					if f > 0 {
						s = fmt.Sprintf("+%s", s)
					}
					return s
				}
				if s, ok := val.(string); ok {
					return s
				}
				return "invalid value"
			},
		},
		{
			Name:   ColPctChange,
			Hidden: opts.HideDiff,
			Align:  text.AlignRight,
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
		ColMoResource,
		ColMoDiffResource,
		ColResourceUnit,
		ColCostPerUnit,
		ColMoCost,
		ColMoDiffCost,
		ColPctChange,
	})

	totalCostImpact := 0.0
	totalCostNew := 0.0
	for _, specData := range specDiffs {
		totalCostImpact += specData.CostChange.TotalMonthlyRate
		totalCostNew += specData.CostAfter.TotalMonthlyRate

		workloadName := fmt.Sprintf("%s %s %s", specData.Namespace, specData.ControllerKind, specData.ControllerName)

		// Don't show resource if there is no cost data before or after
		if !(specData.CostBefore.CPUMonthlyRate == 0 && specData.CostAfter.CPUMonthlyRate == 0) {
			units := "CPU cores"
			avgUnitsNew := specData.CostAfter.MonthlyCPUCoreHours / timeutil.HoursPerMonth
			avgUnitsDiff := specData.CostChange.MonthlyCPUCoreHours / timeutil.HoursPerMonth
			factor := 1.0
			if avgUnitsNew*factor < 1 {
				units = "CPU millicores"
				factor = 1000
			}
			avgUnitsDiff *= factor
			avgUnitsNew *= factor
			costPerUnit := specData.CostChange.CPUMonthlyRate / avgUnitsDiff
			row := table.Row{
				workloadName,
				avgUnitsNew,
				avgUnitsDiff,
				units,
				costPerUnit,
				specData.CostAfter.CPUMonthlyRate,
				specData.CostChange.CPUMonthlyRate,
			}
			if specData.CostBefore.CPUMonthlyRate != 0 {
				row = append(row, specData.CostChange.CPUMonthlyRate/specData.CostBefore.CPUMonthlyRate*100)
			}
			t.AppendRow(row)
		}

		if !(specData.CostBefore.RAMMonthlyRate == 0 && specData.CostAfter.RAMMonthlyRate == 0) {
			units := "RAM GiB"
			avgUnitsNew := specData.CostAfter.MonthlyRAMByteHours / timeutil.HoursPerMonth
			avgUnitsDiff := specData.CostChange.MonthlyRAMByteHours / timeutil.HoursPerMonth
			factor := 1.0 / (1024 * 1024 * 1024)
			if avgUnitsNew*factor < 1 {
				units = "RAM MiB"
				factor = 1.0 / (1024 * 1024)
			}
			avgUnitsDiff *= factor
			avgUnitsNew *= factor
			costPerUnit := specData.CostChange.RAMMonthlyRate / avgUnitsDiff
			row := table.Row{
				workloadName,
				avgUnitsNew,
				avgUnitsDiff,
				units,
				costPerUnit,
				specData.CostAfter.RAMMonthlyRate,
				specData.CostChange.RAMMonthlyRate,
			}
			if specData.CostBefore.RAMMonthlyRate != 0 {
				row = append(row, specData.CostChange.RAMMonthlyRate/specData.CostBefore.RAMMonthlyRate*100)
			}
			t.AppendRow(row)
		}

		if !(specData.CostBefore.GPUMonthlyRate == 0 && specData.CostAfter.GPUMonthlyRate == 0) {
			units := "GPUs"
			avgUnitsNew := specData.CostAfter.MonthlyGPUHours / timeutil.HoursPerMonth
			avgUnitsDiff := specData.CostChange.MonthlyGPUHours / timeutil.HoursPerMonth
			factor := 1.0
			avgUnitsDiff *= factor
			avgUnitsNew *= factor
			costPerUnit := specData.CostChange.GPUMonthlyRate / avgUnitsDiff
			row := table.Row{
				workloadName,
				avgUnitsNew,
				avgUnitsDiff,
				units,
				costPerUnit,
				specData.CostAfter.GPUMonthlyRate,
				specData.CostChange.GPUMonthlyRate,
			}
			if specData.CostBefore.GPUMonthlyRate != 0 {
				row = append(row, specData.CostChange.GPUMonthlyRate/specData.CostBefore.GPUMonthlyRate*100)
			}
			t.AppendRow(row)
		}
		t.AppendSeparator()
	}

	t.AppendFooter(table.Row{
		"Total monthly cost",
		"",
		"",
		"",
		"",
		totalCostNew,
		totalCostImpact,
	})

	return t
}

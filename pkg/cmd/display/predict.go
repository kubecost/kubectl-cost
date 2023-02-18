package display

import (
	"fmt"
	"io"

	"github.com/kubecost/kubectl-cost/pkg/query"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/spf13/cobra"
)

const (
	PredictColWorkload         = "Workload"
	PredictColMoCostCPU        = "CPU/mo"
	PredictColMoCostMemory     = "Mem/mo"
	PredictColMoCostGPU        = "GPU/mo"
	PredictColMoCostTotal      = "Total/mo"
	PredictColMoCostDiffCPU    = "Δ CPU/mo"
	PredictColMoCostDiffMemory = "Δ Mem/mo"
	PredictColMoCostDiffGPU    = "Δ GPU/mo"
	PredictColMoCostDiffTotal  = "Δ Total/mo"
)

type PredictDisplayOptions struct {
	NoShowDiff bool
}

func AddPredictDisplayOptionsFlags(cmd *cobra.Command, options *PredictDisplayOptions) {
	cmd.Flags().BoolVar(&options.NoShowDiff, "no-diff", false, "Set true to not display cost difference information.")
}

func WritePredictionTable(out io.Writer, rowData []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) {
	t := MakePredictionTable(rowData, currencyCode, opts)
	t.SetOutputMirror(out)
	t.Render()
}

func MakePredictionTable(rowData []query.SpecCostDiff, currencyCode string, opts PredictDisplayOptions) table.Writer {
	t := table.NewWriter()

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name: PredictColWorkload,
		},
		{
			Name:        PredictColMoCostCPU,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostMemory,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostGPU,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostTotal,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostDiffCPU,
			Hidden:      opts.NoShowDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostDiffMemory,
			Hidden:      opts.NoShowDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostDiffGPU,
			Hidden:      opts.NoShowDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostDiffTotal,
			Hidden:      opts.NoShowDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
	})

	t.AppendHeader(table.Row{
		PredictColWorkload,
		PredictColMoCostCPU,
		PredictColMoCostMemory,
		PredictColMoCostGPU,
		PredictColMoCostTotal,
		PredictColMoCostDiffCPU,
		PredictColMoCostDiffMemory,
		PredictColMoCostDiffGPU,
		PredictColMoCostDiffTotal,
	})

	t.SortBy([]table.SortBy{
		{
			Name: PredictColMoCostTotal,
			Mode: table.DscNumeric,
		},
	})

	var summedMonthlyCPU float64
	var summedMonthlyMem float64
	var summedMonthlyGPU float64
	var summedMonthlyDiffCPU float64
	var summedMonthlyDiffMemory float64
	var summedMonthlyDiffGPU float64
	var summedMonthlyTotal float64
	var summedMonthlyDiffTotal float64

	for _, r := range rowData {
		row := table.Row{}
		row = append(row, fmt.Sprintf("%s/%s/%s", r.Namespace, r.ControllerKind, r.ControllerName))

		row = append(row, fmt.Sprintf("%.2f %s", r.CostAfter.CPUMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostAfter.RAMMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostAfter.GPUMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostAfter.TotalMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostChange.CPUMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostChange.RAMMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostChange.GPUMonthlyRate, currencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", r.CostChange.TotalMonthlyRate, currencyCode))

		summedMonthlyCPU += r.CostAfter.CPUMonthlyRate
		summedMonthlyMem += r.CostAfter.RAMMonthlyRate
		summedMonthlyGPU += r.CostAfter.GPUMonthlyRate
		summedMonthlyDiffCPU += r.CostChange.CPUMonthlyRate
		summedMonthlyDiffMemory += r.CostChange.RAMMonthlyRate
		summedMonthlyDiffGPU += r.CostChange.GPUMonthlyRate
		summedMonthlyTotal += r.CostAfter.TotalMonthlyRate
		summedMonthlyDiffTotal += r.CostChange.TotalMonthlyRate

		t.AppendRow(row)
	}

	// A summary footer is redundant if there is only one row
	if len(rowData) > 1 {
		footerRow := table.Row{}
		blankRows := 1

		for i := 0; i < blankRows; i++ {
			footerRow = append(footerRow, "")
		}
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyCPU, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyMem, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyGPU, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyTotal, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffCPU, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffMemory, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffGPU, currencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffTotal, currencyCode))
		t.AppendFooter(footerRow)
	}

	return t
}

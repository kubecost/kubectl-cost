package oldpredict

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

const (
	PredictColWorkload         = "Workload"
	PredictColReqCPU           = "CPU"
	PredictColReqMemory        = "Mem"
	PredictColReqGPU           = "GPU"
	PredictColMoCoreHours      = "Mo. core-hrs"
	PredictColMoGibHours       = "Mo. GiB-hrs"
	PredictColMoGPUHours       = "Mo. GPU-hrs"
	PredictColCostCoreHr       = "Cost/core-hr"
	PredictColCostGiBHr        = "Cost/GiB-hr"
	PredictColCostGPUHr        = "Cost/GPU-hr"
	PredictColMoCostCPU        = "CPU/mo"
	PredictColMoCostMemory     = "Mem/mo"
	PredictColMoCostGPU        = "GPU/mo"
	PredictColMoCostTotal      = "Total/mo"
	PredictColMoCostDiffCPU    = "Δ CPU/mo"
	PredictColMoCostDiffMemory = "Δ Mem/mo"
)

type PredictionTableOptions struct {
	CurrencyCode          string
	ShowCostPerResourceHr bool
	NoDiff                bool
}

func writePredictionTable(out io.Writer, rowData []predictRowData, opts PredictionTableOptions) {
	t := makePredictionTable(rowData, opts)
	t.SetOutputMirror(out)
	t.Render()
}

func makePredictionTable(rowData []predictRowData, opts PredictionTableOptions) table.Writer {
	t := table.NewWriter()

	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name: PredictColWorkload,
		},
		{
			Name: PredictColReqCPU,
		},
		{
			Name: PredictColReqMemory,
		},
		{
			Name: PredictColReqGPU,
		},
		{
			Name:   PredictColCostCoreHr,
			Hidden: !opts.ShowCostPerResourceHr,
		},
		{
			Name:   PredictColCostGiBHr,
			Hidden: !opts.ShowCostPerResourceHr,
		},
		{
			Name:   PredictColCostGPUHr,
			Hidden: !opts.ShowCostPerResourceHr,
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
			Name:        PredictColMoCostDiffCPU,
			Hidden:      opts.NoDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostDiffMemory,
			Hidden:      opts.NoDiff,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
		{
			Name:        PredictColMoCostTotal,
			Align:       text.AlignRight,
			AlignFooter: text.AlignRight,
		},
	})

	t.AppendHeader(table.Row{
		PredictColWorkload,
		PredictColReqCPU,
		PredictColReqMemory,
		PredictColReqGPU,
		PredictColCostCoreHr,
		PredictColCostGiBHr,
		PredictColCostGPUHr,
		PredictColMoCostCPU,
		PredictColMoCostMemory,
		PredictColMoCostGPU,
		PredictColMoCostDiffCPU,
		PredictColMoCostDiffMemory,
		PredictColMoCostTotal,
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
	var summedMonthlyTotal float64

	for _, rowDatum := range rowData {
		row := table.Row{}
		row = append(row, fmt.Sprintf("%s/%s/%s", rowDatum.workloadNamespace, rowDatum.workloadType, rowDatum.workloadName))
		row = append(row, rowDatum.totalCPURequested)
		row = append(row, rowDatum.totalMemoryRequested)
		row = append(row, rowDatum.totalGPURequested)

		row = append(row, fmt.Sprintf("%.4f %s", rowDatum.cpuCostMonthly/rowDatum.requestedCPUCoreHours, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.4f %s", (rowDatum.memoryCostMonthly/rowDatum.requestedMemoryByteHours)*1024*1024*1024, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.4f %s", rowDatum.gpuCostMonthly/rowDatum.requestedGPUHours, opts.CurrencyCode))

		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.cpuCostMonthly, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.memoryCostMonthly, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.gpuCostMonthly, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.cpuCostChangeMonthly, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.memoryCostChangeMonthly, opts.CurrencyCode))
		row = append(row, fmt.Sprintf("%.2f %s", rowDatum.totalCostMonthly, opts.CurrencyCode))

		summedMonthlyCPU += rowDatum.cpuCostMonthly
		summedMonthlyMem += rowDatum.memoryCostMonthly
		summedMonthlyGPU += rowDatum.gpuCostMonthly
		summedMonthlyDiffCPU += rowDatum.cpuCostChangeMonthly
		summedMonthlyDiffMemory += rowDatum.memoryCostChangeMonthly
		summedMonthlyTotal += rowDatum.totalCostMonthly

		t.AppendRow(row)
	}

	// A summary footer is redundant if there is only one row
	if len(rowData) > 1 {
		footerRow := table.Row{}
		blankRows := 7

		for i := 0; i < blankRows; i++ {
			footerRow = append(footerRow, "")
		}
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyCPU, opts.CurrencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyMem, opts.CurrencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyGPU, opts.CurrencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffCPU, opts.CurrencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyDiffMemory, opts.CurrencyCode))
		footerRow = append(footerRow, fmt.Sprintf("%.2f %s", summedMonthlyTotal, opts.CurrencyCode))
		t.AppendFooter(footerRow)
	}

	return t
}

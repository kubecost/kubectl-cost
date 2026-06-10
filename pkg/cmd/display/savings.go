package display

import (
	"fmt"
	"io"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/kubecost/kubectl-cost/pkg/query"
)

func WriteSavingsTable(out io.Writer, recs []query.RequestSizingRecommendation, currencyCode string) {
	t := MakeSavingsTable(recs, currencyCode)
	t.SetOutputMirror(out)
	t.Render()
}

func MakeSavingsTable(recs []query.RequestSizingRecommendation, currencyCode string) table.Writer {
	t := table.NewWriter()

	style := table.StyleLight
	style.Options.SeparateColumns = false
	style.Options.DrawBorder = false
	style.Options.SeparateHeader = true
	style.Title.Colors = append(style.Title.Colors, text.Bold)
	t.SetStyle(style)

	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "Namespace", Align: text.AlignLeft},
		{Name: "Controller", Align: text.AlignLeft, WidthMax: 40, WidthMaxEnforcer: text.WrapSoft},
		{Name: "Container", Align: text.AlignLeft},
		{Name: "Current CPU", Align: text.AlignRight},
		{Name: "Rec. CPU", Align: text.AlignRight},
		{Name: "Current RAM", Align: text.AlignRight},
		{Name: "Rec. RAM", Align: text.AlignRight},
		{Name: "CPU Eff.", Align: text.AlignRight},
		{Name: "RAM Eff.", Align: text.AlignRight},
		{
			Name:  "Savings/mo",
			Align: text.AlignRight,
			TransformerFooter: func(val interface{}) string {
				if f, ok := val.(float64); ok {
					return fmt.Sprintf("%.2f %s", f, currencyCode)
				}
				if s, ok := val.(string); ok {
					return s
				}
				return ""
			},
		},
	})

	t.AppendHeader(table.Row{
		"Namespace",
		"Controller",
		"Container",
		"Current CPU",
		"Rec. CPU",
		"Current RAM",
		"Rec. RAM",
		"CPU Eff.",
		"RAM Eff.",
		"Savings/mo",
	})

	// Pre-sort by total monthly savings descending
	sorted := make([]query.RequestSizingRecommendation, len(recs))
	copy(sorted, recs)
	sort.Slice(sorted, func(i, j int) bool {
		si := sorted[i].MonthlySavings.CPU + sorted[i].MonthlySavings.Memory
		sj := sorted[j].MonthlySavings.CPU + sorted[j].MonthlySavings.Memory
		return si > sj
	})

	totalSavings := 0.0
	for _, rec := range sorted {
		controller := fmt.Sprintf("%s/%s", rec.ControllerKind, rec.ControllerName)
		monthlySavings := rec.MonthlySavings.CPU + rec.MonthlySavings.Memory
		totalSavings += monthlySavings

		t.AppendRow(table.Row{
			rec.Namespace,
			controller,
			rec.ContainerName,
			rec.LatestKnownRequest.CPU,
			rec.RecommendedRequest.CPU,
			rec.LatestKnownRequest.Memory,
			rec.RecommendedRequest.Memory,
			fmt.Sprintf("%.0f%%", rec.CurrentEfficiency.CPU*100),
			fmt.Sprintf("%.0f%%", rec.CurrentEfficiency.Memory*100),
			fmt.Sprintf("%.2f %s", monthlySavings, currencyCode),
		})
	}

	t.AppendFooter(table.Row{
		"TOTAL", "", "", "", "", "", "", "", "",
		totalSavings,
	})

	return t
}

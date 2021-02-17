package cmd

import (
	"encoding/csv"
	"fmt"
	"strings"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/gdamore/tcell/v2"
	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type tuiState struct {
	do displayOptions
}

func newCmdTUI(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "interface with the kubecost API with a TUI",
		RunE: func(c *cobra.Command, args []string) error {
			if err := kubeO.Complete(c, args); err != nil {
				return err
			}
			if err := kubeO.Validate(); err != nil {
				return err
			}

			return runTUI(kubeO, displayOptions{})
		},
	}

	return cmd
}

type aggregationTableOptions struct {
	aggregation    string
	headers        []string
	titleExtractor func(string) ([]string, error)
}

func runTUI(ko *KubeOptions, do displayOptions) error {
	// box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	// if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
	// 	return fmt.Errorf("failed to start TUI: %s", err)
	// }

	app := tview.NewApplication()

	table := tview.NewTable()
	tFrame := tview.NewFrame(table)

	displayOptionsList := tview.NewList()

	var aggs map[string]query.Aggregation
	var err error
	var windowIndex int = 0
	var aggregationIndex int = 0

	windowOptions := []string{
		"1d",
		"2d",
		"7d",
	}

	aggregationOptions := []aggregationTableOptions{
		{
			aggregation:    "namespace",
			headers:        []string{"Namespace"},
			titleExtractor: noopTitleExtractor,
		},
		{
			aggregation:    "deployment",
			headers:        []string{"Namespace", "Deployment"},
			titleExtractor: deploymentTitleExtractor,
		},
	}

	requeryData := func() {
		aggs, err = query.QueryAggCostModel(ko.clientset, *ko.configFlags.Namespace, "kubecost-cost-analyzer", windowOptions[windowIndex], aggregationOptions[aggregationIndex].aggregation, "")

		// TODO: handle better
		if err != nil {
			panic(err)
		}
	}

	redrawTable := func() {
		tFrame.Clear()
		table.Clear()

		tWriter := makeAggregationRateTable(aggs, aggregationOptions[aggregationIndex].headers, aggregationOptions[aggregationIndex].titleExtractor, do)
		serializedTable := tWriter.RenderCSV()

		setTableFromCSV(table, serializedTable)

		table.SetTitle(fmt.Sprintf("%s Monthly Rate - Window %s", aggregationOptions[aggregationIndex].aggregation, windowOptions[windowIndex]))
		table.SetBorder(true)
		tFrame.SetBorder(false)
	}

	showCPU := func() {
		do.showCPUCost = !do.showCPUCost
		redrawTable()
	}

	showMemory := func() {
		do.showMemoryCost = !do.showMemoryCost
		redrawTable()
	}

	showPV := func() {
		do.showPVCost = !do.showPVCost
		redrawTable()
	}

	showGPU := func() {
		do.showGPUCost = !do.showGPUCost
		redrawTable()
	}

	showNetwork := func() {
		do.showNetworkCost = !do.showNetworkCost
		redrawTable()
	}

	changeWindow := func() {
		windowIndex = (windowIndex + 1) % len(windowOptions)
		requeryData()
		redrawTable()
	}

	changeAggregation := func() {
		aggregationIndex = (aggregationIndex + 1) % len(aggregationOptions)
		requeryData()
		redrawTable()
	}

	redrawList := func() {
		displayOptionsList.Clear()

		displayOptionsList.ShowSecondaryText(false).
			AddItem("Change Aggregation", "", 'a', changeAggregation).
			AddItem("Change Window", "", 'w', changeWindow).
			AddItem("Show CPU", "", 'c', showCPU).
			AddItem("Show Memory", "", 'm', showMemory).
			AddItem("Show PV", "", 'p', showPV).
			AddItem("Show GPU", "", 'g', showGPU).
			AddItem("Show Network", "", 'n', showNetwork)
	}

	fb := tview.NewFlex().AddItem(tFrame, 20, 1, false).AddItem(displayOptionsList, 0, 1, true)
	fb.SetDirection(tview.FlexRow)

	requeryData()
	redrawTable()
	redrawList()

	if err := app.SetRoot(fb, true).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %s", err)
	}

	return nil
}

func setTableFromCSV(table *tview.Table, csvString string) {
	// make into a Reader so we can use Golang's CSV parsing
	reader := csv.NewReader(strings.NewReader(csvString))
	parsed, err := reader.ReadAll()
	if err != nil {
		// TODO: don't panic
		panic(err)
	}

	headerColor := tcell.ColorYellow

	for rowNum, rowValue := range parsed {
		for colNum, colValue := range rowValue {
			cell := tview.NewTableCell(colValue)
			if rowNum == 0 {
				cell = cell.SetTextColor(headerColor)
			}

			table.SetCell(rowNum, colNum, cell)
		}
	}
}

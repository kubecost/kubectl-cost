package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/gdamore/tcell/v2"
	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/kubecost/opencost/pkg/kubecost"
	"github.com/kubecost/opencost/pkg/log"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type CostOptionsTUI struct {
	query.QueryBackendOptions
	displayOptions
}

func newCmdTUI(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := NewKubeOptions(streams)
	tuiO := &CostOptionsTUI{}

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

			return runTUI(kubeO, tuiO.displayOptions, tuiO.QueryBackendOptions)
		},
	}

	addKubeOptionsFlags(cmd, kubeO)
	addQueryBackendOptionsFlags(cmd, &tuiO.QueryBackendOptions)

	return cmd
}

// aggregationTableOptions is designed in order to encapsulate
// the information necessary to display aggregation options
// and update the aggregation from the TUI.
type aggregationTableOptions struct {
	aggregation    string
	headers        []string
	titleExtractor func(string) ([]string, error)
}

// this is the set of options that the TUI builds the aggregation
// selection from
var aggregationOptions = map[string]aggregationTableOptions{
	"namespace": {
		headers:        []string{"Namespace"},
		titleExtractor: noopTitleExtractor,
	},
	"deployment": {
		headers:        []string{"Namespace", "Deployment"},
		titleExtractor: deploymentTitleExtractor,
	},
	"controller": {
		headers:        []string{"Namespace", "Controller"},
		titleExtractor: controllerTitleExtractor,
	},
	"pod": {
		headers:        []string{"Namespace", "Pod"},
		titleExtractor: podTitleExtractor,
	},
}

// this is the set of options that the TUI builds the window
// selection from
var windowOptions = []string{
	"1d",
	"2d",
	"3d",
	"7d",
	"14d",
	"30d",
}

func populateDisplayOptionsList(displayOptionsList *tview.List, do *displayOptions, redrawTable func(), navigateTable func()) {
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

	showShared := func() {
		do.showSharedCost = !do.showSharedCost
		redrawTable()
	}

	displayOptionsList.ShowSecondaryText(false).
		AddItem("Show CPU", "", 'c', showCPU).
		AddItem("Show Memory", "", 'm', showMemory).
		AddItem("Show PV", "", 'p', showPV).
		AddItem("Show GPU", "", 'g', showGPU).
		AddItem("Show Network", "", 'n', showNetwork).
		AddItem("Show Shared", "", 's', showShared).
		AddItem("Navigate Table", "", 't', navigateTable).
		AddItem("ESC to change other options", "", '-', nil)
}

func buildAggregateByDropdown(aggregation *string, requeryData func()) *tview.DropDown {
	aggregationDropdown := tview.NewDropDown().SetLabel("Aggregate by: ")
	aggregationStrings := []string{}
	for agg, _ := range aggregationOptions {
		aggregationStrings = append(aggregationStrings, agg)
	}

	aggregationEvent := func(selection string, index int) {
		*aggregation = selection
		requeryData()
	}

	aggregationDropdown.SetOptions(aggregationStrings, aggregationEvent)

	return aggregationDropdown
}

func buildWindowDropdown(windowIndex *int, requeryData func()) *tview.DropDown {
	windowDropdown := tview.NewDropDown().SetLabel("Query window: ")
	windowEvent := func(selection string, index int) {
		*windowIndex = index
		requeryData()
	}
	windowDropdown.SetOptions(windowOptions, windowEvent)

	return windowDropdown
}

func runTUI(ko *KubeOptions, do displayOptions, qo query.QueryBackendOptions) error {
	app := tview.NewApplication()

	table := tview.NewTable()

	var allocations map[string]kubecost.Allocation
	var allocMutex sync.Mutex
	var lastUpdated time.Time

	var err error

	var windowIndex int = 0
	var aggregation string = "namespace"

	queryContext, queryCancel := context.WithCancel(context.Background())

	// TODO: use flags for service name
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		RestConfig:          ko.restConfig,
		Ctx:                 queryContext,
		QueryBackendOptions: qo,
	})
	if err != nil {
		return fmt.Errorf("failed to get currency code: %s", err)
	}

	redrawTable := func() {
		table.Clear()

		// This is the magic. Because go-pretty supports rendering a table as CSV,
		// we can re-use all the hard work from building the normal terminal output
		// table here. This TUI library needs us to build tables from a 2D array.
		// The CSV-rendered (string) go-pretty table, nicely sorted and everything,
		// is parsed into a 2D array and then the TUI table is built from that.
		tWriter := makeAllocationTable(aggregation, allocations, do, currencyCode, false, true)
		serializedTable := tWriter.RenderCSV()

		err := setTableFromCSV(table, serializedTable)
		if err != nil {
			log.Errorf("failed to set table from CSV: %s", err)
		}

		table.SetTitle(fmt.Sprintf(" %s Monthly Rate - Window %s - Updated %02d:%02d:%02d ", aggregation, windowOptions[windowIndex], lastUpdated.Hour(), lastUpdated.Minute(), lastUpdated.Second()))
		table.SetBorder(true)
	}

	requeryData := func() {
		// This makes requerying data async, so as to not lock up the UI on
		// large window queries. If a user selects a large window on a large
		// cluster without this, they will think the UI has crashed when it
		// is merely dealing with blocking IO, waiting on the kubecost API
		// and prometheus to aggregate a huge amount of data.
		//
		// TODO: Display an indication to the user that a query is in progress
		go func() {
			// Cancel before the lock so that a previously started query
			// crashes out. This should prevent selecting a huge window
			// from blocking the user from selecting a different window
			// before the query finishes.
			queryCancel()

			allocMutex.Lock()
			defer allocMutex.Unlock()

			queryContext, queryCancel = context.WithCancel(context.Background())

			// TODO: use flags for service name
			allocs, err := query.QueryAllocation(query.AllocationParameters{
				RestConfig:          ko.restConfig,
				Ctx:                 queryContext,
				Window:              windowOptions[windowIndex],
				Aggregate:           aggregation,
				Accumulate:          "true",
				QueryBackendOptions: qo,
			})

			allocations = allocs[0]

			if err != nil && strings.Contains(err.Error(), "context canceled") {
				// do nothing, because the context got canceled to favor a more
				// recent window request from the user
			} else if err != nil {
				log.Errorf("failed to query agg cost model: %s", err)
			} else {
				lastUpdated = time.Now()
				app.QueueUpdateDraw(func() {
					redrawTable()
				})
			}
		}()
	}

	displayOptionsList := tview.NewList()

	// SetDoneFunc sets what happens when the user hits ESC
	// or TAB (if not focused on the list). When finished
	// navigating, we go back to the main options list.
	navigate := func() {
		app.SetFocus(table)
		table.SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(displayOptionsList)
		})
	}
	populateDisplayOptionsList(displayOptionsList, &do, redrawTable, navigate)

	aggregationDropdown := buildAggregateByDropdown(&aggregation, requeryData)
	windowDropdown := buildWindowDropdown(&windowIndex, requeryData)

	// The other DoneFuncs cycle between the options selections,
	// which are the display list and the dropdowns.
	displayOptionsList.SetDoneFunc(func() {
		app.SetFocus(aggregationDropdown)
	})

	aggregationDropdown.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(windowDropdown)
	})

	windowDropdown.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(displayOptionsList)
	})

	optionsFlex := tview.NewFlex()
	optionsFlex.AddItem(displayOptionsList, 0, 1, true)

	dropDownFlex := tview.NewFlex()
	dropDownFlex.SetDirection(tview.FlexRow)
	dropDownFlex.AddItem(aggregationDropdown, 0, 1, true)
	dropDownFlex.AddItem(windowDropdown, 0, 1, true)

	optionsFlex.AddItem(dropDownFlex, 0, 1, true)

	fb := tview.NewFlex().AddItem(table, 0, 1, false).AddItem(optionsFlex, 8, 1, true)
	fb.SetDirection(tview.FlexRow)

	requeryData()

	if err := app.SetRoot(fb, true).Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %s", err)
	}

	return nil
}

func setTableFromCSV(table *tview.Table, csvString string) error {
	// make into a Reader so we can use Golang's CSV parsing
	reader := csv.NewReader(strings.NewReader(csvString))
	parsed, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV string: %s", err)
	}

	headerColor := tcell.ColorYellow

	for rowNum, rowValue := range parsed {
		for colNum, colValue := range rowValue {
			cell := tview.NewTableCell(colValue)

			// Make the header (first row) stand out
			if rowNum == 0 {
				cell = cell.SetTextColor(headerColor)
			}

			table.SetCell(rowNum, colNum, cell)
		}
	}

	return nil
}

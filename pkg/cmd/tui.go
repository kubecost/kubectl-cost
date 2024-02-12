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
	"github.com/kubecost/kubectl-cost/pkg/cmd/display"
	"github.com/kubecost/kubectl-cost/pkg/cmd/utilities"
	"github.com/kubecost/kubectl-cost/pkg/query"
	"github.com/opencost/opencost/pkg/kubecost"
	"github.com/opencost/opencost/pkg/log"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type CostOptionsTUI struct {
	query.QueryBackendOptions
	displayOptions display.AllocationDisplayOptions
}

func newCmdTUI(streams genericclioptions.IOStreams) *cobra.Command {
	kubeO := utilities.NewKubeOptions(streams)
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

			if err := tuiO.QueryBackendOptions.Complete(kubeO.RestConfig); err != nil {
				return fmt.Errorf("completing query options: %s", err)
			}
			if err := tuiO.QueryBackendOptions.Validate(); err != nil {
				return fmt.Errorf("validating query options: %s", err)
			}

			return runTUI(kubeO, tuiO.displayOptions, tuiO.QueryBackendOptions)
		},
	}

	utilities.AddKubeOptionsFlags(cmd, kubeO)
	query.AddQueryBackendOptionsFlags(cmd, &tuiO.QueryBackendOptions)

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

func deploymentTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("deployment title should have 2 fields")
	}

	return sp, nil
}

// see the results of /model/aggregatedCostModel?window=1d&aggregation=controller

func controllerTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("controller title should have 2 fields")
	}

	return sp, nil
}

func podTitleExtractor(aggregationName string) ([]string, error) {
	sp := strings.Split(aggregationName, "/")

	if len(sp) != 2 {
		return nil, fmt.Errorf("pod title should have 2 fields")
	}

	return sp, nil
}

func noopTitleExtractor(aggregationName string) ([]string, error) {
	return []string{aggregationName}, nil
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

func populateDisplayOptionsList(displayOptionsList *tview.List, do *display.AllocationDisplayOptions, redrawTable func(), navigateTable func()) {
	showCPU := func() {
		do.ShowCPUCost = !do.ShowCPUCost
		redrawTable()
	}

	showMemory := func() {
		do.ShowMemoryCost = !do.ShowMemoryCost
		redrawTable()
	}

	showPV := func() {
		do.ShowPVCost = !do.ShowPVCost
		redrawTable()
	}

	showGPU := func() {
		do.ShowGPUCost = !do.ShowGPUCost
		redrawTable()
	}

	showNetwork := func() {
		do.ShowNetworkCost = !do.ShowNetworkCost
		redrawTable()
	}

	showShared := func() {
		do.ShowSharedCost = !do.ShowSharedCost
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

func buildAggregateByDropdown(aggregation *[]string, requeryData func()) *tview.DropDown {
	aggregationDropdown := tview.NewDropDown().SetLabel("Aggregate by: ")
	aggregationStrings := []string{}
	for agg, _ := range aggregationOptions {
		aggregationStrings = append(aggregationStrings, agg)
	}

	aggregationEvent := func(selection string, index int) {
		if selection == "namespace" {
			*aggregation = []string{"cluster", "namespace"}
		} else {
			*aggregation = []string{"cluster", "namespace", selection}
		}
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

func runTUI(ko *utilities.KubeOptions, do display.AllocationDisplayOptions, qo query.QueryBackendOptions) error {
	app := tview.NewApplication()

	table := tview.NewTable()

	var allocations map[string]kubecost.Allocation
	var allocMutex sync.Mutex
	var lastUpdated time.Time

	var err error

	var windowIndex int = 0
	aggregation := []string{"cluster", "namespace"}

	queryContext, queryCancel := context.WithCancel(context.Background())

	// TODO: use flags for service name
	currencyCode, err := query.QueryCurrencyCode(query.CurrencyCodeParameters{
		Ctx:                 queryContext,
		QueryBackendOptions: qo,
	})
	if err != nil {
		log.Debugf("failed to get currency code, displaying as empty string: %s", err)
		currencyCode = ""
	}

	redrawTable := func() {
		table.Clear()

		// This is the magic. Because go-pretty supports rendering a table as CSV,
		// we can re-use all the hard work from building the normal terminal output
		// table here. This TUI library needs us to build tables from a 2D array.
		// The CSV-rendered (string) go-pretty table, nicely sorted and everything,
		// is parsed into a 2D array and then the TUI table is built from that.
		tWriter := display.MakeAllocationTable(aggregation, allocations, do, currencyCode, true)
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
			queriedAllocs, err := query.QueryAllocation(query.AllocationParameters{
				Ctx: queryContext,
				QueryParams: map[string]string{
					"window":      windowOptions[windowIndex],
					"aggregate":   strings.Join(aggregation, ","),
					"accumulate":  "true",
					"includeIdle": "true",
				},
				QueryBackendOptions: qo,
			})

			if err != nil && strings.Contains(err.Error(), "context canceled") {
				// do nothing, because the context got canceled to favor a more
				// recent window request from the user
			} else if err != nil {
				log.Errorf("failed to query agg cost model: %s", err)
			} else if len(queriedAllocs) == 0 {
				log.Errorf("Allocation response was empty. Not updating the table.")
			} else {
				allocations = queriedAllocs[0]

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

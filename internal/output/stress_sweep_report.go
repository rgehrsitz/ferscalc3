package output

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/domain"
)

//go:embed templates/stress_sweep_report.html.tmpl
var stressSweepTemplate string

// StressSweepHTMLReport renders an overlay report showing a stress scenario swept across offsets.
type StressSweepHTMLReport struct {
	Config        calculation.FERSMonteCarloConfig
	Scenario      *domain.StressScenario
	Results       []StressSweepRun
	DisplayName   string
	Description   string
	ProjectionLen int
}

// GenerateHTMLReport writes the overlay HTML file for the sweep results.
func (r *StressSweepHTMLReport) GenerateHTMLReport(path string) error {
	if len(r.Results) == 0 {
		return fmt.Errorf("no sweep results to render")
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create sweep html directory: %w", err)
		}
	}

	data, err := r.buildTemplateData()
	if err != nil {
		return err
	}

	tmpl, err := template.New("stress_sweep").Parse(stressSweepTemplate)
	if err != nil {
		return fmt.Errorf("parse sweep template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute sweep template: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write sweep html: %w", err)
	}

	return nil
}

func (r *StressSweepHTMLReport) buildTemplateData() (stressSweepTemplateData, error) {
	type row struct {
		offset          int
		displayRow      sweepSummaryRow
		netSeries       overlaySeries
		tspSeries       overlaySeries
		medianNetIncome decimal.Decimal
		years           []int
	}

	var builtRows []row
	for _, run := range r.Results {
		offsetLabel := fmt.Sprintf("Offset %d yrs", run.Offset)
		builtRows = append(builtRows, row{
			offset: run.Offset,
			displayRow: sweepSummaryRow{
				OffsetLabel:        offsetLabel,
				SuccessRate:        formatPercent(run.SuccessRate),
				MedianNetIncome:    formatCurrency(run.MedianNetIncome),
				MedianFinalTSP:     formatCurrency(run.MedianFinalTSP),
				TSPDepletionRate:   formatPercent(run.TSPDepletionRate),
				IncomeVolatility:   formatCurrency(run.IncomeVolatility),
				BestCaseNetIncome:  formatCurrency(run.BestCaseNetIncome),
				WorstCaseNetIncome: formatCurrency(run.WorstCaseNetIncome),
			},
			medianNetIncome: run.MedianNetIncome,
			years:           append([]int(nil), run.Years...),
			netSeries: overlaySeries{
				Label: fmt.Sprintf("Offset %d yrs", run.Offset),
				Data:  append([]float64(nil), run.NetMedianSeries...),
			},
			tspSeries: overlaySeries{
				Label: fmt.Sprintf("Offset %d yrs", run.Offset),
				Data:  append([]float64(nil), run.TSPMedianSeries...),
			},
		})
	}

	if len(builtRows) == 0 {
		return stressSweepTemplateData{}, fmt.Errorf("no valid sweep rows")
	}

	sort.Slice(builtRows, func(i, j int) bool {
		return builtRows[i].offset < builtRows[j].offset
	})

	var templateRows []sweepSummaryRow
	netChart := overlayChartData{Series: []overlaySeries{}}
	tspChart := overlayChartData{Series: []overlaySeries{}}

	var worstIdx int
	var worstMedian decimal.Decimal
	worstSet := false
	for idx, entry := range builtRows {
		if !worstSet || entry.medianNetIncome.LessThan(worstMedian) {
			worstMedian = entry.medianNetIncome
			worstIdx = idx
			worstSet = true
		}
	}

	for idx, entry := range builtRows {
		entry.displayRow.IsWorst = idx == worstIdx
		templateRows = append(templateRows, entry.displayRow)

		entry.netSeries.Highlighted = idx == worstIdx
		entry.tspSeries.Highlighted = idx == worstIdx

		if len(netChart.Labels) == 0 {
			netChart.Labels = append(netChart.Labels, entry.years...)
			tspChart.Labels = append(tspChart.Labels, entry.years...)
		}

		netChart.Series = append(netChart.Series, entry.netSeries)
		tspChart.Series = append(tspChart.Series, entry.tspSeries)
	}

	netChartJSON, err := toTemplateJS(netChart)
	if err != nil {
		return stressSweepTemplateData{}, err
	}
	tspChartJSON, err := toTemplateJS(tspChart)
	if err != nil {
		return stressSweepTemplateData{}, err
	}

	title := r.DisplayName
	if title == "" && r.Scenario != nil {
		title = r.Scenario.Name
	}
	if title == "" {
		title = "Stress Scenario Offset Sweep"
	}

	return stressSweepTemplateData{
		ScenarioTitle: fmt.Sprintf("🎯 %s — Offset Sweep", title),
		Description:   r.Description,
		GeneratedAt:   time.Now().Format("January 2, 2006 3:04 PM"),
		SummaryRows:   templateRows,
		NetChartData:  netChartJSON,
		TSPChartData:  tspChartJSON,
		OffsetCount:   len(templateRows),
	}, nil
}

type stressSweepTemplateData struct {
	ScenarioTitle string
	Description   string
	GeneratedAt   string
	SummaryRows   []sweepSummaryRow
	NetChartData  template.JS
	TSPChartData  template.JS
	OffsetCount   int
}

type sweepSummaryRow struct {
	OffsetLabel        string
	SuccessRate        string
	MedianNetIncome    string
	MedianFinalTSP     string
	TSPDepletionRate   string
	IncomeVolatility   string
	BestCaseNetIncome  string
	WorstCaseNetIncome string
	IsWorst            bool
}

func (row sweepSummaryRow) worstScore() decimal.Decimal {
	val, err := decimal.NewFromString(strings.ReplaceAll(row.MedianNetIncome, "$", ""))
	if err != nil {
		return decimal.Zero
	}
	return val
}

type overlayChartData struct {
	Labels []int           `json:"labels"`
	Series []overlaySeries `json:"series"`
}

type overlaySeries struct {
	Label       string    `json:"label"`
	Data        []float64 `json:"data"`
	Highlighted bool      `json:"highlighted"`
}

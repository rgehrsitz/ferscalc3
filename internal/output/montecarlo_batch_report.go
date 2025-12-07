package output

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/shopspring/decimal"
)

//go:embed templates/montecarlo_batch_report.html.tmpl
var monteCarloBatchTemplate string

// ScenarioBatchResult represents the output of a single stress scenario run.
type ScenarioBatchResult struct {
	Key          string                            `json:"key"`
	DisplayName  string                            `json:"display_name"`
	Description  string                            `json:"description"`
	Result       *calculation.FERSMonteCarloResult `json:"result"`
	SweepResults []StressSweepRun                  `json:"sweep_results,omitempty"`
}

// StressSweepRun captures the result of a scenario run at a specific offset.
type StressSweepRun struct {
	Offset             int             `json:"offset"`
	SuccessRate        decimal.Decimal `json:"success_rate"`
	MedianNetIncome    decimal.Decimal `json:"median_net_income"`
	MedianFinalTSP     decimal.Decimal `json:"median_final_tsp"`
	TSPDepletionRate   decimal.Decimal `json:"tsp_depletion_rate"`
	IncomeVolatility   decimal.Decimal `json:"income_volatility"`
	BestCaseNetIncome  decimal.Decimal `json:"best_case_net_income"`
	WorstCaseNetIncome decimal.Decimal `json:"worst_case_net_income"`
	Years              []int           `json:"years,omitempty"`
	NetMedianSeries    []float64       `json:"net_median_series,omitempty"`
	TSPMedianSeries    []float64       `json:"tsp_median_series,omitempty"`
}

// MonteCarloBatchReport renders combined reports for multiple scenarios.
type MonteCarloBatchReport struct {
	Scenarios []ScenarioBatchResult
}

// GenerateHTMLReport renders the combined HTML report.
func (m *MonteCarloBatchReport) GenerateHTMLReport(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create html directory: %w", err)
	}

	data := m.buildTemplateData()
	tmpl, err := template.New("montecarlo_batch").Parse(monteCarloBatchTemplate)
	if err != nil {
		return fmt.Errorf("parse batch template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create html file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("execute batch template: %w", err)
	}

	return nil
}

// GenerateCSVReport writes a summary CSV of all scenarios.
func (m *MonteCarloBatchReport) GenerateCSVReport(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create csv directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create csv file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{
		"Scenario",
		"Description",
		"Success Rate",
		"Median Net Income",
		"Median Final TSP Balance",
		"TSP Depletion Rate",
		"Income Volatility",
		"Best Case Net Income",
		"Worst Case Net Income",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, scenario := range m.Scenarios {
		if scenario.Result == nil {
			continue
		}
		row := []string{
			scenario.DisplayName,
			scenario.Description,
			formatPercent(scenario.Result.SuccessRate),
			formatCurrency(scenario.Result.MedianNetIncome),
			formatCurrency(scenario.Result.MedianFinalTSPBalance),
			formatPercent(scenario.Result.TSPDepletionRate),
			formatCurrency(scenario.Result.IncomeVolatility),
			formatCurrency(scenario.Result.BestCaseScenario),
			formatCurrency(scenario.Result.WorstCaseScenario),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	return nil
}

// GenerateJSONReport writes a machine-readable summary containing full results.
func (m *MonteCarloBatchReport) GenerateJSONReport(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create json directory: %w", err)
	}

	payload := m.buildJSONPayload()
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal batch json: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write batch json: %w", err)
	}

	return nil
}

type batchTemplateData struct {
	GeneratedAt      string
	ScenarioCount    int
	ScenarioRows     []batchScenarioRow
	BestSuccessRate  string
	BestMedianIncome string
}

type batchScenarioRow struct {
	Name               string
	Description        string
	SuccessRate        string
	MedianNetIncome    string
	MedianFinalTSP     string
	TSPDepletionRate   string
	IncomeVolatility   string
	BestCaseNetIncome  string
	WorstCaseNetIncome string
}

func (m *MonteCarloBatchReport) buildTemplateData() batchTemplateData {
	rows := make([]batchScenarioRow, 0, len(m.Scenarios))
	bestSuccessDecimal := decimal.NewFromInt(-1)
	bestMedianIncomeDecimal := decimal.NewFromInt(-1)
	bestSuccessDisplay := "N/A"
	bestMedianDisplay := "N/A"

	for _, scenario := range m.Scenarios {
		if scenario.Result == nil {
			continue
		}
		successDecimal := scenario.Result.SuccessRate
		medianDecimal := scenario.Result.MedianNetIncome
		rows = append(rows, batchScenarioRow{
			Name:               scenario.DisplayName,
			Description:        scenario.Description,
			SuccessRate:        formatPercent(scenario.Result.SuccessRate),
			MedianNetIncome:    formatCurrency(scenario.Result.MedianNetIncome),
			MedianFinalTSP:     formatCurrency(scenario.Result.MedianFinalTSPBalance),
			TSPDepletionRate:   formatPercent(scenario.Result.TSPDepletionRate),
			IncomeVolatility:   formatCurrency(scenario.Result.IncomeVolatility),
			BestCaseNetIncome:  formatCurrency(scenario.Result.BestCaseScenario),
			WorstCaseNetIncome: formatCurrency(scenario.Result.WorstCaseScenario),
		})

		if successDecimal.GreaterThan(bestSuccessDecimal) {
			bestSuccessDecimal = successDecimal
			bestSuccessDisplay = formatPercent(successDecimal)
		}
		if medianDecimal.GreaterThan(bestMedianIncomeDecimal) {
			bestMedianIncomeDecimal = medianDecimal
			bestMedianDisplay = formatCurrency(medianDecimal)
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Name < rows[j].Name
	})

	return batchTemplateData{
		GeneratedAt:      time.Now().Format("January 2, 2006 3:04 PM"),
		ScenarioCount:    len(rows),
		ScenarioRows:     rows,
		BestSuccessRate:  bestSuccessDisplay,
		BestMedianIncome: bestMedianDisplay,
	}
}

type batchJSONPayload struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	ScenarioCount  int                 `json:"scenario_count"`
	ScenarioValues []batchJSONScenario `json:"scenarios"`
}

type batchJSONScenario struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Result      *calculation.FERSMonteCarloResult `json:"result"`
	SweepRuns   []StressSweepRun                  `json:"sweep_results,omitempty"`
}

func (m *MonteCarloBatchReport) buildJSONPayload() batchJSONPayload {
	scenarios := make([]batchJSONScenario, 0, len(m.Scenarios))
	for _, scenario := range m.Scenarios {
		if scenario.Result == nil {
			continue
		}
		scenarios = append(scenarios, batchJSONScenario{
			Name:        scenario.DisplayName,
			Description: scenario.Description,
			Result:      scenario.Result,
			SweepRuns:   scenario.SweepResults,
		})
	}
	return batchJSONPayload{
		GeneratedAt:    time.Now(),
		ScenarioCount:  len(scenarios),
		ScenarioValues: scenarios,
	}
}

func formatPercent(value decimal.Decimal) string {
	return fmt.Sprintf("%.1f%%", value.Mul(decimal.NewFromInt(100)).InexactFloat64())
}

func formatCurrency(value decimal.Decimal) string {
	if value.IsZero() {
		return "$0"
	}
	str := value.StringFixed(0)
	str = addThousandsSeparator(str)
	return "$" + str
}

func addThousandsSeparator(value string) string {
	if value == "" {
		return value
	}
	negative := strings.HasPrefix(value, "-")
	if negative {
		value = value[1:]
	}

	var parts []byte
	count := 0
	for i := len(value) - 1; i >= 0; i-- {
		parts = append(parts, value[i])
		count++
		if count == 3 && i != 0 {
			parts = append(parts, ',')
			count = 0
		}
	}

	// reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	result := string(parts)
	if negative {
		result = "-" + result
	}
	return result
}

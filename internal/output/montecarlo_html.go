package output

import (
	"bytes"
	_ "embed"
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

//go:embed templates/montecarlo_report.html.tmpl
var monteCarloReportTemplate string

// MonteCarloHTMLReport generates an interactive HTML report for FERS Monte Carlo results
type MonteCarloHTMLReport struct {
	Result *calculation.FERSMonteCarloResult
	Config calculation.FERSMonteCarloConfig
}

// GenerateHTMLReport creates an interactive HTML report with charts
func (m *MonteCarloHTMLReport) GenerateHTMLReport(outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	htmlContent, err := m.generateHTMLContent()
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("failed to write HTML report: %w", err)
	}

	return nil
}

func (m *MonteCarloHTMLReport) generateHTMLContent() (string, error) {
	data, err := m.buildTemplateData()
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("montecarlo_report").Parse(monteCarloReportTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse report template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute report template: %w", err)
	}

	return buf.String(), nil
}

func (m *MonteCarloHTMLReport) buildTemplateData() (reportTemplateData, error) {
	if m.Result == nil {
		return reportTemplateData{}, fmt.Errorf("monte carlo result is nil")
	}

	title := "🎯 FERS Monte Carlo Analysis"
	subtitle := "Comprehensive Retirement Scenario Analysis"
	if scenarioName := strings.TrimSpace(m.Result.StressScenarioName); scenarioName != "" {
		title = fmt.Sprintf("🎯 %s", scenarioName)
		subtitle = "Stress Scenario Monte Carlo Analysis"
	} else if scenarioName := strings.TrimSpace(m.Config.StressScenarioName); scenarioName != "" {
		title = fmt.Sprintf("🎯 %s", scenarioName)
		subtitle = "Stress Scenario Monte Carlo Analysis"
	}

	successRatePercent := decimalToFloat(m.Result.SuccessRate.Mul(decimal.NewFromFloat(100)))

	netIncomeChart, err := m.generateNetIncomeHistogram()
	if err != nil {
		return reportTemplateData{}, err
	}
	netIncomeChartJSON, err := toTemplateJS(netIncomeChart)
	if err != nil {
		return reportTemplateData{}, err
	}

	tspBalanceChart, err := m.generateTSPBalanceHistogram()
	if err != nil {
		return reportTemplateData{}, err
	}
	tspBalanceChartJSON, err := toTemplateJS(tspBalanceChart)
	if err != nil {
		return reportTemplateData{}, err
	}

	percentileChart := m.generatePercentileChartData()
	percentileChartJSON, err := toTemplateJS(percentileChart)
	if err != nil {
		return reportTemplateData{}, err
	}

	netIncomeTimeSeries, tspTimeSeries := m.generateTimeSeriesData()
	netIncomeTimeSeriesJSON, err := toTemplateJS(netIncomeTimeSeries)
	if err != nil {
		return reportTemplateData{}, err
	}
	tspTimeSeriesJSON, err := toTemplateJS(tspTimeSeries)
	if err != nil {
		return reportTemplateData{}, err
	}

	data := reportTemplateData{
		ScenarioTitle:          title,
		ScenarioSubtitle:       subtitle,
		SuccessClass:           m.getSuccessRateClass(),
		SuccessRateDisplay:     fmt.Sprintf("%.1f%%", successRatePercent),
		MedianNetIncomeDisplay: "$" + m.formatCurrency(m.Result.MedianNetIncome),
		SimulationCount:        m.Result.NumSimulations,
		RiskLevel:              m.getRiskLevel(),
		Percentiles: percentileView{
			P10: "$" + m.formatCurrency(m.Result.NetIncomePercentiles.P10),
			P25: "$" + m.formatCurrency(m.Result.NetIncomePercentiles.P25),
			P50: "$" + m.formatCurrency(m.Result.NetIncomePercentiles.P50),
			P75: "$" + m.formatCurrency(m.Result.NetIncomePercentiles.P75),
			P90: "$" + m.formatCurrency(m.Result.NetIncomePercentiles.P90),
		},
		PrimaryConcerns:     m.getPrimaryConcerns(),
		MarketSensitivity:   m.getMarketSensitivity(),
		Recommendations:     m.generateRecommendations(),
		GeneratedAt:         time.Now().Format("January 2, 2006 3:04 PM"),
		NetIncomeChartData:  netIncomeChartJSON,
		TSPBalanceChartData: tspBalanceChartJSON,
		PercentileChartData: percentileChartJSON,
		NetIncomeTimeSeries: netIncomeTimeSeriesJSON,
		TSPTimeSeries:       tspTimeSeriesJSON,
	}

	return data, nil
}

func (m *MonteCarloHTMLReport) generateNetIncomeHistogram() (histogramChartData, error) {
	values := make([]decimal.Decimal, 0, len(m.Result.Simulations))
	for _, sim := range m.Result.Simulations {
		if sim.Success {
			values = append(values, sim.NetIncomeMetrics.AverageNetIncome)
		}
	}
	return m.histogramChartData(values, "Simulations", "rgba(52, 152, 219, 0.6)", "rgba(52, 152, 219, 1)")
}

func (m *MonteCarloHTMLReport) generateTSPBalanceHistogram() (histogramChartData, error) {
	values := make([]decimal.Decimal, 0, len(m.Result.Simulations))
	for _, sim := range m.Result.Simulations {
		if sim.Success {
			estimatedBalance := sim.NetIncomeMetrics.AverageNetIncome.Mul(decimal.NewFromInt(int64(sim.TSPMetrics.Longevity)))
			values = append(values, estimatedBalance)
		}
	}
	return m.histogramChartData(values, "Simulations", "rgba(39, 174, 96, 0.6)", "rgba(39, 174, 96, 1)")
}

func (m *MonteCarloHTMLReport) histogramChartData(values []decimal.Decimal, label, backgroundColor, borderColor string) (histogramChartData, error) {
	data := histogramChartData{
		Labels: make([]string, 0),
		Datasets: []chartDataset{{
			Label:           label,
			Data:            make([]int, 0),
			BackgroundColor: backgroundColor,
			BorderColor:     borderColor,
			BorderWidth:     1,
		}},
	}

	if len(values) == 0 {
		return data, nil
	}

	bins := m.createHistogramBins(values, 10)
	labels := make([]string, len(bins))
	counts := make([]int, len(bins))
	for i, bin := range bins {
		labels[i] = "$" + bin.Label
		counts[i] = bin.Count
	}

	data.Labels = labels
	data.Datasets[0].Data = counts

	return data, nil
}

func (m *MonteCarloHTMLReport) generatePercentileChartData() []float64 {
	return []float64{
		decimalToFloat(m.Result.NetIncomePercentiles.P10),
		decimalToFloat(m.Result.NetIncomePercentiles.P25),
		decimalToFloat(m.Result.NetIncomePercentiles.P50),
		decimalToFloat(m.Result.NetIncomePercentiles.P75),
		decimalToFloat(m.Result.NetIncomePercentiles.P90),
	}
}

func (m *MonteCarloHTMLReport) generateTimeSeriesData() (timeSeriesData, timeSeriesData) {
	if len(m.Result.Simulations) == 0 {
		return newTimeSeriesData(0), newTimeSeriesData(0)
	}

	firstSim := m.Result.Simulations[0]
	if len(firstSim.ScenarioResults) == 0 || len(firstSim.ScenarioResults[0].Projection) == 0 {
		return newTimeSeriesData(0), newTimeSeriesData(0)
	}

	projectionLength := len(firstSim.ScenarioResults[0].Projection)
	netData := newTimeSeriesData(projectionLength)
	tspData := newTimeSeriesData(projectionLength)

	yearlyNetIncomes := make([][]decimal.Decimal, projectionLength)
	yearlyTSPBalances := make([][]decimal.Decimal, projectionLength)

	for _, sim := range m.Result.Simulations {
		if len(sim.ScenarioResults) == 0 {
			continue
		}
		scenario := sim.ScenarioResults[0]
		for yearIdx, yearData := range scenario.Projection {
			if yearIdx >= projectionLength {
				continue
			}
			if yearlyNetIncomes[yearIdx] == nil {
				yearlyNetIncomes[yearIdx] = make([]decimal.Decimal, 0, len(m.Result.Simulations))
				yearlyTSPBalances[yearIdx] = make([]decimal.Decimal, 0, len(m.Result.Simulations))
				netData.Years[yearIdx] = yearData.Date.Year()
				tspData.Years[yearIdx] = yearData.Date.Year()
			}
			yearlyNetIncomes[yearIdx] = append(yearlyNetIncomes[yearIdx], yearData.NetIncome)
			totalTSPBalance := yearData.TSPBalancePersonA.Add(yearData.TSPBalancePersonB)
			yearlyTSPBalances[yearIdx] = append(yearlyTSPBalances[yearIdx], totalTSPBalance)
		}
	}

	percentileTargets := []float64{0.10, 0.25, 0.50, 0.75, 0.90}
	for yearIdx := 0; yearIdx < projectionLength; yearIdx++ {
		for pctIdx, pct := range percentileTargets {
			netData.setPercentileValue(pctIdx, yearIdx, percentileFloat(yearlyNetIncomes[yearIdx], pct))
			tspData.setPercentileValue(pctIdx, yearIdx, percentileFloat(yearlyTSPBalances[yearIdx], pct))
		}
	}

	return netData, tspData
}

func (m *MonteCarloHTMLReport) getSuccessRateClass() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	switch {
	case rateFloat >= 90:
		return "success"
	case rateFloat >= 70:
		return "warning"
	default:
		return "danger"
	}
}

func (m *MonteCarloHTMLReport) getRiskLevel() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	switch {
	case rateFloat >= 90:
		return "🟢 Low"
	case rateFloat >= 70:
		return "🟡 Moderate"
	default:
		return "🔴 High"
	}
}

func (m *MonteCarloHTMLReport) getPrimaryConcerns() string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()
	switch {
	case rateFloat >= 90:
		return "Minimal concerns. Your retirement plan appears robust."
	case rateFloat >= 70:
		return "Market volatility could impact retirement income. Consider conservative strategies."
	default:
		return "Significant risk of income shortfall. Immediate action recommended."
	}
}

func (m *MonteCarloHTMLReport) getMarketSensitivity() string {
	median := m.Result.MedianNetIncome
	if median.IsZero() {
		return "Unable to determine"
	}

	incomeRange := m.Result.NetIncomePercentiles.P90.Sub(m.Result.NetIncomePercentiles.P10)
	cv := incomeRange.Div(median)
	cvFloat, _ := cv.Float64()

	switch {
	case cvFloat < 0.5:
		return "Low - Income is relatively stable across market conditions"
	case cvFloat < 1.0:
		return "Moderate - Income varies with market performance"
	default:
		return "High - Income is highly sensitive to market conditions"
	}
}

func (m *MonteCarloHTMLReport) generateRecommendations() []string {
	rate := m.Result.SuccessRate.Mul(decimal.NewFromFloat(100))
	rateFloat, _ := rate.Float64()

	var recommendations []string
	if rateFloat < 90 {
		recommendations = append(recommendations,
			"Consider increasing TSP contributions to improve retirement security",
			"Review withdrawal strategies to optimize income sustainability",
		)
	}

	if rateFloat < 70 {
		recommendations = append(recommendations,
			"Consider delaying retirement to increase benefits",
			"Explore additional income sources or part-time work",
			"Consult with a financial advisor for personalized planning",
		)
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Maintain current retirement strategy",
			"Regularly review and adjust plan as circumstances change",
		)
	}
	return recommendations
}

func (m *MonteCarloHTMLReport) formatCurrency(amount decimal.Decimal) string {
	return amount.StringFixed(0)
}

// HistogramBin represents a bin in a histogram
type HistogramBin struct {
	Label string
	Count int
	Min   decimal.Decimal
	Max   decimal.Decimal
}

func (m *MonteCarloHTMLReport) createHistogramBins(values []decimal.Decimal, numBins int) []HistogramBin {
	if len(values) == 0 {
		return []HistogramBin{}
	}

	min := values[0]
	max := values[0]
	for _, v := range values {
		if v.LessThan(min) {
			min = v
		}
		if v.GreaterThan(max) {
			max = v
		}
	}

	if min.Equal(max) {
		return []HistogramBin{
			{
				Label: min.Div(decimal.NewFromInt(1000)).StringFixed(0) + "K",
				Min:   min,
				Max:   max,
				Count: len(values),
			},
		}
	}

	rangeValue := max.Sub(min)
	binWidth := rangeValue.Div(decimal.NewFromInt(int64(numBins)))
	if binWidth.IsZero() {
		binWidth = decimal.NewFromInt(1)
	}

	bins := make([]HistogramBin, numBins)
	for i := 0; i < numBins; i++ {
		binMin := min.Add(binWidth.Mul(decimal.NewFromInt(int64(i))))
		binMax := binMin.Add(binWidth)

		bins[i] = HistogramBin{
			Label: binMin.Div(decimal.NewFromInt(1000)).StringFixed(0) + "K",
			Min:   binMin,
			Max:   binMax,
			Count: 0,
		}
	}

	for _, value := range values {
		for i := range bins {
			isLast := i == len(bins)-1
			if value.GreaterThanOrEqual(bins[i].Min) && (isLast || value.LessThan(bins[i].Max)) {
				bins[i].Count++
				break
			}
		}
	}

	return bins
}

func percentileFloat(values []decimal.Decimal, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	floatValues := make([]float64, len(values))
	for i, v := range values {
		floatValues[i] = decimalToFloat(v)
	}

	sort.Float64s(floatValues)

	if len(floatValues) == 1 {
		return floatValues[0]
	}

	position := percentile * float64(len(floatValues)-1)
	lowerIndex := int(position)
	upperIndex := lowerIndex + 1

	if upperIndex >= len(floatValues) {
		return floatValues[len(floatValues)-1]
	}

	if position == float64(lowerIndex) {
		return floatValues[lowerIndex]
	}

	weight := position - float64(lowerIndex)
	lower := floatValues[lowerIndex]
	upper := floatValues[upperIndex]

	return lower + (upper-lower)*weight
}

func decimalToFloat(value decimal.Decimal) float64 {
	f, _ := value.Float64()
	return f
}

func toTemplateJS(v interface{}) (template.JS, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(data), nil
}

type reportTemplateData struct {
	ScenarioTitle          string
	ScenarioSubtitle       string
	SuccessClass           string
	SuccessRateDisplay     string
	MedianNetIncomeDisplay string
	SimulationCount        int
	RiskLevel              string
	Percentiles            percentileView
	PrimaryConcerns        string
	MarketSensitivity      string
	Recommendations        []string
	GeneratedAt            string
	NetIncomeChartData     template.JS
	TSPBalanceChartData    template.JS
	PercentileChartData    template.JS
	NetIncomeTimeSeries    template.JS
	TSPTimeSeries          template.JS
}

type percentileView struct {
	P10 string
	P25 string
	P50 string
	P75 string
	P90 string
}

type histogramChartData struct {
	Labels   []string       `json:"labels"`
	Datasets []chartDataset `json:"datasets"`
}

type chartDataset struct {
	Label           string `json:"label"`
	Data            []int  `json:"data"`
	BackgroundColor string `json:"backgroundColor"`
	BorderColor     string `json:"borderColor"`
	BorderWidth     int    `json:"borderWidth"`
}

type timeSeriesData struct {
	Years []int     `json:"years"`
	P10   []float64 `json:"p10"`
	P25   []float64 `json:"p25"`
	P50   []float64 `json:"p50"`
	P75   []float64 `json:"p75"`
	P90   []float64 `json:"p90"`
}

func newTimeSeriesData(length int) timeSeriesData {
	return timeSeriesData{
		Years: make([]int, length),
		P10:   make([]float64, length),
		P25:   make([]float64, length),
		P50:   make([]float64, length),
		P75:   make([]float64, length),
		P90:   make([]float64, length),
	}
}

func (ts *timeSeriesData) setPercentileValue(percentileIndex, yearIndex int, value float64) {
	switch percentileIndex {
	case 0:
		ts.P10[yearIndex] = value
	case 1:
		ts.P25[yearIndex] = value
	case 2:
		ts.P50[yearIndex] = value
	case 3:
		ts.P75[yearIndex] = value
	case 4:
		ts.P90[yearIndex] = value
	}
}

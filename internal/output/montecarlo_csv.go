package output

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/shopspring/decimal"
)

// MonteCarloCSVReport generates CSV exports for FERS Monte Carlo results
type MonteCarloCSVReport struct {
	Result *calculation.FERSMonteCarloResult
	Config calculation.FERSMonteCarloConfig
}

// GenerateSummaryCSV creates a summary CSV with aggregate statistics
func (m *MonteCarloCSVReport) GenerateSummaryCSV(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Metric", "Value", "Description",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write summary data
	summaryData := [][]string{
		{"Success Rate", fmt.Sprintf("%.2f%%", m.Result.SuccessRate.Mul(decimal.NewFromFloat(100)).InexactFloat64()), "Percentage of successful simulations"},
		{"Median Net Income", fmt.Sprintf("$%s", m.Result.MedianNetIncome.StringFixed(0)), "Median annual net income across all simulations"},
		{"Income Volatility", fmt.Sprintf("$%s", m.Result.IncomeVolatility.StringFixed(0)), "Standard deviation of net income"},
		{"TSP Depletion Rate", fmt.Sprintf("%.2f%%", m.Result.TSPDepletionRate.Mul(decimal.NewFromFloat(100)).InexactFloat64()), "Percentage of simulations where TSP was depleted"},
		{"Median TSP Longevity", fmt.Sprintf("%s years", m.Result.TSPLongevityPercentiles.P50.StringFixed(0)), "Median years until TSP depletion"},
		{"10th Percentile Income", fmt.Sprintf("$%s", m.Result.NetIncomePercentiles.P10.StringFixed(0)), "10th percentile of net income"},
		{"25th Percentile Income", fmt.Sprintf("$%s", m.Result.NetIncomePercentiles.P25.StringFixed(0)), "25th percentile of net income"},
		{"75th Percentile Income", fmt.Sprintf("$%s", m.Result.NetIncomePercentiles.P75.StringFixed(0)), "75th percentile of net income"},
		{"90th Percentile Income", fmt.Sprintf("$%s", m.Result.NetIncomePercentiles.P90.StringFixed(0)), "90th percentile of net income"},
		{"Number of Simulations", strconv.Itoa(m.Config.NumSimulations), "Total number of simulations run"},
		{"Data Source", map[bool]string{true: "Historical", false: "Statistical"}[m.Config.UseHistorical], "Source of market data"},
	}

	for _, row := range summaryData {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write data row: %w", err)
		}
	}

	return nil
}

// GenerateDetailedCSV creates a detailed CSV with individual simulation results
func (m *MonteCarloCSVReport) GenerateDetailedCSV(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"SimulationID",
		"Success",
		"FirstYearNetIncome",
		"Year5NetIncome",
		"Year10NetIncome",
		"MinNetIncome",
		"MaxNetIncome",
		"AverageNetIncome",
		"TSPLongevity",
		"TSPDepleted",
		"MarketConditions",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write individual simulation data
	for _, sim := range m.Result.Simulations {
		row := []string{
			strconv.Itoa(sim.SimulationID),
			strconv.FormatBool(sim.Success),
			"$" + sim.NetIncomeMetrics.FirstYearNetIncome.StringFixed(0),
			"$" + sim.NetIncomeMetrics.Year5NetIncome.StringFixed(0),
			"$" + sim.NetIncomeMetrics.Year10NetIncome.StringFixed(0),
			"$" + sim.NetIncomeMetrics.MinNetIncome.StringFixed(0),
			"$" + sim.NetIncomeMetrics.MaxNetIncome.StringFixed(0),
			"$" + sim.NetIncomeMetrics.AverageNetIncome.StringFixed(0),
			strconv.Itoa(sim.TSPMetrics.Longevity),
			strconv.FormatBool(sim.TSPMetrics.Depleted),
			"Historical", // This could be enhanced to show actual market conditions
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write simulation row: %w", err)
		}
	}

	return nil
}

// GeneratePercentileCSV creates a CSV with detailed percentile analysis
func (m *MonteCarloCSVReport) GeneratePercentileCSV(outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Percentile",
		"NetIncome",
		"TSPLongevity",
		"Interpretation",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write percentile data
	percentileData := [][]string{
		{"10th", "$" + m.Result.NetIncomePercentiles.P10.StringFixed(0), m.Result.TSPLongevityPercentiles.P10.StringFixed(0), "Worst 10% of scenarios"},
		{"25th", "$" + m.Result.NetIncomePercentiles.P25.StringFixed(0), m.Result.TSPLongevityPercentiles.P25.StringFixed(0), "Below average scenarios"},
		{"50th (Median)", "$" + m.Result.NetIncomePercentiles.P50.StringFixed(0), m.Result.TSPLongevityPercentiles.P50.StringFixed(0), "Typical scenario"},
		{"75th", "$" + m.Result.NetIncomePercentiles.P75.StringFixed(0), m.Result.TSPLongevityPercentiles.P75.StringFixed(0), "Above average scenarios"},
		{"90th", "$" + m.Result.NetIncomePercentiles.P90.StringFixed(0), m.Result.TSPLongevityPercentiles.P90.StringFixed(0), "Best 10% of scenarios"},
	}

	for _, row := range percentileData {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write percentile row: %w", err)
		}
	}

	return nil
}

// GenerateAllCSVReports creates all CSV reports in a single directory
func (m *MonteCarloCSVReport) GenerateAllCSVReports(outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate summary report
	summaryPath := fmt.Sprintf("%s/monte_carlo_summary.csv", outputDir)
	if err := m.GenerateSummaryCSV(summaryPath); err != nil {
		return fmt.Errorf("failed to generate summary CSV: %w", err)
	}

	// Generate detailed report
	detailedPath := fmt.Sprintf("%s/monte_carlo_detailed.csv", outputDir)
	if err := m.GenerateDetailedCSV(detailedPath); err != nil {
		return fmt.Errorf("failed to generate detailed CSV: %w", err)
	}

	// Generate percentile report
	percentilePath := fmt.Sprintf("%s/monte_carlo_percentiles.csv", outputDir)
	if err := m.GeneratePercentileCSV(percentilePath); err != nil {
		return fmt.Errorf("failed to generate percentile CSV: %w", err)
	}

	return nil
}

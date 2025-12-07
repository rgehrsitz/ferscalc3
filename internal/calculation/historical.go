package calculation

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/shopspring/decimal"
)

// HistoricalDataPoint represents a single year's historical data
type HistoricalDataPoint struct {
	Year int             `json:"year"`
	Data decimal.Decimal `json:"data"`
}

// HistoricalDataSet represents a complete dataset with metadata
type HistoricalDataSet struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	DataPoints  []HistoricalDataPoint  `json:"data_points"`
	MinYear     int                    `json:"min_year"`
	MaxYear     int                    `json:"max_year"`
	Statistics  HistoricalStatistics   `json:"statistics"`
}

// HistoricalStatistics provides statistical summary of the dataset
type HistoricalStatistics struct {
	Mean        decimal.Decimal `json:"mean"`
	Median      decimal.Decimal `json:"median"`
	StdDev      decimal.Decimal `json:"std_dev"`
	Min         decimal.Decimal `json:"min"`
	Max         decimal.Decimal `json:"max"`
	Count       int             `json:"count"`
	MissingYears []int          `json:"missing_years"`
}

// TSPFundData represents historical returns for all TSP funds
type TSPFundData struct {
	CFund *HistoricalDataSet `json:"c_fund"`
	SFund *HistoricalDataSet `json:"s_fund"`
	IFund *HistoricalDataSet `json:"i_fund"`
	FFund *HistoricalDataSet `json:"f_fund"`
	GFund *HistoricalDataSet `json:"g_fund"`
}

// HistoricalDataManager manages all historical datasets
type HistoricalDataManager struct {
	TSPFunds    *TSPFundData    `json:"tsp_funds"`
	Inflation   *HistoricalDataSet `json:"inflation"`
	COLA        *HistoricalDataSet `json:"cola"`
	DataPath    string          `json:"data_path"`
	IsLoaded    bool            `json:"is_loaded"`
}

// NewHistoricalDataManager creates a new historical data manager
func NewHistoricalDataManager(dataPath string) *HistoricalDataManager {
	return &HistoricalDataManager{
		TSPFunds: &TSPFundData{},
		DataPath: dataPath,
		IsLoaded: false,
	}
}

// LoadAllData loads all historical datasets
func (hdm *HistoricalDataManager) LoadAllData() error {
	if hdm.IsLoaded {
		return nil // Already loaded
	}

	// Load TSP fund data
	if err := hdm.loadTSPFundData(); err != nil {
		return fmt.Errorf("failed to load TSP fund data: %w", err)
	}

	// Load inflation data
	if err := hdm.loadInflationData(); err != nil {
		return fmt.Errorf("failed to load inflation data: %w", err)
	}

	// Load COLA data
	if err := hdm.loadCOLAData(); err != nil {
		return fmt.Errorf("failed to load COLA data: %w", err)
	}

	hdm.IsLoaded = true
	return nil
}

// loadTSPFundData loads all TSP fund historical returns
func (hdm *HistoricalDataManager) loadTSPFundData() error {
	funds := map[string]string{
		"c_fund": "c-fund-annual.csv",
		"s_fund": "s-fund-annual.csv", 
		"i_fund": "i-fund-annual.csv",
		"f_fund": "f-fund-annual.csv",
		"g_fund": "g-fund-annual.csv",
	}

	for fundName, fileName := range funds {
		filePath := filepath.Join(hdm.DataPath, "tsp-returns", fileName)
		dataset, err := hdm.loadCSVData(filePath, fundName, "TSP "+fundName+" Fund Annual Returns", "TSP.gov")
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", fundName, err)
		}

		switch fundName {
		case "c_fund":
			hdm.TSPFunds.CFund = dataset
		case "s_fund":
			hdm.TSPFunds.SFund = dataset
		case "i_fund":
			hdm.TSPFunds.IFund = dataset
		case "f_fund":
			hdm.TSPFunds.FFund = dataset
		case "g_fund":
			hdm.TSPFunds.GFund = dataset
		}
	}

	return nil
}

// loadInflationData loads historical CPI-U inflation rates
func (hdm *HistoricalDataManager) loadInflationData() error {
	filePath := filepath.Join(hdm.DataPath, "inflation", "cpi-annual.csv")
	dataset, err := hdm.loadCSVData(filePath, "inflation", "CPI-U Annual Inflation Rates", "BLS.gov")
	if err != nil {
		return fmt.Errorf("failed to load inflation data: %w", err)
	}
	hdm.Inflation = dataset
	return nil
}

// loadCOLAData loads historical Social Security COLA rates
func (hdm *HistoricalDataManager) loadCOLAData() error {
	filePath := filepath.Join(hdm.DataPath, "cola", "ss-cola-annual.csv")
	dataset, err := hdm.loadCSVData(filePath, "cola", "Social Security COLA Annual Rates", "SSA.gov")
	if err != nil {
		return fmt.Errorf("failed to load COLA data: %w", err)
	}
	hdm.COLA = dataset
	return nil
}

// loadCSVData loads data from a CSV file and creates a HistoricalDataSet
func (hdm *HistoricalDataManager) loadCSVData(filePath, name, description, source string) (*HistoricalDataSet, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	if len(header) < 2 {
		return nil, fmt.Errorf("invalid CSV format: expected at least 2 columns")
	}

	var dataPoints []HistoricalDataPoint
	var values []decimal.Decimal

	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read data row: %w", err)
		}

		if len(record) < 2 {
			continue // Skip malformed rows
		}

		year, err := strconv.Atoi(record[0])
		if err != nil {
			continue // Skip rows with invalid year
		}

		value, err := decimal.NewFromString(record[1])
		if err != nil {
			continue // Skip rows with invalid value
		}

		dataPoints = append(dataPoints, HistoricalDataPoint{
			Year: year,
			Data: value,
		})
		values = append(values, value)
	}

	if len(dataPoints) == 0 {
		return nil, fmt.Errorf("no valid data points found in %s", filePath)
	}

	// Calculate statistics
	stats := hdm.calculateStatistics(values, dataPoints)

	// Find min/max years
	minYear := dataPoints[0].Year
	maxYear := dataPoints[0].Year
	for _, dp := range dataPoints {
		if dp.Year < minYear {
			minYear = dp.Year
		}
		if dp.Year > maxYear {
			maxYear = dp.Year
		}
	}

	return &HistoricalDataSet{
		Name:        name,
		Description: description,
		Source:      source,
		DataPoints:  dataPoints,
		MinYear:     minYear,
		MaxYear:     maxYear,
		Statistics:  stats,
	}, nil
}

// calculateStatistics calculates statistical measures for the dataset
func (hdm *HistoricalDataManager) calculateStatistics(values []decimal.Decimal, dataPoints []HistoricalDataPoint) HistoricalStatistics {
	if len(values) == 0 {
		return HistoricalStatistics{}
	}

	// Calculate mean
	var sum decimal.Decimal
	for _, v := range values {
		sum = sum.Add(v)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(values))))

	// Calculate min/max
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

	// Calculate standard deviation
	var varianceSum decimal.Decimal
	for _, v := range values {
		diff := v.Sub(mean)
		varianceSum = varianceSum.Add(diff.Mul(diff))
	}
	variance := varianceSum.Div(decimal.NewFromInt(int64(len(values))))
	// Convert to float for sqrt calculation
	varianceFloat, _ := variance.Float64()
	stdDevFloat := math.Sqrt(varianceFloat)
	stdDev := decimal.NewFromFloat(stdDevFloat)

	// Calculate median (simplified - assumes sorted)
	median := mean // For simplicity, using mean as median approximation

	// Find missing years (assuming continuous range)
	var missingYears []int
	if len(dataPoints) > 1 {
		expectedYears := dataPoints[len(dataPoints)-1].Year - dataPoints[0].Year + 1
		if len(dataPoints) < expectedYears {
			// Find gaps in the data
			for year := dataPoints[0].Year; year <= dataPoints[len(dataPoints)-1].Year; year++ {
				found := false
				for _, dp := range dataPoints {
					if dp.Year == year {
						found = true
						break
					}
				}
				if !found {
					missingYears = append(missingYears, year)
				}
			}
		}
	}

	return HistoricalStatistics{
		Mean:        mean,
		Median:      median,
		StdDev:      stdDev,
		Min:         min,
		Max:         max,
		Count:       len(values),
		MissingYears: missingYears,
	}
}

// GetTSPReturn returns the historical return for a specific TSP fund and year
func (hdm *HistoricalDataManager) GetTSPReturn(fundName string, year int) (decimal.Decimal, error) {
	if !hdm.IsLoaded {
		return decimal.Zero, fmt.Errorf("historical data not loaded")
	}

	var dataset *HistoricalDataSet
	switch fundName {
	case "C", "c", "c_fund":
		dataset = hdm.TSPFunds.CFund
	case "S", "s", "s_fund":
		dataset = hdm.TSPFunds.SFund
	case "I", "i", "i_fund":
		dataset = hdm.TSPFunds.IFund
	case "F", "f", "f_fund":
		dataset = hdm.TSPFunds.FFund
	case "G", "g", "g_fund":
		dataset = hdm.TSPFunds.GFund
	default:
		return decimal.Zero, fmt.Errorf("unknown TSP fund: %s", fundName)
	}

	if dataset == nil {
		return decimal.Zero, fmt.Errorf("dataset not available for fund: %s", fundName)
	}

	for _, dp := range dataset.DataPoints {
		if dp.Year == year {
			return dp.Data, nil
		}
	}

	return decimal.Zero, fmt.Errorf("no data found for fund %s in year %d", fundName, year)
}

// GetInflationRate returns the historical inflation rate for a specific year
func (hdm *HistoricalDataManager) GetInflationRate(year int) (decimal.Decimal, error) {
	if !hdm.IsLoaded || hdm.Inflation == nil {
		return decimal.Zero, fmt.Errorf("inflation data not loaded")
	}

	for _, dp := range hdm.Inflation.DataPoints {
		if dp.Year == year {
			return dp.Data, nil
		}
	}

	return decimal.Zero, fmt.Errorf("no inflation data found for year %d", year)
}

// GetCOLARate returns the historical COLA rate for a specific year
func (hdm *HistoricalDataManager) GetCOLARate(year int) (decimal.Decimal, error) {
	if !hdm.IsLoaded || hdm.COLA == nil {
		return decimal.Zero, fmt.Errorf("COLA data not loaded")
	}

	for _, dp := range hdm.COLA.DataPoints {
		if dp.Year == year {
			return dp.Data, nil
		}
	}

	return decimal.Zero, fmt.Errorf("no COLA data found for year %d", year)
}

// GetRandomHistoricalYear returns a random year from the available historical data
func (hdm *HistoricalDataManager) GetRandomHistoricalYear() (int, error) {
	if !hdm.IsLoaded || hdm.TSPFunds.CFund == nil {
		return 0, fmt.Errorf("historical data not loaded")
	}

	// Use C Fund data as reference for available years
	availableYears := len(hdm.TSPFunds.CFund.DataPoints)
	if availableYears == 0 {
		return 0, fmt.Errorf("no historical data available")
	}

	// For now, return a middle year (this will be enhanced with proper randomization)
	middleIndex := availableYears / 2
	return hdm.TSPFunds.CFund.DataPoints[middleIndex].Year, nil
}

// GetAvailableYears returns the range of available years for historical data
func (hdm *HistoricalDataManager) GetAvailableYears() (int, int, error) {
	if !hdm.IsLoaded || hdm.TSPFunds.CFund == nil {
		return 0, 0, fmt.Errorf("historical data not loaded")
	}

	return hdm.TSPFunds.CFund.MinYear, hdm.TSPFunds.CFund.MaxYear, nil
}

// ValidateDataQuality performs quality checks on the loaded data
func (hdm *HistoricalDataManager) ValidateDataQuality() ([]string, error) {
	if !hdm.IsLoaded {
		return nil, fmt.Errorf("historical data not loaded")
	}

	var issues []string

	// Check for missing years
	if len(hdm.TSPFunds.CFund.Statistics.MissingYears) > 0 {
		issues = append(issues, fmt.Sprintf("Missing years in C Fund data: %v", hdm.TSPFunds.CFund.Statistics.MissingYears))
	}

	// Check for extreme outliers (returns > 100% or < -50%)
	for _, dp := range hdm.TSPFunds.CFund.DataPoints {
		if dp.Data.GreaterThan(decimal.NewFromInt(1)) {
			issues = append(issues, fmt.Sprintf("Extreme positive return in C Fund for year %d: %s", dp.Year, dp.Data.String()))
		}
		if dp.Data.LessThan(decimal.NewFromFloat(-0.5)) {
			issues = append(issues, fmt.Sprintf("Extreme negative return in C Fund for year %d: %s", dp.Year, dp.Data.String()))
		}
	}

	// Check data consistency across funds
	minYear, maxYear, err := hdm.GetAvailableYears()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Error getting year range: %v", err))
	} else {
		expectedYears := maxYear - minYear + 1
		for fundName, dataset := range map[string]*HistoricalDataSet{
			"C Fund": hdm.TSPFunds.CFund,
			"S Fund": hdm.TSPFunds.SFund,
			"I Fund": hdm.TSPFunds.IFund,
			"F Fund": hdm.TSPFunds.FFund,
			"G Fund": hdm.TSPFunds.GFund,
		} {
			if dataset != nil && len(dataset.DataPoints) != expectedYears {
				issues = append(issues, fmt.Sprintf("%s has %d data points, expected %d", fundName, len(dataset.DataPoints), expectedYears))
			}
		}
	}

	return issues, nil
} 
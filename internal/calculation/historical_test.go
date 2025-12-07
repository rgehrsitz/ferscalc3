package calculation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"
)

func TestHistoricalDataManager(t *testing.T) {
	// Create temporary test data directory
	testDataPath := t.TempDir()

	// Create test data files
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	// Test creating new manager
	hdm := NewHistoricalDataManager(testDataPath)
	if hdm == nil {
		t.Fatal("Failed to create HistoricalDataManager")
	}

	if hdm.IsLoaded {
		t.Error("Manager should not be loaded initially")
	}

	// Test loading all data
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load all data: %v", err)
	}

	if !hdm.IsLoaded {
		t.Error("Manager should be loaded after LoadAllData")
	}

	// Test TSP fund data loading
	if hdm.TSPFunds.CFund == nil {
		t.Error("C Fund data should be loaded")
	}
	if hdm.TSPFunds.SFund == nil {
		t.Error("S Fund data should be loaded")
	}
	if hdm.TSPFunds.IFund == nil {
		t.Error("I Fund data should be loaded")
	}
	if hdm.TSPFunds.FFund == nil {
		t.Error("F Fund data should be loaded")
	}
	if hdm.TSPFunds.GFund == nil {
		t.Error("G Fund data should be loaded")
	}

	// Test inflation data loading
	if hdm.Inflation == nil {
		t.Error("Inflation data should be loaded")
	}

	// Test COLA data loading
	if hdm.COLA == nil {
		t.Error("COLA data should be loaded")
	}
}

func TestGetTSPReturn(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// Test valid fund returns
	testCases := []struct {
		fundName string
		year     int
		expected string
	}{
		{"C", 2020, "0.181"},
		{"c", 2020, "0.181"},
		{"c_fund", 2020, "0.181"},
		{"S", 2020, "0.111"},
		{"I", 2020, "0.078"},
		{"F", 2020, "0.078"},
		{"G", 2020, "0.034"},
	}

	for _, tc := range testCases {
		t.Run(tc.fundName+"_"+string(rune(tc.year)), func(t *testing.T) {
			result, err := hdm.GetTSPReturn(tc.fundName, tc.year)
			if err != nil {
				t.Errorf("GetTSPReturn failed for %s fund year %d: %v", tc.fundName, tc.year, err)
				return
			}

			expected, _ := decimal.NewFromString(tc.expected)
			if !result.Equal(expected) {
				t.Errorf("Expected %s, got %s for %s fund year %d", expected, result, tc.fundName, tc.year)
			}
		})
	}

	// Test invalid fund name
	_, err := hdm.GetTSPReturn("INVALID", 2020)
	if err == nil {
		t.Error("Expected error for invalid fund name")
	}

	// Test year not in data
	_, err = hdm.GetTSPReturn("C", 1900)
	if err == nil {
		t.Error("Expected error for year not in data")
	}
}

func TestGetInflationRate(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// Test valid inflation rate
	result, err := hdm.GetInflationRate(2020)
	if err != nil {
		t.Errorf("GetInflationRate failed: %v", err)
	}

	expected, _ := decimal.NewFromString("0.012")
	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s for inflation rate 2020", expected, result)
	}

	// Test year not in data
	_, err = hdm.GetInflationRate(1900)
	if err == nil {
		t.Error("Expected error for year not in data")
	}
}

func TestGetCOLARate(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// Test valid COLA rate
	result, err := hdm.GetCOLARate(2020)
	if err != nil {
		t.Errorf("GetCOLARate failed: %v", err)
	}

	expected, _ := decimal.NewFromString("0.013")
	if !result.Equal(expected) {
		t.Errorf("Expected %s, got %s for COLA rate 2020", expected, result)
	}

	// Test year not in data
	_, err = hdm.GetCOLARate(1900)
	if err == nil {
		t.Error("Expected error for year not in data")
	}
}

func TestGetAvailableYears(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	minYear, maxYear, err := hdm.GetAvailableYears()
	if err != nil {
		t.Errorf("GetAvailableYears failed: %v", err)
	}

	if minYear != 2020 {
		t.Errorf("Expected min year 2020, got %d", minYear)
	}
	if maxYear != 2023 {
		t.Errorf("Expected max year 2023, got %d", maxYear)
	}
}

func TestValidateDataQuality(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	issues, err := hdm.ValidateDataQuality()
	if err != nil {
		t.Errorf("ValidateDataQuality failed: %v", err)
	}

	// With our test data, there should be no quality issues
	if len(issues) > 0 {
		t.Errorf("Expected no quality issues, got: %v", issues)
	}
}

func TestStatisticsCalculation(t *testing.T) {
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	// Test C Fund statistics
	stats := hdm.TSPFunds.CFund.Statistics
	if stats.Count != 4 { // Our test data has 4 years
		t.Errorf("Expected 4 data points, got %d", stats.Count)
	}

	if hdm.TSPFunds.CFund.MinYear != 2020 {
		t.Errorf("Expected min year 2020, got %d", hdm.TSPFunds.CFund.MinYear)
	}
	if hdm.TSPFunds.CFund.MaxYear != 2023 {
		t.Errorf("Expected max year 2023, got %d", hdm.TSPFunds.CFund.MaxYear)
	}

	// Test that statistics are reasonable
	if stats.Mean.LessThan(decimal.NewFromFloat(-0.5)) || stats.Mean.GreaterThan(decimal.NewFromFloat(0.5)) {
		t.Errorf("Mean return seems unreasonable: %s", stats.Mean)
	}

	if stats.StdDev.LessThan(decimal.Zero) {
		t.Errorf("Standard deviation should be positive, got %s", stats.StdDev)
	}
}

// Helper function to create test data files
func createTestDataFiles(dataPath string) error {
	// Create directories
	dirs := []string{"tsp-returns", "inflation", "cola"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(dataPath, dir), 0755); err != nil {
			return err
		}
	}

	// Create test data files with minimal data for testing
	testFiles := map[string]string{
		"tsp-returns/c-fund-annual.csv": "Year,Return\n2020,0.181\n2021,0.287\n2022,-0.182\n2023,0.264",
		"tsp-returns/s-fund-annual.csv": "Year,Return\n2020,0.111\n2021,0.145\n2022,-0.201\n2023,0.214",
		"tsp-returns/i-fund-annual.csv": "Year,Return\n2020,0.078\n2021,0.087\n2022,-0.167\n2023,0.187",
		"tsp-returns/f-fund-annual.csv": "Year,Return\n2020,0.078\n2021,-0.012\n2022,-0.134\n2023,0.045",
		"tsp-returns/g-fund-annual.csv": "Year,Return\n2020,0.034\n2021,0.034\n2022,0.034\n2023,0.034",
		"inflation/cpi-annual.csv":      "Year,InflationRate\n2020,0.012\n2021,0.047\n2022,0.065\n2023,0.031",
		"cola/ss-cola-annual.csv":       "Year,COLARate\n2020,0.013\n2021,0.059\n2022,0.087\n2023,0.032",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(dataPath, filePath)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

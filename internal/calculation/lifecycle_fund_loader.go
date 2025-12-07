package calculation

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// LifecycleFundLoader manages TSP Lifecycle Fund allocation data
type LifecycleFundLoader struct {
	DataPath string
	Funds    map[string]*domain.TSPLifecycleFund
}

// NewLifecycleFundLoader creates a new lifecycle fund loader
func NewLifecycleFundLoader(dataPath string) *LifecycleFundLoader {
	return &LifecycleFundLoader{
		DataPath: dataPath,
		Funds:    make(map[string]*domain.TSPLifecycleFund),
	}
}

// LoadAllLifecycleFunds loads all available lifecycle fund data
func (lfl *LifecycleFundLoader) LoadAllLifecycleFunds() error {
	fundFiles := []string{
		"l2030_allocation.csv",
		"l2035_allocation.csv",
		"l2040_allocation.csv",
		"lincome_allocation.csv",
	}

	for _, filename := range fundFiles {
		fundName := strings.TrimSuffix(filename, "_allocation.csv")
		if err := lfl.loadLifecycleFund(fundName, filename); err != nil {
			return fmt.Errorf("failed to load %s: %w", filename, err)
		}
	}

	return nil
}

// loadLifecycleFund loads a single lifecycle fund from CSV
func (lfl *LifecycleFundLoader) loadLifecycleFund(fundName, filename string) error {
	filePath := filepath.Join(lfl.DataPath, "tsp-returns", filename)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV %s: %w", filePath, err)
	}

	if len(records) < 2 {
		return fmt.Errorf("insufficient data in %s", filePath)
	}

	// Parse header - handle quoted headers
	header := records[0]
	if len(header) != 6 {
		return fmt.Errorf("expected 6 columns in %s, got %d", filePath, len(header))
	}

	// Clean up header values (remove quotes if present)
	for i, col := range header {
		header[i] = strings.Trim(col, `"`)
	}

	// Create lifecycle fund
	lifecycleFund := &domain.TSPLifecycleFund{
		FundName:       fundName,
		AllocationData: make(map[string][]domain.TSPAllocationDataPoint),
	}

	// Parse data rows
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) != 6 {
			continue // Skip malformed rows
		}

		// Parse date (format: "July 2005", "October 2005", etc.)
		dateStr := strings.Trim(row[0], `"`) // Remove quotes if present
		date, err := parseQuarterlyDate(dateStr)
		if err != nil {
			continue // Skip rows with invalid dates
		}

		// Parse allocation percentages - clean up values
		gFund, err := parsePercentage(strings.Trim(row[1], `"`))
		if err != nil {
			continue
		}
		fFund, err := parsePercentage(strings.Trim(row[2], `"`))
		if err != nil {
			continue
		}
		cFund, err := parsePercentage(strings.Trim(row[3], `"`))
		if err != nil {
			continue
		}
		sFund, err := parsePercentage(strings.Trim(row[4], `"`))
		if err != nil {
			continue
		}
		iFund, err := parsePercentage(strings.Trim(row[5], `"`))
		if err != nil {
			continue
		}

		// Convert percentages to decimals (divide by 100)
		allocation := domain.TSPAllocation{
			GFund: gFund.Div(decimal.NewFromInt(100)),
			FFund: fFund.Div(decimal.NewFromInt(100)),
			CFund: cFund.Div(decimal.NewFromInt(100)),
			SFund: sFund.Div(decimal.NewFromInt(100)),
			IFund: iFund.Div(decimal.NewFromInt(100)),
		}

		// Store allocation data
		yearKey := fmt.Sprintf("%d", date.Year())
		lifecycleFund.AllocationData[yearKey] = append(lifecycleFund.AllocationData[yearKey], domain.TSPAllocationDataPoint{
			Date:       date.Format("2006-01-02"),
			Allocation: allocation,
		})
	}

	// Store the lifecycle fund
	lfl.Funds[fundName] = lifecycleFund

	return nil
}

// GetLifecycleFund returns a lifecycle fund by name
func (lfl *LifecycleFundLoader) GetLifecycleFund(fundName string) (*domain.TSPLifecycleFund, error) {
	fund, exists := lfl.Funds[fundName]
	if !exists {
		return nil, fmt.Errorf("lifecycle fund %s not found", fundName)
	}
	return fund, nil
}

// GetAllocationAtDate returns the allocation for a specific fund and date
func (lfl *LifecycleFundLoader) GetAllocationAtDate(fundName string, targetDate time.Time) (*domain.TSPAllocation, error) {
	fund, err := lfl.GetLifecycleFund(fundName)
	if err != nil {
		return nil, err
	}

	// Find the closest date in the allocation data
	var closestDate string
	var minDiff time.Duration

	for dateStr := range fund.AllocationData {
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		diff := targetDate.Sub(date)
		if diff < 0 {
			diff = -diff
		}

		if closestDate == "" || diff < minDiff {
			closestDate = dateStr
			minDiff = diff
		}
	}

	if closestDate == "" {
		return nil, fmt.Errorf("no allocation data found for fund %s", fundName)
	}

	// Return the allocation for the closest date
	allocationData := fund.AllocationData[closestDate]
	if len(allocationData) == 0 {
		return nil, fmt.Errorf("no allocation data found for date %s", closestDate)
	}

	return &allocationData[0].Allocation, nil
}

// parseQuarterlyDate parses dates like "July 2005", "October 2005"
func parseQuarterlyDate(dateStr string) (time.Time, error) {
	// Map month names to numbers
	monthMap := map[string]int{
		"January": 1, "February": 2, "March": 3, "April": 4,
		"May": 5, "June": 6, "July": 7, "August": 8,
		"September": 9, "October": 10, "November": 11, "December": 12,
	}

	parts := strings.Fields(dateStr)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
	}

	monthName := parts[0]
	yearStr := parts[1]

	month, exists := monthMap[monthName]
	if !exists {
		return time.Time{}, fmt.Errorf("invalid month: %s", monthName)
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %s", yearStr)
	}

	// Use the 15th of the month as the representative date
	return time.Date(year, time.Month(month), 15, 0, 0, 0, 0, time.UTC), nil
}

// parsePercentage parses a percentage string to decimal
func parsePercentage(percentStr string) (decimal.Decimal, error) {
	// Remove any trailing % if present
	cleanStr := strings.TrimSuffix(percentStr, "%")

	// Parse as decimal
	value, err := decimal.NewFromString(cleanStr)
	if err != nil {
		return decimal.Zero, fmt.Errorf("invalid percentage: %s", percentStr)
	}

	return value, nil
}

// GetAvailableFunds returns a list of available lifecycle funds
func (lfl *LifecycleFundLoader) GetAvailableFunds() []string {
	funds := make([]string, 0, len(lfl.Funds))
	for fundName := range lfl.Funds {
		funds = append(funds, fundName)
	}
	return funds
}

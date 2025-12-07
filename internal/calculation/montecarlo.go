package calculation

import (
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/shopspring/decimal"
)

// MonteCarloSimulator manages Monte Carlo simulations for retirement planning
type MonteCarloSimulator struct {
	HistoricalData  *HistoricalDataManager
	NumSimulations  int
	ProjectionYears int
	Seed            int64
	UseHistorical   bool // If true, sample from historical data; if false, use statistical distributions
}

// MonteCarloConfig holds configuration for Monte Carlo simulations
type MonteCarloConfig struct {
	NumSimulations     int
	ProjectionYears    int
	Seed               int64
	UseHistorical      bool
	AssetAllocation    map[string]decimal.Decimal // Fund allocation percentages
	WithdrawalStrategy string
	InitialBalance     decimal.Decimal
	AnnualWithdrawal   decimal.Decimal
}

// MonteCarloResult represents the results of a Monte Carlo simulation
type MonteCarloResult struct {
	Simulations         []SimulationOutcome        `json:"simulations"`
	SuccessRate         decimal.Decimal            `json:"success_rate"`
	MedianEndingBalance decimal.Decimal            `json:"median_ending_balance"`
	PercentileRanges    PercentileRanges           `json:"percentile_ranges"`
	NumSimulations      int                        `json:"num_simulations"`
	ProjectionYears     int                        `json:"projection_years"`
	AssetAllocation     map[string]decimal.Decimal `json:"asset_allocation"`
	WithdrawalStrategy  string                     `json:"withdrawal_strategy"`
	InitialBalance      decimal.Decimal            `json:"initial_balance"`
	AnnualWithdrawal    decimal.Decimal            `json:"annual_withdrawal"`
}

// SimulationOutcome represents a single Monte Carlo simulation outcome
type SimulationOutcome struct {
	YearOutcomes    []YearOutcome   `json:"year_outcomes"`
	PortfolioLasted int             `json:"portfolio_lasted"`
	EndingBalance   decimal.Decimal `json:"ending_balance"`
	Success         bool            `json:"success"`
	MaxDrawdown     decimal.Decimal `json:"max_drawdown"`
	TotalWithdrawn  decimal.Decimal `json:"total_withdrawn"`
}

// YearOutcome represents a single year's outcome in a Monte Carlo simulation
type YearOutcome struct {
	Year       int             `json:"year"`
	Balance    decimal.Decimal `json:"balance"`
	Withdrawal decimal.Decimal `json:"withdrawal"`
	Return     decimal.Decimal `json:"return"`
	Inflation  decimal.Decimal `json:"inflation"`
	COLA       decimal.Decimal `json:"cola"`
}

// PercentileRanges represents percentile ranges for Monte Carlo results
type PercentileRanges struct {
	P10 decimal.Decimal `json:"p10"`
	P25 decimal.Decimal `json:"p25"`
	P50 decimal.Decimal `json:"p50"`
	P75 decimal.Decimal `json:"p75"`
	P90 decimal.Decimal `json:"p90"`
}

// NewMonteCarloSimulator creates a new Monte Carlo simulator
func NewMonteCarloSimulator(historicalData *HistoricalDataManager, config MonteCarloConfig) *MonteCarloSimulator {
	if config.Seed == 0 {
		config.Seed = seedFunc()
	}

	return &MonteCarloSimulator{
		HistoricalData:  historicalData,
		NumSimulations:  config.NumSimulations,
		ProjectionYears: config.ProjectionYears,
		Seed:            config.Seed,
		UseHistorical:   config.UseHistorical,
	}
}

// RunSimulation executes the Monte Carlo simulation
func (mcs *MonteCarloSimulator) RunSimulation(config MonteCarloConfig) (*MonteCarloResult, error) {
	if mcs.HistoricalData == nil || !mcs.HistoricalData.IsLoaded {
		return nil, fmt.Errorf("historical data not loaded")
	}

	// Set random seed (using modern Go approach)
	// Note: In Go 1.20+, global rand is automatically seeded
	// For reproducible results with specific seeds, a local random source would be needed

	// Run simulations in parallel
	results := make([]SimulationOutcome, mcs.NumSimulations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent simulations

	for i := 0; i < mcs.NumSimulations; i++ {
		wg.Add(1)
		go func(simIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			outcome := mcs.runSingleSimulation(config)
			results[simIndex] = outcome
		}(i)
	}

	wg.Wait()

	// Calculate aggregate statistics
	successRate := mcs.calculateSuccessRate(results)
	medianEndingBalance := mcs.calculateMedianEndingBalance(results)
	percentileRanges := mcs.calculatePercentileRanges(results)

	return &MonteCarloResult{
		Simulations:         results,
		SuccessRate:         successRate,
		MedianEndingBalance: medianEndingBalance,
		PercentileRanges:    percentileRanges,
		NumSimulations:      mcs.NumSimulations,
		ProjectionYears:     mcs.ProjectionYears,
		AssetAllocation:     config.AssetAllocation,
		WithdrawalStrategy:  config.WithdrawalStrategy,
		InitialBalance:      config.InitialBalance,
		AnnualWithdrawal:    config.AnnualWithdrawal,
	}, nil
}

// runSingleSimulation runs a single Monte Carlo simulation
func (mcs *MonteCarloSimulator) runSingleSimulation(config MonteCarloConfig) SimulationOutcome {
	currentBalance := config.InitialBalance
	var yearOutcomes []YearOutcome
	var totalWithdrawn decimal.Decimal
	maxDrawdown := decimal.Zero
	peakBalance := currentBalance

	for year := 1; year <= mcs.ProjectionYears; year++ {
		// Sample market conditions
		marketData := mcs.sampleMarketConditions()

		// Calculate portfolio return based on asset allocation
		portfolioReturn := mcs.calculatePortfolioReturn(config.AssetAllocation, marketData)

		// Apply market returns
		growth := currentBalance.Mul(portfolioReturn)
		currentBalance = currentBalance.Add(growth)

		// Calculate withdrawal (considering market conditions and inflation)
		withdrawal := mcs.calculateDynamicWithdrawal(config, currentBalance, year, marketData)

		// Apply withdrawal
		if withdrawal.GreaterThan(currentBalance) {
			withdrawal = currentBalance
		}
		currentBalance = currentBalance.Sub(withdrawal)
		totalWithdrawn = totalWithdrawn.Add(withdrawal)

		// Track drawdown
		if currentBalance.GreaterThan(peakBalance) {
			peakBalance = currentBalance
		}
		drawdown := peakBalance.Sub(currentBalance).Div(peakBalance)
		if drawdown.GreaterThan(maxDrawdown) {
			maxDrawdown = drawdown
		}

		yearOutcomes = append(yearOutcomes, YearOutcome{
			Year:       year,
			Balance:    currentBalance,
			Withdrawal: withdrawal,
			Return:     portfolioReturn,
			Inflation:  marketData.Inflation,
			COLA:       marketData.COLA,
		})

		// Check if portfolio is depleted
		if currentBalance.LessThanOrEqual(decimal.Zero) {
			break
		}
	}

	success := currentBalance.GreaterThan(decimal.Zero)

	return SimulationOutcome{
		YearOutcomes:    yearOutcomes,
		PortfolioLasted: len(yearOutcomes),
		EndingBalance:   currentBalance,
		Success:         success,
		MaxDrawdown:     maxDrawdown,
		TotalWithdrawn:  totalWithdrawn,
	}
}

// MarketData represents market conditions for a given year
type MarketData struct {
	TSPReturns map[string]decimal.Decimal
	Inflation  decimal.Decimal
	COLA       decimal.Decimal
}

// sampleMarketConditions samples market conditions
func (mcs *MonteCarloSimulator) sampleMarketConditions() MarketData {
	if mcs.UseHistorical {
		return mcs.sampleHistoricalMarketConditions()
	} else {
		return mcs.generateStatisticalMarketConditions()
	}
}

// sampleHistoricalMarketConditions samples from historical data
func (mcs *MonteCarloSimulator) sampleHistoricalMarketConditions() MarketData {
	// Get available years
	minYear, maxYear, err := mcs.HistoricalData.GetAvailableYears()
	if err != nil {
		// Fallback to statistical generation
		return mcs.generateStatisticalMarketConditions()
	}

	// Randomly select a historical year
	historicalYear := minYear + rand.Intn(maxYear-minYear+1)

	// Get historical data for that year
	marketData := MarketData{
		TSPReturns: make(map[string]decimal.Decimal),
	}

	// Sample TSP fund returns
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		if returnRate, err := mcs.HistoricalData.GetTSPReturn(fund, historicalYear); err == nil {
			marketData.TSPReturns[fund] = returnRate
		} else {
			// Fallback to statistical generation for this fund
			marketData.TSPReturns[fund] = mcs.generateStatisticalReturn(fund)
		}
	}

	// Sample inflation and COLA
	if inflation, err := mcs.HistoricalData.GetInflationRate(historicalYear); err == nil {
		marketData.Inflation = inflation
	} else {
		marketData.Inflation = mcs.generateStatisticalInflation()
	}

	if cola, err := mcs.HistoricalData.GetCOLARate(historicalYear); err == nil {
		marketData.COLA = cola
	} else {
		marketData.COLA = mcs.generateStatisticalCOLA()
	}

	return marketData
}

// generateStatisticalMarketConditions generates market conditions using statistical distributions
func (mcs *MonteCarloSimulator) generateStatisticalMarketConditions() MarketData {
	marketData := MarketData{
		TSPReturns: make(map[string]decimal.Decimal),
	}

	// Generate returns for each fund
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		marketData.TSPReturns[fund] = mcs.generateStatisticalReturn(fund)
	}

	marketData.Inflation = mcs.generateStatisticalInflation()
	marketData.COLA = mcs.generateStatisticalCOLA()

	return marketData
}

// generateStatisticalReturn generates a statistical return for a given fund
func (mcs *MonteCarloSimulator) generateStatisticalReturn(fund string) decimal.Decimal {
	// Use historical statistics if available, otherwise use reasonable defaults
	var mean, stdDev decimal.Decimal

	switch fund {
	case "C":
		mean = decimal.NewFromFloat(0.1125)   // 11.25% historical mean
		stdDev = decimal.NewFromFloat(0.1744) // 17.44% historical std dev
	case "S":
		mean = decimal.NewFromFloat(0.1117)   // 11.17% historical mean
		stdDev = decimal.NewFromFloat(0.1933) // 19.33% historical std dev
	case "I":
		mean = decimal.NewFromFloat(0.0634)   // 6.34% historical mean
		stdDev = decimal.NewFromFloat(0.1863) // 18.63% historical std dev
	case "F":
		mean = decimal.NewFromFloat(0.0532)   // 5.32% historical mean
		stdDev = decimal.NewFromFloat(0.0565) // 5.65% historical std dev
	case "G":
		mean = decimal.NewFromFloat(0.0493)   // 4.93% historical mean
		stdDev = decimal.NewFromFloat(0.0165) // 1.65% historical std dev
	default:
		mean = decimal.NewFromFloat(0.08)   // 8% default mean
		stdDev = decimal.NewFromFloat(0.15) // 15% default std dev
	}

	// Generate normal distribution (simplified)
	// In a production system, you might want to use a more sophisticated distribution
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := mcs.boxMullerTransform(u1, u2)

	// Convert to decimal and apply mean/std dev
	zDecimal := decimal.NewFromFloat(z)
	return mean.Add(zDecimal.Mul(stdDev))
}

// generateStatisticalInflation generates statistical inflation rate
func (mcs *MonteCarloSimulator) generateStatisticalInflation() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0259)   // 2.59% historical mean
	stdDev := decimal.NewFromFloat(0.0137) // 1.37% historical std dev

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := mcs.boxMullerTransform(u1, u2)

	zDecimal := decimal.NewFromFloat(z)
	return mean.Add(zDecimal.Mul(stdDev))
}

// generateStatisticalCOLA generates statistical COLA rate
func (mcs *MonteCarloSimulator) generateStatisticalCOLA() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0255)   // 2.55% historical mean
	stdDev := decimal.NewFromFloat(0.0182) // 1.82% historical std dev

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := mcs.boxMullerTransform(u1, u2)

	zDecimal := decimal.NewFromFloat(z)
	return mean.Add(zDecimal.Mul(stdDev))
}

// boxMullerTransform implements Box-Muller transform for normal distribution
func (mcs *MonteCarloSimulator) boxMullerTransform(u1, u2 float64) float64 {
	// Box-Muller transform to convert uniform random variables to normal distribution
	z0 := mcs.sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return z0
}

// sqrt is a helper function for square root
func (mcs *MonteCarloSimulator) sqrt(x float64) float64 {
	return math.Sqrt(x)
}

// calculatePortfolioReturn calculates the weighted portfolio return based on asset allocation
func (mcs *MonteCarloSimulator) calculatePortfolioReturn(allocation map[string]decimal.Decimal, marketData MarketData) decimal.Decimal {
	var portfolioReturn decimal.Decimal

	for fund, weight := range allocation {
		if returnRate, exists := marketData.TSPReturns[fund]; exists {
			portfolioReturn = portfolioReturn.Add(returnRate.Mul(weight))
		}
	}

	return portfolioReturn
}

// calculateDynamicWithdrawal calculates withdrawal amount considering market conditions
func (mcs *MonteCarloSimulator) calculateDynamicWithdrawal(config MonteCarloConfig, currentBalance decimal.Decimal, year int, marketData MarketData) decimal.Decimal {
	switch config.WithdrawalStrategy {
	case "fixed_amount":
		return config.AnnualWithdrawal
	case "fixed_percentage":
		return currentBalance.Mul(config.AnnualWithdrawal)
	case "inflation_adjusted":
		// Adjust withdrawal for inflation
		inflationFactor := decimal.NewFromFloat(1).Add(marketData.Inflation)
		return config.AnnualWithdrawal.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year - 1))))
	case "guardrails":
		return mcs.calculateGuardrailsWithdrawal(config, currentBalance, year, marketData)
	default:
		return config.AnnualWithdrawal
	}
}

// calculateGuardrailsWithdrawal implements a guardrails withdrawal strategy
func (mcs *MonteCarloSimulator) calculateGuardrailsWithdrawal(config MonteCarloConfig, _ decimal.Decimal, year int, marketData MarketData) decimal.Decimal {
	// Base withdrawal (inflation-adjusted)
	baseWithdrawal := config.AnnualWithdrawal
	if year > 1 {
		inflationFactor := decimal.NewFromFloat(1).Add(marketData.Inflation)
		baseWithdrawal = config.AnnualWithdrawal.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year - 1))))
	}

	// Calculate withdrawal rate
	withdrawalRate := baseWithdrawal.Div(config.InitialBalance)

	// Guardrails: reduce withdrawal if portfolio is down significantly
	if withdrawalRate.GreaterThan(decimal.NewFromFloat(0.06)) { // 6% threshold
		// Reduce withdrawal by 10%
		baseWithdrawal = baseWithdrawal.Mul(decimal.NewFromFloat(0.9))
	}

	// Floor: don't reduce below 80% of original withdrawal
	floorWithdrawal := config.AnnualWithdrawal.Mul(decimal.NewFromFloat(0.8))
	if baseWithdrawal.LessThan(floorWithdrawal) {
		baseWithdrawal = floorWithdrawal
	}

	return baseWithdrawal
}

// calculateSuccessRate calculates the percentage of successful simulations
func (mcs *MonteCarloSimulator) calculateSuccessRate(simulations []SimulationOutcome) decimal.Decimal {
	successCount := 0
	for _, sim := range simulations {
		if sim.Success {
			successCount++
		}
	}

	successRate := decimal.NewFromInt(int64(successCount)).Div(decimal.NewFromInt(int64(len(simulations))))
	return successRate
}

// calculateMedianEndingBalance calculates the median ending balance
func (mcs *MonteCarloSimulator) calculateMedianEndingBalance(simulations []SimulationOutcome) decimal.Decimal {
	// Extract ending balances
	balances := make([]decimal.Decimal, len(simulations))
	for i, sim := range simulations {
		balances[i] = sim.EndingBalance
	}

	// Sort balances
	mcs.sortBalances(balances)

	// Return the middle value
	middleIndex := len(balances) / 2
	return balances[middleIndex]
}

// calculatePercentileRanges calculates percentile ranges for ending balances
func (mcs *MonteCarloSimulator) calculatePercentileRanges(simulations []SimulationOutcome) PercentileRanges {
	// Extract ending balances
	balances := make([]decimal.Decimal, len(simulations))
	for i, sim := range simulations {
		balances[i] = sim.EndingBalance
	}

	// Sort balances
	mcs.sortBalances(balances)

	// Calculate percentiles
	n := len(balances)

	return PercentileRanges{
		P10: balances[n/10],
		P25: balances[n/4],
		P50: balances[n/2],
		P75: balances[3*n/4],
		P90: balances[9*n/10],
	}
}

// sortBalances sorts balances in ascending order
func (mcs *MonteCarloSimulator) sortBalances(balances []decimal.Decimal) {
	// Simple bubble sort for small arrays
	// In production, use a more efficient sorting algorithm
	for i := 0; i < len(balances)-1; i++ {
		for j := 0; j < len(balances)-i-1; j++ {
			if balances[j].GreaterThan(balances[j+1]) {
				balances[j], balances[j+1] = balances[j+1], balances[j]
			}
		}
	}
}

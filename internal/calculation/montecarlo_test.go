package calculation

import (
	"fmt"
	"testing"

	"github.com/shopspring/decimal"
)

func TestMonteCarloSimulator(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	// Test configuration
	config := MonteCarloConfig{
		NumSimulations:  100,
		ProjectionYears: 25,
		Seed:            12345,
		UseHistorical:   true,
		AssetAllocation: map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(0.6), // 60% C Fund
			"S": decimal.NewFromFloat(0.2), // 20% S Fund
			"I": decimal.NewFromFloat(0.1), // 10% I Fund
			"F": decimal.NewFromFloat(0.1), // 10% F Fund
		},
		WithdrawalStrategy: "fixed_amount",
		InitialBalance:     decimal.NewFromInt(1000000), // $1M
		AnnualWithdrawal:   decimal.NewFromInt(40000),   // $40K (4% rule)
	}

	// Create simulator
	simulator := NewMonteCarloSimulator(hdm, config)
	if simulator == nil {
		t.Fatal("Failed to create Monte Carlo simulator")
	}

	// Run simulation
	result, err := simulator.RunSimulation(config)
	if err != nil {
		t.Fatalf("Failed to run simulation: %v", err)
	}

	// Validate results
	if result == nil {
		t.Fatal("Simulation result is nil")
	}

	if result.NumSimulations != config.NumSimulations {
		t.Errorf("Expected %d simulations, got %d", config.NumSimulations, result.NumSimulations)
	}

	if result.ProjectionYears != config.ProjectionYears {
		t.Errorf("Expected %d projection years, got %d", config.ProjectionYears, result.ProjectionYears)
	}

	if len(result.Simulations) != config.NumSimulations {
		t.Errorf("Expected %d simulation outcomes, got %d", config.NumSimulations, len(result.Simulations))
	}

	// Check that success rate is reasonable (between 0 and 1)
	if result.SuccessRate.LessThan(decimal.Zero) || result.SuccessRate.GreaterThan(decimal.NewFromInt(1)) {
		t.Errorf("Success rate should be between 0 and 1, got %s", result.SuccessRate)
	}

	// Check that we have some successful simulations
	successCount := 0
	for _, sim := range result.Simulations {
		if sim.Success {
			successCount++
		}
	}
	if successCount == 0 {
		t.Error("Expected at least some successful simulations")
	}
}

func TestMonteCarloStatisticalMode(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	// Test configuration with statistical mode
	config := MonteCarloConfig{
		NumSimulations:  50,
		ProjectionYears: 20,
		Seed:            54321,
		UseHistorical:   false, // Use statistical distributions
		AssetAllocation: map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(0.5), // 50% C Fund
			"G": decimal.NewFromFloat(0.5), // 50% G Fund
		},
		WithdrawalStrategy: "inflation_adjusted",
		InitialBalance:     decimal.NewFromInt(500000), // $500K
		AnnualWithdrawal:   decimal.NewFromInt(20000),  // $20K
	}

	// Create simulator
	simulator := NewMonteCarloSimulator(hdm, config)

	// Run simulation
	result, err := simulator.RunSimulation(config)
	if err != nil {
		t.Fatalf("Failed to run statistical simulation: %v", err)
	}

	// Validate results
	if result == nil {
		t.Fatal("Statistical simulation result is nil")
	}

	if len(result.Simulations) != config.NumSimulations {
		t.Errorf("Expected %d simulation outcomes, got %d", config.NumSimulations, len(result.Simulations))
	}

	// Check that success rate is reasonable
	if result.SuccessRate.LessThan(decimal.Zero) || result.SuccessRate.GreaterThan(decimal.NewFromInt(1)) {
		t.Errorf("Success rate should be between 0 and 1, got %s", result.SuccessRate)
	}
}

func TestMonteCarloWithdrawalStrategies(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	// Test different withdrawal strategies
	strategies := []string{"fixed_amount", "fixed_percentage", "inflation_adjusted", "guardrails"}

	for _, strategy := range strategies {
		t.Run(strategy, func(t *testing.T) {
			config := MonteCarloConfig{
				NumSimulations:  25,
				ProjectionYears: 15,
				Seed:            99999,
				UseHistorical:   true,
				AssetAllocation: map[string]decimal.Decimal{
					"C": decimal.NewFromFloat(0.7), // 70% C Fund
					"F": decimal.NewFromFloat(0.3), // 30% F Fund
				},
				WithdrawalStrategy: strategy,
				InitialBalance:     decimal.NewFromInt(750000), // $750K
				AnnualWithdrawal:   decimal.NewFromInt(30000),  // $30K
			}

			simulator := NewMonteCarloSimulator(hdm, config)
			result, err := simulator.RunSimulation(config)
			if err != nil {
				t.Fatalf("Failed to run simulation with strategy %s: %v", strategy, err)
			}

			if result == nil {
				t.Fatalf("Result is nil for strategy %s", strategy)
			}

			if len(result.Simulations) != config.NumSimulations {
				t.Errorf("Expected %d simulations for strategy %s, got %d", config.NumSimulations, strategy, len(result.Simulations))
			}

			// Check that withdrawal strategy is set correctly
			if result.WithdrawalStrategy != strategy {
				t.Errorf("Expected withdrawal strategy %s, got %s", strategy, result.WithdrawalStrategy)
			}
		})
	}
}

func TestMonteCarloAssetAllocations(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	// Test different asset allocations
	allocations := []map[string]decimal.Decimal{
		{
			"C": decimal.NewFromFloat(1.0), // 100% C Fund (aggressive)
		},
		{
			"G": decimal.NewFromFloat(1.0), // 100% G Fund (conservative)
		},
		{
			"C": decimal.NewFromFloat(0.4), // 40% C Fund
			"S": decimal.NewFromFloat(0.2), // 20% S Fund
			"I": decimal.NewFromFloat(0.2), // 20% I Fund
			"F": decimal.NewFromFloat(0.2), // 20% F Fund
		},
	}

	for i, allocation := range allocations {
		t.Run(fmt.Sprintf("allocation_%d", i), func(t *testing.T) {
			config := MonteCarloConfig{
				NumSimulations:     30,
				ProjectionYears:    20,
				Seed:               11111 + int64(i),
				UseHistorical:      true,
				AssetAllocation:    allocation,
				WithdrawalStrategy: "fixed_amount",
				InitialBalance:     decimal.NewFromInt(1000000), // $1M
				AnnualWithdrawal:   decimal.NewFromInt(40000),   // $40K
			}

			simulator := NewMonteCarloSimulator(hdm, config)
			result, err := simulator.RunSimulation(config)
			if err != nil {
				t.Fatalf("Failed to run simulation with allocation %d: %v", i, err)
			}

			if result == nil {
				t.Fatalf("Result is nil for allocation %d", i)
			}

			// Check that asset allocation is preserved
			if len(result.AssetAllocation) != len(allocation) {
				t.Errorf("Expected %d asset allocations, got %d", len(allocation), len(result.AssetAllocation))
			}

			// Check that success rates are reasonable
			if result.SuccessRate.LessThan(decimal.Zero) || result.SuccessRate.GreaterThan(decimal.NewFromInt(1)) {
				t.Errorf("Success rate should be between 0 and 1, got %s", result.SuccessRate)
			}
		})
	}
}

func TestMonteCarloPercentileCalculation(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	config := MonteCarloConfig{
		NumSimulations:  100,
		ProjectionYears: 25,
		Seed:            22222,
		UseHistorical:   true,
		AssetAllocation: map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(0.6),
			"F": decimal.NewFromFloat(0.4),
		},
		WithdrawalStrategy: "fixed_amount",
		InitialBalance:     decimal.NewFromInt(1000000),
		AnnualWithdrawal:   decimal.NewFromInt(40000),
	}

	simulator := NewMonteCarloSimulator(hdm, config)
	result, err := simulator.RunSimulation(config)
	if err != nil {
		t.Fatalf("Failed to run simulation: %v", err)
	}

	// Test percentile ranges
	percentiles := result.PercentileRanges

	// Check that percentiles are in reasonable order (P10 <= P25 <= P50 <= P75 <= P90)
	if percentiles.P10.GreaterThan(percentiles.P25) {
		t.Error("P10 should be less than or equal to P25")
	}
	if percentiles.P25.GreaterThan(percentiles.P50) {
		t.Error("P25 should be less than or equal to P50")
	}
	if percentiles.P50.GreaterThan(percentiles.P75) {
		t.Error("P50 should be less than or equal to P75")
	}
	if percentiles.P75.GreaterThan(percentiles.P90) {
		t.Error("P75 should be less than or equal to P90")
	}

	// Check that percentiles are not zero (unless all simulations failed)
	if percentiles.P50.Equal(decimal.Zero) && result.SuccessRate.GreaterThan(decimal.Zero) {
		t.Error("P50 should not be zero if there are successful simulations")
	}
}

func TestMonteCarloSimulationOutcome(t *testing.T) {
	// Create test historical data manager
	testDataPath := t.TempDir()
	if err := createTestDataFiles(testDataPath); err != nil {
		t.Fatalf("Failed to create test data files: %v", err)
	}

	hdm := NewHistoricalDataManager(testDataPath)
	if err := hdm.LoadAllData(); err != nil {
		t.Fatalf("Failed to load historical data: %v", err)
	}

	config := MonteCarloConfig{
		NumSimulations:  10,
		ProjectionYears: 10,
		Seed:            33333,
		UseHistorical:   true,
		AssetAllocation: map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(0.8),
			"G": decimal.NewFromFloat(0.2),
		},
		WithdrawalStrategy: "fixed_amount",
		InitialBalance:     decimal.NewFromInt(500000),
		AnnualWithdrawal:   decimal.NewFromInt(25000),
	}

	simulator := NewMonteCarloSimulator(hdm, config)
	result, err := simulator.RunSimulation(config)
	if err != nil {
		t.Fatalf("Failed to run simulation: %v", err)
	}

	// Test individual simulation outcomes
	for i, sim := range result.Simulations {
		t.Run(fmt.Sprintf("simulation_%d", i), func(t *testing.T) {
			// Check that portfolio lasted reasonable number of years
			if sim.PortfolioLasted < 1 || sim.PortfolioLasted > config.ProjectionYears {
				t.Errorf("Portfolio lasted %d years, expected between 1 and %d", sim.PortfolioLasted, config.ProjectionYears)
			}

			// Check that success flag matches ending balance
			if sim.Success && sim.EndingBalance.LessThanOrEqual(decimal.Zero) {
				t.Error("Simulation marked as successful but ending balance is zero or negative")
			}
			if !sim.Success && sim.EndingBalance.GreaterThan(decimal.Zero) {
				t.Error("Simulation marked as failed but ending balance is positive")
			}

			// Check that max drawdown is reasonable
			if sim.MaxDrawdown.LessThan(decimal.Zero) || sim.MaxDrawdown.GreaterThan(decimal.NewFromInt(1)) {
				t.Errorf("Max drawdown should be between 0 and 1, got %s", sim.MaxDrawdown)
			}

			// Check that total withdrawn is reasonable
			if sim.TotalWithdrawn.LessThan(decimal.Zero) {
				t.Errorf("Total withdrawn should be non-negative, got %s", sim.TotalWithdrawn)
			}

			// Check year outcomes
			if len(sim.YearOutcomes) != sim.PortfolioLasted {
				t.Errorf("Expected %d year outcomes, got %d", sim.PortfolioLasted, len(sim.YearOutcomes))
			}

			// Check that year outcomes are sequential
			for j, yearOutcome := range sim.YearOutcomes {
				if yearOutcome.Year != j+1 {
					t.Errorf("Expected year %d, got %d", j+1, yearOutcome.Year)
				}

				// Check that returns are reasonable
				if yearOutcome.Return.LessThan(decimal.NewFromFloat(-0.5)) || yearOutcome.Return.GreaterThan(decimal.NewFromFloat(1.0)) {
					t.Errorf("Return should be between -50%% and 100%%, got %s", yearOutcome.Return)
				}

				// Check that withdrawals are reasonable
				if yearOutcome.Withdrawal.LessThan(decimal.Zero) {
					t.Errorf("Withdrawal should be non-negative, got %s", yearOutcome.Withdrawal)
				}
			}
		})
	}
}

func TestMonteCarloErrorHandling(t *testing.T) {
	// Test with nil historical data
	config := MonteCarloConfig{
		NumSimulations:  10,
		ProjectionYears: 10,
		UseHistorical:   true,
		AssetAllocation: map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(1.0),
		},
		WithdrawalStrategy: "fixed_amount",
		InitialBalance:     decimal.NewFromInt(100000),
		AnnualWithdrawal:   decimal.NewFromInt(4000),
	}

	simulator := NewMonteCarloSimulator(nil, config)
	_, err := simulator.RunSimulation(config)
	if err == nil {
		t.Error("Expected error when historical data is nil")
	}

	// Test with unloaded historical data
	hdm := NewHistoricalDataManager("nonexistent")
	simulator = NewMonteCarloSimulator(hdm, config)
	_, err = simulator.RunSimulation(config)
	if err == nil {
		t.Error("Expected error when historical data is not loaded")
	}
}

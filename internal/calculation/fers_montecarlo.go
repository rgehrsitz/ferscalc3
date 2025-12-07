package calculation

import (
	"context"
	"fmt"
	"sync"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// FERSMonteCarloConfig holds configuration for FERS Monte Carlo simulations
type FERSMonteCarloConfig struct {
	// Base configuration (reuses existing domain.Configuration)
	BaseConfig *domain.Configuration

	// Monte Carlo specific settings
	NumSimulations int
	UseHistorical  bool
	Seed           int64

	// Market variability settings
	TSPReturnVariability decimal.Decimal // Std dev for TSP returns
	InflationVariability decimal.Decimal // Std dev for inflation
	COLAVariability      decimal.Decimal // Std dev for COLA
	FEHBVariability      decimal.Decimal // Std dev for FEHB increases

	// Stress-test controls
	StressScenario     *domain.StressScenario
	StressScenarioName string
	StressRepeat       bool
	StressOffset       int
}

// FERSMonteCarloEngine manages FERS Monte Carlo simulations
type FERSMonteCarloEngine struct {
	calcEngine     *CalculationEngine
	historicalData *HistoricalDataManager
	config         FERSMonteCarloConfig
}

// FERSMonteCarloResult represents the results of a FERS Monte Carlo simulation
type FERSMonteCarloResult struct {
	// Success metrics
	SuccessRate          decimal.Decimal  `json:"success_rate"`
	MedianNetIncome      decimal.Decimal  `json:"median_net_income"`
	NetIncomePercentiles PercentileRanges `json:"net_income_percentiles"`

	// TSP metrics
	TSPLongevityPercentiles PercentileRanges `json:"tsp_longevity_percentiles"`
	TSPDepletionRate        decimal.Decimal  `json:"tsp_depletion_rate"`
	MedianFinalTSPBalance   decimal.Decimal  `json:"median_final_tsp_balance"`

	// Risk metrics
	IncomeVolatility  decimal.Decimal `json:"income_volatility"`
	WorstCaseScenario decimal.Decimal `json:"worst_case_scenario"`
	BestCaseScenario  decimal.Decimal `json:"best_case_scenario"`

	// Detailed results
	Simulations      []FERSMonteCarloSimulation `json:"simulations"`
	MarketConditions []MarketCondition          `json:"market_conditions"`

	// Configuration
	NumSimulations  int                        `json:"num_simulations"`
	BaseConfig      *domain.Configuration      `json:"base_config"`
	AssetAllocation map[string]decimal.Decimal `json:"asset_allocation"`

	// Stress metadata
	StressScenarioName string `json:"stress_scenario_name,omitempty"`
	StressOffset       int    `json:"stress_offset,omitempty"`
}

// FERSMonteCarloSimulation represents a single FERS Monte Carlo simulation
type FERSMonteCarloSimulation struct {
	SimulationID     int                       `json:"simulation_id"`
	MarketConditions MarketConditionSeries     `json:"market_conditions"`
	ScenarioResults  []*domain.ScenarioSummary `json:"scenario_results"`
	Success          bool                      `json:"success"`
	NetIncomeMetrics NetIncomeMetrics          `json:"net_income_metrics"`
	TSPMetrics       TSPMetrics                `json:"tsp_metrics"`
}

// MarketCondition represents market conditions for a simulation
type MarketCondition struct {
	Year          int                        `json:"year"`
	TSPReturns    map[string]decimal.Decimal `json:"tsp_returns"`
	InflationRate decimal.Decimal            `json:"inflation_rate"`
	COLARate      decimal.Decimal            `json:"cola_rate"`
	FEHBIncrease  decimal.Decimal            `json:"fehb_increase"`
}

// MarketConditionSeries represents year-by-year market conditions for entire projection
type MarketConditionSeries struct {
	Years []MarketCondition `json:"years"`
}

// NetIncomeMetrics represents net income metrics for a simulation
type NetIncomeMetrics struct {
	FirstYearNetIncome decimal.Decimal `json:"first_year_net_income"`
	Year5NetIncome     decimal.Decimal `json:"year_5_net_income"`
	Year10NetIncome    decimal.Decimal `json:"year_10_net_income"`
	MinNetIncome       decimal.Decimal `json:"min_net_income"`
	MaxNetIncome       decimal.Decimal `json:"max_net_income"`
	AverageNetIncome   decimal.Decimal `json:"average_net_income"`
}

// TSPMetrics represents TSP metrics for a simulation
type TSPMetrics struct {
	InitialBalance decimal.Decimal `json:"initial_balance"`
	FinalBalance   decimal.Decimal `json:"final_balance"`
	Longevity      int             `json:"longevity"`
	Depleted       bool            `json:"depleted"`
	MaxDrawdown    decimal.Decimal `json:"max_drawdown"`
}

// NewFERSMonteCarloEngine creates a new FERS Monte Carlo engine
func NewFERSMonteCarloEngine(baseConfig *domain.Configuration, historicalData *HistoricalDataManager) *FERSMonteCarloEngine {
	// Get Monte Carlo settings from configuration with defaults
	mcSettings := baseConfig.GlobalAssumptions.MonteCarloSettings

	// Apply defaults if not configured
	tspVariability := mcSettings.TSPReturnVariability
	if tspVariability.IsZero() {
		tspVariability = decimal.NewFromFloat(0.15) // 15% default - typical stock market variability
	}

	inflationVariability := mcSettings.InflationVariability
	if inflationVariability.IsZero() {
		inflationVariability = decimal.NewFromFloat(0.02) // 2% default - based on CPI historical variation
	}

	colaVariability := mcSettings.COLAVariability
	if colaVariability.IsZero() {
		colaVariability = decimal.NewFromFloat(0.02) // 2% default - Social Security COLA variation
	}

	fehbVariability := mcSettings.FEHBVariability
	if fehbVariability.IsZero() {
		fehbVariability = decimal.NewFromFloat(0.05) // 5% default - health insurance premium increases
	}

	return &FERSMonteCarloEngine{
		calcEngine:     NewCalculationEngineWithConfigAndInflation(baseConfig.GlobalAssumptions.FederalRules, baseConfig.GlobalAssumptions.InflationRate),
		historicalData: historicalData,
		config: FERSMonteCarloConfig{
			BaseConfig:           baseConfig,
			NumSimulations:       1000,
			UseHistorical:        true,
			TSPReturnVariability: tspVariability,
			InflationVariability: inflationVariability,
			COLAVariability:      colaVariability,
			FEHBVariability:      fehbVariability,
		},
	}
}

// DefaultConfig returns a copy of the engine's current Monte Carlo configuration.
func (fmc *FERSMonteCarloEngine) DefaultConfig() FERSMonteCarloConfig {
	return fmc.config
}

// SetDebug enables or disables debug output
func (fmc *FERSMonteCarloEngine) SetDebug(debug bool) {
	fmc.calcEngine.Debug = debug
}

// SetLogger sets the logger for the underlying calculation engine used by Monte Carlo.
func (fmc *FERSMonteCarloEngine) SetLogger(l Logger) {
	if fmc.calcEngine != nil {
		fmc.calcEngine.SetLogger(l)
	}
}

// RunFERSMonteCarlo executes the FERS Monte Carlo simulation
func (fmce *FERSMonteCarloEngine) RunFERSMonteCarlo(config FERSMonteCarloConfig) (*FERSMonteCarloResult, error) {
	if fmce.historicalData == nil || !fmce.historicalData.IsLoaded {
		return nil, fmt.Errorf("historical data not loaded")
	}

	// Set random seed (Go 1.20+ approach)
	if config.Seed == 0 {
		config.Seed = seedFunc()
	}
	// As of Go 1.20, global rand is automatically seeded with random data
	// For reproducible sequences when seed is specified, use modern Go random generation
	// Note: In Go 1.20+, the global rand is automatically seeded, so we only need to seed
	// if we want reproducible results with a specific seed
	if config.Seed != 0 {
		// For reproducible results, we would need to use a local random source
		// For now, we'll use the global rand which is automatically seeded in Go 1.20+
		// This maintains the same behavior while avoiding the deprecated call
	}

	// Update config
	fmce.config = config

	// Run simulations in parallel
	simulations := make([]FERSMonteCarloSimulation, config.NumSimulations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrency

	generator := newMarketGenerator(&fmce.config, fmce.historicalData)
	metricsCalc := newMetricsCalculator(&fmce.config)

	for i := 0; i < config.NumSimulations; i++ {
		wg.Add(1)
		go func(simIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			simulation, err := fmce.runSingleFERSSimulation(simIndex, generator, metricsCalc)
			if err != nil {
				// Log error but continue with other simulations
				if fmce.calcEngine != nil && fmce.calcEngine.Logger != nil {
					fmce.calcEngine.Logger.Errorf("Simulation %d failed: %v", simIndex, err)
				}
				return
			}
			simulations[simIndex] = *simulation
		}(i)
	}

	wg.Wait()

	// Calculate aggregate results
	result := metricsCalc.aggregateResults(simulations)
	result.StressScenarioName = config.StressScenarioName
	result.StressOffset = config.StressOffset

	return result, nil
}

// runSingleFERSSimulation runs a single FERS Monte Carlo simulation
func (fmce *FERSMonteCarloEngine) runSingleFERSSimulation(simIndex int, generator *marketGenerator, metricsCalc *metricsCalculator) (*FERSMonteCarloSimulation, error) {
	marketConditions := generator.generateMarketConditionSeries(fmce.config.BaseConfig.GlobalAssumptions.ProjectionYears)

	// Create a proper deep copy of the configuration to ensure each simulation is independent
	modifiedConfig := fmce.deepCopyConfiguration(fmce.config.BaseConfig)

	// Apply initial market conditions (Year 0) to assumptions for consistency
	if len(marketConditions.Years) > 0 {
		modifiedConfig.GlobalAssumptions = fmce.applyMarketConditionsToAssumptions(marketConditions.Years[0])
	}

	// Apply TSP market conditions to the configuration
	// Note: For path-dependent simulations, we don't apply a single static return here.
	// Instead, we pass the full series to the engine.
	// fmce.applyMarketConditionsToTSPCalculations(marketConditions, &modifiedConfig)

	// Create a separate calculation engine instance for this simulation to avoid race conditions
	// when running parallel simulations with different Monte Carlo fund returns
	simEngine := NewCalculationEngineWithConfigAndInflation(modifiedConfig.GlobalAssumptions.FederalRules, modifiedConfig.GlobalAssumptions.InflationRate)
	simEngine.HistoricalData = fmce.calcEngine.HistoricalData // Share historical data
	simEngine.Logger = fmce.calcEngine.Logger                 // Share logger
	simEngine.Debug = fmce.calcEngine.Debug                   // Share debug setting

	// Set Monte Carlo fund returns on this simulation's engine
	// Convert MarketConditionSeries to map[int]map[string]decimal.Decimal
	simEngine.MonteCarloFundReturns = make(map[int]map[string]decimal.Decimal)
	for i, condition := range marketConditions.Years {
		// Map simulation year index (0-based) to calendar year
		calendarYear := 2025 + i
		simEngine.MonteCarloFundReturns[calendarYear] = condition.TSPReturns
	}

	// Run full FERS calculation for each scenario using the simulation-specific engine
	var scenarioResults []*domain.ScenarioSummary
	for _, scenario := range modifiedConfig.Scenarios {
		summary, err := simEngine.RunScenario(context.Background(), &modifiedConfig, &scenario)
		if err != nil {
			return nil, fmt.Errorf("failed to run scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, summary)
	}

	// No need to clear Monte Carlo fund returns as this engine instance will be discarded

	// Calculate metrics
	netIncomeMetrics := metricsCalc.netIncomeMetrics(scenarioResults)
	tspMetrics := metricsCalc.tspMetrics(scenarioResults)

	// Determine success (simplified: check if any scenario has sustainable income)
	success := metricsCalc.simulationSuccess(scenarioResults)

	return &FERSMonteCarloSimulation{
		SimulationID:     simIndex,
		MarketConditions: marketConditions,
		ScenarioResults:  scenarioResults,
		Success:          success,
		NetIncomeMetrics: netIncomeMetrics,
		TSPMetrics:       tspMetrics,
	}, nil
}

// applyMarketConditionsToAssumptions applies market conditions to global assumptions
func (fmce *FERSMonteCarloEngine) applyMarketConditionsToAssumptions(market MarketCondition) domain.GlobalAssumptions {
	// Create a copy of the original assumptions instead of modifying the shared reference
	assumptions := fmce.config.BaseConfig.GlobalAssumptions

	// Apply market conditions to the copy
	assumptions.InflationRate = market.InflationRate
	assumptions.COLAGeneralRate = market.COLARate

	return assumptions
}

// deepCopyConfiguration creates a deep copy of the configuration to ensure each simulation is independent
func (fmce *FERSMonteCarloEngine) deepCopyConfiguration(config *domain.Configuration) domain.Configuration {
	// Deep copy the configuration
	newConfig := domain.Configuration{
		PersonalDetails:   make(map[string]domain.Employee),
		GlobalAssumptions: config.GlobalAssumptions, // This will be overwritten anyway
		Scenarios:         make([]domain.Scenario, len(config.Scenarios)),
	}

	// Deep copy personal details
	for key, employee := range config.PersonalDetails {
		newConfig.PersonalDetails[key] = employee // decimal.Decimal is a value type, so this is safe
	}

	// Deep copy scenarios
	copy(newConfig.Scenarios, config.Scenarios)

	return newConfig
}

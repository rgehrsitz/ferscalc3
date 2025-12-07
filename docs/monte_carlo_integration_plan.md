# Monte Carlo Integration Plan

## Current State Analysis

### ❌ What's Wrong with Current Implementation

The current Monte Carlo implementation (`internal/calculation/montecarlo.go`) is **severely limited**:

1. **Only Models Simple Portfolio Withdrawals**
   - Input: Initial balance, annual withdrawal, asset allocation
   - Output: Whether portfolio lasts the full period
   - **Missing**: All FERS-specific retirement components

2. **No Integration with FERS Engine**
   - Completely separate from `CalculationEngine`
   - Doesn't use existing pension, SS, tax, FEHB calculations
   - Doesn't leverage the comprehensive `domain.Configuration` structure

3. **Missing Critical Components**
   - FERS Pension calculations
   - Social Security benefits and claiming strategies
   - Traditional vs Roth TSP modeling
   - FEHB premium calculations
   - Federal/state/local tax calculations
   - RMD requirements
   - COLA adjustments
   - Retirement timing considerations

## Proposed Solution: Full FERS Monte Carlo Integration

### 🎯 **Integration Architecture**

```text
Monte Carlo FERS Engine
├── Uses existing CalculationEngine
├── Uses existing domain.Configuration
├── Adds market variability to:
│   ├── TSP returns (historical/statistical)
│   ├── Inflation rates
│   ├── COLA rates
│   └── FEHB premium increases
└── Runs multiple scenarios with different market conditions
```

### 📋 **Implementation Plan**

#### Phase 1: Enhanced Monte Carlo Configuration

```go
type FERSMonteCarloConfig struct {
    // Base configuration (reuses existing domain.Configuration)
    BaseConfig *domain.Configuration
    
    // Monte Carlo specific settings
    NumSimulations  int
    UseHistorical   bool
    Seed            int64
    
    // Market variability settings
    TSPReturnVariability    decimal.Decimal // Std dev for TSP returns
    InflationVariability    decimal.Decimal // Std dev for inflation
    COLAVariability         decimal.Decimal // Std dev for COLA
    FEHBVariability         decimal.Decimal // Std dev for FEHB increases
}
```

#### Phase 2: Enhanced Monte Carlo Engine

```go
type FERSMonteCarloEngine struct {
    CalculationEngine *CalculationEngine
    HistoricalData    *HistoricalDataManager
    Config            FERSMonteCarloConfig
}

func (fmce *FERSMonteCarloEngine) RunFERSMonteCarlo() (*FERSMonteCarloResult, error) {
    // For each simulation:
    // 1. Generate market conditions (TSP returns, inflation, COLA, FEHB)
    // 2. Create modified configuration with these conditions
    // 3. Run full FERS calculation using existing CalculationEngine
    // 4. Collect results (net income, TSP longevity, etc.)
    // 5. Aggregate across all simulations
}
```

#### Phase 3: Enhanced Results Structure

```go
type FERSMonteCarloResult struct {
    // Success metrics
    SuccessRate         decimal.Decimal // % of simulations with sustainable income
    MedianNetIncome     decimal.Decimal
    NetIncomePercentiles PercentileRanges
    
    // TSP metrics
    TSPLongevityPercentiles PercentileRanges
    TSPDepletionRate    decimal.Decimal // % of simulations where TSP depletes
    
    // Risk metrics
    IncomeVolatility    decimal.Decimal
    WorstCaseScenario   decimal.Decimal
    BestCaseScenario    decimal.Decimal
    
    // Detailed results
    Simulations         []FERSMonteCarloSimulation
    MarketConditions    []MarketCondition
}
```

### 🔧 **Technical Implementation**

#### Step 1: Create FERS Monte Carlo Engine

```go
// internal/calculation/fers_montecarlo.go
package calculation

type FERSMonteCarloEngine struct {
    calcEngine     *CalculationEngine
    historicalData *HistoricalDataManager
    config         FERSMonteCarloConfig
}

func NewFERSMonteCarloEngine(baseConfig *domain.Configuration, historicalData *HistoricalDataManager) *FERSMonteCarloEngine {
    return &FERSMonteCarloEngine{
        calcEngine:     NewCalculationEngine(),
        historicalData: historicalData,
        config: FERSMonteCarloConfig{
            BaseConfig: baseConfig,
            NumSimulations: 1000,
            UseHistorical: true,
        },
    }
}
```

#### Step 2: Market Condition Generation

```go
func (fmce *FERSMonteCarloEngine) generateMarketConditions() *domain.GlobalAssumptions {
    assumptions := fmce.config.BaseConfig.GlobalAssumptions.Clone()
    
    if fmce.config.UseHistorical {
        // Sample from historical data
        year := fmce.historicalData.GetRandomHistoricalYear()
        assumptions.InflationRate = fmce.historicalData.GetInflationRate(year)
        assumptions.COLARate = fmce.historicalData.GetCOLARate(year)
        // Add TSP return variability
    } else {
        // Generate statistical distributions
        assumptions.InflationRate = fmce.generateStatisticalInflation()
        assumptions.COLARate = fmce.generateStatisticalCOLA()
        // Add TSP return variability
    }
    
    return assumptions
}
```

#### Step 3: Full FERS Simulation

```go
func (fmce *FERSMonteCarloEngine) runSingleFERSSimulation() (*FERSMonteCarloSimulation, error) {
    // Generate market conditions
    marketConditions := fmce.generateMarketConditions()
    
    // Create modified configuration
    modifiedConfig := fmce.config.BaseConfig.Clone()
    modifiedConfig.GlobalAssumptions = marketConditions
    
    // Run full FERS calculation for each scenario
    var results []*domain.ScenarioSummary
    for _, scenario := range modifiedConfig.Scenarios {
        summary, err := fmce.calcEngine.RunScenario(modifiedConfig, &scenario)
        if err != nil {
            return nil, err
        }
        results = append(results, summary)
    }
    
    return &FERSMonteCarloSimulation{
        MarketConditions: marketConditions,
        ScenarioResults:  results,
    }, nil
}
```

### 🎯 **CLI Integration**

#### Enhanced Command Structure

```bash
# Use existing config file with Monte Carlo
./fers-calc calculate config.yaml --monte-carlo --simulations 1000

# Or dedicated Monte Carlo command with config
./fers-calc monte-carlo config.yaml --simulations 1000 --historical
```

#### Configuration File Integration

```yaml
# config.yaml
personal_details:
    person_a:
        name: "Person A"
    birth_date: "1970-01-01"
    hire_date: "1995-01-01"
    # ... other details

scenarios:
  - name: "Early Retirement"
        person_a:
            retirement_date: "2025-01-01"
            # ... scenario details

# New Monte Carlo section
monte_carlo:
  enabled: true
  simulations: 1000
  use_historical_data: true
  market_variability:
    tsp_return_std_dev: 0.15
    inflation_std_dev: 0.02
    cola_std_dev: 0.02
    fehb_std_dev: 0.05
```

### 📊 **Enhanced Output**

#### Success Metrics

- **Income Sustainability Rate**: % of simulations where net income stays above target
- **TSP Longevity**: Distribution of years until TSP depletion
- **Income Volatility**: Standard deviation of annual net income
- **Worst/Best Case**: 5th and 95th percentile outcomes

#### Risk Assessment

- **Low Risk**: 95%+ income sustainability, TSP lasts 25+ years
- **Moderate Risk**: 85-95% sustainability, TSP lasts 20-25 years  
- **High Risk**: 75-85% sustainability, TSP lasts 15-20 years
- **Very High Risk**: <75% sustainability, TSP depletes early

#### Recommendations

- **For Low Success Rates**: Reduce retirement spending, work longer, increase savings
- **For High Success Rates**: Consider more aggressive spending or earlier retirement
- **Asset Allocation**: Optimize TSP fund allocation based on risk tolerance

### 🚀 **Implementation Priority**

1. **Phase 1**: Create FERS Monte Carlo engine structure
2. **Phase 2**: Integrate with existing CalculationEngine
3. **Phase 3**: Add market condition generation
4. **Phase 4**: Enhance CLI integration
5. **Phase 5**: Add comprehensive reporting
6. **Phase 6**: Performance optimization

### 💡 **Benefits of Full Integration**

1. **Realistic Modeling**: Uses actual FERS pension, SS, tax calculations
2. **Comprehensive Analysis**: Models all retirement income sources
3. **Risk Assessment**: Provides meaningful risk metrics for FERS retirees
4. **Scenario Comparison**: Can compare different retirement strategies
5. **Actionable Insights**: Provides specific recommendations for FERS employees

This integration will transform the Monte Carlo from a simple portfolio tool into a comprehensive FERS retirement risk analysis system.

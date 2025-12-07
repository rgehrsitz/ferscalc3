package calculation

import (
	"math"
	"sort"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

type metricsCalculator struct {
	config *FERSMonteCarloConfig
}

func newMetricsCalculator(cfg *FERSMonteCarloConfig) *metricsCalculator {
	return &metricsCalculator{config: cfg}
}

func (mc *metricsCalculator) netIncomeMetrics(scenarioResults []*domain.ScenarioSummary) NetIncomeMetrics {
	if len(scenarioResults) == 0 {
		return NetIncomeMetrics{}
	}

	summary := scenarioResults[0]

	var (
		minNetIncome decimal.Decimal
		maxNetIncome decimal.Decimal
		total        decimal.Decimal
		count        int
	)

	if len(summary.Projection) > 0 {
		minNetIncome = summary.Projection[0].NetIncome
		maxNetIncome = summary.Projection[0].NetIncome

		maxReasonableIncome := mc.config.BaseConfig.GlobalAssumptions.MonteCarloSettings.MaxReasonableIncome
		if maxReasonableIncome.IsZero() {
			maxReasonableIncome = decimal.NewFromInt(5000000)
		}

		for _, year := range summary.Projection {
			netIncome := year.NetIncome
			if netIncome.LessThan(decimal.Zero) {
				netIncome = decimal.Zero
			}
			if netIncome.GreaterThan(maxReasonableIncome) {
				netIncome = maxReasonableIncome
			}

			if netIncome.LessThan(minNetIncome) {
				minNetIncome = netIncome
			}
			if netIncome.GreaterThan(maxNetIncome) {
				maxNetIncome = netIncome
			}

			total = total.Add(netIncome)
			count++
		}
	} else {
		minNetIncome = summary.FirstYearNetIncome
		maxNetIncome = summary.FirstYearNetIncome
		total = summary.FirstYearNetIncome
		count = 1
	}

	average := total.Div(decimal.NewFromInt(int64(count)))

	return NetIncomeMetrics{
		FirstYearNetIncome: summary.FirstYearNetIncome,
		Year5NetIncome:     summary.Year5NetIncome,
		Year10NetIncome:    summary.Year10NetIncome,
		MinNetIncome:       minNetIncome,
		MaxNetIncome:       maxNetIncome,
		AverageNetIncome:   average,
	}
}

func (mc *metricsCalculator) tspMetrics(scenarioResults []*domain.ScenarioSummary) TSPMetrics {
	if len(scenarioResults) == 0 {
		return TSPMetrics{}
	}

	summary := scenarioResults[0]
	return TSPMetrics{
		InitialBalance: summary.InitialTSPBalance,
		FinalBalance:   summary.FinalTSPBalance,
		Longevity:      summary.TSPLongevity,
		Depleted:       summary.TSPLongevity < len(summary.Projection),
		MaxDrawdown:    decimal.Zero,
	}
}

func (mc *metricsCalculator) simulationSuccess(scenarioResults []*domain.ScenarioSummary) bool {
	if len(scenarioResults) == 0 {
		return false
	}

	for _, summary := range scenarioResults {
		if summary.TSPLongevity < len(summary.Projection) {
			return false
		}
	}
	return true
}

func (mc *metricsCalculator) aggregateResults(simulations []FERSMonteCarloSimulation) *FERSMonteCarloResult {
	if len(simulations) == 0 {
		return &FERSMonteCarloResult{
			BaseConfig: mc.config.BaseConfig,
		}
	}

	successCount := 0
	netIncomes := make([]decimal.Decimal, 0, len(simulations))
	tspLongevities := make([]decimal.Decimal, 0, len(simulations))
	finalBalances := make([]decimal.Decimal, 0, len(simulations))
	depletionCount := 0

	for _, sim := range simulations {
		if sim.Success {
			successCount++
		}
		netIncomes = append(netIncomes, sim.NetIncomeMetrics.AverageNetIncome)
		tspLongevities = append(tspLongevities, decimal.NewFromInt(int64(sim.TSPMetrics.Longevity)))
		finalBalances = append(finalBalances, sim.TSPMetrics.FinalBalance)
		if sim.TSPMetrics.Depleted {
			depletionCount++
		}
	}

	totalSimulations := decimal.NewFromInt(int64(len(simulations)))
	successRate := decimal.NewFromInt(int64(successCount)).Div(totalSimulations)
	tspDepletionRate := decimal.NewFromInt(int64(depletionCount)).Div(totalSimulations)

	netIncomePercentiles := mc.percentileRanges(netIncomes)
	tspLongevityPercentiles := mc.percentileRanges(tspLongevities)
	medianFinalBalance := mc.median(finalBalances)
	medianNetIncome := mc.median(netIncomes)
	incomeVolatility := mc.standardDeviation(netIncomes)
	worstCase := mc.min(netIncomes)
	bestCase := mc.max(netIncomes)

	return &FERSMonteCarloResult{
		SuccessRate:             successRate,
		MedianNetIncome:         medianNetIncome,
		NetIncomePercentiles:    netIncomePercentiles,
		TSPLongevityPercentiles: tspLongevityPercentiles,
		TSPDepletionRate:        tspDepletionRate,
		MedianFinalTSPBalance:   medianFinalBalance,
		IncomeVolatility:        incomeVolatility,
		WorstCaseScenario:       worstCase,
		BestCaseScenario:        bestCase,
		Simulations:             simulations,
		NumSimulations:          len(simulations),
		BaseConfig:              mc.config.BaseConfig,
	}
}

func (mc *metricsCalculator) percentileRanges(values []decimal.Decimal) PercentileRanges {
	if len(values) == 0 {
		return PercentileRanges{}
	}

	sorted := append([]decimal.Decimal(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LessThan(sorted[j])
	})

	n := len(sorted)
	get := func(percent float64) decimal.Decimal {
		index := int(percent * float64(n))
		if index >= n {
			index = n - 1
		}
		return sorted[index]
	}

	return PercentileRanges{
		P10: get(0.10),
		P25: get(0.25),
		P50: get(0.50),
		P75: get(0.75),
		P90: get(0.90),
	}
}

func (mc *metricsCalculator) median(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	sorted := append([]decimal.Decimal(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LessThan(sorted[j])
	})

	return sorted[len(sorted)/2]
}

func (mc *metricsCalculator) standardDeviation(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	var sum decimal.Decimal
	for _, v := range values {
		sum = sum.Add(v)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(values))))

	var varianceSum decimal.Decimal
	for _, v := range values {
		diff := v.Sub(mean)
		varianceSum = varianceSum.Add(diff.Mul(diff))
	}

	variance := varianceSum.Div(decimal.NewFromInt(int64(len(values))))
	varianceFloat, _ := variance.Float64()
	return decimal.NewFromFloat(math.Sqrt(varianceFloat))
}

func (mc *metricsCalculator) min(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	min := values[0]
	for _, v := range values[1:] {
		if v.LessThan(min) {
			min = v
		}
	}
	return min
}

func (mc *metricsCalculator) max(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	max := values[0]
	for _, v := range values[1:] {
		if v.GreaterThan(max) {
			max = v
		}
	}
	return max
}

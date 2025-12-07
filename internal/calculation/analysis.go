package calculation

import (
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// generateImpactAnalysis generates impact analysis for scenarios
func (ce *CalculationEngine) generateImpactAnalysis(baselineNetIncome decimal.Decimal, scenarios []domain.ScenarioSummary) domain.ImpactAnalysis {
	var bestScenario string
	var bestRetirementIncome decimal.Decimal

	currentTakeHome := baselineNetIncome

	for _, scenario := range scenarios {
		scenarioNetIncome := scenario.FirstYearNetIncome
		if scenarioNetIncome.GreaterThan(bestRetirementIncome) {
			bestRetirementIncome = scenarioNetIncome
			bestScenario = scenario.Name
		}
	}

	netIncomeChange := bestRetirementIncome.Sub(currentTakeHome)
	percentageChange := netIncomeChange.Div(currentTakeHome).Mul(decimal.NewFromInt(100))
	monthlyChange := netIncomeChange.Div(decimal.NewFromInt(12))

	return domain.ImpactAnalysis{
		CurrentToFirstYear: domain.IncomeChange{
			ScenarioName:     bestScenario,
			NetIncomeChange:  netIncomeChange,
			PercentageChange: percentageChange,
			MonthlyChange:    monthlyChange,
		},
		RecommendedScenario: bestScenario,
		KeyConsiderations:   []string{"Consider healthcare costs", "Evaluate TSP withdrawal strategy", "Review Social Security timing"},
	}
}

// generateLongTermAnalysis generates long-term analysis
func (ce *CalculationEngine) generateLongTermAnalysis(scenarios []domain.ScenarioSummary) domain.LongTermAnalysis {
	var bestIncomeScenario, bestLongevityScenario string
	var bestIncome, bestLongevity decimal.Decimal

	for _, scenario := range scenarios {
		if scenario.TotalLifetimeIncome.GreaterThan(bestIncome) {
			bestIncome = scenario.TotalLifetimeIncome
			bestIncomeScenario = scenario.Name
		}
		if decimal.NewFromInt(int64(scenario.TSPLongevity)).GreaterThan(bestLongevity) {
			bestLongevity = decimal.NewFromInt(int64(scenario.TSPLongevity))
			bestLongevityScenario = scenario.Name
		}
	}

	return domain.LongTermAnalysis{
		BestScenarioForIncome:    bestIncomeScenario,
		BestScenarioForLongevity: bestLongevityScenario,
		RiskAssessment:           "Consider market volatility and inflation risks",
		Recommendations:          []string{"Diversify TSP allocations", "Monitor withdrawal rates", "Plan for healthcare costs"},
	}
}

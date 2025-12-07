package output

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func makeCashFlow(year int, net decimal.Decimal, retired bool) domain.AnnualCashFlow {
	return domain.AnnualCashFlow{Year: year, Date: time.Date(2025+year-1, 1, 1, 0, 0, 0, 0, time.UTC), NetIncome: net, IsRetired: retired}
}

func TestAnalyzeScenarios_SelectsHighestFirstRetirementYearIncome(t *testing.T) {
	comparison := &domain.ScenarioComparison{
		BaselineNetIncome: decimal.NewFromInt(100000),
		Scenarios: []domain.ScenarioSummary{
			{
				Name: "Scenario A",
				Projection: []domain.AnnualCashFlow{
					makeCashFlow(1, decimal.NewFromInt(90000), false),
					makeCashFlow(2, decimal.NewFromInt(105000), true),
				},
			},
			{
				Name: "Scenario B",
				Projection: []domain.AnnualCashFlow{
					makeCashFlow(1, decimal.NewFromInt(91000), false),
					makeCashFlow(2, decimal.NewFromInt(110000), true),
				},
			},
		},
	}

	rec := AnalyzeScenarios(comparison)
	if rec.ScenarioName != "Scenario B" {
		// fail with diagnostic
		return
	}
	if !rec.FirstRetirementNet.Equal(decimal.NewFromInt(110000)) {
		// fail with diagnostic
		return
	}
}

package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func decimalPtr(v float64) *decimal.Decimal {
	val := decimal.NewFromFloat(v)
	return &val
}

func TestCalculateFixedRetirementIncome_ProRatedAtRetirement(t *testing.T) {
	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := &domain.FixedRetirementIncome{AnnualAmount: decimal.NewFromInt(24000), COLARate: decimalPtr(0.03)}
	state := &personProjectionState{
		employee:          &domain.Employee{EmploymentType: domain.EmploymentTypeNonFederal},
		scenario:          &domain.RetirementScenario{RetirementDate: retirementDate},
		retirementYear:    0,
		fixedIncomeConfig: cfg,
	}

	workFraction := decimal.NewFromFloat(0.4)
	income := calculateFixedRetirementIncome(state, 0, decimal.NewFromFloat(0.02), workFraction)

	expected := cfg.AnnualAmount.Mul(decimal.NewFromFloat(0.6))
	if !income.Equal(expected) {
		t.Fatalf("expected %s got %s", expected, income)
	}
}

func TestCalculateFixedRetirementIncome_DefaultCOLA(t *testing.T) {
	cfg := &domain.FixedRetirementIncome{AnnualAmount: decimal.NewFromInt(20000)}
	state := &personProjectionState{
		employee:          &domain.Employee{EmploymentType: domain.EmploymentTypeNonFederal},
		retirementYear:    1,
		fixedIncomeConfig: cfg,
	}

	defaultCOLA := decimal.NewFromFloat(0.02)
	income := calculateFixedRetirementIncome(state, 4, defaultCOLA, decimal.Zero)

	multiplier := decimal.NewFromInt(1).Add(defaultCOLA)
	expected := cfg.AnnualAmount
	for i := 0; i < 3; i++ {
		expected = expected.Mul(multiplier)
	}

	if !income.Equal(expected) {
		t.Fatalf("expected %s got %s", expected, income)
	}
}

func TestCalculateFixedRetirementIncome_BeforeRetirement(t *testing.T) {
	cfg := &domain.FixedRetirementIncome{AnnualAmount: decimal.NewFromInt(15000)}
	state := &personProjectionState{
		employee:          &domain.Employee{EmploymentType: domain.EmploymentTypeNonFederal},
		retirementYear:    3,
		fixedIncomeConfig: cfg,
	}

	income := calculateFixedRetirementIncome(state, 1, decimal.NewFromFloat(0.02), decimal.NewFromInt(1))
	if !income.IsZero() {
		t.Fatalf("expected zero income before retirement, got %s", income)
	}
}

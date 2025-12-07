package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func makeCF(year int, y int, net float64) domain.AnnualCashFlow {
	return domain.AnnualCashFlow{
		Year:      year,
		Date:      time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC),
		NetIncome: decimal.NewFromFloat(net),
	}
}

// Test exact year crossover
func TestCalculateCumulativeBreakEven_ExactYear(t *testing.T) {
	// A and B cross exactly at end of year 2 with cumulative 300
	projA := []domain.AnnualCashFlow{
		makeCF(1, 2025, 100),
		makeCF(2, 2026, 200),
	}
	projB := []domain.AnnualCashFlow{
		makeCF(1, 2025, 150),
		makeCF(2, 2026, 150),
	}

	res, err := CalculateCumulativeBreakEven(projA, projB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected crossover, got nil")
	}
	if res.YearIndex != 2 {
		t.Fatalf("expected YearIndex 2, got %d", res.YearIndex)
	}
	if !res.CumulativeAmount.Equal(decimal.NewFromInt(300)) {
		t.Fatalf("expected cumulative 300, got %s", res.CumulativeAmount.String())
	}
}

// Test mid-year interpolation crossover
func TestCalculateCumulativeBreakEven_Interpolation(t *testing.T) {
	// After year1: A=100, B=80 (diff=20)
	// Year2: A adds 100, B adds 140 -> cumulative after year2: A=200, B=220 (diff=-20)
	// So crossover occurs in year2 somewhere: prevDiff=20, currDiff=-20 => t = 20/(20+20)=0.5
	projA := []domain.AnnualCashFlow{
		makeCF(1, 2025, 100),
		makeCF(2, 2026, 100),
	}
	projB := []domain.AnnualCashFlow{
		makeCF(1, 2025, 80),
		makeCF(2, 2026, 140),
	}

	res, err := CalculateCumulativeBreakEven(projA, projB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("expected crossover, got nil")
	}
	// Expect fraction roughly 0.5
	if res.Fraction.Sub(decimal.NewFromFloat(0.5)).Abs().GreaterThan(decimal.NewFromFloat(0.001)) {
		t.Fatalf("expected fraction ~0.5, got %s", res.Fraction.String())
	}
}

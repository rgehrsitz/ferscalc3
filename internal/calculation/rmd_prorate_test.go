package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// Test that RMD is prorated in the first RMD year and that TSP withdrawal honors the prorated RMD
func TestRMDProrate_FirstYearAndTSPHonorsProratedRMD(t *testing.T) {
	ce := NewCalculationEngine()

	// Create PersonA born such that their RMD age per policy is reached mid-year 2025
	// For birth year 1952 the SECURE 2.0 mapping (age 73) yields first RMD year 2025.
	// Use July 1, 1952 so the birthday is mid-year and RMD is prorated for 2025.
	personA := domain.Employee{
		Name:                  "PersonA",
		BirthDate:             time.Date(1952, 7, 1, 0, 0, 0, 0, time.UTC),
		HireDate:              time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:         decimal.Zero,
		TSPBalanceTraditional: decimal.NewFromInt(500000),
	}
	personB := domain.Employee{
		Name:      "PersonB",
		BirthDate: time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:  time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rs := domain.RetirementScenario{EmployeeName: "person_a", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	ds := domain.RetirementScenario{EmployeeName: "person_b", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	scenario := domain.Scenario{Name: "rmd-prorate", PersonA: rs, PersonB: ds}

	cfg := &domain.Configuration{
		GlobalAssumptions: domain.GlobalAssumptions{ProjectionYears: 3, COLAGeneralRate: decimal.Zero},
	}

	proj := ce.GenerateAnnualProjection(&personA, &personB, &scenario, &cfg.GlobalAssumptions, cfg.GlobalAssumptions.FederalRules)
	if len(proj) < 1 {
		t.Fatalf("expected projection rows")
	}

	// For 2025 (index 0), PersonA turns RMD age on July 1 so prorated RMD should be fullRMD * frac
	row := proj[0]
	fullRMD := CalculateRMD(personA.TSPBalanceTraditional, personA.BirthDate.Year(), dateutil.GetRMDAge(personA.BirthDate.Year()))
	if fullRMD.LessThanOrEqual(decimal.Zero) {
		t.Fatalf("expected non-zero full RMD for balance, got %s", fullRMD.String())
	}
	// Compute expected prorated RMD fraction
	yearEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	birthdayThisYear := time.Date(2025, personA.BirthDate.Month(), personA.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
	daysAfter := yearEnd.Sub(birthdayThisYear).Hours() / 24.0
	daysInYear := float64(dateutil.DaysInYear(2025))
	frac := daysAfter / daysInYear
	expectedProratedRMD := fullRMD.Mul(decimal.NewFromFloat(frac))

	// TSP 4% rule withdrawal for 500k is 20,000. If prorated RMD > 20,000 then withdrawal should equal prorated RMD; otherwise 4% rule applies.
	fourPercent := personA.TSPBalanceTraditional.Mul(decimal.NewFromFloat(0.04))
	expectedWithdrawal := fourPercent
	if expectedProratedRMD.GreaterThan(fourPercent) {
		expectedWithdrawal = expectedProratedRMD
	}

	// Verify the projection row's RMDAmount matches the expected prorated RMD
	if row.RMDAmount.Sub(expectedProratedRMD).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		t.Fatalf("RMDAmount mismatch; expected %s, got %s", expectedProratedRMD.StringFixed(2), row.RMDAmount.StringFixed(2))
	}

	diff := row.TSPWithdrawalPersonA.Sub(expectedWithdrawal).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		t.Fatalf("TSP withdrawal mismatch; expected %s, got %s (diff %s)", expectedWithdrawal.StringFixed(2), row.TSPWithdrawalPersonA.StringFixed(2), diff.StringFixed(2))
	}
}

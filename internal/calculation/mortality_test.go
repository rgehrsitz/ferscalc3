package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TestMortalityBasic verifies incomes cease and survivor benefits adjust after death.
func TestMortalityBasic(t *testing.T) {
	// Setup employees (neutral identifiers)
	personA := domain.Employee{BirthDate: time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC), HireDate: time.Date(1987, 6, 22, 0, 0, 0, 0, time.UTC), CurrentSalary: decimal.NewFromInt(100000), High3Salary: decimal.NewFromInt(100000), TSPBalanceTraditional: decimal.NewFromInt(500000), TSPBalanceRoth: decimal.Zero, TSPContributionPercent: decimal.NewFromFloat(0.1), SSBenefit62: decimal.NewFromInt(2000), SSBenefitFRA: decimal.NewFromInt(3000), SSBenefit70: decimal.NewFromInt(4000)}
	personB := domain.Employee{BirthDate: time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC), HireDate: time.Date(1995, 7, 11, 0, 0, 0, 0, time.UTC), CurrentSalary: decimal.NewFromInt(90000), High3Salary: decimal.NewFromInt(90000), TSPBalanceTraditional: decimal.NewFromInt(400000), TSPBalanceRoth: decimal.Zero, TSPContributionPercent: decimal.NewFromFloat(0.1), SSBenefit62: decimal.NewFromInt(1800), SSBenefitFRA: decimal.NewFromInt(2800), SSBenefit70: decimal.NewFromInt(3600)}

	deathDate := time.Date(2030, 6, 30, 0, 0, 0, 0, time.UTC)
	scenario := domain.Scenario{
		Name:      "Mortality Test",
		PersonA:   domain.RetirementScenario{EmployeeName: "person_a", RetirementDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"},
		PersonB:   domain.RetirementScenario{EmployeeName: "person_b", RetirementDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"},
		Mortality: &domain.ScenarioMortality{PersonA: &domain.MortalitySpec{DeathDate: &deathDate}, Assumptions: &domain.MortalityAssumptions{SurvivorSpendingFactor: decimal.NewFromFloat(0.9), TSPSpousalTransfer: "merge", FilingStatusSwitch: "next_year"}},
	}
	assumptions := domain.GlobalAssumptions{ProjectionYears: 15, InflationRate: decimal.NewFromFloat(0.02), FEHBPremiumInflation: decimal.NewFromFloat(0.04), TSPReturnPreRetirement: decimal.NewFromFloat(0.05), TSPReturnPostRetirement: decimal.NewFromFloat(0.04), COLAGeneralRate: decimal.NewFromFloat(0.02)}

	engine := NewCalculationEngine()
	projection := engine.GenerateAnnualProjection(&personA, &personB, &scenario, &assumptions, domain.FederalRules{})

	if len(projection) < 6 {
		t.Fatalf("expected >=6 years, got %d", len(projection))
	}

	// Year index for 2030 relative to base year 2025
	deathYearIdx := deathDate.Year() - ProjectionBaseYear
	if deathYearIdx < 0 || deathYearIdx >= len(projection) {
		t.Fatalf("death year index out of range: %d", deathYearIdx)
	}

	before := projection[deathYearIdx-1]
	after := projection[deathYearIdx]
	next := projection[deathYearIdx+1]

	if after.PersonADeceased != true {
		t.Fatalf("expected PersonA deceased in death year")
	}
	if before.PersonADeceased {
		t.Fatalf("PersonA should be alive year before death")
	}

	// PersonA pension and SS should be zero after death year start
	if after.PensionPersonA.GreaterThan(decimal.Zero) {
		t.Fatalf("pension should cease after death; got %s", after.PensionPersonA)
	}
	if after.SSBenefitPersonA.GreaterThan(decimal.Zero) {
		t.Fatalf("SS should cease after death; got %s", after.SSBenefitPersonA)
	}

	// Survivor (PersonB) should have SS at least her own prior-year or PersonA whichever higher (simplified rule)
	if after.SSBenefitPersonB.LessThan(before.SSBenefitPersonB) {
		t.Fatalf("survivor SS should not decrease; before %s after %s", before.SSBenefitPersonB, after.SSBenefitPersonB)
	}

	// Filing status should switch the year AFTER death per next_year policy
	if after.FilingStatusSingle {
		t.Fatalf("filing status should remain MFJ in death year for next_year policy")
	}
	if !next.FilingStatusSingle {
		t.Fatalf("filing status should be single the year after death")
	}
}

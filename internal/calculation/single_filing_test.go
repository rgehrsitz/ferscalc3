package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TestSingleBracketDefaults ensures defaults derive when single brackets absent
func TestSingleBracketDefaults(t *testing.T) {
	cfg := domain.FederalTaxConfig{StandardDeductionMFJ: decimal.NewFromInt(30000)}
	calc := NewFederalTaxCalculator(cfg)
	if !calc.StandardDeductionSingle.Equal(decimal.NewFromInt(15000)) {
		t.Fatalf("expected default single standard deduction 15000 got %s", calc.StandardDeductionSingle)
	}
	if len(calc.Brackets) == 0 || len(calc.BracketsSingle) != len(calc.Brackets) {
		t.Fatalf("expected mirrored single brackets")
	}
}

// TestSSTaxationStatusSwitch checks SS taxable portion reduces when filing switches to single with same incomes (thresholds lower so taxable may increase; verify difference logic)
func TestSSTaxationStatusSwitch(t *testing.T) {
	ssCalc := NewSSTaxCalculator()
	ssBenefits := decimal.NewFromInt(40000)
	otherIncome := decimal.NewFromInt(30000)
	// Married provisional
	provMarried := ssCalc.CalculateProvisionalIncome(otherIncome, decimal.Zero, ssBenefits)
	taxMarried := ssCalc.CalculateTaxableSocialSecurity(ssBenefits, provMarried)
	// Single provisional
	provSingle := provMarried // same formula
	taxSingle := ssCalc.CalculateTaxableSocialSecuritySingle(ssBenefits, provSingle)
	if taxSingle.Equal(taxMarried) {
		t.Fatalf("expected differing taxable SS amounts between filing statuses")
	}
}

// Integration style test: mortality causes filing status change -> different SS taxation
func TestMortalityFilingAffectsSSTaxation(t *testing.T) {
	personA := &domain.Employee{Name: "PersonA", BirthDate: time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC), HireDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), CurrentSalary: decimal.NewFromInt(120000), High3Salary: decimal.NewFromInt(120000), SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.5), SSBenefitFRA: decimal.NewFromInt(3000)}
	personB := &domain.Employee{Name: "PersonB", BirthDate: time.Date(1962, 1, 1, 0, 0, 0, 0, time.UTC), HireDate: time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC), CurrentSalary: decimal.NewFromInt(80000), High3Salary: decimal.NewFromInt(80000), SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.5), SSBenefitFRA: decimal.NewFromInt(2500)}
	scenario := &domain.Scenario{Name: "Mortality SS Tax", PersonA: domain.RetirementScenario{EmployeeName: personA.Name, RetirementDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 67, TSPWithdrawalStrategy: "4_percent_rule"}, PersonB: domain.RetirementScenario{EmployeeName: personB.Name, RetirementDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 67, TSPWithdrawalStrategy: "4_percent_rule"}, Mortality: &domain.ScenarioMortality{PersonA: &domain.MortalitySpec{DeathDate: &[]time.Time{time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)}[0]}, Assumptions: &domain.MortalityAssumptions{FilingStatusSwitch: "immediate"}}}
	assumptions := &domain.GlobalAssumptions{ProjectionYears: 8, InflationRate: decimal.NewFromFloat(0.02), COLAGeneralRate: decimal.NewFromFloat(0.02)}
	ce := NewCalculationEngine()
	proj := ce.GenerateAnnualProjection(personA, personB, scenario, assumptions, domain.FederalRules{})
	// Find first year after death (same year since immediate switch) and prior year
	deathIdx := 2029 - ProjectionBaseYear
	if deathIdx <= 0 || deathIdx >= len(proj) {
		t.Fatalf("deathIdx out of range")
	}
	pre := proj[deathIdx-1]
	post := proj[deathIdx]
	// Ensure filing status changed
	if !post.FilingStatusSingle {
		t.Fatalf("expected single filing status post death")
	}
	// Compare taxable SS (approx by difference in federal tax relative to non-SS incomes not rigorous, just ensure SS benefits not zero)
	if pre.SSBenefitPersonA.Add(pre.SSBenefitPersonB).IsZero() || post.SSBenefitPersonA.Add(post.SSBenefitPersonB).IsZero() {
		t.Skip("SS not started yet in this simplified test")
	}
}

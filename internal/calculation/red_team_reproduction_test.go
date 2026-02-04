package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TestIRMAACumulativeSurcharge_Reproduction demonstrates the cumulative IRMAA bug.
// The current implementation sums up surcharges for all exceeded thresholds,
// whereas IRMAA should typically select the specific tier's surcharge.
func TestIRMAACumulativeSurcharge_Reproduction(t *testing.T) {
	// Setup: Create a MedicareCalculator with 2 thresholds
	// Tier 1: > 10000 -> Surcharge 50
	// Tier 2: > 20000 -> Surcharge 100
	// If MAGI is 25000, we expect surcharge to be 100 (Tier 2).
	// If bug exists, it will be 50 + 100 = 150.
	// NOTE: This assumes default behavior where MonthlySurcharge is the TOTAL for that tier.
	// If MonthlySurcharge was intended to be incremental, the code is right but the data model is confusing.
	// However, assuming standard IRMAA presentation where valid values are discrete amounts.

	mc := &MedicareCalculator{
		BasePremium2025: decimal.NewFromInt(174), // 2024/2025 approx base
		IRMAAThresholds: []IRMAAThreshold{
			{
				IncomeThresholdSingle: decimal.NewFromInt(10000),
				IncomeThresholdJoint:  decimal.NewFromInt(20000),
				MonthlySurcharge:      decimal.NewFromInt(50),
			},
			{
				IncomeThresholdSingle: decimal.NewFromInt(20000),
				IncomeThresholdJoint:  decimal.NewFromInt(40000),
				MonthlySurcharge:      decimal.NewFromInt(100),
			},
		},
	}

	magi := decimal.NewFromInt(25000)
	isMarriedFilingJointly := false // Use Single thresholds

	// Exceeds 10000 (Surcharge 50) AND
	// Exceeds 20000 (Surcharge 100)

	surcharge := mc.irmaaSurchargeForMAGI(magi, isMarriedFilingJointly)
	expected := decimal.NewFromInt(100) // The tier 2 surcharge

	if !surcharge.Equal(expected) {
		t.Fatalf("IRMAA surcharge is %s, expected %s (Non-cumulative)", surcharge.String(), expected.String())
	}
}

// TestFixedAnnuityPrincipal_Reproduction demonstrates that Fixed Annuity does not remove principal.
// We expect that if 100% of TSP is annuitized, the remaining TSP balance should be zero (or near zero).
func TestFixedAnnuityPrincipal_Reproduction(t *testing.T) {
	// Setup basic objects
	initialBalance := decimal.NewFromInt(100000)
	employee := &domain.Employee{
		BirthDate:             time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
		TSPBalanceTraditional: initialBalance,
		TSPBalanceRoth:        decimal.Zero,
	}

	// Create scenario with Fixed Annuity Strategy
	premiumPercent := decimal.NewFromInt(1) // 100% premium
	scenario := &domain.RetirementScenario{
		RetirementDate:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		TSPWithdrawalStrategy: "fixed_annuity",
		AnnuityPremiumPercent: &premiumPercent,
	}

	ce := &CalculationEngine{}
	strategy := ce.createTSPStrategy(scenario, employee.TSPBalanceTraditional, decimal.NewFromFloat(0.02))

	// Verify strategy created
	if _, ok := strategy.(*FixedAnnuity); !ok {
		t.Fatalf("Failed to create fixed annuity strategy")
	}

	state := newPersonProjectionState(employee, scenario, strategy, nil, nil, "Test", ssProrationFractional, ProjectionBaseYear)
	shouldBeBalance := initialBalance.Sub(initialBalance.Mul(premiumPercent)) // Should be 0
	currentRecordedBalance := state.traditional

	if !shouldBeBalance.Equal(currentRecordedBalance) {
		t.Fatalf("Balance is %s, expected %s (Principal should be removed)", currentRecordedBalance.String(), shouldBeBalance.String())
	}
}

func TestFixedAnnuityDoesNotReduceRemainingBalance(t *testing.T) {
	initialBalance := decimal.NewFromInt(100000)
	employee := &domain.Employee{
		BirthDate:             time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
		TSPBalanceTraditional: initialBalance,
		TSPBalanceRoth:        decimal.Zero,
	}

	premiumPercent := decimal.NewFromFloat(0.5)
	scenario := &domain.RetirementScenario{
		RetirementDate:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		TSPWithdrawalStrategy: "fixed_annuity",
		AnnuityPremiumPercent: &premiumPercent,
	}

	ce := &CalculationEngine{}
	strategy := ce.createTSPStrategy(scenario, employee.TSPBalanceTraditional, decimal.NewFromFloat(0.02))
	state := newPersonProjectionState(employee, scenario, strategy, nil, nil, "Test", ssProrationFractional, ProjectionBaseYear)

	projectionDate := time.Date(ProjectionBaseYear, 1, 1, 0, 0, 0, 0, time.UTC)
	result := personAnnualResult{
		isRetired:    true,
		workFraction: decimal.Zero,
		ageStart:     65,
		rmd:          decimal.Zero,
	}

	result.tspWithdrawal = ce.calculateTSPWithdrawalForPerson(&state, state.retirementYear, projectionDate, decimal.Zero, &result)
	assumptions := &domain.GlobalAssumptions{
		TSPReturnPostRetirement: decimal.Zero,
	}

	ce.updateTSPBalancesForPerson(&state, projectionDate, assumptions, &result)

	expectedRemaining := initialBalance.Mul(decimal.NewFromFloat(0.5))
	if !state.traditional.Equal(expectedRemaining) {
		t.Fatalf("remaining balance = %s, expected %s", state.traditional.StringFixed(2), expectedRemaining.StringFixed(2))
	}
}

// TestTaxBracketGap_Reproduction demonstrates the missing dollar in tax brackets.
func TestTaxBracketGap_Reproduction(t *testing.T) {
	// Brackets: 0-100 (10%), 100-200 (20%)
	brackets := []TaxBracket{
		{decimal.Zero, decimal.NewFromInt(100), decimal.NewFromFloat(0.10)},
		{decimal.NewFromInt(100), decimal.NewFromInt(200), decimal.NewFromFloat(0.20)},
	}
	ftc := &FederalTaxCalculator{
		Brackets:          brackets,
		StandardDeduction: decimal.Zero,
	}

	// Test Income: 101.
	// Expected Tax:
	// Bracket 1: 100 * 0.10 = 10.
	// Bracket 2: (101 - 100) * 0.20 = 0.20
	// Code:
	// Loop 1: Min 0. Income > 0. Tax += (Min(101, 100) - 0) * 0.10 = 100 * 0.10 = 10.
	// Loop 2: Min 101. Income (101) <= Min (101). BREAK.
	// Result: Tax = 10.
	// Missing tax on the dollar at 101.

	income := decimal.NewFromInt(101)
	tax := ftc.CalculateFederalTax(income, 0, 0)

	// Expected: 100 * 0.10 + (101 is in 20% bracket?)
	// Usually tax brackets are:
	// 0 up to 100.
	// Over 100 up to 200.
	// If I have 101, I have 1 dollar in the 20% bracket (or 101-100 = 1).
	// So expected tax = 10 + 1 * 0.20 = 10.20.

	expected := decimal.NewFromFloat(10.20)

	if !tax.Equal(expected) {
		t.Fatalf("Tax is %s, expected %s (Gap at boundary)", tax.String(), expected.String())
	}
}

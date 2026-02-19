package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestValidateFERSEligibility_SickLeaveRefactor explicitly tests that Sick Leave is NOT counted for eligibility.
// Fix: ValidateFERSEligibility should use CreditableService, not YearsOfService.
func TestValidateFERSEligibility_SickLeaveRefactor(t *testing.T) {
	// Employee with 4 years actual service, but 1 year (2080 hours) sick leave.
	// Total YearsOfService = 5.
	// CreditableService = 4.
	// Minimum service for MRA+10 or Age 62+5 is 5 years.
	// This employee should be INELIGIBLE because sick leave excludes from eligibility.

	birthDate := time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC) // 4.5 years to 2024-01-01
	retirementDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	employee := &domain.Employee{
		Name:           "Sick Leave Tester",
		BirthDate:      birthDate,
		HireDate:       hireDate,
		EmploymentType: domain.EmploymentTypeFederal,
		SickLeaveHours: decimal.NewFromInt(2080), // ~0.71 years
	}

	// Verify CreditableService is 4.58 (approx)
	creditable := employee.CreditableService(retirementDate)
	assert.True(t, creditable.LessThan(decimal.NewFromInt(5)), "Creditable service should be < 5 years (actual: %s)", creditable)

	// Verify YearsOfService is 5
	totalService := employee.YearsOfService(retirementDate)
	assert.True(t, totalService.GreaterThanOrEqual(decimal.NewFromInt(5)), "Total service should be 5 years")

	// Verify Eligibility (Should be False)
	isEligible, reason := ValidateFERSEligibility(employee, retirementDate)
	assert.False(t, isEligible, "Employee with < 5 years creditable service should not be eligible, even if sick leave pushes total > 5")
	assert.Contains(t, reason, "less than 5 years", "Reason should mention service requirement")
}

// TestSRSEarningsTest verifies that the FERS Special Retirement Supplement is reduced by earnings.
func TestSRSEarningsTest(t *testing.T) {
	// Setup: Employee eligible for SRS
	// SRS amount (base) = $1500/month * 12 = $18,000
	// Earnings Limit = $23,400

	employee := &domain.Employee{
		EmploymentType: domain.EmploymentTypeFederal,
		BirthDate:      time.Date(1968, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), // 30+ service
		SSBenefit62:    decimal.NewFromInt(2000),                    // Monthly
	}

	retirementDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	inflation := decimal.Zero
	srsLimit := decimal.NewFromInt(23400)

	tests := []struct {
		name         string
		earnedIncome decimal.Decimal
		expectedSRS  bool // approximate check or non-zero
		shouldReduce bool
	}{
		{
			name:         "Earnings below limit",
			earnedIncome: decimal.NewFromInt(10000),
			shouldReduce: false,
		},
		{
			name:         "Earnings equal limit",
			earnedIncome: decimal.NewFromInt(23400),
			shouldReduce: false,
		},
		{
			name:         "Earnings above limit",
			earnedIncome: decimal.NewFromInt(33400), // $10,000 over
			shouldReduce: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srs := CalculateFERSSupplementYear(employee, retirementDate, 1, inflation, tt.earnedIncome, srsLimit)

			// Base SRS (approx): 2000 * 12 * (36/40) = 24000 * 0.9 = 21600
			baseSRS := CalculateFERSSupplementYear(employee, retirementDate, 1, inflation, decimal.Zero, srsLimit)

			if tt.shouldReduce {
				assert.True(t, srs.LessThan(baseSRS), "SRS should be reduced by excess earnings")
				// Check exact reduction amount: ($33400 - $23400) / 2 = $5000 reduction
				// Expected: 21600 - 5000 = 16600
				excess := tt.earnedIncome.Sub(srsLimit)
				reduction := excess.Div(decimal.NewFromInt(2))
				expected := baseSRS.Sub(reduction)

				assert.True(t, srs.Equal(expected), "SRS reduction calculation incorrect. Got %s, Expected %s", srs, expected)
			} else {
				assert.True(t, srs.Equal(baseSRS), "SRS should not be reduced")
			}
		})
	}
}

// TestNewJerseyPensionExclusion verifies the 3-tier exclusion logic.
func TestNewJerseyPensionExclusion(t *testing.T) {
	calc := NewNewJerseyTaxCalculator()

	tests := []struct {
		name          string
		wageIncome    decimal.Decimal
		pensionIncome decimal.Decimal
		tspIncome     decimal.Decimal
		isRetired     bool
		expectedTax   decimal.Decimal // We verify logic by checking taxable basis inferring from tax or stepping through?
		// Logic:
		// Tier 1 (<=100k): $100k exclusion
		// Tier 2 (<=125k): $50k exclusion
		// Tier 3 (<=150k): $25k exclusion
		// Tier 4 (>150k): $0 exclusion
	}{
		{
			name:          "Tier 1: Income $90k (All Pension) -> Fully Excluded",
			wageIncome:    decimal.Zero,
			pensionIncome: decimal.NewFromInt(90000),
			tspIncome:     decimal.Zero,
			isRetired:     true,
			// Taxable = 90k - 90k (min(100k, 90k)) = 0
			// Tax = 0
		},
		{
			name:          "Tier 2: Income $110k (All Pension) -> 50k Excluded",
			wageIncome:    decimal.Zero,
			pensionIncome: decimal.NewFromInt(110000),
			tspIncome:     decimal.Zero,
			isRetired:     true,
			// Taxable = 110k - 50k = 60k
			// Tax calc:
			// 20k @ 1.4% = 280
			// 30k @ 1.75% = 525 (20k-50k)
			// 10k @ 2.45% = 245 (50k-70k brackets changed in my impl?? Let's check impl)
			// My Impl: 20k, 50k(1.75), 70k(2.45), 80k(3.5), 150k(5.525), 500k(6.37)
			// 0-20k (1.4%) = 280
			// 20k-50k (1.75%) = 525
			// 50k-60k (2.45%) = 245
			// Total = 1050
		},
		{
			name:          "Tier 4: Income $160k (All Pension) -> 0 Excluded",
			wageIncome:    decimal.Zero,
			pensionIncome: decimal.NewFromInt(160000),
			tspIncome:     decimal.Zero,
			isRetired:     true,
			// Taxable = 160k
			// Should be much higher tax
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			income := domain.TaxableIncome{
				WageIncome:         tt.wageIncome,
				FERSPension:        tt.pensionIncome,
				TSPWithdrawalsTrad: tt.tspIncome,
			}

			tax := calc.CalculateTax(income, StateTaxContext{
				IsRetired:                   tt.isRetired,
				FilingStatus:                "mfj",
				EligibleRetirementExclusion: tt.isRetired,
			})

			if tt.name == "Tier 1: Income $90k (All Pension) -> Fully Excluded" {
				assert.True(t, tax.IsZero(), "Tax should be zero for fully excluded pension")
			}

			if tt.name == "Tier 2: Income $110k (All Pension) -> 50k Excluded" {
				// We expect non-zero tax, specifically around 1050
				// But we mainly want to verify it's not taxing the full 110k
				// Full 110k tax would be much higher (~4000+)
				assert.True(t, tax.LessThan(decimal.NewFromInt(2000)), "Tax should reflect significant exclusion")
				assert.True(t, tax.GreaterThan(decimal.NewFromInt(500)), "Tax should not be zero")
			}

			if tt.name == "Tier 4: Income $160k (All Pension) -> 0 Excluded" {
				// Tax on 160k vs 110k should be huge jump due to loss of exclusion
				assert.True(t, tax.GreaterThan(decimal.NewFromInt(5000)), "Tax should be substantial (no exclusion)")
			}
		})
	}
}

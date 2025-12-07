package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCalculateFERSPension(t *testing.T) {
	// Test the multiplier determination function directly
	tests := []struct {
		name               string
		retirementAge      int
		serviceYears       decimal.Decimal
		expectedMultiplier decimal.Decimal
	}{
		{
			name:               "Standard multiplier at 60",
			retirementAge:      60,
			serviceYears:       decimal.NewFromFloat(30.0),
			expectedMultiplier: decimal.NewFromFloat(0.01),
		},
		{
			name:               "Enhanced multiplier at 62 with 20+ years",
			retirementAge:      62,
			serviceYears:       decimal.NewFromFloat(30.0),
			expectedMultiplier: decimal.NewFromFloat(0.011),
		},
		{
			name:               "Standard multiplier at 62 with less than 20 years",
			retirementAge:      62,
			serviceYears:       decimal.NewFromFloat(15.0),
			expectedMultiplier: decimal.NewFromFloat(0.01),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiplier := determineMultiplier(tt.retirementAge, tt.serviceYears)
			assert.True(t, multiplier.Equal(tt.expectedMultiplier),
				"Expected %s, got %s", tt.expectedMultiplier, multiplier)
		})
	}
}

func TestCalculateFERSPensionNonFederal(t *testing.T) {
	employee := &domain.Employee{
		EmploymentType: domain.EmploymentTypeNonFederal,
		BirthDate:      time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		High3Salary:    decimal.NewFromInt(100000),
	}
	calc := CalculateFERSPension(employee, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.True(t, calc.ReducedPension.IsZero(), "non-federal employees should not generate FERS pensions")
}

func TestApplyFERSPensionCOLA(t *testing.T) {
	tests := []struct {
		name            string
		currentPension  decimal.Decimal
		inflationRate   decimal.Decimal
		annuitantAge    int
		expectedPension decimal.Decimal
	}{
		{
			name:            "No COLA before age 62",
			currentPension:  decimal.NewFromInt(30000),
			inflationRate:   decimal.NewFromFloat(0.03),
			annuitantAge:    60,
			expectedPension: decimal.NewFromInt(30000),
		},
		{
			name:            "Full COLA at age 62 with 2% inflation",
			currentPension:  decimal.NewFromInt(30000),
			inflationRate:   decimal.NewFromFloat(0.02),
			annuitantAge:    62,
			expectedPension: decimal.NewFromInt(30600), // 30000 * 1.02
		},
		{
			name:            "Capped COLA at age 62 with 2.5% inflation",
			currentPension:  decimal.NewFromInt(30000),
			inflationRate:   decimal.NewFromFloat(0.025),
			annuitantAge:    62,
			expectedPension: decimal.NewFromInt(30600), // 30000 * 1.02 (capped at 2%)
		},
		{
			name:            "Reduced COLA at age 62 with 4% inflation",
			currentPension:  decimal.NewFromInt(30000),
			inflationRate:   decimal.NewFromFloat(0.04),
			annuitantAge:    62,
			expectedPension: decimal.NewFromInt(30900), // 30000 * 1.03 (4% - 1%)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyFERSPensionCOLA(tt.currentPension, tt.inflationRate, tt.annuitantAge)
			assert.True(t, result.Equal(tt.expectedPension),
				"Expected %s, got %s", tt.expectedPension, result)
		})
	}
}

func TestCalculateFERSSpecialRetirementSupplement(t *testing.T) {
	tests := []struct {
		name          string
		ssBenefitAt62 decimal.Decimal
		serviceYears  decimal.Decimal
		currentAge    int
		expectedSRS   decimal.Decimal
	}{
		{
			name:          "SRS at age 60",
			ssBenefitAt62: decimal.NewFromInt(2000), // Monthly
			serviceYears:  decimal.NewFromFloat(30.0),
			currentAge:    60,
			expectedSRS:   decimal.NewFromInt(18000), // 2000 * 12 * (30/40)
		},
		{
			name:          "No SRS at age 62",
			ssBenefitAt62: decimal.NewFromInt(2000),
			serviceYears:  decimal.NewFromFloat(30.0),
			currentAge:    62,
			expectedSRS:   decimal.Zero,
		},
		{
			name:          "SRS with 20 years service",
			ssBenefitAt62: decimal.NewFromInt(2000),
			serviceYears:  decimal.NewFromFloat(20.0),
			currentAge:    60,
			expectedSRS:   decimal.NewFromInt(12000), // 2000 * 12 * (20/40)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateFERSSpecialRetirementSupplement(tt.ssBenefitAt62, tt.serviceYears, tt.currentAge)
			assert.True(t, result.Equal(tt.expectedSRS),
				"Expected %s, got %s", tt.expectedSRS, result)
		})
	}
}

func TestCalculateFERSSupplementYearNonFederal(t *testing.T) {
	employee := &domain.Employee{
		EmploymentType: domain.EmploymentTypeNonFederal,
		BirthDate:      time.Date(1965, 6, 15, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 3, 20, 0, 0, 0, 0, time.UTC),
		SSBenefit62:    decimal.NewFromInt(2000),
	}
	amount := CalculateFERSSupplementYear(employee, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), 0, decimal.NewFromFloat(0.02))
	assert.True(t, amount.IsZero(), "non-federal employees should not receive FERS supplement")
}

func TestValidateFERSEligibility(t *testing.T) {
	tests := []struct {
		name           string
		birthDate      time.Time
		hireDate       time.Time
		retirementDate time.Time
		expectedValid  bool
		expectedReason string
	}{
		{
			name:           "Eligible at age 62 with 5+ years",
			birthDate:      time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expectedValid:  true,
			expectedReason: "Eligible for immediate annuity at age 62+",
		},
		{
			name:           "Not eligible - too young",
			birthDate:      time.Date(1970, 6, 15, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expectedValid:  false,
			expectedReason: "Employee has not reached Minimum Retirement Age",
		},
		{
			name:           "Not eligible - insufficient service",
			birthDate:      time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(2023, 3, 20, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expectedValid:  false,
			expectedReason: "Employee has less than 5 years of service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employee := &domain.Employee{
				BirthDate: tt.birthDate,
				HireDate:  tt.hireDate,
			}

			valid, reason := ValidateFERSEligibility(employee, tt.retirementDate)
			assert.Equal(t, tt.expectedValid, valid)
			assert.Contains(t, reason, tt.expectedReason)
		})
	}
}

func TestCalculatePensionReduction(t *testing.T) {
	tests := []struct {
		name              string
		birthDate         time.Time
		hireDate          time.Time
		retirementDate    time.Time
		expectedReduction decimal.Decimal
	}{
		{
			name:              "No reduction at age 62+ with 5+ years",
			birthDate:         time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			hireDate:          time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
			retirementDate:    time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			expectedReduction: decimal.Zero,
		},
		{
			name:              "No reduction at MRA+ with 20+ years",
			birthDate:         time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
			hireDate:          time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
			retirementDate:    time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC),
			expectedReduction: decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employee := &domain.Employee{
				BirthDate: tt.birthDate,
				HireDate:  tt.hireDate,
			}

			reduction := CalculatePensionReduction(employee, tt.retirementDate)
			assert.True(t, reduction.Equal(tt.expectedReduction),
				"Expected %s, got %s", tt.expectedReduction, reduction)
		})
	}
}

func TestProjectFERSPension(t *testing.T) {
	employee := &domain.Employee{
		High3Salary: decimal.NewFromInt(95000),
		BirthDate:   time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC),
		HireDate:    time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC),
	}

	retirementDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	inflationRate := decimal.NewFromFloat(0.025)
	projectionYears := 5

	projections := ProjectFERSPension(employee, retirementDate, projectionYears, inflationRate)

	assert.Len(t, projections, projectionYears)

	// First year should be the base pension
	firstYearPension := CalculateFERSPension(employee, retirementDate)
	assert.True(t, projections[0].Equal(firstYearPension.ReducedPension),
		"Expected %s, got %s", firstYearPension.ReducedPension, projections[0])

	// Subsequent years should be higher due to COLA
	for i := 1; i < len(projections); i++ {
		assert.True(t, projections[i].GreaterThan(projections[i-1]),
			"Year %d pension should be greater than year %d", i+1, i)
	}
}

// TestFERSPensionOfficialExamples tests FERS pension calculations using official OPM examples
func TestFERSPensionOfficialExamples(t *testing.T) {
	tests := []struct {
		name           string
		birthDate      time.Time
		hireDate       time.Time
		retirementDate time.Time
		high3Salary    decimal.Decimal
		expectedAnnual decimal.Decimal
		description    string
	}{
		{
			name:           "OPM Example: Age 62, 25 years service, $80k salary",
			birthDate:      time.Date(1963, 1, 1, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			high3Salary:    decimal.NewFromInt(80000),
			expectedAnnual: decimal.NewFromInt(22000), // 25 * 80000 * 0.011 = 22000
			description:    "Enhanced multiplier (1.1%) at age 62 with 20+ years",
		},
		{
			name:           "OPM Example: Age 60, 30 years service, $95k salary",
			birthDate:      time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			high3Salary:    decimal.NewFromInt(95000),
			expectedAnnual: decimal.NewFromInt(28500), // 30 * 95000 * 0.01 = 28500
			description:    "Standard multiplier (1.0%) at age 60 (before 62)",
		},
		{
			name:           "Person A Actual Scenario: Age 60, 38.44 years, $190k salary",
			birthDate:      time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(1987, 6, 22, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			high3Salary:    decimal.NewFromInt(190000),
			expectedAnnual: decimal.NewFromFloat(73045.31), // 38.44 * 190000 * 0.01 ≈ 73045
			description:    "Real scenario: Standard multiplier due to age < 62",
		},
		{
			name:           "Person B Actual Scenario: Age 62, ~30.1 years, $164k salary",
			birthDate:      time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(1995, 7, 11, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC),
			high3Salary:    decimal.NewFromInt(164000),
			expectedAnnual: decimal.NewFromFloat(54350), // ~30.1 * 164000 * 0.011 ≈ 54350
			description:    "Real scenario: Enhanced multiplier at age 62 with 20+ years",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employee := &domain.Employee{
				BirthDate:                      tt.birthDate,
				HireDate:                       tt.hireDate,
				High3Salary:                    tt.high3Salary,
				SurvivorBenefitElectionPercent: decimal.Zero,
			}

			result := CalculateFERSPension(employee, tt.retirementDate)

			// Allow for small rounding differences (within $50)
			difference := result.ReducedPension.Sub(tt.expectedAnnual).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(50)),
				"%s: Expected %s, got %s (difference: %s)",
				tt.description, tt.expectedAnnual.StringFixed(2),
				result.ReducedPension.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestFERSEligibilityScenarios tests various FERS eligibility scenarios
func TestFERSEligibilityScenarios(t *testing.T) {
	tests := []struct {
		name           string
		birthDate      time.Time
		hireDate       time.Time
		retirementDate time.Time
		expectedValid  bool
		expectedReason string
	}{
		{
			name:           "Immediate: Age 62+ with 5 years",
			birthDate:      time.Date(1963, 1, 1, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expectedValid:  true,
			expectedReason: "Eligible for immediate annuity at age 62+",
		},
		{
			name:           "Immediate: Age 60 with 20 years",
			birthDate:      time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
			hireDate:       time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedValid:  true,
			expectedReason: "Eligible for immediate annuity at MRA with 10+ years",
		},
		{
			name:           "Immediate: MRA with 30 years",
			birthDate:      time.Date(1967, 1, 1, 0, 0, 0, 0, time.UTC), // MRA = 56.5, rounds to 56
			hireDate:       time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // Age 58 - above MRA
			expectedValid:  true,
			expectedReason: "Eligible for immediate annuity at MRA with 10+ years",
		},
		{
			name:           "Not eligible: Under MRA",
			birthDate:      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), // MRA = 57
			hireDate:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			retirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // Age 55
			expectedValid:  false,
			expectedReason: "Employee has not reached Minimum Retirement Age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employee := &domain.Employee{
				BirthDate: tt.birthDate,
				HireDate:  tt.hireDate,
			}

			valid, reason := ValidateFERSEligibility(employee, tt.retirementDate)
			assert.Equal(t, tt.expectedValid, valid, "Validity check failed for %s", tt.name)
			assert.Contains(t, reason, tt.expectedReason, "Reason check failed for %s", tt.name)
		})
	}
}

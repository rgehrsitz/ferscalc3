package calculation

import (
	"testing"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestFederalTaxCalculation tests federal income tax calculations using 2025 tax brackets
func TestFederalTaxCalculation(t *testing.T) {
	calculator := NewFederalTaxCalculator2025()

	tests := []struct {
		name        string
		grossIncome decimal.Decimal
		age1        int
		age2        int
		expectedTax decimal.Decimal
		description string
	}{
		{
			name:        "No tax below standard deduction",
			grossIncome: decimal.NewFromInt(25000),
			age1:        45,
			age2:        43,
			expectedTax: decimal.Zero,
			description: "Income below $30,000 standard deduction",
		},
		{
			name:        "Low tax bracket",
			grossIncome: decimal.NewFromInt(50000),
			age1:        45,
			age2:        43,
			expectedTax: decimal.NewFromInt(2000), // (50000-30000) * 0.10
			description: "Income in 10% bracket only",
		},
		{
			name:        "Multiple tax brackets",
			grossIncome: decimal.NewFromInt(100000),
			age1:        45,
			age2:        43,
			expectedTax: decimal.NewFromFloat(7936), // (100000-30000): 23200*0.10 + 46800*0.12 = 7936
			description: "Income spanning multiple tax brackets",
		},
		{
			name:        "High income scenario",
			grossIncome: decimal.NewFromInt(300000),
			age1:        45,
			age2:        43,
			expectedTax: decimal.NewFromFloat(50885), // 270000 taxable across all brackets
			description: "High income in 24% bracket",
		},
		{
			name:        "Senior additional deduction",
			grossIncome: decimal.NewFromInt(80000),
			age1:        66,                         // Over 65
			age2:        64,                         // Under 65
			expectedTax: decimal.NewFromFloat(5350), // (80000-31550): 23200*0.10 + 25250*0.12 = 5350
			description: "Additional standard deduction for senior",
		},
		{
			name:        "Both seniors additional deduction",
			grossIncome: decimal.NewFromInt(80000),
			age1:        66,                         // Over 65
			age2:        67,                         // Over 65
			expectedTax: decimal.NewFromFloat(5164), // (80000-33100): 23200*0.10 + 23700*0.12 = 5164
			description: "Additional standard deduction for both seniors",
		},
		{
			name:        "PersonA and PersonB current income",
			grossIncome: decimal.NewFromFloat(367399), // Their actual combined gross
			age1:        60,
			age2:        62,
			expectedTax: decimal.NewFromFloat(67061), // (367399-30000) across all brackets
			description: "Real scenario: PersonA and PersonB's current income",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tax := calculator.CalculateFederalTax(tt.grossIncome, tt.age1, tt.age2)

			// Allow for rounding differences in federal tax calculations (within $100)
			difference := tax.Sub(tt.expectedTax).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(100)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedTax.StringFixed(2), tax.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestPennsylvaniaTaxCalculation tests PA state tax calculations
func TestPennsylvaniaTaxCalculation(t *testing.T) {
	calculator := NewPennsylvaniaTaxCalculator()

	tests := []struct {
		name        string
		income      domain.TaxableIncome
		isRetired   bool
		expectedTax decimal.Decimal
		description string
	}{
		{
			name: "Working income only",
			income: domain.TaxableIncome{
				WageIncome:         decimal.NewFromInt(100000),
				FERSPension:        decimal.Zero,
				TSPWithdrawalsTrad: decimal.Zero,
				TaxableSSBenefits:  decimal.Zero,
			},
			isRetired:   false,
			expectedTax: decimal.NewFromFloat(3070), // 100000 * 0.0307
			description: "3.07% on wages while working",
		},
		{
			name: "Retirement income - no tax",
			income: domain.TaxableIncome{
				WageIncome:         decimal.Zero,
				FERSPension:        decimal.NewFromInt(50000),
				TSPWithdrawalsTrad: decimal.NewFromInt(30000),
				TaxableSSBenefits:  decimal.NewFromInt(20000),
			},
			isRetired:   true,
			expectedTax: decimal.Zero, // PA doesn't tax retirement income
			description: "No PA tax on retirement income",
		},
		{
			name: "Mixed income in retirement",
			income: domain.TaxableIncome{
				WageIncome:         decimal.NewFromInt(20000), // Part-time work
				FERSPension:        decimal.NewFromInt(40000),
				TSPWithdrawalsTrad: decimal.NewFromInt(25000),
				InterestIncome:     decimal.NewFromInt(5000),
			},
			isRetired:   true,
			expectedTax: decimal.NewFromFloat(767.50), // (20000 + 5000) * 0.0307
			description: "PA tax only on wages and interest in retirement",
		},
		{
			name: "Person A and Person B working scenario",
			income: domain.TaxableIncome{
				WageIncome: decimal.NewFromFloat(367399), // Combined wages
			},
			isRetired:   false,
			expectedTax: decimal.NewFromFloat(11279.15), // 367399 * 0.0307
			description: "Actual PA tax for PersonA and PersonB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tax := calculator.CalculateTax(tt.income, tt.isRetired)

			// Allow for rounding differences (within $1)
			difference := tax.Sub(tt.expectedTax).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(1)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedTax.StringFixed(2), tax.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestUpperMakefieldEIT tests local Earned Income Tax
func TestUpperMakefieldEIT(t *testing.T) {
	calculator := NewUpperMakefieldEITCalculator()

	tests := []struct {
		name        string
		wageIncome  decimal.Decimal
		isRetired   bool
		expectedTax decimal.Decimal
		description string
	}{
		{
			name:        "Working income",
			wageIncome:  decimal.NewFromInt(100000),
			isRetired:   false,
			expectedTax: decimal.NewFromInt(1000), // 100000 * 0.01
			description: "1% EIT on working income",
		},
		{
			name:        "Retirement - no EIT",
			wageIncome:  decimal.Zero,
			isRetired:   true,
			expectedTax: decimal.Zero,
			description: "No EIT in retirement",
		},
		{
			name:        "Part-time work in retirement",
			wageIncome:  decimal.NewFromInt(20000),
			isRetired:   true,
			expectedTax: decimal.Zero, // EIT doesn't apply in retirement
			description: "No EIT even on wages in retirement",
		},
		{
			name:        "Person A and Person B working scenario",
			wageIncome:  decimal.NewFromFloat(367399),
			isRetired:   false,
			expectedTax: decimal.NewFromFloat(3673.99), // 367399 * 0.01
			description: "Actual EIT for Person A and Person B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tax := calculator.CalculateEIT(tt.wageIncome, tt.isRetired)

			// Allow for rounding differences (within $0.01)
			difference := tax.Sub(tt.expectedTax).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromFloat(0.01)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedTax.StringFixed(2), tax.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestFICACalculation tests FICA tax calculations
func TestFICACalculation(t *testing.T) {
	calculator := NewFICACalculator2025()

	tests := []struct {
		name                string
		wages               decimal.Decimal
		totalHouseholdWages decimal.Decimal
		expectedFICA        decimal.Decimal
		description         string
	}{
		{
			name:                "Low income - no additional Medicare",
			wages:               decimal.NewFromInt(50000),
			totalHouseholdWages: decimal.NewFromInt(50000),
			expectedFICA:        decimal.NewFromFloat(3825), // 50000 * (0.062 + 0.0145)
			description:         "Standard FICA on $50k",
		},
		{
			name:                "At SS wage base",
			wages:               decimal.NewFromInt(176100), // 2025 SS wage base
			totalHouseholdWages: decimal.NewFromInt(176100),
			expectedFICA:        decimal.NewFromFloat(13471.65), // 176100 * (0.062 + 0.0145)
			description:         "FICA at Social Security wage base limit",
		},
		{
			name:                "Above SS wage base",
			wages:               decimal.NewFromInt(200000),
			totalHouseholdWages: decimal.NewFromInt(200000),
			expectedFICA:        decimal.NewFromFloat(13818.20), // SS capped at 176100 + Medicare on full 200000
			description:         "FICA with SS cap but full Medicare",
		},
		{
			name:                "High income - additional Medicare",
			wages:               decimal.NewFromInt(300000),
			totalHouseholdWages: decimal.NewFromInt(300000),
			expectedFICA:        decimal.NewFromFloat(15718.20), // SS capped at 176100 + Medicare + Additional Medicare
			description:         "High income with additional 0.9% Medicare tax",
		},
		{
			name:                "Person A and Person B working scenario",
			wages:               decimal.NewFromFloat(190779),   // Person A's salary
			totalHouseholdWages: decimal.NewFromFloat(367399),   // Combined
			expectedFICA:        decimal.NewFromFloat(14233.30), // Person A's portion of household FICA with proportional additional Medicare
			description:         "Person A's FICA from actual scenario",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fica := calculator.CalculateFICA(tt.wages, tt.totalHouseholdWages)

			// Use expectedFICA
			expected := tt.expectedFICA

			// Allow for rounding differences in FICA calculations (within $50)
			difference := fica.Sub(expected).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(50)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				expected.StringFixed(2), fica.StringFixed(2), difference.StringFixed(2))
		})
	}
}

func TestNewFICACalculatorUsesConfigYear(t *testing.T) {
	config := domain.FICATaxConfig{
		Year:                   2030,
		SocialSecurityWageBase: decimal.NewFromInt(200000),
		SocialSecurityRate:     decimal.NewFromFloat(0.062),
		MedicareRate:           decimal.NewFromFloat(0.0145),
		AdditionalMedicareRate: decimal.NewFromFloat(0.009),
		HighIncomeThresholdMFJ: decimal.NewFromInt(400000),
	}

	calculator := NewFICACalculator(config)

	assert.Equal(t, 2030, calculator.Year)
	assert.True(t, calculator.SSWageBase.Equal(config.SocialSecurityWageBase))
	assert.True(t, calculator.HighIncomeThreshold.Equal(config.HighIncomeThresholdMFJ))
}

func TestNewFICACalculatorDefaultsYear(t *testing.T) {
	config := domain.FICATaxConfig{
		SocialSecurityWageBase: decimal.NewFromInt(176100),
		SocialSecurityRate:     decimal.NewFromFloat(0.062),
		MedicareRate:           decimal.NewFromFloat(0.0145),
		AdditionalMedicareRate: decimal.NewFromFloat(0.009),
		HighIncomeThresholdMFJ: decimal.NewFromInt(250000),
	}

	calculator := NewFICACalculator(config)

	assert.Equal(t, ProjectionBaseYear, calculator.Year)
	assert.True(t, calculator.SSWageBase.Equal(config.SocialSecurityWageBase))
	assert.True(t, calculator.HighIncomeThreshold.Equal(config.HighIncomeThresholdMFJ))
}

// TestFICAWithProration tests FICA calculations with partial year work
func TestFICAWithProration(t *testing.T) {
	calculator := NewFICACalculator2025()

	tests := []struct {
		name                string
		annualWages         decimal.Decimal
		totalHouseholdWages decimal.Decimal
		workFraction        decimal.Decimal
		expectedFICA        decimal.Decimal
		description         string
	}{
		{
			name:                "Half year work",
			annualWages:         decimal.NewFromInt(100000),
			totalHouseholdWages: decimal.NewFromInt(150000),
			workFraction:        decimal.NewFromFloat(0.5),
			expectedFICA:        decimal.NewFromFloat(3825), // (100000 * 0.5) * (0.062 + 0.0145)
			description:         "Working half the year",
		},
		{
			name:                "Quarter year work",
			annualWages:         decimal.NewFromInt(200000),
			totalHouseholdWages: decimal.NewFromInt(300000),
			workFraction:        decimal.NewFromFloat(0.25),
			expectedFICA:        decimal.NewFromFloat(3825), // (200000 * 0.25) * (0.062 + 0.0145)
			description:         "Working quarter of the year",
		},
		{
			name:                "PersonA partial retirement year",
			annualWages:         decimal.NewFromFloat(190779),
			totalHouseholdWages: decimal.NewFromFloat(367399),
			workFraction:        decimal.NewFromFloat(0.917), // Working until Dec 1
			expectedFICA:        decimal.NewFromFloat(13787), // Includes additional Medicare tax for high earners
			description:         "PersonA working until December 1st",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fica := calculator.CalculateFICAWithProration(tt.annualWages, tt.totalHouseholdWages, tt.workFraction)

			// Allow for rounding differences in prorated FICA calculations (within $100)
			difference := fica.Sub(tt.expectedFICA).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(100)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedFICA.StringFixed(2), fica.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestSocialSecurityTaxationComprehensive tests comprehensive SS taxation scenarios
func TestSocialSecurityTaxationComprehensive(t *testing.T) {
	calculator := NewSSTaxCalculator()

	tests := []struct {
		name               string
		annualSSBenefit    decimal.Decimal
		otherIncome        decimal.Decimal
		nontaxableInterest decimal.Decimal
		expectedTaxable    decimal.Decimal
		description        string
	}{
		{
			name:               "Low income - no SS taxation",
			annualSSBenefit:    decimal.NewFromInt(18000),
			otherIncome:        decimal.NewFromInt(15000),
			nontaxableInterest: decimal.Zero,
			expectedTaxable:    decimal.Zero, // Provisional income = 24000 < 32000
			description:        "Below first threshold",
		},
		{
			name:               "Middle income - 50% SS taxation",
			annualSSBenefit:    decimal.NewFromInt(24000),
			otherIncome:        decimal.NewFromInt(22000), // Fixed: was 20000, needed 22000 for provisional income of 34000
			nontaxableInterest: decimal.Zero,
			expectedTaxable:    decimal.NewFromInt(1000), // 50% of excess over 32k
			description:        "Between thresholds: 50% taxation",
		},
		{
			name:               "High income - 85% SS taxation",
			annualSSBenefit:    decimal.NewFromInt(36000),
			otherIncome:        decimal.NewFromInt(50000),
			nontaxableInterest: decimal.Zero,
			expectedTaxable:    decimal.NewFromInt(30600), // 85% of benefits
			description:        "Above second threshold: 85% taxation",
		},
		{
			name:               "PersonA and PersonB scenario",
			annualSSBenefit:    decimal.NewFromFloat(44683.25), // Combined SS benefits in scenario
			otherIncome:        decimal.NewFromFloat(150000),   // Combined pensions + TSP
			nontaxableInterest: decimal.Zero,
			expectedTaxable:    decimal.NewFromFloat(37980.76), // 85% of benefits
			description:        "Real retirement scenario SS taxation",
		},
		{
			name:               "With tax-free municipal bonds",
			annualSSBenefit:    decimal.NewFromInt(30000),
			otherIncome:        decimal.NewFromInt(30000),
			nontaxableInterest: decimal.NewFromInt(10000), // Municipal bond interest
			expectedTaxable:    decimal.NewFromInt(25500), // 85% taxation due to higher provisional income
			description:        "Municipal bond interest affects SS taxation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provisionalIncome := calculator.CalculateProvisionalIncome(
				tt.otherIncome, tt.nontaxableInterest, tt.annualSSBenefit)

			taxableAmount := calculator.CalculateTaxableSocialSecurity(
				tt.annualSSBenefit, provisionalIncome)

			// Allow for rounding differences in SS taxation calculations (within $100)
			difference := taxableAmount.Sub(tt.expectedTaxable).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromInt(100)),
				"%s: Expected %s taxable, got %s (difference: %s)", tt.description,
				tt.expectedTaxable.StringFixed(2), taxableAmount.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestComprehensiveTaxCalculation tests the full tax calculation system
func TestComprehensiveTaxCalculation(t *testing.T) {
	calculator := NewComprehensiveTaxCalculator()

	tests := []struct {
		name         string
		income       domain.TaxableIncome
		isRetired    bool
		age1         int
		age2         int
		totalWages   decimal.Decimal
		expectedFed  decimal.Decimal
		expectedSt   decimal.Decimal
		expectedLoc  decimal.Decimal
		expectedFICA decimal.Decimal
		description  string
	}{
		{
			name: "Working couple - current scenario",
			income: domain.TaxableIncome{
				Salary:             decimal.NewFromFloat(367399),
				WageIncome:         decimal.NewFromFloat(367399),
				FERSPension:        decimal.Zero,
				TSPWithdrawalsTrad: decimal.Zero,
				TaxableSSBenefits:  decimal.Zero,
			},
			isRetired:    false,
			age1:         60,
			age2:         62,
			totalWages:   decimal.NewFromFloat(367399),
			expectedFed:  decimal.NewFromFloat(67061), // Federal tax with standard deduction
			expectedSt:   decimal.NewFromFloat(11279), // PA state tax
			expectedLoc:  decimal.NewFromFloat(3674),  // Local EIT
			expectedFICA: decimal.NewFromFloat(17302), // FICA taxes with additional Medicare
			description:  "PersonA and PersonB current working scenario",
		},
		{
			name: "Retirement scenario - no wages",
			income: domain.TaxableIncome{
				Salary:             decimal.Zero,
				WageIncome:         decimal.Zero,
				FERSPension:        decimal.NewFromFloat(75000),
				TSPWithdrawalsTrad: decimal.NewFromFloat(60000),
				TaxableSSBenefits:  decimal.NewFromFloat(30000),
			},
			isRetired:    true,
			age1:         65,
			age2:         67,
			totalWages:   decimal.Zero,
			expectedFed:  decimal.NewFromFloat(19124), // Federal tax on retirement income with standard deduction
			expectedSt:   decimal.Zero,                // PA doesn't tax retirement income
			expectedLoc:  decimal.Zero,                // No local tax in retirement
			expectedFICA: decimal.Zero,                // No FICA in retirement
			description:  "Typical retirement tax scenario",
		},
		{
			name: "Transition year - partial work",
			income: domain.TaxableIncome{
				Salary:             decimal.NewFromFloat(100000), // Partial year work
				WageIncome:         decimal.NewFromFloat(100000),
				FERSPension:        decimal.NewFromFloat(25000), // Started pension
				TSPWithdrawalsTrad: decimal.NewFromFloat(20000), // Started withdrawals
				TaxableSSBenefits:  decimal.Zero,                // SS not started yet
			},
			isRetired:    false, // Transition year
			age1:         60,
			age2:         62,
			totalWages:   decimal.NewFromFloat(100000),
			expectedFed:  decimal.NewFromFloat(15406), // Federal tax on mixed income with standard deduction
			expectedSt:   decimal.NewFromFloat(3070),  // PA tax on wages only
			expectedLoc:  decimal.NewFromFloat(1000),  // Local tax on wages
			expectedFICA: decimal.NewFromFloat(7650),  // FICA on wages only
			description:  "Transition year with mixed income sources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			federal, state, local, fica := calculator.CalculateTotalTaxes(
				tt.income, tt.isRetired, tt.age1, tt.age2, tt.totalWages)

			// Allow for larger rounding differences due to complex calculations
			tolerance := decimal.NewFromInt(200)

			fedDiff := federal.Sub(tt.expectedFed).Abs()
			assert.True(t, fedDiff.LessThan(tolerance),
				"%s - Federal: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedFed.StringFixed(2), federal.StringFixed(2), fedDiff.StringFixed(2))

			stDiff := state.Sub(tt.expectedSt).Abs()
			assert.True(t, stDiff.LessThan(tolerance),
				"%s - State: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedSt.StringFixed(2), state.StringFixed(2), stDiff.StringFixed(2))

			locDiff := local.Sub(tt.expectedLoc).Abs()
			assert.True(t, locDiff.LessThan(tolerance),
				"%s - Local: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedLoc.StringFixed(2), local.StringFixed(2), locDiff.StringFixed(2))

			ficaDiff := fica.Sub(tt.expectedFICA).Abs()
			assert.True(t, ficaDiff.LessThan(tolerance),
				"%s - FICA: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedFICA.StringFixed(2), fica.StringFixed(2), ficaDiff.StringFixed(2))
		})
	}
}

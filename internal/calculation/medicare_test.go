package calculation

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMedicareCalculator_CalculatePartBPremium(t *testing.T) {
	mc := NewMedicareCalculator()

	tests := []struct {
		name                   string
		magi                   decimal.Decimal
		isMarriedFilingJointly bool
		expectedPremium        decimal.Decimal
		description            string
	}{
		{
			name:                   "Low income - no IRMAA",
			magi:                   decimal.NewFromInt(50000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(185.00),
			description:            "Base premium only for low income",
		},
		{
			name:                   "Single filer - first IRMAA tier",
			magi:                   decimal.NewFromInt(110000),
			isMarriedFilingJointly: false,
			expectedPremium:        decimal.NewFromFloat(254.90), // 185 + 69.90
			description:            "Single filer in first IRMAA tier",
		},
		{
			name:                   "Joint filer - first IRMAA tier",
			magi:                   decimal.NewFromInt(220000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(254.90), // 185 + 69.90
			description:            "Joint filer in first IRMAA tier",
		},
		{
			name:                   "Joint filer - second IRMAA tier",
			magi:                   decimal.NewFromInt(280000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(429.60), // 185 + 69.90 + 174.70
			description:            "Joint filer in second IRMAA tier",
		},
		{
			name:                   "Joint filer - third IRMAA tier",
			magi:                   decimal.NewFromInt(350000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(709.10), // 185 + 69.90 + 174.70 + 279.50
			description:            "Joint filer in third IRMAA tier",
		},
		{
			name:                   "Joint filer - fourth IRMAA tier",
			magi:                   decimal.NewFromInt(400000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(1093.40), // 185 + 69.90 + 174.70 + 279.50 + 384.30
			description:            "Joint filer in fourth IRMAA tier",
		},
		{
			name:                   "Joint filer - highest IRMAA tier",
			magi:                   decimal.NewFromInt(800000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(1582.50), // 185 + 69.90 + 174.70 + 279.50 + 384.30 + 489.10
			description:            "Joint filer in highest IRMAA tier",
		},
		{
			name:                   "Person A and Person B scenario - high income",
			magi:                   decimal.NewFromInt(300000),
			isMarriedFilingJointly: true,
			expectedPremium:        decimal.NewFromFloat(429.60), // 185 + 69.90 + 174.70 (reaches 2nd tier)
			description:            "Realistic scenario for Person A and Person B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			premium := mc.CalculatePartBPremium(tt.magi, tt.isMarriedFilingJointly)

			if !premium.Equal(tt.expectedPremium) {
				t.Errorf("CalculatePartBPremium() = %v, want %v", premium, tt.expectedPremium)
			}

			t.Logf("MAGI: $%s, Premium: $%s/month (%s)",
				tt.magi.StringFixed(0), premium.StringFixed(2), tt.description)
		})
	}
}

func TestMedicareCalculator_CalculateAnnualPartBCost(t *testing.T) {
	mc := NewMedicareCalculator()

	tests := []struct {
		name                   string
		magi                   decimal.Decimal
		isMarriedFilingJointly bool
		expectedAnnualCost     decimal.Decimal
	}{
		{
			name:                   "Low income - no IRMAA",
			magi:                   decimal.NewFromInt(50000),
			isMarriedFilingJointly: true,
			expectedAnnualCost:     decimal.NewFromFloat(2220.00), // 185 * 12
		},
		{
			name:                   "High income - multiple IRMAA tiers",
			magi:                   decimal.NewFromInt(400000),
			isMarriedFilingJointly: true,
			expectedAnnualCost:     decimal.NewFromFloat(13120.80), // 1093.40 * 12
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annualCost := mc.CalculateAnnualPartBCost(tt.magi, tt.isMarriedFilingJointly)

			if !annualCost.Equal(tt.expectedAnnualCost) {
				t.Errorf("CalculateAnnualPartBCost() = %v, want %v", annualCost, tt.expectedAnnualCost)
			}

			t.Logf("MAGI: $%s, Annual Cost: $%s (%s per month)",
				tt.magi.StringFixed(0), annualCost.StringFixed(2),
				annualCost.Div(decimal.NewFromInt(12)).StringFixed(2))
		})
	}
}

func TestEstimateMAGI(t *testing.T) {
	tests := []struct {
		name              string
		pensionIncome     decimal.Decimal
		tspWithdrawals    decimal.Decimal
		taxableSSBenefits decimal.Decimal
		otherIncome       decimal.Decimal
		expectedMAGI      decimal.Decimal
	}{
		{
			name:              "Typical retirement income",
			pensionIncome:     decimal.NewFromInt(60000),
			tspWithdrawals:    decimal.NewFromInt(40000),
			taxableSSBenefits: decimal.NewFromInt(25000),
			otherIncome:       decimal.NewFromInt(5000),
			expectedMAGI:      decimal.NewFromInt(130000),
		},
		{
			name:              "High retirement income - IRMAA eligible",
			pensionIncome:     decimal.NewFromInt(80000),
			tspWithdrawals:    decimal.NewFromInt(120000),
			taxableSSBenefits: decimal.NewFromInt(35000),
			otherIncome:       decimal.NewFromInt(15000),
			expectedMAGI:      decimal.NewFromInt(250000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			magi := EstimateMAGI(tt.pensionIncome, tt.tspWithdrawals, tt.taxableSSBenefits, tt.otherIncome)

			if !magi.Equal(tt.expectedMAGI) {
				t.Errorf("EstimateMAGI() = %v, want %v", magi, tt.expectedMAGI)
			}

			t.Logf("Pension: $%s, TSP: $%s, SS: $%s, Other: $%s => MAGI: $%s",
				tt.pensionIncome.StringFixed(0), tt.tspWithdrawals.StringFixed(0),
				tt.taxableSSBenefits.StringFixed(0), tt.otherIncome.StringFixed(0),
				magi.StringFixed(0))
		})
	}
}

// TestMedicareRealWorldScenario tests Medicare calculations with realistic Person A/B income levels
func TestMedicareRealWorldScenario(t *testing.T) {
	mc := NewMedicareCalculator()

	// Realistic scenario: Combined pension ~$80k, TSP withdrawals ~$60k, SS ~$40k
	pensionIncome := decimal.NewFromInt(80000)
	tspWithdrawals := decimal.NewFromInt(60000)
	taxableSSBenefits := decimal.NewFromInt(30000) // Assume 85% of SS is taxable
	otherIncome := decimal.Zero

	estimatedMAGI := EstimateMAGI(pensionIncome, tspWithdrawals, taxableSSBenefits, otherIncome)

	// Calculate Medicare premium per person
	monthlyPremiumPerPerson := mc.CalculatePartBPremium(estimatedMAGI, true)
	annualPremiumPerPerson := mc.CalculateAnnualPartBCost(estimatedMAGI, true)

	// Total for both Person A and Person B
	totalAnnualPremium := annualPremiumPerPerson.Mul(decimal.NewFromInt(2))

	t.Logf("=== Medicare Cost Analysis for Person A & Person B ===")
	t.Logf("Estimated MAGI: $%s", estimatedMAGI.StringFixed(0))
	t.Logf("Monthly premium per person: $%s", monthlyPremiumPerPerson.StringFixed(2))
	t.Logf("Annual premium per person: $%s", annualPremiumPerPerson.StringFixed(2))
	t.Logf("Combined annual Medicare cost: $%s", totalAnnualPremium.StringFixed(2))

	// Verify IRMAA is being applied for high income
	if estimatedMAGI.GreaterThan(decimal.NewFromInt(206000)) {
		if monthlyPremiumPerPerson.LessThanOrEqual(decimal.NewFromFloat(185.00)) {
			t.Error("Expected IRMAA surcharge for high income, but got base premium only")
		}
		t.Logf("IRMAA surcharge applied: $%s/month per person",
			monthlyPremiumPerPerson.Sub(decimal.NewFromFloat(185.00)).StringFixed(2))
	}
}

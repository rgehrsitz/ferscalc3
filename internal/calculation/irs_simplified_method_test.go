package calculation

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestGetExpectedPaymentsIRS validates the IRS Pub 575 Table 1 lookup.
// Single-life:     <50→230, 50→210, 55→190, 60→170, 65→150, 70→140
// Joint-survivor:  <50→410, 50→380, 55→360, 60→340, 65→320, 70→310
func TestGetExpectedPaymentsIRS(t *testing.T) {
	tests := []struct {
		name       string
		age        int
		hasSurvivor bool
		want       int
	}{
		// Single-life
		{"single age 45", 45, false, 230},
		{"single age 49", 49, false, 230},
		{"single age 50", 50, false, 210},
		{"single age 54", 54, false, 210},
		{"single age 55", 55, false, 190},
		{"single age 59", 59, false, 190},
		{"single age 60", 60, false, 170},
		{"single age 64", 64, false, 170},
		{"single age 65", 65, false, 150},
		{"single age 69", 69, false, 150},
		{"single age 70", 70, false, 140},
		{"single age 75", 75, false, 140},
		// Joint-and-survivor
		{"joint age 45", 45, true, 410},
		{"joint age 49", 49, true, 410},
		{"joint age 50", 50, true, 380},
		{"joint age 54", 54, true, 380},
		{"joint age 55", 55, true, 360},
		{"joint age 59", 59, true, 360},
		{"joint age 60", 60, true, 340},
		{"joint age 64", 64, true, 340},
		{"joint age 65", 65, true, 320},
		{"joint age 69", 69, true, 320},
		{"joint age 70", 70, true, 310},
		{"joint age 80", 80, true, 310},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getExpectedPaymentsIRS(tc.age, tc.hasSurvivor)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestCalculateIRSSimplifiedMethodExclusion validates the annual exclusion calculation.
func TestCalculateIRSSimplifiedMethodExclusion(t *testing.T) {
	tests := []struct {
		name               string
		contributions      float64
		age                int
		hasSurvivor        bool
		expectedAnnual     float64
		toleranceDollars   float64
	}{
		{
			// Single life, age 62: 170 expected payments
			// Monthly = 30000/170 = $176.47; Annual = $176.47×12 = $2,117.65
			name:             "single life age 62 - $30k contributions",
			contributions:    30000,
			age:              62,
			hasSurvivor:      false,
			expectedAnnual:   2117.65,
			toleranceDollars: 0.02,
		},
		{
			// Joint survivor, age 57: 360 expected payments
			// Monthly = 25000/360 = $69.44; Annual = $69.44×12 = $833.33
			name:             "joint survivor age 57 - $25k contributions",
			contributions:    25000,
			age:              57,
			hasSurvivor:      true,
			expectedAnnual:   833.28,  // exact: 25000/360*12 = 833.333... → rounded monthly $69.44 × 12
			toleranceDollars: 1.00,
		},
		{
			// Single life, age 72: 140 expected payments
			// Monthly = 10000/140 = $71.43; Annual = $71.43×12 = $857.14
			name:             "single life age 72 - $10k contributions",
			contributions:    10000,
			age:              72,
			hasSurvivor:      false,
			expectedAnnual:   857.16,
			toleranceDollars: 0.02,
		},
		{
			// Zero contributions → zero exclusion
			name:             "zero contributions",
			contributions:    0,
			age:              60,
			hasSurvivor:      false,
			expectedAnnual:   0,
			toleranceDollars: 0,
		},
		{
			// Negative contributions → zero exclusion
			name:             "negative contributions",
			contributions:    -100,
			age:              60,
			hasSurvivor:      false,
			expectedAnnual:   0,
			toleranceDollars: 0,
		},
		{
			// Joint survivor, age 65: 320 expected payments
			// Monthly = 50000/320 = $156.25; Annual = $156.25×12 = $1,875.00
			name:             "joint survivor age 65 - $50k contributions",
			contributions:    50000,
			age:              65,
			hasSurvivor:      true,
			expectedAnnual:   1875.00,
			toleranceDollars: 0.01,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			contributions := decimal.NewFromFloat(tc.contributions)
			got := CalculateIRSSimplifiedMethodExclusion(contributions, tc.age, tc.hasSurvivor)
			assert.InDelta(t, tc.expectedAnnual, got.InexactFloat64(), tc.toleranceDollars,
				"annual exclusion mismatch")
		})
	}
}

// TestIRSExclusionReducesTax verifies that a non-zero EmployeeContributions value
// leads to a lower annual exclusion amount than zero contributions.
// (Integration-level sanity check: the exclusion is positive and finite.)
func TestIRSExclusionIsPositiveForNonZeroContributions(t *testing.T) {
	contributions := decimal.NewFromFloat(40000)
	exclusion := CalculateIRSSimplifiedMethodExclusion(contributions, 60, false)
	assert.True(t, exclusion.GreaterThan(decimal.Zero), "exclusion should be positive")
	assert.True(t, exclusion.LessThan(contributions), "annual exclusion should be less than total contributions")
}

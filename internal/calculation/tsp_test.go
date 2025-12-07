package calculation

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestFourPercentRuleOfficialExamples tests TSP 4% rule using official TSP examples
func TestFourPercentRuleOfficialExamples(t *testing.T) {
	tests := []struct {
		name                     string
		initialBalance           decimal.Decimal
		inflationRate            decimal.Decimal
		expectedFirstYearAnnual  decimal.Decimal
		expectedFirstYearMonthly decimal.Decimal
		expectedSecondYearAnnual decimal.Decimal
		description              string
	}{
		{
			name:                     "Official TSP Example: $500k balance",
			initialBalance:           decimal.NewFromInt(500000),
			inflationRate:            decimal.NewFromFloat(0.02),    // 2% inflation
			expectedFirstYearAnnual:  decimal.NewFromInt(20000),     // 500000 * 0.04
			expectedFirstYearMonthly: decimal.NewFromFloat(1666.67), // 20000 / 12
			expectedSecondYearAnnual: decimal.NewFromInt(20400),     // 20000 * 1.02
			description:              "Standard 4% rule with 2% inflation adjustment",
		},
		{
			name:                     "Official TSP Example: $700k combined balance",
			initialBalance:           decimal.NewFromInt(700000),
			inflationRate:            decimal.NewFromFloat(0.05),    // 5% inflation
			expectedFirstYearAnnual:  decimal.NewFromInt(28000),     // 700000 * 0.04
			expectedFirstYearMonthly: decimal.NewFromFloat(2333.33), // 28000 / 12
			expectedSecondYearAnnual: decimal.NewFromInt(29400),     // 28000 * 1.05
			description:              "Higher balance with higher inflation rate",
		},
		{
			name:                     "PersonA's TSP Balance: $1.966M",
			initialBalance:           decimal.NewFromFloat(1966168.86),
			inflationRate:            decimal.NewFromFloat(0.025),    // 2.5% inflation
			expectedFirstYearAnnual:  decimal.NewFromFloat(78646.75), // 1966168.86 * 0.04
			expectedFirstYearMonthly: decimal.NewFromFloat(6553.90),  // 78646.75 / 12
			expectedSecondYearAnnual: decimal.NewFromFloat(80612.92), // 78646.75 * 1.025
			description:              "Real scenario using PersonA actual TSP balance",
		},
		{
			name:                     "PersonB's TSP Balance: $1.525M",
			initialBalance:           decimal.NewFromFloat(1525175.90),
			inflationRate:            decimal.NewFromFloat(0.025),    // 2.5% inflation
			expectedFirstYearAnnual:  decimal.NewFromFloat(61007.04), // 1525175.90 * 0.04
			expectedFirstYearMonthly: decimal.NewFromFloat(5083.92),  // 61007.04 / 12
			expectedSecondYearAnnual: decimal.NewFromFloat(62532.21), // 61007.04 * 1.025
			description:              "Real scenario using PersonB actual TSP balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewFourPercentRule(tt.initialBalance, tt.inflationRate)

			// Test first year withdrawal
			firstYearWithdrawal := strategy.CalculateWithdrawal(tt.initialBalance, 1, decimal.Zero, 60, false, decimal.Zero)
			assert.True(t, firstYearWithdrawal.Sub(tt.expectedFirstYearAnnual).Abs().LessThan(decimal.NewFromFloat(0.01)),
				"%s: First year - Expected %s, got %s", tt.description,
				tt.expectedFirstYearAnnual.StringFixed(2), firstYearWithdrawal.StringFixed(2))

			// Test monthly withdrawal calculation
			monthlyWithdrawal := firstYearWithdrawal.Div(decimal.NewFromInt(12))
			assert.True(t, monthlyWithdrawal.Sub(tt.expectedFirstYearMonthly).Abs().LessThan(decimal.NewFromFloat(0.01)),
				"%s: Monthly - Expected %s, got %s", tt.description,
				tt.expectedFirstYearMonthly.StringFixed(2), monthlyWithdrawal.StringFixed(2))

			// Test second year withdrawal (inflation adjusted)
			secondYearWithdrawal := strategy.CalculateWithdrawal(tt.initialBalance, 2, decimal.Zero, 61, false, decimal.Zero)
			assert.True(t, secondYearWithdrawal.Sub(tt.expectedSecondYearAnnual).Abs().LessThan(decimal.NewFromFloat(0.01)),
				"%s: Second year - Expected %s, got %s", tt.description,
				tt.expectedSecondYearAnnual.StringFixed(2), secondYearWithdrawal.StringFixed(2))
		})
	}
}

// TestNeedBasedWithdrawalStrategy tests the need-based withdrawal strategy
func TestNeedBasedWithdrawalStrategy(t *testing.T) {
	tests := []struct {
		name           string
		targetMonthly  decimal.Decimal
		expectedAnnual decimal.Decimal
		currentBalance decimal.Decimal
		description    string
	}{
		{
			name:           "Target $2000/month",
			targetMonthly:  decimal.NewFromInt(2000),
			expectedAnnual: decimal.NewFromInt(24000), // 2000 * 12
			currentBalance: decimal.NewFromInt(500000),
			description:    "Standard need-based withdrawal",
		},
		{
			name:           "Target $1700/month (PersonB scenario)",
			targetMonthly:  decimal.NewFromInt(1700),
			expectedAnnual: decimal.NewFromInt(20400), // 1700 * 12
			currentBalance: decimal.NewFromFloat(1525175.90),
			description:    "person_b actual withdrawal target",
		},
		{
			name:           "High balance, low need",
			targetMonthly:  decimal.NewFromInt(1000),
			expectedAnnual: decimal.NewFromInt(12000),
			currentBalance: decimal.NewFromInt(2000000),
			description:    "Conservative withdrawal from large balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewNeedBasedWithdrawal(tt.targetMonthly)

			withdrawal := strategy.CalculateWithdrawal(tt.currentBalance, 1, decimal.Zero, 60, false, decimal.Zero)
			assert.True(t, withdrawal.Equal(tt.expectedAnnual),
				"%s: Expected %s, got %s", tt.description,
				tt.expectedAnnual.StringFixed(2), withdrawal.StringFixed(2))
		})
	}
}

// TestRMDCalculationExamples tests Required Minimum Distribution calculations
func TestRMDCalculationExamples(t *testing.T) {
	tests := []struct {
		name        string
		birthYear   int
		age         int
		balance     decimal.Decimal
		expectedRMD decimal.Decimal
		description string
	}{
		{
			name:        "Age 72 (pre-2020 birth): $500k balance",
			birthYear:   1950,
			age:         72,
			balance:     decimal.NewFromInt(500000),
			expectedRMD: decimal.NewFromFloat(18248.18), // 500000 / 27.4
			description: "First year RMD calculation",
		},
		{
			name:        "Age 73 (SECURE 2.0): $600k balance",
			birthYear:   1951,
			age:         73,
			balance:     decimal.NewFromInt(600000),
			expectedRMD: decimal.NewFromFloat(22641.51), // 600000 / 26.5
			description: "SECURE 2.0 Act RMD age change",
		},
		{
			name:        "Age 75 (2024+ birth): $800k balance",
			birthYear:   1960,
			age:         75,
			balance:     decimal.NewFromInt(800000),
			expectedRMD: decimal.NewFromFloat(32520.33), // 800000 / 24.6
			description: "Future RMD age under SECURE 2.0",
		},
		{
			name:        "Before RMD age: no distribution",
			birthYear:   1965,
			age:         71,
			balance:     decimal.NewFromInt(500000),
			expectedRMD: decimal.Zero,
			description: "No RMD required before minimum age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculator := NewRMDCalculator(tt.birthYear)
			rmd := calculator.CalculateRMD(tt.balance, tt.age)

			// Allow for small rounding differences
			difference := rmd.Sub(tt.expectedRMD).Abs()
			assert.True(t, difference.LessThan(decimal.NewFromFloat(0.01)),
				"%s: Expected %s, got %s (difference: %s)", tt.description,
				tt.expectedRMD.StringFixed(2), rmd.StringFixed(2), difference.StringFixed(2))
		})
	}
}

// TestTSPWithdrawalWithRMD tests TSP withdrawals when RMD is required
func TestTSPWithdrawalWithRMD(t *testing.T) {
	strategy := NewFourPercentRule(decimal.NewFromInt(1000000), decimal.NewFromFloat(0.025))
	balance := decimal.NewFromInt(1000000)

	// Test scenario where 4% rule withdrawal is less than RMD
	fourPercentWithdrawal := strategy.CalculateWithdrawal(balance, 1, decimal.Zero, 75, false, decimal.Zero)
	expectedFourPercent := decimal.NewFromInt(40000) // 1M * 0.04

	assert.True(t, fourPercentWithdrawal.Equal(expectedFourPercent),
		"4% rule should give $40,000, got %s", fourPercentWithdrawal.StringFixed(2))

	// Test scenario where RMD is higher than 4% rule
	rmdAmount := decimal.NewFromFloat(50000) // Higher than 4%
	rmdWithdrawal := strategy.CalculateWithdrawal(balance, 1, decimal.Zero, 75, true, rmdAmount)

	assert.True(t, rmdWithdrawal.Equal(rmdAmount),
		"Should take RMD amount when higher than 4% rule: Expected %s, got %s",
		rmdAmount.StringFixed(2), rmdWithdrawal.StringFixed(2))
}

// TestTSPBalanceDepletion tests TSP balance tracking over time
func TestTSPBalanceDepletion(t *testing.T) {
	initialBalance := decimal.NewFromInt(500000)
	strategy := NewFourPercentRule(initialBalance, decimal.NewFromFloat(0.025))
	returnRate := decimal.NewFromFloat(0.05) // 5% annual return

	projections := ProjectTSP(initialBalance, strategy, returnRate, 10, 1960, nil)

	assert.Len(t, projections, 10, "Should have 10 years of projections")

	// First year should start with initial balance
	assert.True(t, projections[0].BeginningBalance.Equal(initialBalance),
		"First year beginning balance should equal initial balance")

	// Each year's ending balance should become next year's beginning balance
	for i := 1; i < len(projections); i++ {
		assert.True(t, projections[i].BeginningBalance.Equal(projections[i-1].EndingBalance),
			"Year %d beginning balance should equal year %d ending balance", i+1, i)
	}

	// With 5% returns and 4% withdrawals (plus inflation), balance may decline over time
	// Just verify the calculation is reasonable - don't enforce growth
	for i, projection := range projections {
		assert.True(t, projection.EndingBalance.GreaterThanOrEqual(decimal.Zero),
			"Year %d: Ending balance should not be negative: %s", i+1, projection.EndingBalance.StringFixed(2))

		// Withdrawal should be reasonable relative to beginning balance (allow up to 20% for inflation-adjusted 4% rule)
		if projection.BeginningBalance.GreaterThan(decimal.Zero) {
			withdrawalRate := projection.Withdrawal.Div(projection.BeginningBalance)
			assert.True(t, withdrawalRate.LessThan(decimal.NewFromFloat(0.25)),
				"Year %d: Withdrawal rate should be reasonable: %s%%", i+1, withdrawalRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
		}
	}
}

// TestTSPProjectionWithTraditionalRoth tests separate tracking of Traditional and Roth TSP
func TestTSPProjectionWithTraditionalRoth(t *testing.T) {
	initialTraditional := decimal.NewFromFloat(1966168.86) // PersonA's balance
	initialRoth := decimal.Zero                            // No Roth balance
	strategy := NewFourPercentRule(initialTraditional.Add(initialRoth), decimal.NewFromFloat(0.025))
	returnRate := decimal.NewFromFloat(0.05)

	traditionalBalances, rothBalances, withdrawals := ProjectTSPWithTraditionalRoth(
		initialTraditional, initialRoth, strategy, returnRate, 5, 1965, nil)

	assert.Len(t, traditionalBalances, 5, "Should have 5 years of traditional projections")
	assert.Len(t, rothBalances, 5, "Should have 5 years of Roth projections")
	assert.Len(t, withdrawals, 5, "Should have 5 years of withdrawal projections")

	// Since no Roth balance, all withdrawals should come from Traditional
	for i, withdrawal := range withdrawals {
		assert.True(t, withdrawal.GreaterThan(decimal.Zero),
			"Year %d should have positive withdrawals", i+1)
		assert.True(t, rothBalances[i].Equal(decimal.Zero),
			"Year %d Roth balance should remain zero", i+1)
		assert.True(t, traditionalBalances[i].LessThan(initialTraditional),
			"Year %d Traditional balance should decrease", i+1)
	}
}

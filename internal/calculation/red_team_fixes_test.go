package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TestFinding3_RMDFirstYearProration verifies that the first RMD year returns the FULL RMD amount
// and is NOT prorated, per IRS Publication 590-B.
func TestFinding3_RMDFirstYearProration(t *testing.T) {
	// Setup: Person born Jan 15, 1952.
	// RMD Age is 73 (SECURE 2.0).
	// Turns 73 in 2025.
	birthDate := time.Date(1952, 1, 15, 0, 0, 0, 0, time.UTC)

	// Balance: $1,000,000 Traditional
	tradBalance := decimal.NewFromInt(1000000)

	// Employee
	emp := &domain.Employee{
		BirthDate:             birthDate,
		TSPBalanceTraditional: tradBalance,
	}

	// Projection year 2025 (Year 0 relative to 2025 start)
	projDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	// Setup state
	state := &personProjectionState{
		employee:    emp,
		traditional: tradBalance,
	}

	// Result struct
	// In the code, ageStart is used.
	// 1952 + 73 = 2025 = age 73 start.
	result := &personAnnualResult{
		ageStart:  73,
		ageEnd:    73,
		isRetired: true,
	}

	// Calculate RMD
	// Age 73 distribution period is 26.5
	expectedRMD := tradBalance.Div(decimal.NewFromFloat(26.5))

	rmd := calculateRMDForYear(state, projDate, yearEnd, result)

	if !rmd.Equal(expectedRMD) {
		t.Errorf("RMD First Year check failed: Expected full amount %s, got %s. (Did proration occur?)", expectedRMD.StringFixed(2), rmd.StringFixed(2))
	}

	// Verify previous year (age 72) is zero
	resultPre := &personAnnualResult{
		ageStart:  72,
		ageEnd:    72,
		isRetired: true,
	}
	rmdPre := calculateRMDForYear(state, projDate, yearEnd, resultPre)
	if !rmdPre.IsZero() {
		t.Errorf("RMD Pre-73 check failed: Expected 0, got %s", rmdPre.StringFixed(2))
	}
}

// TestFinding4_TSPWithdrawalOrdering verifies that withdrawals come from Traditional (up to RMD) then Roth.
func TestFinding4_TSPWithdrawalOrdering(t *testing.T) {
	ce := &CalculationEngine{}

	// Setup:
	// Trad: $500,000
	// Roth: $100,000
	// RMD: $20,000
	// Total Withdrawal Needed: $50,000

	// Logic Expectation:
	// 1. RMD ($20k) from Traditional.
	// 2. Remaining ($30k) from Roth.
	// 3. Growth applied.

	// If logic was "Trad First" (Old way):
	// 1. $50k from Trad.
	// 2. $0 from Roth.

	emp := &domain.Employee{
		// Needs allocation to trigger the optimized path
		TSPAllocation: &domain.TSPAllocation{
			GFund: decimal.NewFromInt(1.0), // 100% G Fund for simpler growth calc
		},
	}

	state := &personProjectionState{
		employee:    emp,
		traditional: decimal.NewFromInt(500000),
		roth:        decimal.NewFromInt(100000),
	}

	result := &personAnnualResult{
		isRetired:     true,
		tspWithdrawal: decimal.NewFromInt(50000),
		rmd:           decimal.NewFromInt(20000),
	}

	assumptions := &domain.GlobalAssumptions{}

	ce.updateTSPBalancesForPerson(state, time.Now(), assumptions, result)

	// Check balances.
	// G Fund fallback return is approx 4.93% (0.0493) or similar.
	// Let's assume some growth happened.

	// Thresholds:
	// If Roth was used ($30k withdrawal), End Roth should be roughly (100k - 30k) * growth = 70k * 1.05 = ~73.5k
	// If Roth NOT used, End Roth should be roughly 100k * 1.05 = 105k

	if state.roth.GreaterThan(decimal.NewFromInt(90000)) {
		t.Errorf("Finding 4 Fail: Roth balance %s suggests it was NOT used for excess withdrawal. Optimization likely not working.", state.roth.StringFixed(2))
	} else {
		t.Logf("Finding 4 Success: Roth balance %s indicates correct withdrawal ordering.", state.roth.StringFixed(2))
	}
}

// TestFinding5_SSProration verify SS proration when retiring mid-year after birthday.
func TestFinding5_SSProration(t *testing.T) {
	// Setup:
	// Born: Jan 15, 1963.
	// Scenario Start: 2025.
	// Retirement: Oct 31, 2025 (Age 62).
	// SS Start Age: 62.

	// Timeline:
	// Jan 15: Birthday (Turns 62).
	// Oct 31: Retires.
	// Benefits Start: Nov 1 (First full month of retirement).
	// Payable Months: Nov, Dec (2 months).

	// OLD Logic would check "if RetirementDate.Before(BirthdayThisYear)".
	// Oct 31 is AFTER Jan 15. So it would NOT enter the proration block.
	// It would return full annual amount (12 months).

	// NEW Logic should explicitly check monthsOfBenefits logic and apply it.

	birthDate := time.Date(1963, 1, 15, 0, 0, 0, 0, time.UTC)
	retDate := time.Date(2025, 10, 31, 0, 0, 0, 0, time.UTC)

	emp := &domain.Employee{
		BirthDate:    birthDate,
		SSBenefitFRA: decimal.NewFromInt(3000),
		SSBenefit62:  decimal.NewFromInt(2100), // $2100/mo
	}

	scenario := &domain.RetirementScenario{
		RetirementDate: retDate,
		SSStartAge:     62,
	}

	state := &personProjectionState{
		employee:       emp,
		scenario:       scenario,
		retirementYear: 0,
		ssProration:    ssProrationMonthlyAfterRetirement,
	}

	projDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	yearEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	result := &personAnnualResult{
		isRetired: true,
	}

	// Expected: 2100 * 2 = 4200.
	ss := calculateSocialSecurityForYear(state, 0, projDate, yearEnd, decimal.Zero, result)

	// With monthly calc, calculateSocialSecurityForYear calls CalculateSSBenefitForYear which returns annual.
	// Then checks proration.

	// If logic fails (returns full year): 2100 * 12 = 25200.

	expected := decimal.NewFromInt(4200)

	if !ss.Round(2).Equal(expected) {
		t.Errorf("Finding 5 Fail (SS Proration): Expected %s (2 months), got %s. Proration logic likely incorrect for post-birthday retirement.", expected.StringFixed(2), ss.StringFixed(2))
	} else {
		t.Logf("Finding 5 Success: SS Proration calculated correctly as %s", ss.StringFixed(2))
	}
}

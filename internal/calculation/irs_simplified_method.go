package calculation

import "github.com/shopspring/decimal"

// CalculateIRSSimplifiedMethodExclusion computes the annual tax-free portion of a FERS pension
// using the IRS Simplified Method (IRS Publication 721, Table 1).
//
// Federal employees make after-tax contributions to FERS throughout their careers. Upon
// retirement, a portion of each monthly pension payment is a tax-free return of those
// contributions. This exclusion reduces the taxable portion of the pension dollar-for-dollar
// until all contributions have been recovered.
//
// Parameters:
//   - totalContributions: employee's total after-tax contributions to FERS (decimal.Decimal)
//   - annuitantAge: age at retirement / annuity start date (int)
//   - hasSurvivor: true if any survivor benefit is elected (bool)
//
// Returns: annual exclusion amount as decimal.Decimal (the monthly exclusion × 12).
//
// References:
//   - IRS Publication 721, "Tax Guide to U.S. Civil Service Retirement Benefits"
//   - IRS Publication 575, "Pension and Annuity Income", Table 1
//   - 26 USC § 72 — Annuities; Certain Proceeds of Endowment and Life Insurance Contracts
func CalculateIRSSimplifiedMethodExclusion(
	totalContributions decimal.Decimal,
	annuitantAge int,
	hasSurvivor bool,
) decimal.Decimal {
	if totalContributions.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	numPayments := getExpectedPaymentsIRS(annuitantAge, hasSurvivor)
	if numPayments == 0 {
		return decimal.Zero
	}

	// Monthly exclusion = totalContributions / expectedMonthlyPayments
	// Annual exclusion  = monthly × 12
	monthlyExclusion := totalContributions.Div(decimal.NewFromInt(int64(numPayments))).Round(2)
	annualExclusion := monthlyExclusion.Mul(decimal.NewFromInt(12))

	return annualExclusion
}

// getExpectedPaymentsIRS returns the number of expected monthly payments from IRS Pub 575 Table 1.
//
// The IRS assigns a fixed number of expected monthly payments based on the annuitant's age at
// the annuity start date and whether there is a survivor beneficiary:
//
// Single-life (no survivor):
//
//	Age < 50  → 230 months
//	Age 50–54 → 210 months
//	Age 55–59 → 190 months
//	Age 60–64 → 170 months
//	Age 65–69 → 150 months
//	Age 70+   → 140 months
//
// Joint-and-survivor (survivor elected):
//
//	Age < 50  → 410 months
//	Age 50–54 → 380 months
//	Age 55–59 → 360 months
//	Age 60–64 → 340 months
//	Age 65–69 → 320 months
//	Age 70+   → 310 months
func getExpectedPaymentsIRS(age int, hasSurvivor bool) int {
	if hasSurvivor {
		switch {
		case age >= 70:
			return 310
		case age >= 65:
			return 320
		case age >= 60:
			return 340
		case age >= 55:
			return 360
		case age >= 50:
			return 380
		default: // age < 50
			return 410
		}
	}
	// Single-life (no survivor benefit elected)
	switch {
	case age >= 70:
		return 140
	case age >= 65:
		return 150
	case age >= 60:
		return 170
	case age >= 55:
		return 190
	case age >= 50:
		return 210
	default: // age < 50
		return 230
	}
}

package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// FERSInputs represents the inputs needed for FERS pension calculation
type FERSInputs struct {
	High3Salary      decimal.Decimal
	YearsOfService   decimal.Decimal
	RetirementAge    int
	MRA              int
	SurvivorElection decimal.Decimal
}

// FERSPensionCalculation represents the complete FERS pension calculation result
type FERSPensionCalculation struct {
	High3Salary      decimal.Decimal
	ServiceYears     decimal.Decimal
	ServiceMonths    int
	RetirementAge    int
	Multiplier       decimal.Decimal
	AnnualPension    decimal.Decimal
	SurvivorElection decimal.Decimal // Input election percent (0, 0.25, 0.50 typical)
	ReducedPension   decimal.Decimal // Retiree's payable pension after survivor reduction
	SurvivorAnnuity  decimal.Decimal // Amount payable to surviving spouse after death (unreduced base * elected pct)
}

// CalculateFERSPension calculates the annual FERS pension
//
// IMPORTANT: This function returns the FULL ANNUAL pension amount based on the retirement date,
// regardless of when in the year retirement occurs. Partial-year adjustments (proration) for
// mid-year retirements are the responsibility of the calling code (typically in the projection layer).
//
// The survivor annuity calculation uses the unreduced annual pension as its base, which is correct
// per FERS rules. If the retiree dies mid-year during their first year of retirement, the projection
// layer will apply appropriate proration to both the retiree's pension and survivor benefits.
//
// For edge cases involving mid-year retirement followed by death in the same year, verify that:
//  1. The retiree receives prorated pension for the portion of the year after retirement
//  2. The survivor receives prorated survivor annuity for the portion after the retiree's death
//  3. Both calculations use the full annual amounts as their base
func CalculateFERSPension(employee *domain.Employee, retirementDate time.Time) FERSPensionCalculation {
	if employee == nil || employee.EmploymentCategory() != domain.EmploymentTypeFederal {
		return FERSPensionCalculation{}
	}
	// Calculate years of service
	serviceYears := employee.YearsOfService(retirementDate)
	retirementAge := employee.Age(retirementDate)

	// Determine multiplier based on age and service
	multiplier := determineMultiplier(retirementAge, serviceYears)

	// Calculate base pension (unreduced)
	annualPension := employee.High3Salary.Mul(serviceYears).Mul(multiplier)

	// Apply MRA+10 reduction when applicable
	reductionRate := CalculatePensionReduction(employee, retirementDate)
	if reductionRate.GreaterThan(decimal.Zero) {
		reductionFactor := decimal.NewFromInt(1).Sub(reductionRate)
		if reductionFactor.IsNegative() {
			reductionFactor = decimal.Zero
		}
		annualPension = annualPension.Mul(reductionFactor)
	}

	// Survivor rules (simplified FERS):
	// If elect 50% survivor annuity -> retiree pension reduced by 10%
	// If elect 25% survivor annuity -> retiree pension reduced by 5%
	// Assume input SurvivorBenefitElectionPercent holds desired survivor percent of base (0, 0.25, 0.50).
	// NOTE: Survivor annuity is based on the unreduced annual pension (before survivor election reduction)
	reducedPension := annualPension
	survivorAnnuity := decimal.Zero
	election := employee.SurvivorBenefitElectionPercent
	if election.GreaterThan(decimal.Zero) {
		// Normalize election to standard values
		if election.GreaterThan(decimal.NewFromFloat(0.4)) {
			election = decimal.NewFromFloat(0.5)
		}
		if election.GreaterThan(decimal.NewFromFloat(0.20)) && election.LessThan(decimal.NewFromFloat(0.30)) {
			election = decimal.NewFromFloat(0.25)
		}
		if election.Equals(decimal.NewFromFloat(0.5)) {
			reducedPension = annualPension.Mul(decimal.NewFromFloat(0.90)) // 10% reduction
			survivorAnnuity = annualPension.Mul(decimal.NewFromFloat(0.50))
		} else if election.Equals(decimal.NewFromFloat(0.25)) {
			reducedPension = annualPension.Mul(decimal.NewFromFloat(0.95)) // 5% reduction
			survivorAnnuity = annualPension.Mul(decimal.NewFromFloat(0.25))
		} else {
			// Unsupported value - treat as no survivor
			election = decimal.Zero
		}
	}

	return FERSPensionCalculation{
		High3Salary:      employee.High3Salary,
		ServiceYears:     serviceYears,
		RetirementAge:    retirementAge,
		Multiplier:       multiplier,
		AnnualPension:    annualPension,
		SurvivorElection: election,
		ReducedPension:   reducedPension,
		SurvivorAnnuity:  survivorAnnuity,
	}
}

// determineMultiplier determines the FERS pension multiplier based on age and service
func determineMultiplier(retirementAge int, serviceYears decimal.Decimal) decimal.Decimal {
	// Enhanced multiplier: 1.1% if age >= 62 with 20+ years of service
	if retirementAge >= 62 && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(20)) {
		return decimal.NewFromFloat(0.011)
	}

	// Standard multiplier: 1.0% for all other cases
	return decimal.NewFromFloat(0.010)
}

// ApplyFERSPensionCOLA applies the FERS COLA rules
//
// Per OPM regulations (5 CFR § 842.403-842.404):
//   - COLA is NOT applied until the annuitant reaches age 62
//   - The first COLA is payable in December of the year following the year the retiree turns 62,
//     OR December following the first year of retirement, whichever is later
//
// This implementation applies COLA starting in the calendar year AFTER turning 62.
// For retirees who turn 62 mid-year, they receive NO COLA for that partial year, then
// full COLA adjustments beginning the following year. This aligns with OPM guidance.
//
// Annual COLA Rules (when eligible):
// - If CPI change (inflation) is 2% or less, COLA is the actual CPI change
// - If CPI change is between 2% and 3%, COLA is 2%
// - If CPI change is greater than 3%, COLA is CPI change minus 1%
//
// Reference: OPM CSRS and FERS Handbook, Chapter 81
func ApplyFERSPensionCOLA(currentPension decimal.Decimal, inflationRate decimal.Decimal, annuitantAge int) decimal.Decimal {
	if annuitantAge < 62 {
		return currentPension // No COLA until age 62
	}

	var colaRate decimal.Decimal
	if inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.02)) {
		colaRate = inflationRate // Full CPI increase
	} else if inflationRate.GreaterThan(decimal.NewFromFloat(0.02)) && inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.03)) {
		colaRate = decimal.NewFromFloat(0.02) // Capped at 2%
	} else { // inflationRate > 0.03
		colaRate = inflationRate.Sub(decimal.NewFromFloat(0.01)) // CPI minus 1%
	}

	return currentPension.Mul(decimal.NewFromFloat(1.0).Add(colaRate))
}

// CalculateFERSSpecialRetirementSupplement calculates the FERS Special Retirement Supplement (SRS)
// SRS is paid to FERS retirees who retire before age 62 with MRA+ service
// It is equivalent to the Social Security benefit earned during federal service
// Formula: Estimated SS Benefit at Age 62 * (FERS Service Years / 40)
// SRS stops at age 62
func CalculateFERSSpecialRetirementSupplement(ssBenefitAt62 decimal.Decimal, fersServiceYears decimal.Decimal, currentAge int) decimal.Decimal {
	if currentAge >= 62 {
		return decimal.Zero // SRS stops at age 62
	}

	// Calculate the proportion of federal service to total working years (assumed 40)
	serviceProportion := fersServiceYears.Div(decimal.NewFromInt(40))

	// Calculate SRS as annual amount
	annualSRS := ssBenefitAt62.Mul(decimal.NewFromInt(12)).Mul(serviceProportion)

	return annualSRS
}

// ProjectFERSPension projects the FERS pension over multiple years with COLA adjustments
func ProjectFERSPension(employee *domain.Employee, retirementDate time.Time, projectionYears int, inflationRate decimal.Decimal) []decimal.Decimal {
	// Calculate initial pension
	initialCalculation := CalculateFERSPension(employee, retirementDate)
	initialPension := initialCalculation.ReducedPension

	projections := make([]decimal.Decimal, projectionYears)

	// First year is the base pension without COLA
	projections[0] = initialPension

	// Apply COLA starting from year 1
	currentPension := initialPension
	for year := 1; year < projectionYears; year++ {
		projectionDate := retirementDate.AddDate(year, 0, 0)
		age := employee.Age(projectionDate)

		// Apply COLA for this year
		currentPension = ApplyFERSPensionCOLA(currentPension, inflationRate, age)
		projections[year] = currentPension
	}

	return projections
}

// CalculatePensionForYear calculates the pension amount for a specific year in the projection
func CalculatePensionForYear(employee *domain.Employee, retirementDate time.Time, year int, inflationRate decimal.Decimal) decimal.Decimal {
	if employee == nil || employee.EmploymentCategory() != domain.EmploymentTypeFederal {
		return decimal.Zero
	}
	// Calculate initial pension
	initialCalculation := CalculateFERSPension(employee, retirementDate)
	initialPension := initialCalculation.ReducedPension

	// Year 0 is the base pension without COLA
	if year == 0 {
		return initialPension
	}

	// Apply COLA for each year up to the target year
	currentPension := initialPension

	for y := 1; y <= year; y++ {
		// Calculate age in the projection year (not just years since retirement)
		projectionDate := retirementDate.AddDate(y, 0, 0)
		age := employee.Age(projectionDate)

		currentPension = ApplyFERSPensionCOLA(currentPension, inflationRate, age)
	}

	return currentPension
}

// ValidateFERSEligibility checks if an employee is eligible for FERS retirement
func ValidateFERSEligibility(employee *domain.Employee, retirementDate time.Time) (bool, string) {
	age := employee.Age(retirementDate)
	// Use CreditableService (excludes sick leave) for eligibility determination
	serviceYears := employee.CreditableService(retirementDate)
	mra := dateutil.MinimumRetirementAge(employee.BirthDate)
	hasReachedMRA := dateutil.HasReachedRetirementAge(employee.BirthDate, retirementDate, mra)

	// Check minimum age and service requirements
	if !hasReachedMRA {
		return false, "Employee has not reached Minimum Retirement Age"
	}

	if serviceYears.LessThan(decimal.NewFromInt(5)) {
		return false, "Employee has less than 5 years of service"
	}

	// Check for immediate annuity eligibility
	if age >= 62 && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(5)) {
		return true, "Eligible for immediate annuity at age 62+"
	}

	if hasReachedMRA && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(10)) {
		return true, "Eligible for immediate annuity at MRA with 10+ years"
	}

	if hasReachedMRA && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(5)) && serviceYears.LessThan(decimal.NewFromInt(10)) {
		return true, "Eligible for deferred annuity (reduced benefits)"
	}

	return false, "Not eligible for immediate annuity"
}

// CalculatePensionReduction calculates any reduction in pension benefits
func CalculatePensionReduction(employee *domain.Employee, retirementDate time.Time) decimal.Decimal {
	age := employee.Age(retirementDate)
	serviceYears := employee.YearsOfService(retirementDate)
	mra := dateutil.MinimumRetirementAge(employee.BirthDate)
	hasReachedMRA := dateutil.HasReachedRetirementAge(employee.BirthDate, retirementDate, mra)

	// No reduction if age 62+ with 5+ years, or MRA+ with 20+ years
	if (age >= 62 && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(5))) ||
		(hasReachedMRA && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(20))) {
		return decimal.Zero
	}

	// Reduction applies for MRA+ with 10-20 years of service
	if hasReachedMRA && serviceYears.GreaterThanOrEqual(decimal.NewFromInt(10)) && serviceYears.LessThan(decimal.NewFromInt(20)) {
		// 5% reduction for each year under age 62
		yearsUnder62 := 62 - age
		reductionRate := decimal.NewFromInt(int64(yearsUnder62)).Mul(decimal.NewFromFloat(0.05))
		return reductionRate
	}

	return decimal.Zero
}

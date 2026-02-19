package calculation

import (
	"fmt"
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
//   - For REDUCED annuitants (MRA+10 early retirement): COLA is NOT applied until age 62
//   - For UNREDUCED annuitants (MRA+30, age 60+/20, age 62+/5, or special provisions):
//     COLA is applied starting the first December after at least one full year of retirement
//
// The isReducedAnnuity parameter controls which rule applies:
//   - true  → COLA deferred until age 62 (MRA+10 reduced annuity)
//   - false → COLA applied regardless of age (unreduced immediate annuity, survivor annuity)
//
// Annual COLA Rules (when eligible):
// - If CPI change (inflation) is 2% or less, COLA is the actual CPI change
// - If CPI change is between 2% and 3%, COLA is 2%
// - If CPI change is greater than 3%, COLA is CPI change minus 1%
//
// Reference: OPM CSRS and FERS Handbook, Chapter 81
func ApplyFERSPensionCOLA(currentPension decimal.Decimal, inflationRate decimal.Decimal, annuitantAge int, isReducedAnnuity bool) decimal.Decimal {
	if isReducedAnnuity && annuitantAge < 62 {
		return currentPension // No COLA until age 62 for reduced annuitants
	}

	// FERS COLA can never be negative per OPM rules (5 CFR § 842.403).
	// If CPI change is zero or negative, pension remains unchanged.
	if inflationRate.LessThanOrEqual(decimal.Zero) {
		return currentPension
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

	// Determine if this is a reduced annuity (MRA+10) — affects COLA eligibility before age 62
	isReduced := CalculatePensionReduction(employee, retirementDate).GreaterThan(decimal.Zero)

	projections := make([]decimal.Decimal, projectionYears)

	// First year is the base pension without COLA
	projections[0] = initialPension

	// Apply COLA starting from year 1
	currentPension := initialPension
	for year := 1; year < projectionYears; year++ {
		projectionDate := retirementDate.AddDate(year, 0, 0)
		age := employee.Age(projectionDate)

		// Apply COLA for this year
		currentPension = ApplyFERSPensionCOLA(currentPension, inflationRate, age, isReduced)
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

	// Determine if this is a reduced annuity (MRA+10) — affects COLA eligibility before age 62
	isReduced := CalculatePensionReduction(employee, retirementDate).GreaterThan(decimal.Zero)

	// Apply COLA for each year up to the target year
	currentPension := initialPension

	for y := 1; y <= year; y++ {
		// Calculate age in the projection year (not just years since retirement)
		projectionDate := retirementDate.AddDate(y, 0, 0)
		age := employee.Age(projectionDate)

		currentPension = ApplyFERSPensionCOLA(currentPension, inflationRate, age, isReduced)
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

// ─────────────────────────────────────────────────────────────────────────────
// Audit Trail Support
// ─────────────────────────────────────────────────────────────────────────────

// FERSPensionAuditResult wraps FERSPensionCalculation with an optional step-by-step
// audit trail that documents each formula, its inputs, and the OPM regulations applied.
type FERSPensionAuditResult struct {
	FERSPensionCalculation
	// AuditTrail is populated when CalculateFERSPensionWithAudit is called.
	// It is nil when the standard CalculateFERSPension path is used.
	AuditTrail *domain.CalculationAuditTrail `json:"auditTrail,omitempty"`
}

// CalculateFERSPensionWithAudit performs the same calculation as CalculateFERSPension but also
// produces a structured audit trail documenting every step, formula, and regulatory reference.
//
// This function is additive — it calls CalculateFERSPension internally and wraps the result.
// It is suitable for:
//   - Web API responses where the client requests ?audit=true
//   - Test assertions verifying calculation correctness
//   - Printing human-readable calculation breakdowns for users
func CalculateFERSPensionWithAudit(employee *domain.Employee, retirementDate time.Time) FERSPensionAuditResult {
	trail := &domain.CalculationAuditTrail{
		CalculationType: "FERS Pension",
		OPMReferences: []string{
			"OPM FERS Handbook Chapter 50 – Computation",
			"5 USC 8415 – Computation of basic annuity",
			"5 CFR 842.403-842.404 – COLA rules",
			"5 CFR 630.301 – Sick leave conversion (2,087 hours = 1 year)",
		},
	}
	stepNum := 1

	// ── Step 1: Sick Leave Service Credit ────────────────────────────────────
	const sickLeaveHoursPerYear = 2087
	sickLeaveHours := employee.SickLeaveHours
	sickLeaveYears := decimal.Zero
	if sickLeaveHours.GreaterThan(decimal.Zero) {
		sickLeaveYears = sickLeaveHours.Div(decimal.NewFromInt(sickLeaveHoursPerYear))
	}
	trail.Steps = append(trail.Steps, domain.AuditStep{
		StepNumber:  stepNum,
		StepName:    "Sick Leave Service Credit",
		Description: "Convert unused sick leave hours to additional service credit years",
		Formula:     "sickLeaveYears = unusedSickLeaveHours ÷ 2,087",
		Inputs: map[string]interface{}{
			"unusedSickLeaveHours": sickLeaveHours.InexactFloat64(),
			"hoursPerYear":         sickLeaveHoursPerYear,
		},
		Calculation: fmt.Sprintf("%.0f ÷ %d = %.4f years", sickLeaveHours.InexactFloat64(), sickLeaveHoursPerYear, sickLeaveYears.InexactFloat64()),
		Result:      sickLeaveYears.InexactFloat64(),
		Notes:       "Per OPM: 2,087 hours = 1 year of service credit (5 CFR § 630.301)",
	})
	stepNum++

	// ── Step 2: Total Creditable Service ─────────────────────────────────────
	baseService := employee.CreditableService(retirementDate)
	totalService := baseService.Add(sickLeaveYears)
	trail.Steps = append(trail.Steps, domain.AuditStep{
		StepNumber:  stepNum,
		StepName:    "Total Creditable Service",
		Description: "Add sick leave credit to base service years for pension computation",
		Formula:     "totalService = baseService + sickLeaveCredit",
		Inputs: map[string]interface{}{
			"baseServiceYears": baseService.InexactFloat64(),
			"sickLeaveCredit":  sickLeaveYears.InexactFloat64(),
		},
		Calculation: fmt.Sprintf("%.4f + %.4f = %.4f years", baseService.InexactFloat64(), sickLeaveYears.InexactFloat64(), totalService.InexactFloat64()),
		Result:      totalService.InexactFloat64(),
	})
	stepNum++

	// ── Step 3: Annuity Multiplier Selection ─────────────────────────────────
	retirementAge := employee.Age(retirementDate)
	multiplier := determineMultiplier(retirementAge, totalService)
	multiplierNote := "Standard FERS multiplier: 1.0%"
	if multiplier.Equal(decimal.NewFromFloat(0.011)) {
		multiplierNote = "Enhanced multiplier: 1.1% — age ≥ 62 with ≥ 20 years service (5 USC § 8415(b))"
	}
	trail.Steps = append(trail.Steps, domain.AuditStep{
		StepNumber:  stepNum,
		StepName:    "Annuity Multiplier Selection",
		Description: "Select 1.0% or 1.1% multiplier based on age and service at retirement",
		Formula:     "if (age ≥ 62 AND totalService ≥ 20) → 1.1%; otherwise → 1.0%",
		Inputs: map[string]interface{}{
			"retirementAge":    retirementAge,
			"totalServiceYear": totalService.InexactFloat64(),
		},
		Calculation: fmt.Sprintf("age %d ≥ 62? %v  AND  service %.2f ≥ 20? %v  →  %.1f%%",
			retirementAge, retirementAge >= 62,
			totalService.InexactFloat64(), totalService.GreaterThanOrEqual(decimal.NewFromInt(20)),
			multiplier.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		Result: multiplier.InexactFloat64(),
		Notes:  multiplierNote,
	})
	stepNum++

	// ── Step 4: Basic Annual Annuity ──────────────────────────────────────────
	annualPension := employee.High3Salary.Mul(totalService).Mul(multiplier)
	trail.Steps = append(trail.Steps, domain.AuditStep{
		StepNumber:  stepNum,
		StepName:    "Basic Annual Annuity",
		Description: "Compute gross annual pension before any reductions",
		Formula:     "annualPension = High3Salary × totalServiceYears × multiplier",
		Inputs: map[string]interface{}{
			"high3Salary":      employee.High3Salary.InexactFloat64(),
			"totalServiceYear": totalService.InexactFloat64(),
			"multiplier":       multiplier.InexactFloat64(),
		},
		Calculation: fmt.Sprintf("$%.2f × %.4f × %.3f = $%.2f",
			employee.High3Salary.InexactFloat64(), totalService.InexactFloat64(),
			multiplier.InexactFloat64(), annualPension.InexactFloat64()),
		Result: annualPension.InexactFloat64(),
		Notes:  "Reference: 5 USC § 8415",
	})
	stepNum++

	// ── Step 5: MRA+10 Early Retirement Reduction (if applicable) ─────────────
	reductionRate := CalculatePensionReduction(employee, retirementDate)
	if reductionRate.GreaterThan(decimal.Zero) {
		reductionAmount := annualPension.Mul(reductionRate)
		yearsUnder62 := 62 - retirementAge
		trail.Steps = append(trail.Steps, domain.AuditStep{
			StepNumber:  stepNum,
			StepName:    "MRA+10 Early Retirement Reduction",
			Description: "Apply permanent reduction for MRA+10 retirement with fewer than 20 years",
			Formula:     "reductionRate = yearsUnder62 × 5%;  reducedPension = annualPension × (1 − reductionRate)",
			Inputs: map[string]interface{}{
				"retirementAge": retirementAge,
				"yearsUnder62":  yearsUnder62,
				"reductionRate": fmt.Sprintf("%.0f%%", reductionRate.Mul(decimal.NewFromInt(100)).InexactFloat64()),
			},
			Calculation: fmt.Sprintf("%d years under 62 × 5%% = %.0f%% reduction; $%.2f × (1 − %.2f) = $%.2f",
				yearsUnder62, reductionRate.Mul(decimal.NewFromInt(100)).InexactFloat64(),
				annualPension.InexactFloat64(), reductionRate.InexactFloat64(),
				annualPension.Mul(decimal.NewFromFloat(1.0).Sub(reductionRate)).InexactFloat64()),
			Result: reductionAmount.InexactFloat64(),
			Notes:  "This is a PERMANENT reduction. Reference: 5 USC § 8414(b)(1)(B)",
		})
		trail.Warnings = append(trail.Warnings,
			fmt.Sprintf("Permanent early retirement reduction of %.0f%% applied ($%.2f/yr) because retirement is %d year(s) before age 62.",
				reductionRate.Mul(decimal.NewFromInt(100)).InexactFloat64(),
				reductionAmount.InexactFloat64(),
				yearsUnder62))
		stepNum++
	}

	// ── Step 6: Survivor Benefit Cost (if elected) ────────────────────────────
	election := employee.SurvivorBenefitElectionPercent
	if election.GreaterThan(decimal.Zero) {
		reductionFactor := decimal.Zero
		if election.GreaterThan(decimal.NewFromFloat(0.4)) {
			reductionFactor = decimal.NewFromFloat(0.10) // 50% election → 10% cost
		} else if election.GreaterThan(decimal.NewFromFloat(0.2)) {
			reductionFactor = decimal.NewFromFloat(0.05) // 25% election → 5% cost
		}
		if reductionFactor.GreaterThan(decimal.Zero) {
			reducedBase := annualPension
			if reductionRate.GreaterThan(decimal.Zero) {
				reducedBase = annualPension.Mul(decimal.NewFromFloat(1.0).Sub(reductionRate))
			}
			survivorCost := reducedBase.Mul(reductionFactor)
			trail.Steps = append(trail.Steps, domain.AuditStep{
				StepNumber:  stepNum,
				StepName:    "Survivor Benefit Cost",
				Description: "Monthly reduction to retiree's pension for elected survivor annuity",
				Formula:     "survivorCost = reducedPension × reductionPct",
				Inputs: map[string]interface{}{
					"electedSurvivorPercent": fmt.Sprintf("%.0f%%", election.Mul(decimal.NewFromInt(100)).InexactFloat64()),
					"retireeReductionPct":    fmt.Sprintf("%.0f%%", reductionFactor.Mul(decimal.NewFromInt(100)).InexactFloat64()),
					"reducedPension":         reducedBase.InexactFloat64(),
				},
				Calculation: fmt.Sprintf("$%.2f × %.0f%% = $%.2f/yr reduction",
					reducedBase.InexactFloat64(),
					reductionFactor.Mul(decimal.NewFromInt(100)).InexactFloat64(),
					survivorCost.InexactFloat64()),
				Result: survivorCost.InexactFloat64(),
				Notes:  "Survivor annuity is based on unreduced pension per OPM FERS rules",
			})
			stepNum++
		}
	}

	// ── Step 7: FERS Special Retirement Supplement (if eligible) ──────────────
	if retirementAge < 62 {
		ssBenefit62 := employee.SSBenefit62 // monthly
		if ssBenefit62.GreaterThan(decimal.Zero) {
			fersService := baseService // actual FERS service (excluding sick leave) for SRS formula
			srs := CalculateFERSSpecialRetirementSupplement(ssBenefit62, fersService, retirementAge)
			trail.Steps = append(trail.Steps, domain.AuditStep{
				StepNumber:  stepNum,
				StepName:    "FERS Special Retirement Supplement (SRS)",
				Description: "Bridge supplement paid until age 62 approximating Social Security earned during federal service",
				Formula:     "annualSRS = ssBenefitAt62Monthly × 12 × (fersServiceYears ÷ 40)",
				Inputs: map[string]interface{}{
					"ssBenefitAt62Monthly": ssBenefit62.InexactFloat64(),
					"fersServiceYears":     fersService.InexactFloat64(),
				},
				Calculation: fmt.Sprintf("$%.2f × 12 × (%.2f ÷ 40) = $%.2f/yr",
					ssBenefit62.InexactFloat64(), fersService.InexactFloat64(), srs.InexactFloat64()),
				Result: srs.InexactFloat64(),
				Notes:  "SRS stops at age 62. Subject to earnings test if retiree has post-retirement wages.",
			})
			stepNum++
		}
	}

	// ── Step 8: IRS Simplified Method Exclusion (if applicable) ──────────────
	if employee.EmployeeContributions.GreaterThan(decimal.Zero) {
		hasSurvivor := employee.SurvivorBenefitElectionPercent.GreaterThan(decimal.Zero)
		annualExclusion := CalculateIRSSimplifiedMethodExclusion(employee.EmployeeContributions, retirementAge, hasSurvivor)
		expectedPayments := getExpectedPaymentsIRS(retirementAge, hasSurvivor)
		trail.Steps = append(trail.Steps, domain.AuditStep{
			StepNumber:  stepNum,
			StepName:    "IRS Simplified Method Exclusion",
			Description: "Annual tax-free portion of pension — return of after-tax employee contributions",
			Formula:     "monthlyExclusion = totalContributions ÷ expectedMonthlyPayments;  annualExclusion = monthlyExclusion × 12",
			Inputs: map[string]interface{}{
				"totalContributions":    employee.EmployeeContributions.InexactFloat64(),
				"retirementAge":         retirementAge,
				"hasSurvivor":           hasSurvivor,
				"expectedMonthlyPmt":    expectedPayments,
			},
			Calculation: fmt.Sprintf("$%.2f ÷ %d × 12 = $%.2f/yr exclusion",
				employee.EmployeeContributions.InexactFloat64(), expectedPayments, annualExclusion.InexactFloat64()),
			Result: annualExclusion.InexactFloat64(),
			Notes:  "Reduces taxable pension each year. Gross pension is unchanged. Reference: IRS Pub 721, 26 USC § 72.",
		})
		stepNum++
	}

	// ── Compute final result via the canonical function ───────────────────────
	pensionCalc := CalculateFERSPension(employee, retirementDate)

	trail.FinalResult = pensionCalc.ReducedPension.InexactFloat64()
	trail.InputSummary = fmt.Sprintf(
		"Employee retiring at age %d with %.2f years of service (plus %.2f yr sick leave credit), High-3 salary $%.2f",
		retirementAge,
		baseService.InexactFloat64(),
		sickLeaveYears.InexactFloat64(),
		employee.High3Salary.InexactFloat64(),
	)

	return FERSPensionAuditResult{
		FERSPensionCalculation: pensionCalc,
		AuditTrail:             trail,
	}
}

package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// SocialSecurityCalculator handles Social Security benefit calculations
type SocialSecurityCalculator struct {
	BirthYear         int
	FullRetirementAge dateutil.RetirementAge
	BenefitAtFRA      decimal.Decimal
}

// NewSocialSecurityCalculator creates a new Social Security calculator
func NewSocialSecurityCalculator(birthYear int, benefitAtFRA decimal.Decimal) *SocialSecurityCalculator {
	return &SocialSecurityCalculator{
		BirthYear:         birthYear,
		FullRetirementAge: dateutil.FullRetirementAge(time.Date(birthYear, 1, 1, 0, 0, 0, 0, time.UTC)),
		BenefitAtFRA:      benefitAtFRA,
	}
}

// CalculateBenefitAtAge calculates the Social Security benefit at a specific claiming age.
//
// NOTE: The claimingAge parameter is an integer (whole years), which means this function
// cannot model mid-year claiming (e.g., claiming at age 62 and 4 months). SSA reduction
// and delayed-credit calculations are month-sensitive in practice. This approximation is
// acceptable for long-range projections but may introduce small errors (~1-2%) for
// individuals whose optimal claiming age falls between integer years.
//
// Future enhancement: Accept a fractional claiming age or (year, month) pair to enable
// month-level granularity for early/delayed retirement credit calculations.
func (ssc *SocialSecurityCalculator) CalculateBenefitAtAge(claimingAge int) decimal.Decimal {
	if claimingAge < 62 {
		return decimal.Zero
	}

	fullRetirementMonths := ssc.FullRetirementAge.TotalMonths()
	claimingMonths := claimingAge * 12

	if claimingMonths < fullRetirementMonths {
		// Early retirement reduction
		monthsEarly := fullRetirementMonths - claimingMonths
		var reductionRate decimal.Decimal

		if monthsEarly <= 36 {
			// 5/9 of 1% per month for first 36 months
			reductionRate = decimal.NewFromFloat(5.0 / 9.0 / 100.0).Mul(decimal.NewFromInt(int64(monthsEarly)))
		} else {
			// 5/9 of 1% for first 36 months, 5/12 of 1% for additional months
			firstReduction := decimal.NewFromFloat(5.0 / 9.0 / 100.0).Mul(decimal.NewFromInt(36))
			additionalMonths := monthsEarly - 36
			additionalReduction := decimal.NewFromFloat(5.0 / 12.0 / 100.0).Mul(decimal.NewFromInt(int64(additionalMonths)))
			reductionRate = firstReduction.Add(additionalReduction)
		}

		return ssc.BenefitAtFRA.Mul(decimal.NewFromFloat(1).Sub(reductionRate))
	}

	if claimingMonths > fullRetirementMonths {
		// Delayed retirement credits: 8% per year (2/3% per month)
		monthsDelayed := claimingMonths - fullRetirementMonths
		if monthsDelayed > 48 { // Cap at age 70
			monthsDelayed = 48
		}

		delayCredit := decimal.NewFromFloat(2.0 / 3.0 / 100.0).Mul(decimal.NewFromInt(int64(monthsDelayed)))
		return ssc.BenefitAtFRA.Mul(decimal.NewFromFloat(1).Add(delayCredit))
	}

	return ssc.BenefitAtFRA // At Full Retirement Age
}

// CalculateMonthlySSBenefitAtAge calculates the monthly SS benefit based on claiming age
func CalculateMonthlySSBenefitAtAge(baseFRA decimal.Decimal, birthDate time.Time, claimingAge int) decimal.Decimal {
	ssc := NewSocialSecurityCalculator(birthDate.Year(), baseFRA)
	return ssc.CalculateBenefitAtAge(claimingAge)
}

// ApplySSCOLA applies the annual Social Security COLA
func ApplySSCOLA(currentBenefit decimal.Decimal, colaRate decimal.Decimal) decimal.Decimal {
	return currentBenefit.Mul(decimal.NewFromFloat(1.0).Add(colaRate))
}

// ProjectSocialSecurityBenefits projects Social Security benefits over multiple years
func ProjectSocialSecurityBenefits(employee *domain.Employee, ssStartAge int, projectionYears int, colaRate decimal.Decimal) []decimal.Decimal {
	projections := make([]decimal.Decimal, projectionYears)

	// Calculate initial benefit at claiming age
	initialBenefit := CalculateMonthlySSBenefitAtAge(employee.SSBenefitFRA, employee.BirthDate, ssStartAge)
	currentBenefit := initialBenefit

	for year := 0; year < projectionYears; year++ {
		projectionDate := nowFunc().AddDate(year, 0, 0)
		age := employee.Age(projectionDate)

		// Check if Social Security has started
		if age >= ssStartAge {
			// Apply COLA for each year after the first
			if year > 0 {
				currentBenefit = ApplySSCOLA(currentBenefit, colaRate)
			}
			projections[year] = currentBenefit.Mul(decimal.NewFromInt(12)) // Convert to annual
		} else {
			projections[year] = decimal.Zero
		}
	}

	return projections
}

// SSTaxCalculator handles Social Security taxation calculations
type SSTaxCalculator struct{}

// NewSSTaxCalculator creates a new Social Security tax calculator
func NewSSTaxCalculator() *SSTaxCalculator {
	return &SSTaxCalculator{}
}

// CalculateTaxableSocialSecurity determines the federally taxable portion of SS benefits
// Provisional Income = (AGI - deductions) + Non-taxable interest + 1/2 of Social Security benefits
// Thresholds for Married Filing Jointly:
// - Provisional Income <= $32,000: 0% of SS benefits are taxable
// - Provisional Income > $32,000 and <= $44,000: Up to 50% of SS benefits are taxable
// - Provisional Income > $44,000: Up to 85% of SS benefits are taxable
func (sstc *SSTaxCalculator) CalculateTaxableSocialSecurity(totalSSBenefitAnnual decimal.Decimal, provisionalIncome decimal.Decimal) decimal.Decimal {
	return calculateTaxableSocialSecurityWithThresholds(
		totalSSBenefitAnnual,
		provisionalIncome,
		decimal.NewFromInt(32000),
		decimal.NewFromInt(44000),
	)
}

// CalculateTaxableSocialSecuritySingle determines the federally taxable portion for single filers
func (sstc *SSTaxCalculator) CalculateTaxableSocialSecuritySingle(totalSSBenefitAnnual decimal.Decimal, provisionalIncome decimal.Decimal) decimal.Decimal {
	return calculateTaxableSocialSecurityWithThresholds(
		totalSSBenefitAnnual,
		provisionalIncome,
		decimal.NewFromInt(25000),
		decimal.NewFromInt(34000),
	)
}

func calculateTaxableSocialSecurityWithThresholds(totalSSBenefitAnnual, provisionalIncome, threshold1, threshold2 decimal.Decimal) decimal.Decimal {
	if provisionalIncome.LessThanOrEqual(threshold1) {
		return decimal.Zero
	}

	halfBenefit := totalSSBenefitAnnual.Mul(decimal.NewFromFloat(0.5))
	excessOverFirst := provisionalIncome.Sub(threshold1)
	halfExcess := excessOverFirst.Mul(decimal.NewFromFloat(0.5))

	if provisionalIncome.LessThanOrEqual(threshold2) {
		return decimal.Min(halfExcess, halfBenefit)
	}

	// Above the second threshold: IRS worksheet logic
	basePortion := decimal.Min(
		halfBenefit,
		threshold2.Sub(threshold1).Mul(decimal.NewFromFloat(0.5)),
	)

	excessOverSecond := provisionalIncome.Sub(threshold2)
	additionalPortion := excessOverSecond.Mul(decimal.NewFromFloat(0.85))

	candidate := basePortion.Add(additionalPortion)
	maxTaxable := totalSSBenefitAnnual.Mul(decimal.NewFromFloat(0.85))

	return decimal.Min(candidate, maxTaxable)
}

// CalculateProvisionalIncome calculates the provisional income for Social Security taxation
func (sstc *SSTaxCalculator) CalculateProvisionalIncome(agi decimal.Decimal, nontaxableInterest decimal.Decimal, ssBenefits decimal.Decimal) decimal.Decimal {
	// Provisional Income = AGI + Non-taxable interest + 1/2 of Social Security benefits
	return agi.Add(nontaxableInterest).Add(ssBenefits.Mul(decimal.NewFromFloat(0.5)))
}

// InterpolateSSBenefit interpolates Social Security benefits between known ages
func InterpolateSSBenefit(benefit62, benefitFRA, benefit70 decimal.Decimal, claimingAge int) decimal.Decimal {
	fra := 67 // Assuming 1960+ birth year for simplicity

	if claimingAge <= 62 {
		return benefit62
	}
	if claimingAge >= 70 {
		return benefit70
	}
	if claimingAge == fra {
		return benefitFRA
	}

	if claimingAge < fra {
		monthsEarly := (fra - claimingAge) * 12
		reduction := scaledEarlyRetirementReduction(monthsEarly, benefit62, benefitFRA, fra)
		factor := decimal.NewFromInt(1).Sub(reduction)
		interpolated := benefitFRA.Mul(factor)
		if interpolated.LessThan(benefit62) {
			return benefit62
		}
		if interpolated.GreaterThan(benefitFRA) {
			return benefitFRA
		}
		return interpolated
	}

	monthsDelayed := (claimingAge - fra) * 12
	increase := scaledDelayedRetirementIncrease(monthsDelayed, benefit70, benefitFRA, fra)
	factor := decimal.NewFromInt(1).Add(increase)
	interpolated := benefitFRA.Mul(factor)
	if interpolated.GreaterThan(benefit70) {
		return benefit70
	}
	if interpolated.LessThan(benefitFRA) {
		return benefitFRA
	}
	return interpolated
}

func scaledEarlyRetirementReduction(monthsEarly int, benefit62, benefitFRA decimal.Decimal, fra int) decimal.Decimal {
	if monthsEarly <= 0 || benefitFRA.IsZero() {
		return decimal.Zero
	}
	totalMonths := (fra - 62) * 12
	ssaReduction := ssEarlyReductionRate(monthsEarly)
	totalSSA := ssEarlyReductionRate(totalMonths)
	if totalSSA.IsZero() {
		return decimal.Zero
	}

	actualTotalReduction := decimal.NewFromInt(1)
	if !benefitFRA.IsZero() {
		actualTotalReduction = actualTotalReduction.Sub(benefit62.Div(benefitFRA))
	}
	if actualTotalReduction.IsNegative() {
		actualTotalReduction = decimal.Zero
	}
	if actualTotalReduction.IsZero() {
		return decimal.Zero
	}

	scale := actualTotalReduction.Div(totalSSA)
	return ssaReduction.Mul(scale)
}

func scaledDelayedRetirementIncrease(monthsDelayed int, benefit70, benefitFRA decimal.Decimal, fra int) decimal.Decimal {
	if monthsDelayed <= 0 || benefitFRA.IsZero() {
		return decimal.Zero
	}
	maxMonths := (70 - fra) * 12
	if monthsDelayed > maxMonths {
		monthsDelayed = maxMonths
	}

	ssaIncrease := ssDelayedIncreaseRate(monthsDelayed)
	totalSSA := ssDelayedIncreaseRate(maxMonths)
	if totalSSA.IsZero() {
		return decimal.Zero
	}

	actualTotalIncrease := decimal.Zero
	if !benefitFRA.IsZero() {
		actualTotalIncrease = benefit70.Div(benefitFRA).Sub(decimal.NewFromInt(1))
	}
	if actualTotalIncrease.IsNegative() {
		actualTotalIncrease = decimal.Zero
	}
	if actualTotalIncrease.IsZero() {
		return decimal.Zero
	}

	scale := actualTotalIncrease.Div(totalSSA)
	return ssaIncrease.Mul(scale)
}

func ssEarlyReductionRate(months int) decimal.Decimal {
	if months <= 0 {
		return decimal.Zero
	}
	firstTierMonths := months
	if firstTierMonths > 36 {
		firstTierMonths = 36
	}
	secondTierMonths := 0
	if months > 36 {
		secondTierMonths = months - 36
	}

	rateFirst := decimal.NewFromFloat(5.0 / 9.0 / 100.0)
	rateSecond := decimal.NewFromFloat(5.0 / 12.0 / 100.0)

	reduction := rateFirst.Mul(decimal.NewFromInt(int64(firstTierMonths)))
	if secondTierMonths > 0 {
		reduction = reduction.Add(rateSecond.Mul(decimal.NewFromInt(int64(secondTierMonths))))
	}
	return reduction
}

func ssDelayedIncreaseRate(months int) decimal.Decimal {
	if months <= 0 {
		return decimal.Zero
	}
	rate := decimal.NewFromFloat(2.0 / 3.0 / 100.0)
	return rate.Mul(decimal.NewFromInt(int64(months)))
}

// CalculateSurvivorSSBenefit computes the survivor benefit based on deceased primary benefit and survivor age.
// Simplified FERS/SS rule: Survivor can receive up to 100% of deceased's benefit if at or after survivor FRA.
// Early survivor reduction: approximately 28.5% maximum reduction if claimed at 60 (i.e. 71.5% of full).
// We interpolate linearly between age 60 (71.5%) and survivor FRA (~67).
func CalculateSurvivorSSBenefit(deceasedCurrent decimal.Decimal, survivorAgeYears int, survivorFRA dateutil.RetirementAge) decimal.Decimal {
	if deceasedCurrent.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	fraMonths := survivorFRA.TotalMonths()
	survivorMonths := survivorAgeYears * 12
	if survivorMonths >= fraMonths {
		return deceasedCurrent
	}
	if survivorAgeYears < 60 {
		return decimal.Zero
	} // not yet eligible (simplified, ignoring child-in-care cases)
	// Linear interpolation from 60 -> FRA: factor from 0.715 -> 1.0
	totalMonths := fraMonths - 60*12
	if totalMonths <= 0 {
		return deceasedCurrent
	}
	monthsFrom60 := survivorMonths - 60*12
	if monthsFrom60 < 0 {
		monthsFrom60 = 0
	}
	ratio := decimal.NewFromInt(int64(monthsFrom60)).Div(decimal.NewFromInt(int64(totalMonths)))
	minFactor := decimal.NewFromFloat(0.715)
	factor := minFactor.Add(decimal.NewFromFloat(1.0).Sub(minFactor).Mul(ratio))
	return deceasedCurrent.Mul(factor)
}

// CalculateSSBenefitForYear calculates the Social Security benefit for a specific year
func CalculateSSBenefitForYear(employee *domain.Employee, ssStartAge int, year int, colaRate decimal.Decimal) decimal.Decimal {
	// Start projection from 2025, not current year
	projectionStartYear := 2025

	// Use end of year for age calculation to account for people who turn eligible during the year
	endOfYearDate := time.Date(projectionStartYear+year, 12, 31, 0, 0, 0, 0, time.UTC)
	age := employee.Age(endOfYearDate)

	// Check if Social Security has started
	if age < ssStartAge {
		return decimal.Zero
	}

	// Calculate initial benefit at claiming age
	initialBenefit := CalculateMonthlySSBenefitAtAge(employee.SSBenefitFRA, employee.BirthDate, ssStartAge)

	// Apply COLA for each year after the first year of benefits
	currentBenefit := initialBenefit
	yearsSinceStart := age - ssStartAge

	// Only apply COLA if this is not the first year of benefits
	if yearsSinceStart > 0 {
		for y := 0; y < yearsSinceStart; y++ {
			currentBenefit = ApplySSCOLA(currentBenefit, colaRate)
		}
	}

	return currentBenefit.Mul(decimal.NewFromInt(12)) // Convert to annual
}

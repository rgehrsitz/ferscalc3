package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// CalculateFERSSupplementYear calculates the FERS Special Retirement Supplement for a given year offset
// Includes earnings test reduction if applicable
func CalculateFERSSupplementYear(employee *domain.Employee, retirementDate time.Time, yearsSinceRetirement int, inflationRate decimal.Decimal, earnedIncome decimal.Decimal, earningsLimit decimal.Decimal) decimal.Decimal {
	if employee == nil || employee.EmploymentCategory() != domain.EmploymentTypeFederal {
		return decimal.Zero
	}
	if yearsSinceRetirement < 0 {
		return decimal.Zero
	}

	projectionDate := retirementDate.AddDate(yearsSinceRetirement, 0, 0)
	yearStart := time.Date(projectionDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	ageAtYearStart := employee.Age(yearStart)
	if ageAtYearStart >= 62 {
		return decimal.Zero // SRS stops at age 62
	}

	serviceYears := employee.YearsOfService(retirementDate)
	ageAtProjection := employee.Age(projectionDate)
	if ageAtProjection > 61 {
		ageAtProjection = 61
	}
	srs := CalculateFERSSpecialRetirementSupplement(employee.SSBenefit62, serviceYears, ageAtProjection)

	for y := 0; y < yearsSinceRetirement; y++ {
		srs = srs.Mul(decimal.NewFromFloat(1).Add(inflationRate))
	}

	// Apply partial year proration for the year turning 62
	birthdayThisYear := time.Date(projectionDate.Year(), employee.BirthDate.Month(), employee.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
	if employee.Age(birthdayThisYear) >= 62 {
		daysBeforeBirthday := birthdayThisYear.Sub(yearStart).Hours() / 24.0
		daysInYear := float64(dateutil.DaysInYear(projectionDate.Year()))
		if daysBeforeBirthday < 0 {
			daysBeforeBirthday = 0
		}
		if daysInYear > 0 {
			fraction := daysBeforeBirthday / daysInYear
			if fraction < 0 {
				fraction = 0
			}
			if fraction > 1 {
				fraction = 1
			}
			srs = srs.Mul(decimal.NewFromFloat(fraction))
		}
	}

	// Apply Earnings Test Reduction
	// Reduction is $1 for every $2 earned above the limit
	if earnedIncome.GreaterThan(earningsLimit) && earningsLimit.GreaterThan(decimal.Zero) {
		excessEarnings := earnedIncome.Sub(earningsLimit)
		reduction := excessEarnings.Div(decimal.NewFromInt(2))
		srs = srs.Sub(reduction)
		if srs.IsNegative() {
			srs = decimal.Zero
		}
	}

	return srs
}

// CalculateFEHBPremium calculates FEHB premium for a given year
func CalculateFEHBPremium(employee *domain.Employee, year int, premiumInflation decimal.Decimal, fehbConfig domain.FEHBConfig) decimal.Decimal {
	inflationFactor := decimal.NewFromFloat(1).Add(premiumInflation)
	adjustedPremium := employee.FEHBPremiumPerPayPeriod.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year))))
	return adjustedPremium.Mul(decimal.NewFromInt(int64(fehbConfig.PayPeriodsPerYear)))
}

// CalculateRMD wraps RMD calculation with birth year
func CalculateRMD(balance decimal.Decimal, birthYear, age int) decimal.Decimal {
	rmdCalc := NewRMDCalculator(birthYear)
	return rmdCalc.CalculateRMD(balance, age)
}

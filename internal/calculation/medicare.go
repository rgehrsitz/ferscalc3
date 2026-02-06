package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// MedicareCalculator handles Medicare Part B premium calculations including IRMAA
type MedicareCalculator struct {
	BasePremium2025      decimal.Decimal
	PremiumInflationRate decimal.Decimal
	IRMAAThresholds      []IRMAAThreshold
}

// IRMAAThreshold represents an IRMAA income threshold and corresponding surcharge
type IRMAAThreshold struct {
	IncomeThresholdSingle decimal.Decimal // For single filers
	IncomeThresholdJoint  decimal.Decimal // For married filing jointly
	MonthlySurcharge      decimal.Decimal // Additional monthly premium per person
}

// NewMedicareCalculator creates a new Medicare calculator with 2025 rates
func NewMedicareCalculator() *MedicareCalculator {
	return &MedicareCalculator{
		BasePremium2025:      decimal.NewFromFloat(185.00), // 2025 base Part B premium
		PremiumInflationRate: decimal.NewFromFloat(0.055),
		IRMAAThresholds: []IRMAAThreshold{
			// 2025 IRMAA thresholds (based on 2023 MAGI)
			{
				IncomeThresholdSingle: decimal.NewFromInt(103000),
				IncomeThresholdJoint:  decimal.NewFromInt(206000),
				MonthlySurcharge:      decimal.NewFromFloat(69.90),
			},
			{
				IncomeThresholdSingle: decimal.NewFromInt(129000),
				IncomeThresholdJoint:  decimal.NewFromInt(258000),
				MonthlySurcharge:      decimal.NewFromFloat(174.70),
			},
			{
				IncomeThresholdSingle: decimal.NewFromInt(161000),
				IncomeThresholdJoint:  decimal.NewFromInt(322000),
				MonthlySurcharge:      decimal.NewFromFloat(279.50),
			},
			{
				IncomeThresholdSingle: decimal.NewFromInt(193000),
				IncomeThresholdJoint:  decimal.NewFromInt(386000),
				MonthlySurcharge:      decimal.NewFromFloat(384.30),
			},
			{
				IncomeThresholdSingle: decimal.NewFromInt(500000),
				IncomeThresholdJoint:  decimal.NewFromInt(750000),
				MonthlySurcharge:      decimal.NewFromFloat(489.10),
			},
		},
	}
}

// NewMedicareCalculatorWithConfig creates a new Medicare calculator with configurable values
func NewMedicareCalculatorWithConfig(config domain.MedicareConfig) *MedicareCalculator {
	defaultCalc := NewMedicareCalculator()
	calc := &MedicareCalculator{
		BasePremium2025:      defaultCalc.BasePremium2025,
		PremiumInflationRate: defaultCalc.PremiumInflationRate,
		IRMAAThresholds:      append([]IRMAAThreshold(nil), defaultCalc.IRMAAThresholds...),
	}

	if !config.BasePremium2025.IsZero() {
		calc.BasePremium2025 = config.BasePremium2025
	}
	if !config.PremiumInflationRate.IsZero() {
		calc.PremiumInflationRate = config.PremiumInflationRate
	}
	if len(config.IRMAAThresholds) > 0 {
		var thresholds []IRMAAThreshold
		for _, threshold := range config.IRMAAThresholds {
			thresholds = append(thresholds, IRMAAThreshold{
				IncomeThresholdSingle: threshold.IncomeThresholdSingle,
				IncomeThresholdJoint:  threshold.IncomeThresholdJoint,
				MonthlySurcharge:      threshold.MonthlySurcharge,
			})
		}
		calc.IRMAAThresholds = thresholds
	}

	return calc
}

// CalculatePartBPremium calculates Medicare Part B premium including IRMAA surcharge
// based on Modified Adjusted Gross Income (MAGI) from 2 years prior
func (mc *MedicareCalculator) CalculatePartBPremium(magi decimal.Decimal, isMarriedFilingJointly bool) decimal.Decimal {
	premium := mc.BasePremium2025

	// Find applicable IRMAA surcharge (single highest tier, not cumulative)
	premium = premium.Add(mc.irmaaSurchargeForMAGI(magi, isMarriedFilingJointly))

	return premium
}

// CalculateAnnualPartBCost calculates annual Medicare Part B cost with sophisticated IRMAA
func (mc *MedicareCalculator) CalculateAnnualPartBCost(estimatedMAGI decimal.Decimal, isMarriedFilingJointly bool) decimal.Decimal {
	// Base premium for 2025
	basePremium := mc.BasePremium2025

	// Calculate IRMAA surcharge based on MAGI
	irmaaSurcharge := mc.irmaaSurchargeForMAGI(estimatedMAGI, isMarriedFilingJointly)

	// Apply IRMAA surcharge
	totalMonthlyPremium := basePremium.Add(irmaaSurcharge)

	// Convert to annual cost
	annualCost := totalMonthlyPremium.Mul(decimal.NewFromInt(12))

	return annualCost
}

// irmaaSurchargeForMAGI returns the single highest IRMAA tier surcharge for the given MAGI.
func (mc *MedicareCalculator) irmaaSurchargeForMAGI(estimatedMAGI decimal.Decimal, isMarriedFilingJointly bool) decimal.Decimal {
	var surcharge decimal.Decimal

	for _, threshold := range mc.IRMAAThresholds {
		var incomeThreshold decimal.Decimal
		if isMarriedFilingJointly {
			incomeThreshold = threshold.IncomeThresholdJoint
		} else {
			incomeThreshold = threshold.IncomeThresholdSingle
		}

		if estimatedMAGI.GreaterThan(incomeThreshold) {
			surcharge = threshold.MonthlySurcharge
		} else {
			break
		}
	}

	return surcharge
}

// CalculateMedicarePremiumWithInflation calculates Medicare premium with inflation adjustment
func (mc *MedicareCalculator) CalculateMedicarePremiumWithInflation(estimatedMAGI decimal.Decimal, isMarriedFilingJointly bool, yearsFrom2025 int) decimal.Decimal {
	// Base calculation
	baseAnnualCost := mc.CalculateAnnualPartBCost(estimatedMAGI, isMarriedFilingJointly)

	// Apply inflation adjustment (Medicare premiums typically increase faster than general inflation)
	medicareInflationRate := mc.PremiumInflationRate
	inflationFactor := decimal.NewFromInt(1).Add(medicareInflationRate).Pow(decimal.NewFromInt(int64(yearsFrom2025)))

	adjustedAnnualCost := baseAnnualCost.Mul(inflationFactor)

	return adjustedAnnualCost
}

// EstimateMAGI estimates Modified Adjusted Gross Income for IRMAA calculation
// This is a simplified calculation - real MAGI includes various adjustments
func EstimateMAGI(pensionIncome, tspWithdrawals, taxableSSBenefits, otherIncome decimal.Decimal) decimal.Decimal {
	// Simplified MAGI calculation for retirement income
	// In reality, MAGI includes additional items like tax-exempt interest, etc.
	return pensionIncome.Add(tspWithdrawals).Add(taxableSSBenefits).Add(otherIncome)
}

// IsMedicareEligible checks if someone is eligible for Medicare (age 65+).
// Delegates to dateutil.IsMedicareEligible for consistent age calculation.
func IsMedicareEligible(birthDate, atDate time.Time) bool {
	return dateutil.IsMedicareEligible(birthDate, atDate)
}

// calculateMedicarePremium calculates Medicare Part B premiums with IRMAA considerations
// based on current year income (simplified - real IRMAA uses 2-year-old MAGI)
// calculateMedicarePremium calculates Medicare Part B premiums with IRMAA considerations
// using an externally provided estimatedMAGI (should be MAGI from two years prior per SSA rules)
// and returns per-person annual premiums (A, B). projectionDate is used for eligibility and
// to compute inflation-adjusted premium growth since 2025.
func (ce *CalculationEngine) calculateMedicarePremium(personA, personB *domain.Employee, projectionDate time.Time, estimatedMAGI decimal.Decimal, isMarriedFilingJointly bool) (decimal.Decimal, decimal.Decimal) {

	var personAPremium decimal.Decimal
	var personBPremium decimal.Decimal

	yearsFrom2025 := projectionDate.Year() - 2025

	// Note: estimatedMAGI is expected to already include taxable SS effects. If an external
	// caller provides only partial MAGI, we could estimate here, but caller will pass full MAGI.

	// Check eligibility and compute per-person premium (with inflation adjustment)
	if IsMedicareEligible(personA.BirthDate, projectionDate) {
		// Use the inflation-aware calculation
		personAPremium = ce.MedicareCalc.CalculateMedicarePremiumWithInflation(estimatedMAGI, isMarriedFilingJointly, yearsFrom2025)
	}
	if IsMedicareEligible(personB.BirthDate, projectionDate) {
		personBPremium = ce.MedicareCalc.CalculateMedicarePremiumWithInflation(estimatedMAGI, isMarriedFilingJointly, yearsFrom2025)
	}

	return personAPremium, personBPremium
}

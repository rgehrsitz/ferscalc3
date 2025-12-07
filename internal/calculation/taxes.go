package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// TAX CALCULATION ASSUMPTIONS:
//
// 1. Federal Tax Brackets: Uses 2025 tax brackets for all projection years
//    - No inflation indexing applied to future years
//    - Standard deduction: $30,000 (2025 MFJ estimate)
//    - Additional standard deduction for age 65+: $1,550 per person
//
// 2. Pennsylvania State Tax: 3.07% flat tax rate (no inflation adjustment)
//
// 3. Upper Makefield EIT: 1% flat tax on earned income only
//    - Does not apply to retirement income (pensions, TSP, SS)
//
// 4. Medicare Part B & IRMAA: Placeholder implementation
//    - Base premium: $185/month per person (2025 estimate)
//    - IRMAA surcharge: $200/month placeholder (needs AGI-based calculation)
//
// Inflation indexing: Tax brackets and standard deductions are adjusted annually based on inflation.
// This implementation applies cumulative inflation from ProjectionBaseYear (2025) to future years.

// TaxBracket represents a federal tax bracket
type TaxBracket struct {
	Min  decimal.Decimal
	Max  decimal.Decimal
	Rate decimal.Decimal
}

// FederalTaxCalculator handles federal income tax calculations
type FederalTaxCalculator struct {
	Year                    int
	StandardDeduction       decimal.Decimal
	StandardDeductionSingle decimal.Decimal
	Brackets                []TaxBracket
	BracketsSingle          []TaxBracket
	AdditionalStdDed        decimal.Decimal // For age 65+
}

// NewFederalTaxCalculator2025 creates a new federal tax calculator for 2025
func NewFederalTaxCalculator2025() *FederalTaxCalculator {
	return &FederalTaxCalculator{
		Year:              2025,
		StandardDeduction: decimal.NewFromInt(30000), // MFJ 2025 estimated
		AdditionalStdDed:  decimal.NewFromInt(1550),  // Per person 65+
		Brackets: []TaxBracket{
			{decimal.Zero, decimal.NewFromInt(23200), decimal.NewFromFloat(0.10)},
			{decimal.NewFromInt(23201), decimal.NewFromInt(94300), decimal.NewFromFloat(0.12)},
			{decimal.NewFromInt(94301), decimal.NewFromInt(201050), decimal.NewFromFloat(0.22)},
			{decimal.NewFromInt(201051), decimal.NewFromInt(383900), decimal.NewFromFloat(0.24)},
			{decimal.NewFromInt(383901), decimal.NewFromInt(487450), decimal.NewFromFloat(0.32)},
			{decimal.NewFromInt(487451), decimal.NewFromInt(731200), decimal.NewFromFloat(0.35)},
			{decimal.NewFromInt(731201), decimal.NewFromInt(999999999), decimal.NewFromFloat(0.37)},
		},
	}
}

// NewFederalTaxCalculator creates a new federal tax calculator with configurable values
func NewFederalTaxCalculator(config domain.FederalTaxConfig) *FederalTaxCalculator {
	var bracketsMFJ []TaxBracket
	for _, b := range config.TaxBrackets2025 {
		bracketsMFJ = append(bracketsMFJ, TaxBracket{Min: b.Min, Max: b.Max, Rate: b.Rate})
	}
	if len(bracketsMFJ) == 0 { // fallback defaults
		bracketsMFJ = []TaxBracket{
			{decimal.Zero, decimal.NewFromInt(23200), decimal.NewFromFloat(0.10)},
			{decimal.NewFromInt(23201), decimal.NewFromInt(94300), decimal.NewFromFloat(0.12)},
			{decimal.NewFromInt(94301), decimal.NewFromInt(201050), decimal.NewFromFloat(0.22)},
			{decimal.NewFromInt(201051), decimal.NewFromInt(383900), decimal.NewFromFloat(0.24)},
			{decimal.NewFromInt(383901), decimal.NewFromInt(487450), decimal.NewFromFloat(0.32)},
			{decimal.NewFromInt(487451), decimal.NewFromInt(731200), decimal.NewFromFloat(0.35)},
			{decimal.NewFromInt(731201), decimal.NewFromInt(999999999), decimal.NewFromFloat(0.37)},
		}
	}
	var bracketsSingle []TaxBracket
	for _, b := range config.TaxBrackets2025Single {
		bracketsSingle = append(bracketsSingle, TaxBracket{Min: b.Min, Max: b.Max, Rate: b.Rate})
	}
	// Provide defaults if single not supplied
	stdSingle := config.StandardDeductionSingle
	if stdSingle.IsZero() && !config.StandardDeductionMFJ.IsZero() {
		stdSingle = config.StandardDeductionMFJ.Div(decimal.NewFromInt(2))
	}
	if len(bracketsSingle) == 0 && len(bracketsMFJ) > 0 {
		for _, b := range bracketsMFJ {
			bracketsSingle = append(bracketsSingle, TaxBracket{Min: b.Min.Div(decimal.NewFromInt(2)), Max: b.Max.Div(decimal.NewFromInt(2)), Rate: b.Rate})
		}
	}
	return &FederalTaxCalculator{Year: 2025, StandardDeduction: config.StandardDeductionMFJ, StandardDeductionSingle: stdSingle, AdditionalStdDed: config.AdditionalStandardDeduction, Brackets: bracketsMFJ, BracketsSingle: bracketsSingle}
}

// CalculateFederalTax calculates federal income tax
func (ftc *FederalTaxCalculator) CalculateFederalTax(grossIncome decimal.Decimal, age1, age2 int) decimal.Decimal {
	standardDed := ftc.StandardDeduction

	// Additional standard deduction for seniors
	if age1 >= 65 {
		standardDed = standardDed.Add(ftc.AdditionalStdDed)
	}
	if age2 >= 65 {
		standardDed = standardDed.Add(ftc.AdditionalStdDed)
	}

	taxableIncome := grossIncome.Sub(standardDed)
	if taxableIncome.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	var totalTax decimal.Decimal
	for _, bracket := range ftc.Brackets {
		if taxableIncome.LessThanOrEqual(bracket.Min) {
			break
		}
		incomeInBracket := decimal.Min(taxableIncome, bracket.Max).Sub(bracket.Min)
		if incomeInBracket.GreaterThan(decimal.Zero) {
			totalTax = totalTax.Add(incomeInBracket.Mul(bracket.Rate))
		}
	}

	return totalTax
}

// StateTaxCalculator defines the interface for state tax calculations
type StateTaxCalculator interface {
	CalculateTax(income domain.TaxableIncome, isRetired bool) decimal.Decimal
}

// PennsylvaniaTaxCalculator handles Pennsylvania state tax calculations
type PennsylvaniaTaxCalculator struct {
	Rate decimal.Decimal
}

// NewPennsylvaniaTaxCalculator creates a new Pennsylvania tax calculator
func NewPennsylvaniaTaxCalculator() *PennsylvaniaTaxCalculator {
	return &PennsylvaniaTaxCalculator{
		Rate: decimal.NewFromFloat(0.0307), // Default rate
	}
}

// NewPennsylvaniaTaxCalculatorWithConfig creates a new Pennsylvania tax calculator with configurable rate
func NewPennsylvaniaTaxCalculatorWithConfig(config domain.StateLocalTaxConfig) *PennsylvaniaTaxCalculator {
	return &PennsylvaniaTaxCalculator{
		Rate: config.PennsylvaniaRate,
	}
}

// CalculateTax calculates Pennsylvania state income tax
// PA has a flat tax rate (currently 3.07%)
// Key Exclusions: PA does NOT tax FERS pensions, TSP withdrawals, or Social Security benefits
// Only earned income (salary) is typically taxed
func (ptc *PennsylvaniaTaxCalculator) CalculateTax(income domain.TaxableIncome, isRetired bool) decimal.Decimal {
	if isRetired {
		// PA exempts retirement income: pensions, TSP, Social Security
		// Only tax earned income (wages) and interest income
		taxablePA := income.WageIncome.Add(income.InterestIncome).Add(income.OtherTaxableIncome)
		return taxablePA.Mul(ptc.Rate)
	}

	// While working: tax wages at configured rate
	return income.WageIncome.Mul(ptc.Rate)
}

// NewJerseyTaxCalculator handles New Jersey state tax calculations
type NewJerseyTaxCalculator struct {
	// For simplicity in this phase, we might use a single effective rate or a simplified bracket set
	// NJ has progressive rates from 1.4% to 10.75%
	// NJ excludes SS.
	// NJ has a pension exclusion for those 62+ with income <= $150k (phased out)
}

// NewNewJerseyTaxCalculator creates a new NJ tax calculator
func NewNewJerseyTaxCalculator() *NewJerseyTaxCalculator {
	return &NewJerseyTaxCalculator{}
}

// CalculateTax calculates New Jersey state income tax
func (njtc *NewJerseyTaxCalculator) CalculateTax(income domain.TaxableIncome, isRetired bool) decimal.Decimal {
	// Simplified NJ Tax Logic for Phase 1
	// 1. Social Security is exempt.
	// 2. Pension/Retirement income exclusion (simplified):
	//    If total income is reasonable, we assume some exclusion.
	//    For now, let's apply a simplified progressive rate on taxable income.

	// Taxable base: Salary + Pension + TSP + Other (SS is exempt)
	taxableIncome := income.Salary.Add(income.FERSPension).Add(income.TSPWithdrawalsTrad).Add(income.OtherTaxableIncome)

	// Simple progressive brackets (approximate for 2024/2025)
	// 0 - 20k: 1.4%
	// 20k - 35k: 1.75%
	// 35k - 40k: 3.5%
	// 40k - 75k: 5.525%
	// 75k - 500k: 6.37%
	// 500k+: 8.97% (ignoring top bracket for now)

	// We'll implement a simple bracket calculation here
	// Note: This is a simplification. Real NJ tax has filing status differences.

	var tax decimal.Decimal
	remaining := taxableIncome

	brackets := []struct {
		limit decimal.Decimal
		rate  decimal.Decimal
	}{
		{decimal.NewFromInt(20000), decimal.NewFromFloat(0.014)},
		{decimal.NewFromInt(35000), decimal.NewFromFloat(0.0175)},
		{decimal.NewFromInt(40000), decimal.NewFromFloat(0.035)},
		{decimal.NewFromInt(75000), decimal.NewFromFloat(0.05525)},
		{decimal.NewFromInt(500000), decimal.NewFromFloat(0.0637)},
		{decimal.NewFromInt(999999999), decimal.NewFromFloat(0.0897)},
	}

	prevLimit := decimal.Zero
	for _, b := range brackets {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}

		width := b.limit.Sub(prevLimit)
		taxableInBracket := decimal.Min(remaining, width)

		tax = tax.Add(taxableInBracket.Mul(b.rate))

		remaining = remaining.Sub(taxableInBracket)
		prevLimit = b.limit
	}

	return tax
}

// UpperMakefieldEITCalculator handles Upper Makefield Township local tax calculations
type UpperMakefieldEITCalculator struct {
	Rate decimal.Decimal
}

// NewUpperMakefieldEITCalculator creates a new Upper Makefield EIT calculator
func NewUpperMakefieldEITCalculator() *UpperMakefieldEITCalculator {
	return &UpperMakefieldEITCalculator{
		Rate: decimal.NewFromFloat(0.01), // Default rate
	}
}

// NewUpperMakefieldEITCalculatorWithConfig creates a new Upper Makefield EIT calculator with configurable rate
func NewUpperMakefieldEITCalculatorWithConfig(config domain.StateLocalTaxConfig) *UpperMakefieldEITCalculator {
	return &UpperMakefieldEITCalculator{
		Rate: config.UpperMakefieldEITRate,
	}
}

// CalculateEIT calculates the Earned Income Tax for Upper Makefield Township
// EIT only applies to earned income, not retirement income
func (ume *UpperMakefieldEITCalculator) CalculateEIT(wageIncome decimal.Decimal, isRetired bool) decimal.Decimal {
	if isRetired {
		return decimal.Zero // EIT only applies to earned income
	}

	return wageIncome.Mul(ume.Rate)
}

// FICACalculator handles FICA tax calculations
type FICACalculator struct {
	Year                int
	SSWageBase          decimal.Decimal
	SSRate              decimal.Decimal
	MedicareRate        decimal.Decimal
	AdditionalRate      decimal.Decimal
	HighIncomeThreshold decimal.Decimal
}

// NewFICACalculator2025 creates a new FICA calculator for 2025
func NewFICACalculator2025() *FICACalculator {
	return &FICACalculator{
		Year:                2025,
		SSWageBase:          decimal.NewFromInt(176100), // 2025 official
		SSRate:              decimal.NewFromFloat(0.062),
		MedicareRate:        decimal.NewFromFloat(0.0145),
		AdditionalRate:      decimal.NewFromFloat(0.009),
		HighIncomeThreshold: decimal.NewFromInt(250000), // MFJ
	}
}

// NewFICACalculator creates a new FICA calculator with configurable values
func NewFICACalculator(config domain.FICATaxConfig) *FICACalculator {
	year := config.Year
	if year == 0 {
		year = ProjectionBaseYear
	}

	return &FICACalculator{
		Year:                year,
		SSWageBase:          config.SocialSecurityWageBase,
		SSRate:              config.SocialSecurityRate,
		MedicareRate:        config.MedicareRate,
		AdditionalRate:      config.AdditionalMedicareRate,
		HighIncomeThreshold: config.HighIncomeThresholdMFJ,
	}
}

// CalculateFICA calculates FICA taxes (Social Security and Medicare)
func (fc *FICACalculator) CalculateFICA(wages decimal.Decimal, totalHouseholdWages decimal.Decimal) decimal.Decimal {
	// Social Security tax (capped per individual)
	ssWages := decimal.Min(wages, fc.SSWageBase)
	ssTax := ssWages.Mul(fc.SSRate)

	// Medicare tax (no cap)
	medicareTax := wages.Mul(fc.MedicareRate)

	// Additional Medicare tax for high earners - proportionally allocated
	var additionalMedicare decimal.Decimal
	if totalHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
		excessWages := totalHouseholdWages.Sub(fc.HighIncomeThreshold)
		totalAdditionalMedicare := excessWages.Mul(fc.AdditionalRate)
		// Allocate proportionally based on individual wages
		wagesProportion := wages.Div(totalHouseholdWages)
		additionalMedicare = totalAdditionalMedicare.Mul(wagesProportion)
	}

	return ssTax.Add(medicareTax).Add(additionalMedicare)
}

// CalculateFICAWithProration calculates FICA taxes with proration for partial year work
func (fc *FICACalculator) CalculateFICAWithProration(wages decimal.Decimal, totalHouseholdWages decimal.Decimal, workFraction decimal.Decimal) decimal.Decimal {
	// Apply work fraction to wages first
	proratedWages := wages.Mul(workFraction)
	proratedHouseholdWages := totalHouseholdWages.Mul(workFraction)

	// Social Security tax (capped per individual, then prorated)
	ssWages := decimal.Min(proratedWages, fc.SSWageBase)
	ssTax := ssWages.Mul(fc.SSRate)

	// Medicare tax (no cap, prorated)
	medicareTax := proratedWages.Mul(fc.MedicareRate)

	// Additional Medicare tax for high earners (prorated and proportionally allocated)
	var additionalMedicare decimal.Decimal
	if proratedHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
		excessWages := proratedHouseholdWages.Sub(fc.HighIncomeThreshold)
		totalAdditionalMedicare := excessWages.Mul(fc.AdditionalRate)
		// Allocate proportionally based on individual prorated wages
		wagesProportion := proratedWages.Div(proratedHouseholdWages)
		additionalMedicare = totalAdditionalMedicare.Mul(wagesProportion)
	}

	return ssTax.Add(medicareTax).Add(additionalMedicare)
}

// ComprehensiveTaxCalculator handles all tax calculations
type ComprehensiveTaxCalculator struct {
	FederalTaxCalc *FederalTaxCalculator
	StateTaxCalc   StateTaxCalculator
	LocalTaxCalc   *UpperMakefieldEITCalculator
	FICATaxCalc    *FICACalculator
	SSTaxCalc      *SSTaxCalculator
	InflationRate  decimal.Decimal // Annual inflation rate for indexing tax brackets
}

// NewComprehensiveTaxCalculator creates a new comprehensive tax calculator
func NewComprehensiveTaxCalculator() *ComprehensiveTaxCalculator {
	return &ComprehensiveTaxCalculator{
		FederalTaxCalc: NewFederalTaxCalculator2025(),
		StateTaxCalc:   NewPennsylvaniaTaxCalculator(),
		LocalTaxCalc:   NewUpperMakefieldEITCalculator(),
		FICATaxCalc:    NewFICACalculator2025(),
		SSTaxCalc:      NewSSTaxCalculator(),
		InflationRate:  decimal.NewFromFloat(DefaultCOLARate), // Default 3% inflation
	}
}

// NewComprehensiveTaxCalculatorWithConfig creates a new comprehensive tax calculator with configurable values
func NewComprehensiveTaxCalculatorWithConfig(federalRules domain.FederalRules, state string, inflationRate decimal.Decimal) *ComprehensiveTaxCalculator {
	var stateCalc StateTaxCalculator
	switch state {
	case "New Jersey":
		stateCalc = NewNewJerseyTaxCalculator()
	case "Pennsylvania":
		stateCalc = NewPennsylvaniaTaxCalculatorWithConfig(federalRules.StateLocalTaxConfig)
	default:
		// Default to PA if unknown, or maybe a generic zero tax calculator?
		// For now, default to PA as it was the previous behavior
		stateCalc = NewPennsylvaniaTaxCalculatorWithConfig(federalRules.StateLocalTaxConfig)
	}
	return &ComprehensiveTaxCalculator{
		FederalTaxCalc: NewFederalTaxCalculator(federalRules.FederalTaxConfig),
		StateTaxCalc:   stateCalc,
		LocalTaxCalc:   NewUpperMakefieldEITCalculatorWithConfig(federalRules.StateLocalTaxConfig),
		FICATaxCalc:    NewFICACalculator(federalRules.FICATaxConfig),
		SSTaxCalc:      NewSSTaxCalculator(),
		InflationRate:  inflationRate,
	}
}

// CalculateTotalTaxes calculates all applicable taxes with inflation-adjusted tax brackets
// This method is for calculating current year (year 0) taxes
func (ctc *ComprehensiveTaxCalculator) CalculateTotalTaxes(taxableIncome domain.TaxableIncome, isRetired bool, agePersonA, agePersonB int, workingIncome decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	// Calculate federal tax for current year (no inflation adjustment needed for year 0)
	federalTax := ctc.calculateFederalTaxWithInflation(taxableIncome, agePersonA, agePersonB, 0)

	// Calculate state tax
	stateTax := ctc.StateTaxCalc.CalculateTax(taxableIncome, isRetired)

	// Calculate local tax (only on earned income)
	localTax := ctc.LocalTaxCalc.CalculateEIT(workingIncome, isRetired)

	// Calculate FICA tax (only on earned income)
	ficaTax := ctc.FICATaxCalc.CalculateFICA(workingIncome, workingIncome)

	return federalTax, stateTax, localTax, ficaTax
}

// calculateFederalTaxWithInflation calculates federal tax with inflation-adjusted brackets
func (ctc *ComprehensiveTaxCalculator) calculateFederalTaxWithInflation(taxableIncome domain.TaxableIncome, agePersonA, agePersonB, projectionYear int) decimal.Decimal {
	// Calculate total taxable income
	totalIncome := taxableIncome.Salary.Add(taxableIncome.FERSPension).Add(taxableIncome.TSPWithdrawalsTrad).Add(taxableIncome.TaxableSSBenefits).Add(taxableIncome.OtherTaxableIncome)

	// Apply standard deduction with age-based adjustments
	standardDeduction := ctc.FederalTaxCalc.StandardDeduction

	// Add additional standard deduction for taxpayers 65 and older
	if agePersonA >= 65 {
		standardDeduction = standardDeduction.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}
	if agePersonB >= 65 {
		standardDeduction = standardDeduction.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}

	// Calculate inflation adjustment factor based on projection year
	// Tax brackets and standard deductions are indexed to inflation
	// projectionYear is the offset from ProjectionBaseYear (0 = 2025, 1 = 2026, etc.)
	var inflationAdjustment decimal.Decimal
	if projectionYear > 0 && !ctc.InflationRate.IsZero() {
		// Cumulative inflation: (1 + inflation)^years
		inflationAdjustment = decimal.NewFromFloat(1.0).Add(ctc.InflationRate).Pow(decimal.NewFromInt(int64(projectionYear)))
	} else {
		inflationAdjustment = decimal.NewFromFloat(1.0)
	}

	// Apply inflation adjustment to standard deduction
	standardDeduction = standardDeduction.Mul(inflationAdjustment)

	// Calculate adjusted gross income
	agi := totalIncome.Sub(standardDeduction)
	if agi.LessThan(decimal.Zero) {
		agi = decimal.Zero
	}

	// Calculate tax using inflation-adjusted brackets
	tax := decimal.Zero
	remainingIncome := agi

	for _, bracket := range ctc.FederalTaxCalc.Brackets {
		// Apply inflation adjustment to bracket thresholds
		adjustedMin := bracket.Min.Mul(inflationAdjustment)
		adjustedMax := bracket.Max.Mul(inflationAdjustment)

		if remainingIncome.LessThanOrEqual(decimal.Zero) {
			break
		}

		// Determine the width of this bracket
		bracketWidth := adjustedMax.Sub(adjustedMin)
		if bracketWidth.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// The amount taxed in this bracket is limited by the remaining income
		// and the width of the bracket. Do not subtract adjustedMin from remainingIncome
		// because remainingIncome already represents income above all previous brackets.
		incomeInBracket := decimal.Min(remainingIncome, bracketWidth)

		// Only tax amounts once the taxpayer's income exceeds the start of this bracket
		if agi.GreaterThan(adjustedMin) && incomeInBracket.GreaterThan(decimal.Zero) {
			tax = tax.Add(incomeInBracket.Mul(bracket.Rate))
			remainingIncome = remainingIncome.Sub(incomeInBracket)
		}
	}

	return tax
}

// calculateFederalTaxWithStatus allows specifying filing status ("mfj" or "single") and number of seniors 65+.
func (ctc *ComprehensiveTaxCalculator) calculateFederalTaxWithStatus(agiComponents domain.TaxableIncome, filingStatus string, seniors, projectionYear int) decimal.Decimal {
	totalIncome := agiComponents.Salary.Add(agiComponents.FERSPension).Add(agiComponents.TSPWithdrawalsTrad).Add(agiComponents.TaxableSSBenefits).Add(agiComponents.OtherTaxableIncome)

	// Standard deduction based on filing status
	standardDed := ctc.FederalTaxCalc.StandardDeduction
	brackets := ctc.FederalTaxCalc.Brackets
	if filingStatus == "single" {
		standardDed = ctc.FederalTaxCalc.StandardDeductionSingle
		if len(ctc.FederalTaxCalc.BracketsSingle) > 0 {
			brackets = ctc.FederalTaxCalc.BracketsSingle
		}
	}
	for i := 0; i < seniors; i++ {
		standardDed = standardDed.Add(ctc.FederalTaxCalc.AdditionalStdDed)
	}

	// Calculate inflation adjustment factor
	var inflationAdjustment decimal.Decimal
	if projectionYear > 0 && !ctc.InflationRate.IsZero() {
		inflationAdjustment = decimal.NewFromFloat(1.0).Add(ctc.InflationRate).Pow(decimal.NewFromInt(int64(projectionYear)))
	} else {
		inflationAdjustment = decimal.NewFromFloat(1.0)
	}

	// Apply inflation adjustment to standard deduction
	standardDed = standardDed.Mul(inflationAdjustment)

	agi := totalIncome.Sub(standardDed)
	if agi.LessThan(decimal.Zero) {
		agi = decimal.Zero
	}
	remaining := agi
	tax := decimal.Zero
	for _, b := range brackets {
		adjMin := b.Min.Mul(inflationAdjustment)
		adjMax := b.Max.Mul(inflationAdjustment)
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		width := adjMax.Sub(adjMin)
		if width.LessThanOrEqual(decimal.Zero) {
			continue
		}
		incomeInBracket := decimal.Min(remaining, width)
		if agi.GreaterThan(adjMin) && incomeInBracket.GreaterThan(decimal.Zero) {
			tax = tax.Add(incomeInBracket.Mul(b.Rate))
			remaining = remaining.Sub(incomeInBracket)
		}
	}
	return tax
}

// CalculateTaxableIncome creates a TaxableIncome struct from cash flow data
func CalculateTaxableIncome(cashFlow domain.AnnualCashFlow, isRetired bool) domain.TaxableIncome {
	return domain.TaxableIncome{
		Salary:             decimal.Zero,
		FERSPension:        cashFlow.PensionPersonA.Add(cashFlow.PensionPersonB).Add(cashFlow.SurvivorPensionPersonA).Add(cashFlow.SurvivorPensionPersonB),
		TSPWithdrawalsTrad: cashFlow.TSPWithdrawalPersonA.Add(cashFlow.TSPWithdrawalPersonB),
		TaxableSSBenefits:  cashFlow.SSBenefitPersonA.Add(cashFlow.SSBenefitPersonB),
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         decimal.Zero,
		InterestIncome:     decimal.Zero,
	}
}

// CalculateCurrentTaxableIncome calculates taxable income for current employment
func CalculateCurrentTaxableIncome(personASalary, personBSalary decimal.Decimal) domain.TaxableIncome {
	totalSalary := personASalary.Add(personBSalary)

	return domain.TaxableIncome{
		Salary:             totalSalary,
		FERSPension:        decimal.Zero,
		TSPWithdrawalsTrad: decimal.Zero,
		TaxableSSBenefits:  decimal.Zero,
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         totalSalary,
		InterestIncome:     decimal.Zero,
	}
}

// TaxCalculationInput contains all inputs needed for tax calculations
type TaxCalculationInput struct {
	PersonA, PersonB *domain.Employee
	Scenario         *domain.Scenario
	Year             int
	IsRetired        bool
	Pensions         [2]decimal.Decimal // [PersonA, PersonB]
	SurvivorPensions [2]decimal.Decimal // [PersonA, PersonB]
	TSPWithdrawals   [2]decimal.Decimal // [PersonA, PersonB]
	SocialSecurity   [2]decimal.Decimal // [PersonA, PersonB]
	WorkingIncome    [2]decimal.Decimal // [PersonA, PersonB]
}

// TaxCalculationResult contains all tax calculation outputs
type TaxCalculationResult struct {
	FederalTax         decimal.Decimal
	StateTax           decimal.Decimal
	LocalTax           decimal.Decimal
	FICATax            decimal.Decimal
	TaxableIncomeTotal decimal.Decimal
	StandardDeduction  decimal.Decimal
	FilingStatus       string
	Seniors            int
}

// CalculateSocialSecurityTaxation calculates the taxable portion of Social Security benefits
func (ctc *ComprehensiveTaxCalculator) CalculateSocialSecurityTaxation(ssBenefits decimal.Decimal, otherIncome decimal.Decimal) decimal.Decimal {
	// Calculate provisional income
	provisionalIncome := ctc.SSTaxCalc.CalculateProvisionalIncome(otherIncome, decimal.Zero, ssBenefits)

	// Calculate taxable portion
	return ctc.SSTaxCalc.CalculateTaxableSocialSecurity(ssBenefits, provisionalIncome)
}

// calculateTaxes calculates all applicable taxes
func (ce *CalculationEngine) calculateTaxes(input TaxCalculationInput) TaxCalculationResult {
	ctx := newTaxComputationContext(input.PersonA, input.PersonB, input.Scenario, input.Year, input.IsRetired,
		input.Pensions[0], input.Pensions[1],
		input.SurvivorPensions[0], input.SurvivorPensions[1],
		input.TSPWithdrawals[0], input.TSPWithdrawals[1],
		input.SocialSecurity[0], input.SocialSecurity[1],
		input.WorkingIncome[0], input.WorkingIncome[1],
	)

	var result taxResult
	switch {
	case ctx.isTransitionYear():
		result = ce.calculateTransitionYearTaxes(ctx)
	case ctx.isRetirementYear():
		result = ce.calculateRetirementYearTaxes(ctx)
	default:
		result = ce.calculateWorkingYearTaxes(ctx)
	}

	return TaxCalculationResult{
		FederalTax:         result.federal,
		StateTax:           result.state,
		LocalTax:           result.local,
		FICATax:            result.fica,
		TaxableIncomeTotal: result.taxableTotal,
		StandardDeduction:  result.standardDeduction,
		FilingStatus:       result.filingStatus,
		Seniors:            result.seniors,
	}
}

type taxResult struct {
	federal           decimal.Decimal
	state             decimal.Decimal
	local             decimal.Decimal
	fica              decimal.Decimal
	taxableTotal      decimal.Decimal
	standardDeduction decimal.Decimal
	filingStatus      string
	seniors           int
}

type taxComputationContext struct {
	personA, personB *domain.Employee
	scenario         *domain.Scenario
	year             int
	projectionDate   time.Time
	filingStatus     string
	seniors          int
	personADeceased  bool
	personBDeceased  bool
	isRetired        bool
	pensionA         decimal.Decimal
	pensionB         decimal.Decimal
	survivorPensionA decimal.Decimal
	survivorPensionB decimal.Decimal
	tspWithdrawalA   decimal.Decimal
	tspWithdrawalB   decimal.Decimal
	socialSecurityA  decimal.Decimal
	socialSecurityB  decimal.Decimal
	workingIncomeA   decimal.Decimal
	workingIncomeB   decimal.Decimal
	currentSalaryA   decimal.Decimal
	currentSalaryB   decimal.Decimal
}

func newTaxComputationContext(
	personA, personB *domain.Employee,
	scenario *domain.Scenario,
	year int,
	isRetired bool,
	pensionA, pensionB,
	survivorPensionA, survivorPensionB,
	tspWithdrawalA, tspWithdrawalB,
	socialSecurityA, socialSecurityB decimal.Decimal,
	workingIncomeA, workingIncomeB decimal.Decimal,
) taxComputationContext {
	projectionDate := time.Date(ProjectionBaseYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
	ageA := personA.Age(projectionDate)
	ageB := personB.Age(projectionDate)

	filingStatus, seniors, personADeceased, personBDeceased := determineFilingStatusAndSeniors(scenario, personA, personB, year, ageA, ageB)

	return taxComputationContext{
		personA:          personA,
		personB:          personB,
		scenario:         scenario,
		year:             year,
		projectionDate:   projectionDate,
		filingStatus:     filingStatus,
		seniors:          seniors,
		personADeceased:  personADeceased,
		personBDeceased:  personBDeceased,
		isRetired:        isRetired,
		pensionA:         pensionA,
		pensionB:         pensionB,
		survivorPensionA: survivorPensionA,
		survivorPensionB: survivorPensionB,
		tspWithdrawalA:   tspWithdrawalA,
		tspWithdrawalB:   tspWithdrawalB,
		socialSecurityA:  socialSecurityA,
		socialSecurityB:  socialSecurityB,
		workingIncomeA:   workingIncomeA,
		workingIncomeB:   workingIncomeB,
		currentSalaryA:   personA.CurrentSalary,
		currentSalaryB:   personB.CurrentSalary,
	}
}

func determineFilingStatusAndSeniors(scenario *domain.Scenario, personA, personB *domain.Employee, year, ageA, ageB int) (string, int, bool, bool) {
	filingStatus := "mfj"
	seniors := 0
	if ageA >= 65 {
		seniors++
	}
	if ageB >= 65 {
		seniors++
	}

	const mortalityBufferYears = 5
	personADeathIndex, personBDeathIndex := deriveDeathYearIndexes(scenario, personA, personB, year+1+mortalityBufferYears)
	personADeceased := personADeathIndex != nil && year >= *personADeathIndex
	personBDeceased := personBDeathIndex != nil && year >= *personBDeathIndex

	if (personADeceased || personBDeceased) && !(personADeceased && personBDeceased) {
		if scenario != nil && scenario.Mortality != nil && scenario.Mortality.Assumptions != nil {
			switch scenario.Mortality.Assumptions.FilingStatusSwitch {
			case "immediate":
				filingStatus = "single"
				seniors = survivingSeniorCount(personADeceased, personBDeceased, ageA, ageB)
			case "next_year":
				deathYear := 0
				if personADeceased && personADeathIndex != nil {
					deathYear = *personADeathIndex
				}
				if personBDeceased && personBDeathIndex != nil {
					deathYear = *personBDeathIndex
				}
				if year > deathYear {
					filingStatus = "single"
					seniors = survivingSeniorCount(personADeceased, personBDeceased, ageA, ageB)
				}
			}
		}
	}

	return filingStatus, seniors, personADeceased, personBDeceased
}

func survivingSeniorCount(personADeceased, personBDeceased bool, ageA, ageB int) int {
	if personADeceased && personBDeceased {
		return 0
	}

	if !personADeceased && ageA >= 65 {
		return 1
	}
	if !personBDeceased && ageB >= 65 {
		return 1
	}
	return 0
}

func (ctx taxComputationContext) totalWorkingIncome() decimal.Decimal {
	return ctx.workingIncomeA.Add(ctx.workingIncomeB)
}

func (ctx taxComputationContext) totalRetirementIncome() decimal.Decimal {
	return ctx.pensionA.
		Add(ctx.pensionB).
		Add(ctx.survivorPensionA).
		Add(ctx.survivorPensionB).
		Add(ctx.tspWithdrawalA).
		Add(ctx.tspWithdrawalB)
}

func (ctx taxComputationContext) totalPensionWithSurvivor() decimal.Decimal {
	return ctx.pensionA.Add(ctx.pensionB).Add(ctx.survivorPensionA).Add(ctx.survivorPensionB)
}

func (ctx taxComputationContext) totalSocialSecurity() decimal.Decimal {
	return ctx.socialSecurityA.Add(ctx.socialSecurityB)
}

func (ctx taxComputationContext) totalTSPWithdrawals() decimal.Decimal {
	return ctx.tspWithdrawalA.Add(ctx.tspWithdrawalB)
}

func (ctx taxComputationContext) combinedCurrentSalary() decimal.Decimal {
	return ctx.currentSalaryA.Add(ctx.currentSalaryB)
}

func (ctx taxComputationContext) hasWorkingIncome() bool {
	return ctx.workingIncomeA.GreaterThan(decimal.Zero) || ctx.workingIncomeB.GreaterThan(decimal.Zero)
}

func (ctx taxComputationContext) hasRetirementIncome() bool {
	return ctx.pensionA.GreaterThan(decimal.Zero) ||
		ctx.pensionB.GreaterThan(decimal.Zero) ||
		ctx.survivorPensionA.GreaterThan(decimal.Zero) ||
		ctx.survivorPensionB.GreaterThan(decimal.Zero) ||
		ctx.tspWithdrawalA.GreaterThan(decimal.Zero) ||
		ctx.tspWithdrawalB.GreaterThan(decimal.Zero) ||
		ctx.socialSecurityA.GreaterThan(decimal.Zero) ||
		ctx.socialSecurityB.GreaterThan(decimal.Zero)
}

func (ctx taxComputationContext) isTransitionYear() bool {
	return ctx.hasWorkingIncome() && ctx.hasRetirementIncome()
}

func (ctx taxComputationContext) isRetirementYear() bool {
	return ctx.isRetired
}

func (ce *CalculationEngine) calculateTransitionYearTaxes(ctx taxComputationContext) taxResult {
	totalWorkingIncome := ctx.totalWorkingIncome()
	totalRetirementIncome := ctx.totalRetirementIncome()
	totalSS := ctx.totalSocialSecurity()

	provisional := ce.TaxCalc.SSTaxCalc.CalculateProvisionalIncome(totalRetirementIncome, decimal.Zero, totalSS)

	var taxableSS decimal.Decimal
	if ctx.filingStatus == "single" {
		taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecuritySingle(totalSS, provisional)
	} else {
		taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecurity(totalSS, provisional)
	}

	taxableIncome := domain.TaxableIncome{
		Salary:             totalWorkingIncome,
		FERSPension:        ctx.totalPensionWithSurvivor(),
		TSPWithdrawalsTrad: ctx.totalTSPWithdrawals(),
		TaxableSSBenefits:  taxableSS,
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         totalWorkingIncome,
		InterestIncome:     decimal.Zero,
	}

	federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(taxableIncome, ctx.filingStatus, ctx.seniors, ctx.year)
	stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(taxableIncome, false)
	localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(totalWorkingIncome, false)
	personAFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(ctx.workingIncomeA, totalWorkingIncome)
	personBFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(ctx.workingIncomeB, totalWorkingIncome)
	ficaTax := personAFICA.Add(personBFICA)
	standardDeduction := ce.standardDeductionFor(ctx.filingStatus, ctx.seniors, ctx.year)

	taxableTotal := taxableIncome.Salary.
		Add(taxableIncome.FERSPension).
		Add(taxableIncome.TSPWithdrawalsTrad).
		Add(taxableIncome.TaxableSSBenefits)

	return taxResult{
		federal:           federalTax,
		state:             stateTax,
		local:             localTax,
		fica:              ficaTax,
		taxableTotal:      taxableTotal,
		standardDeduction: standardDeduction,
		filingStatus:      ctx.filingStatus,
		seniors:           ctx.seniors,
	}
}

func (ce *CalculationEngine) calculateRetirementYearTaxes(ctx taxComputationContext) taxResult {
	otherIncome := ctx.totalRetirementIncome()
	totalSS := ctx.totalSocialSecurity()

	provisional := ce.TaxCalc.SSTaxCalc.CalculateProvisionalIncome(otherIncome, decimal.Zero, totalSS)

	var taxableSS decimal.Decimal
	if ctx.filingStatus == "single" {
		taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecuritySingle(totalSS, provisional)
	} else {
		taxableSS = ce.TaxCalc.SSTaxCalc.CalculateTaxableSocialSecurity(totalSS, provisional)
	}

	taxableIncome := domain.TaxableIncome{
		Salary:             decimal.Zero,
		FERSPension:        ctx.totalPensionWithSurvivor(),
		TSPWithdrawalsTrad: ctx.totalTSPWithdrawals(),
		TaxableSSBenefits:  taxableSS,
		OtherTaxableIncome: decimal.Zero,
		WageIncome:         decimal.Zero,
		InterestIncome:     decimal.Zero,
	}

	federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(taxableIncome, ctx.filingStatus, ctx.seniors, ctx.year)
	stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(taxableIncome, true)
	localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(decimal.Zero, true)
	standardDeduction := ce.standardDeductionFor(ctx.filingStatus, ctx.seniors, ctx.year)

	taxableTotal := taxableIncome.Salary.
		Add(taxableIncome.FERSPension).
		Add(taxableIncome.TSPWithdrawalsTrad).
		Add(taxableIncome.TaxableSSBenefits)

	return taxResult{
		federal:           federalTax,
		state:             stateTax,
		local:             localTax,
		fica:              decimal.Zero,
		taxableTotal:      taxableTotal,
		standardDeduction: standardDeduction,
		filingStatus:      ctx.filingStatus,
		seniors:           ctx.seniors,
	}
}

func (ce *CalculationEngine) calculateWorkingYearTaxes(ctx taxComputationContext) taxResult {
	currentTaxableIncome := CalculateCurrentTaxableIncome(ctx.currentSalaryA, ctx.currentSalaryB)

	federalTax := ce.TaxCalc.calculateFederalTaxWithStatus(currentTaxableIncome, ctx.filingStatus, ctx.seniors, ctx.year)
	stateTax := ce.TaxCalc.StateTaxCalc.CalculateTax(currentTaxableIncome, false)
	localTax := ce.TaxCalc.LocalTaxCalc.CalculateEIT(ctx.combinedCurrentSalary(), false)

	totalCurrentSalary := ctx.currentSalaryA.Add(ctx.currentSalaryB)
	personAFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(ctx.currentSalaryA, totalCurrentSalary)
	personBFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(ctx.currentSalaryB, totalCurrentSalary)
	ficaTax := personAFICA.Add(personBFICA)

	standardDeduction := ce.standardDeductionFor(ctx.filingStatus, ctx.seniors, ctx.year)

	return taxResult{
		federal:           federalTax,
		state:             stateTax,
		local:             localTax,
		fica:              ficaTax,
		taxableTotal:      currentTaxableIncome.Salary,
		standardDeduction: standardDeduction,
		filingStatus:      ctx.filingStatus,
		seniors:           ctx.seniors,
	}
}

func (ce *CalculationEngine) standardDeductionFor(filingStatus string, seniors, projectionYear int) decimal.Decimal {
	standardDeduction := ce.TaxCalc.FederalTaxCalc.StandardDeduction
	if filingStatus == "single" {
		standardDeduction = ce.TaxCalc.FederalTaxCalc.StandardDeductionSingle
	}
	for i := 0; i < seniors; i++ {
		standardDeduction = standardDeduction.Add(ce.TaxCalc.FederalTaxCalc.AdditionalStdDed)
	}

	// Apply inflation adjustment
	if projectionYear > 0 && !ce.TaxCalc.InflationRate.IsZero() {
		inflationAdjustment := decimal.NewFromFloat(1.0).Add(ce.TaxCalc.InflationRate).Pow(decimal.NewFromInt(int64(projectionYear)))
		standardDeduction = standardDeduction.Mul(inflationAdjustment)
	}

	return standardDeduction
}

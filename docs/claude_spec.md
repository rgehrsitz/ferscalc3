# Comprehensive FERS Retirement Planning Calculator - Technical Specification

## Executive Summary

This specification combines comprehensive research on Federal Employee Retirement System regulations with detailed implementation requirements for building a sophisticated retirement planning calculator in Go. The tool will provide accurate "apples-to-apples" comparisons between current net income and multiple retirement scenarios, incorporating all recent regulatory changes, tax implications, and federal-specific benefits.

## Core Objectives

1. **Accurate Net Income Comparison**: Reflect all relevant taxes (federal, state, local) and deductions in current vs. retirement scenarios
2. **Multiple Scenario Support**: Compare at least two retirement scenarios simultaneously with current baseline
3. **Recent Regulatory Compliance**: Incorporate 2025 WEP/GPO repeal and SECURE 2.0 changes
4. **User-Adjustable Parameters**: Enable customization of inflation, COLAs, TSP returns, and withdrawal strategies
5. **Long-Term Projection**: Provide 25+ year projections emphasizing immediate post-retirement years
6. **Monte Carlo Capability**: Optional probabilistic analysis using historical market data

## Recent Regulatory Changes Requiring Implementation

### Critical 2025 Updates Affecting Federal Retirement Planning

**Social Security Fairness Act Implementation**: The January 2025 repeal of WEP/GPO affects 3.2 million beneficiaries with average increases of $7,300 annually. The calculator must remove previous Social Security reduction penalties for federal retirees.

**SECURE 2.0 Act Changes**: Required Minimum Distributions now begin at age 73 (or 75 for those born 1960+), with only traditional TSP balances counting toward RMDs. Roth TSP funds are exempt from RMDs during owner's lifetime.

**2025 Contribution Limits**: TSP limits reach $23,500 regular plus $7,500 catch-up (50+), with new "super catch-up" $11,250 for ages 60-63.

## Input Data Structure and Configuration

The calculator accepts structured JSON/YAML input enabling comprehensive scenario modeling.

### Personal Profile Structure

```yaml
personal_details:
    person_a:
    birth_date: "1963-06-15"
    hire_date: "1985-03-20"
    employment_type: "federal"
    current_salary: 95000
    high_3_salary: 93000  # Optional, calculated if not provided
    tsp_balance_traditional: 450000
    tsp_balance_roth: 50000
    tsp_contribution_percent: 15
    ss_benefit_fra: 2400  # Monthly at Full Retirement Age
    ss_benefit_62: 1680   # Monthly at age 62
    ss_benefit_70: 2976   # Monthly at age 70
  
    person_b:
    birth_date: "1965-08-22"
    hire_date: "1988-07-10"
    employment_type: "federal"
    current_salary: 87000
    high_3_salary: 85000
    tsp_balance_traditional: 380000
    tsp_balance_roth: 45000
    tsp_contribution_percent: 12
    ss_benefit_fra: 2200
    ss_benefit_62: 1540
    ss_benefit_70: 2728

shared_details:
  location:
    state: "PA"
    county: "Bucks"
    municipality: "Upper Makefield Township"
  fehb_plan: "AETNA High Option"
  fehb_premium_per_pay_period: 875 # This is the per-pay-period amount, which will be annualized by multiplying by 26.
  survivor_benefit_election: 0  # 0% survivor benefits elected

Set `employment_type: "non-federal"` for civilian spouses; the engine will skip FERS/TSP accrual logic but still track wages, Social Security, and taxes. When a non-federal spouse has a constant pension or annuity, add a `fixed_retirement_income` block either under the employee or under an individual scenario to describe the annual amount and optional COLA override:

```yaml
fixed_retirement_income:
    annual_amount: 24000
    cola_rate: 0.02  # Optional; defaults to general COLA when omitted
```

### Scenario Configuration

```yaml
global_assumptions:
  inflation_rate: 0.025
  fehb_premium_inflation: 0.065  # Healthcare inflation higher than general
  tsp_return_pre_retirement: 0.055
  tsp_return_post_retirement: 0.045
  cola_method: "fers_rules"  # or "fixed_percent"
  projection_years: 25

scenarios:
  - name: "Early Retirement 2025"
    person_a:
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    person_b:
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
  
    - name: "Delayed Retirement 2028"
        person_a:
            retirement_date: "2028-12-31"
            ss_start_age: 67
            tsp_withdrawal_strategy: "need_based"
            tsp_withdrawal_target: 3000  # Monthly target until SS
        person_b:
            retirement_date: "2028-12-31"
            ss_start_age: 62
            tsp_withdrawal_strategy: "4_percent_rule"
```

## Income Stream Calculations

### 1. FERS Pension (Annuity) Calculations

#### Core Pension Formula

```text
Annual Pension = High-3 Average Salary × Years of Service × Multiplier
```

#### Multiplier Logic

- Standard: 1.0% per year of service
- Enhanced: 1.1% per year if retiring at age 62+ with 20+ years service
- Implementation must check both age at retirement and years of service

#### Key Implementation Requirements

```go
type PensionCalculation struct {
    High3Salary      decimal.Decimal
    ServiceYears     int
    ServiceMonths    int
    RetirementAge    int
    Multiplier       decimal.Decimal
    AnnualPension    decimal.Decimal
    SurvivorElection decimal.Decimal
}

func CalculateFERSPension(employee Employee, retirementDate time.Time) PensionCalculation {
    // Calculate years/months of service (floor to whole months)
    serviceTime := calculateServiceTime(employee.HireDate, retirementDate)
    
    // Determine multiplier based on age and service
    multiplier := determineMultiplier(retirementAge, serviceTime.Years)
    
    // Apply survivor benefit reduction if elected (0% in this case)
    pension := high3 * serviceYears * multiplier * (1 - survivorReduction)
    
    return PensionCalculation{...}
}
```

#### FERS Special Retirement Supplement (SRS)

- Applies if retiring before age 62 with immediate annuity
- Formula: `SS_Benefit_at_62 × (FERS_Service_Years ÷ 40)`
- Subject to earnings test after Minimum Retirement Age ($23,400 limit for 2025)
- Terminates at age 62 regardless of SS claiming decision

#### FERS COLA Implementation

```go
func ApplyFERSCOLA(pensionAmount decimal.Decimal, cpiIncrease decimal.Decimal, retireeAge int) decimal.Decimal {
    // FERS COLAs only apply after age 62
    if retireeAge < 62 {
        return pensionAmount
    }
    
    var colaRate decimal.Decimal
    if cpiIncrease.LessThanOrEqual(decimal.NewFromFloat(0.02)) {
        colaRate = cpiIncrease  // Full CPI increase
    } else if cpiIncrease.LessThanOrEqual(decimal.NewFromFloat(0.03)) {
        colaRate = decimal.NewFromFloat(0.02)  // Capped at 2%
    } else {
        colaRate = cpiIncrease.Sub(decimal.NewFromFloat(0.01))  // CPI minus 1%
    }
    
    return pensionAmount.Mul(decimal.NewFromFloat(1).Add(colaRate))
}
```

### 2. TSP Modeling with Historical Performance Integration

#### Historical Data for Modeling

- C Fund (S&P 500): Long-term average 12.29% (1988-2020), 2024 return 24.96%
- F Fund (Bond Index): Long-term average 6.29%, 2024 return 1.33%
- G Fund (Government Securities): Long-term average 4.7%, stable but low returns
- S Fund (Small Cap): 2024 return 16.93%, higher volatility
- I Fund (International): More volatile, requires careful modeling

#### Withdrawal Strategy Implementation

```go
type TSPWithdrawalStrategy interface {
    CalculateWithdrawal(balance decimal.Decimal, year int, targetIncome decimal.Decimal) decimal.Decimal
}

type FourPercentRule struct {
    InitialWithdrawal decimal.Decimal
    InflationRate     decimal.Decimal
}

func (fpr *FourPercentRule) CalculateWithdrawal(balance decimal.Decimal, year int, targetIncome decimal.Decimal) decimal.Decimal {
    if year == 1 {
        fpr.InitialWithdrawal = balance.Mul(decimal.NewFromFloat(0.04))
        return fpr.InitialWithdrawal
    }
    
    // Inflate previous year's withdrawal
    inflationFactor := decimal.NewFromFloat(1).Add(fpr.InflationRate)
    return fpr.InitialWithdrawal.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year-1))))
}

type NeedBasedWithdrawal struct {
    TargetMonthlyIncome decimal.Decimal
}

func (nbw *NeedBasedWithdrawal) CalculateWithdrawal(balance decimal.Decimal, year int, targetIncome decimal.Decimal) decimal.Decimal {
    // Calculate gap between other income sources and target
    annualTarget := nbw.TargetMonthlyIncome.Mul(decimal.NewFromInt(12))
    return annualTarget.Sub(targetIncome)
}
```

#### Required Minimum Distribution Compliance

```go
type RMDCalculator struct {
    BirthYear int
}

func (rmd *RMDCalculator) GetRMDAge() int {
    if rmd.BirthYear >= 1960 {
        return 75  // SECURE 2.0 change
    } else if rmd.BirthYear >= 1951 {
        return 73
    }
    return 72
}

func (rmd *RMDCalculator) CalculateRMD(traditionalBalance decimal.Decimal, age int) decimal.Decimal {
    if age < rmd.GetRMDAge() {
        return decimal.Zero
    }
    
    // IRS Uniform Lifetime Table (simplified)
    distributionPeriods := map[int]decimal.Decimal{
        73: decimal.NewFromFloat(26.5),
        74: decimal.NewFromFloat(25.5),
        75: decimal.NewFromFloat(24.6),
        // ... continue with full table
    }
    
    if period, exists := distributionPeriods[age]; exists {
        return traditionalBalance.Div(period)
    }
    
    return decimal.Zero
}
```

#### TSP Growth and Balance Tracking

```go
type TSPProjection struct {
    Year              int
    BeginningBalance  decimal.Decimal
    Growth           decimal.Decimal
    Withdrawal       decimal.Decimal
    RMD              decimal.Decimal
    EndingBalance    decimal.Decimal
    TraditionalPct   decimal.Decimal
    RothPct          decimal.Decimal
}

func ProjectTSP(initialBalance decimal.Decimal, strategy TSPWithdrawalStrategy, returnRate decimal.Decimal, years int) []TSPProjection {
    projections := make([]TSPProjection, years)
    currentBalance := initialBalance
    
    for year := 1; year <= years; year++ {
        growth := currentBalance.Mul(returnRate)
        withdrawal := strategy.CalculateWithdrawal(currentBalance, year, decimal.Zero)
        
        // Ensure withdrawal doesn't exceed balance
        if withdrawal.GreaterThan(currentBalance.Add(growth)) {
            withdrawal = currentBalance.Add(growth)
        }
        
        endingBalance := currentBalance.Add(growth).Sub(withdrawal)
        
        projections[year-1] = TSPProjection{
            Year:             year,
            BeginningBalance: currentBalance,
            Growth:          growth,
            Withdrawal:      withdrawal,
            EndingBalance:   endingBalance,
        }
        
        currentBalance = endingBalance
    }
    
    return projections
}
```

### 3. Social Security Optimization with 2025 Updates

#### Post-WEP/GPO Repeal Benefits

With the 2025 Social Security Fairness Act, federal employees no longer face benefit reductions. The calculator must implement full Social Security benefits without WEP/GPO penalties.

#### Benefit Calculation by Claiming Age

```go
type SocialSecurityCalculator struct {
    BirthYear           int
    FullRetirementAge   int
    BenefitAtFRA       decimal.Decimal
}

func (ssc *SocialSecurityCalculator) CalculateBenefitAtAge(claimingAge int) decimal.Decimal {
    if claimingAge < 62 {
        return decimal.Zero
    }
    
    if claimingAge < ssc.FullRetirementAge {
        // Early retirement reduction
        monthsEarly := (ssc.FullRetirementAge - claimingAge) * 12
        var reductionRate decimal.Decimal
        
        if monthsEarly <= 36 {
            // 5/9 of 1% per month for first 36 months
            reductionRate = decimal.NewFromFloat(5.0/9.0/100.0).Mul(decimal.NewFromInt(int64(monthsEarly)))
        } else {
            // 5/9 of 1% for first 36 months, 5/12 of 1% for additional months
            firstReduction := decimal.NewFromFloat(5.0/9.0/100.0).Mul(decimal.NewFromInt(36))
            additionalMonths := monthsEarly - 36
            additionalReduction := decimal.NewFromFloat(5.0/12.0/100.0).Mul(decimal.NewFromInt(int64(additionalMonths)))
            reductionRate = firstReduction.Add(additionalReduction)
        }
        
        return ssc.BenefitAtFRA.Mul(decimal.NewFromFloat(1).Sub(reductionRate))
    }
    
    if claimingAge > ssc.FullRetirementAge {
        // Delayed retirement credits: 8% per year (2/3% per month)
        monthsDelayed := (claimingAge - ssc.FullRetirementAge) * 12
        if monthsDelayed > 48 { // Cap at age 70
            monthsDelayed = 48
        }
        
        delayCredit := decimal.NewFromFloat(2.0/3.0/100.0).Mul(decimal.NewFromInt(int64(monthsDelayed)))
        return ssc.BenefitAtFRA.Mul(decimal.NewFromFloat(1).Add(delayCredit))
    }
    
    return ssc.BenefitAtFRA // At Full Retirement Age
}
```

#### Social Security Taxation (Critical for Net Income Calculation)

```go
type SSTaxCalculator struct{}

func (sstc *SSTaxCalculator) CalculateTaxablePortion(ssAmount decimal.Decimal, otherIncome decimal.Decimal, filingJoint bool) decimal.Decimal {
    combinedIncome := otherIncome.Add(ssAmount.Mul(decimal.NewFromFloat(0.5)))
    
    var threshold1, threshold2 decimal.Decimal
    if filingJoint {
        threshold1 = decimal.NewFromInt(32000)
        threshold2 = decimal.NewFromInt(44000)
    } else {
        threshold1 = decimal.NewFromInt(25000)
        threshold2 = decimal.NewFromInt(34000)
    }
    
    if combinedIncome.LessThanOrEqual(threshold1) {
        return decimal.Zero
    }
    
    if combinedIncome.LessThanOrEqual(threshold2) {
        // Up to 50% taxable
        excessOverThreshold1 := combinedIncome.Sub(threshold1)
        halfSS := ssAmount.Mul(decimal.NewFromFloat(0.5))
        return decimal.Min(excessOverThreshold1.Mul(decimal.NewFromFloat(0.5)), halfSS)
    }
    
    // Up to 85% taxable
    excessOverThreshold2 := combinedIncome.Sub(threshold2)
    fromFirstTier := threshold2.Sub(threshold1).Mul(decimal.NewFromFloat(0.5))
    fromSecondTier := excessOverThreshold2.Mul(decimal.NewFromFloat(0.85))
    totalTaxable := fromFirstTier.Add(fromSecondTier)
    
    maxTaxable := ssAmount.Mul(decimal.NewFromFloat(0.85))
    return decimal.Min(totalTaxable, maxTaxable)
}
```

### 4. Comprehensive Tax Modeling

#### Federal Income Tax Implementation

```go
type TaxBracket struct {
    Min  decimal.Decimal
    Max  decimal.Decimal
    Rate decimal.Decimal
}

type FederalTaxCalculator struct {
    Year              int
    StandardDeduction decimal.Decimal
    Brackets          []TaxBracket
    AdditionalStdDed  decimal.Decimal // For age 65+
}

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
            {decimal.NewFromInt(731201), decimal.MaxValue, decimal.NewFromFloat(0.37)},
        },
    }
}

func (ftc *FederalTaxCalculator) CalculateTax(grossIncome decimal.Decimal, age1, age2 int) decimal.Decimal {
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
        
        taxableInBracket := decimal.Min(taxableIncome, bracket.Max).Sub(bracket.Min)
        if taxableInBracket.GreaterThan(decimal.Zero) {
            totalTax = totalTax.Add(taxableInBracket.Mul(bracket.Rate))
        }
    }
    
    return totalTax
}
```

#### Pennsylvania State Tax (Critical Retirement Benefit)

```go
type PennsylvaniaTaxCalculator struct{}

func (ptc *PennsylvaniaTaxCalculator) CalculateTax(income TaxableIncome, isRetired bool) decimal.Decimal {
    paRate := decimal.NewFromFloat(0.0307)
    
    if isRetired {
        // PA exempts retirement income: pensions, TSP, Social Security
        taxablePA := income.WageIncome.Add(income.InterestIncome).Add(income.OtherTaxableIncome)
        return taxablePA.Mul(paRate)
    }
    
    // While working: tax wages at 3.07%
    return income.WageIncome.Mul(paRate)
}
```

#### Upper Makefield Township Local Tax

```go
type UpperMakefieldEITCalculator struct{}

func (ume *UpperMakefieldEITCalculator) CalculateEIT(wageIncome decimal.Decimal, isRetired bool) decimal.Decimal {
    if isRetired {
        return decimal.Zero // EIT only applies to earned income
    }
    
    eitRate := decimal.NewFromFloat(0.01) // 1% on earned income
    return wageIncome.Mul(eitRate)
}
```

#### FICA Calculations (Current Employment Only)

```go
type FICACalculator struct {
    Year           int
    SSWageBase     decimal.Decimal
    SSRate         decimal.Decimal
    MedicareRate   decimal.Decimal
    AdditionalRate decimal.Decimal
    HighIncomeThreshold decimal.Decimal
}

func NewFICACalculator2025() *FICACalculator {
    return &FICACalculator{
        Year:                2025,
        SSWageBase:          decimal.NewFromInt(168600), // 2025 estimated
        SSRate:              decimal.NewFromFloat(0.062),
        MedicareRate:        decimal.NewFromFloat(0.0145),
        AdditionalRate:      decimal.NewFromFloat(0.009),
        HighIncomeThreshold: decimal.NewFromInt(250000), // MFJ
    }
}

func (fc *FICACalculator) CalculateFICA(wages decimal.Decimal, totalHouseholdWages decimal.Decimal) decimal.Decimal {
    // Social Security tax (capped)
    ssWages := decimal.Min(wages, fc.SSWageBase)
    ssTax := ssWages.Mul(fc.SSRate)
    
    // Medicare tax (no cap)
    medicareTax := wages.Mul(fc.MedicareRate)
    
    // Additional Medicare tax for high earners
    var additionalMedicare decimal.Decimal
    if totalHouseholdWages.GreaterThan(fc.HighIncomeThreshold) {
        excessWages := totalHouseholdWages.Sub(fc.HighIncomeThreshold)
        applicableExcess := decimal.Min(excessWages, wages)
        additionalMedicare = applicableExcess.Mul(fc.AdditionalRate)
    }
    
    return ssTax.Add(medicareTax).Add(additionalMedicare)
}
```

### 5. Healthcare Cost Modeling (FEHB/Medicare Integration)

#### FEHB Premium Calculations in Retirement

```go
type FEHBCalculator struct {
    PlanName            string
    MonthlyPremium      decimal.Decimal
    PremiumInflationRate decimal.Decimal
    IsPretaxWhileWorking bool
}

func (fehb *FEHBCalculator) CalculateRetirementCost(year int, isMedicare bool) decimal.Decimal {
    // Inflate premium from base year
    inflationFactor := decimal.NewFromFloat(1).Add(fehb.PremiumInflationRate)
    adjustedPremium := fehb.MonthlyPremium.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year-1))))
    
    // FEHB rates same for retirees as employees, but lose pre-tax benefit
    annualPremium := adjustedPremium.Mul(decimal.NewFromInt(12))
    
    if isMedicare {
        // Optional: Medicare Part B coordination
        // FEHB remains primary, but may reduce some costs
        return annualPremium
    }
    
    return annualPremium
}
```

#### Medicare Integration at Age 65

```go
type MedicareCalculator struct {
    PartBPremium     decimal.Decimal
    IRMAATiers       []IRMAAThreshold
}

type IRMAAThreshold struct {
    IncomeMin    decimal.Decimal
    IncomeMax    decimal.Decimal
    Surcharge    decimal.Decimal
}

func (mc *MedicareCalculator) CalculatePartBCost(modifiedAGI decimal.Decimal, filingJoint bool) decimal.Decimal {
    basePremium := mc.PartBPremium.Mul(decimal.NewFromInt(12))
    
    // Apply IRMAA surcharges for high-income retirees
    for _, tier := range mc.IRMAATiers {
        if modifiedAGI.GreaterThanOrEqual(tier.IncomeMin) && 
           (tier.IncomeMax.IsZero() || modifiedAGI.LessThan(tier.IncomeMax)) {
            return basePremium.Add(tier.Surcharge.Mul(decimal.NewFromInt(12)))
        }
    }
    
    return basePremium
}
```

### 6. Monte Carlo Simulation Implementation

#### Historical Data Integration

```go
type HistoricalData struct {
    Year        int
    StockReturn decimal.Decimal
    BondReturn  decimal.Decimal
    Inflation   decimal.Decimal
}

type MonteCarloSimulator struct {
    HistoricalData    []HistoricalData
    NumSimulations    int
    ProjectionYears   int
    RetirementProfile RetirementScenario
}

func (mcs *MonteCarloSimulator) RunSimulation() MonteCarloResults {
    results := make([]SimulationOutcome, mcs.NumSimulations)
    
    for sim := 0; sim < mcs.NumSimulations; sim++ {
        outcome := mcs.runSingleSimulation()
        results[sim] = outcome
    }
    
    return MonteCarloResults{
        Simulations:         results,
        SuccessRate:        mcs.calculateSuccessRate(results),
        MedianEndingBalance: mcs.calculateMedian(results),
        PercentileRanges:   mcs.calculatePercentiles(results),
    }
}

func (mcs *MonteCarloSimulator) runSingleSimulation() SimulationOutcome {
    currentBalance := mcs.RetirementProfile.InitialTSPBalance
    var outcomes []YearOutcome
    
    for year := 1; year <= mcs.ProjectionYears; year++ {
        // Randomly sample historical data or generate correlated returns
        marketData := mcs.sampleMarketConditions()
        
        // Apply market returns
        growth := currentBalance.Mul(marketData.Return)
        
        // Calculate withdrawal (considering market conditions)
        withdrawal := mcs.calculateDynamicWithdrawal(currentBalance, year, marketData)
        
        currentBalance = currentBalance.Add(growth).Sub(withdrawal)
        
        outcomes = append(outcomes, YearOutcome{
            Year:      year,
            Balance:   currentBalance,
            Withdrawal: withdrawal,
            Return:    marketData.Return,
        })
        
        if currentBalance.LessThanOrEqual(decimal.Zero) {
            break // Portfolio depleted
        }
    }
    
    return SimulationOutcome{
        YearOutcomes:   outcomes,
        PortfolioLasted: len(outcomes),
        EndingBalance:  currentBalance,
    }
}
```

### 7. Output and Visualization Framework

#### Comprehensive Annual Cash Flow Table

```go
type AnnualCashFlow struct {
    Year                int
    Age1                int
    Age2                int
    PensionPersonA       decimal.Decimal
    PensionPersonB       decimal.Decimal
    TSPWithdrawalPersonA decimal.Decimal
    TSPWithdrawalPersonB decimal.Decimal
    SSBenefitPersonA     decimal.Decimal
    SSBenefitPersonB     decimal.Decimal
    FERSSupplementPersonA decimal.Decimal
    FERSSupplementPersonB  decimal.Decimal
    TotalGrossIncome    decimal.Decimal
    FederalTax          decimal.Decimal
    StateTax            decimal.Decimal
    LocalTax            decimal.Decimal
    FEHBPremium         decimal.Decimal
    MedicarePremium     decimal.Decimal
    NetIncome           decimal.Decimal
    TSPBalancePersonA    decimal.Decimal
    TSPBalancePersonB    decimal.Decimal
}

func GenerateAnnualProjection(scenario RetirementScenario, years int) []AnnualCashFlow {
    projection := make([]AnnualCashFlow, years)
    
    for year := 1; year <= years; year++ {
        cf := AnnualCashFlow{Year: year}
        
        // Calculate each income component
    cf.PensionPersonA = calculatePensionForYear(scenario.PersonA, year)
    cf.PensionPersonB = calculatePensionForYear(scenario.PersonB, year)
        
        // ... (implement other calculations)
        
        // Calculate net income
    totalGross := cf.PensionPersonA.Add(cf.PensionPersonB).Add(cf.TSPWithdrawalPersonA).
             Add(cf.TSPWithdrawalPersonB).Add(cf.SSBenefitPersonA).Add(cf.SSBenefitPersonB)
        
        cf.TotalGrossIncome = totalGross
        
        // Calculate taxes and deductions
        cf.FederalTax = calculateFederalTax(totalGross, cf.Age1, cf.Age2)
        cf.StateTax = calculateStateTax(totalGross, true) // Retired
        cf.LocalTax = decimal.Zero // No EIT on retirement income
        
        cf.NetIncome = cf.TotalGrossIncome.Sub(cf.FederalTax).Sub(cf.StateTax).
                      Sub(cf.LocalTax).Sub(cf.FEHBPremium).Sub(cf.MedicarePremium)
        
        projection[year-1] = cf
    }
    
    return projection
}
```

#### Scenario Comparison and Summary

```go
type ScenarioComparison struct {
    BaselineNetIncome    decimal.Decimal
    Scenarios           []ScenarioSummary
    ImmediateImpact     ImpactAnalysis
    LongTermProjection  LongTermAnalysis
}

type ScenarioSummary struct {
    Name                string
    FirstYearNetIncome  decimal.Decimal
    Year5NetIncome      decimal.Decimal
    TotalLifetimeIncome decimal.Decimal
    TSPLongevity        int
    SuccessRate         decimal.Decimal // From Monte Carlo
}

type ImpactAnalysis struct {
    CurrentToFirstYear   IncomeChange
    CurrentToSteadyState IncomeChange
    RecommendedScenario  string
    KeyConsiderations   []string
}

func GenerateScenarioComparison(baseline AnnualCashFlow, scenarios []RetirementScenario) ScenarioComparison {
    comparison := ScenarioComparison{
        BaselineNetIncome: baseline.NetIncome,
        Scenarios:        make([]ScenarioSummary, len(scenarios)),
    }
    
    for i, scenario := range scenarios {
        projection := GenerateAnnualProjection(scenario, 25)
        
        summary := ScenarioSummary{
            Name:               scenario.Name,
            FirstYearNetIncome: projection[0].NetIncome,
            Year5NetIncome:     projection[4].NetIncome,
        }
        
        // Calculate total lifetime income (present value)
        var totalPV decimal.Decimal
        for _, year := range projection {
            discountFactor := decimal.NewFromFloat(1.03).Pow(decimal.NewFromInt(int64(year.Year)))
            totalPV = totalPV.Add(year.NetIncome.Div(discountFactor))
        }
        summary.TotalLifetimeIncome = totalPV
        
        // Determine TSP longevity
        for j, year := range projection {
        if year.TSPBalancePersonA.Add(year.TSPBalancePersonB).LessThanOrEqual(decimal.Zero) {
                summary.TSPLongevity = j + 1
                break
            }
        }
        if summary.TSPLongevity == 0 {
            summary.TSPLongevity = 25 // Lasted full projection
        }
        
        comparison.Scenarios[i] = summary
    }
    
    // Generate impact analysis
    comparison.ImmediateImpact = analyzeImmediateImpact(baseline, comparison.Scenarios)
    
    return comparison
}
```

### 8. Go-Specific Implementation Architecture

#### Recommended Project Structure

```text
fers-retirement-calculator/
├── cmd/
│   └── cli/
│       └── main.go                 # CLI entry point
├── internal/
│   ├── domain/
│   │   ├── employee.go            # Core entities
│   │   ├── scenario.go
│   │   └── projection.go
│   ├── calculation/
│   │   ├── fers.go                # FERS pension calculations
│   │   ├── tsp.go                 # TSP modeling
│   │   ├── socialsecurity.go      # SS calculations
│   │   └── taxes.go               # Tax calculations
│   ├── simulation/
│   │   ├── montecarlo.go          # Monte Carlo implementation
│   │   └── historical.go          # Historical data management
│   ├── output/
│   │   ├── report.go              # Report generation
│   │   ├── charts.go              # Chart generation
│   │   └── export.go              # Export formats
│   └── config/
│       ├── input.go               # Input parsing
│       └── validation.go          # Input validation
├── pkg/
│   ├── decimal/                   # Decimal wrapper utilities
│   └── dateutil/                  # Date calculation utilities
├── data/
│   ├── historical/                # Historical market data
│   ├── tax-tables/                # Tax bracket data
│   └── defaults/                  # Default assumptions
├── web/                           # Optional web interface
├── test/
│   ├── testdata/                  # Test scenarios
│   └── integration/               # Integration tests
└── docs/
    ├── examples/                  # Example input files
    └── calculations/              # Calculation documentation
```

#### Core Dependencies

```go
// go.mod
module github.com/yourorg/fers-retirement-calculator

go 1.21

require (
    github.com/govalues/decimal v0.1.29     // High-precision decimal math
    github.com/spf13/cobra v1.8.0           // CLI framework
    github.com/spf13/viper v1.18.2          // Configuration management
    gopkg.in/yaml.v3 v3.0.1                 // YAML parsing
    github.com/go-echarts/go-echarts/v2 v2.3.3  // Chart generation
    github.com/stretchr/testify v1.8.4      // Testing framework
)
```

#### Financial Precision Implementation

```go
package decimal

import (
    "github.com/govalues/decimal"
)

// Wrapper for financial calculations with proper rounding
type Money struct {
    decimal.Decimal
}

func NewMoney(value float64) Money {
    return Money{decimal.NewFromFloat(value)}
}

func (m Money) Round() Money {
    // Always round to cents using banker's rounding
    return Money{m.Decimal.Round(2)}
}

func (m Money) Annual() Money {
    return Money{m.Decimal.Mul(decimal.NewFromInt(12))}
}

func (m Money) Monthly() Money {
    return Money{m.Decimal.Div(decimal.NewFromInt(12))}
}

// Tax-aware calculations
func (m Money) ApplyTaxRate(rate decimal.Decimal) Money {
    tax := m.Decimal.Mul(rate)
    return Money{m.Decimal.Sub(tax)}
}
```

#### CLI Interface Design

```go
package main

import (
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "fers-calc",
    Short: "FERS Retirement Calculator",
    Long:  "Comprehensive retirement planning calculator for federal employees",
}

var calculateCmd = &cobra.Command{
    Use:   "calculate [input-file]",
    Short: "Calculate retirement scenarios",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        inputFile := args[0]
        
        // Parse input
        config, err := config.LoadFromFile(inputFile)
        if err != nil {
            log.Fatal(err)
        }
        
        // Validate input
        if err := config.Validate(); err != nil {
            log.Fatal(err)
        }
        
        // Run calculations
        results := calculation.RunScenarios(config)
        
        // Generate output
        outputFormat, _ := cmd.Flags().GetString("format")
        output.GenerateReport(results, outputFormat)
    },
}

func init() {
    calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, html, pdf)")
    calculateCmd.Flags().BoolP("monte-carlo", "m", false, "Run Monte Carlo simulation")
    calculateCmd.Flags().IntP("simulations", "s", 1000, "Number of Monte Carlo simulations")
    
    rootCmd.AddCommand(calculateCmd)
}
```

### 9. Testing Strategy and Validation

#### Comprehensive Test Coverage

```go
package calculation_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/govalues/decimal"
)

func TestFERSPensionCalculation(t *testing.T) {
    tests := []struct {
        name             string
        high3Salary      decimal.Decimal
        serviceYears     int
        retirementAge    int
        expectedPension  decimal.Decimal
        expectedMultiplier decimal.Decimal
    }{
        {
            name:            "Standard multiplier at 60",
            high3Salary:     decimal.NewFromInt(95000),
            serviceYears:    30,
            retirementAge:   60,
            expectedPension: decimal.NewFromInt(28500), // 95000 * 30 * 0.01
            expectedMultiplier: decimal.NewFromFloat(0.01),
        },
        {
            name:            "Enhanced multiplier at 62 with 20+ years",
            high3Salary:     decimal.NewFromInt(95000),
            serviceYears:    30,
            retirementAge:   62,
            expectedPension: decimal.NewFromInt(31350), // 95000 * 30 * 0.011
            expectedMultiplier: decimal.NewFromFloat(0.011),
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            employee := Employee{
                High3Salary:   tt.high3Salary,
                ServiceYears:  tt.serviceYears,
                RetirementAge: tt.retirementAge,
            }
            
            pension := CalculateFERSPension(employee)
            
            assert.Equal(t, tt.expectedMultiplier, pension.Multiplier)
            assert.True(t, pension.AnnualPension.Equal(tt.expectedPension),
                "Expected %s, got %s", tt.expectedPension, pension.AnnualPension)
        })
    }
}

// Integration test with real scenario data
func TestCompleteScenarioCalculation(t *testing.T) {
    // Load test scenario from YAML
    scenario := loadTestScenario("testdata/complete_scenario.yaml")
    
    // Run full calculation
    results := calculation.RunScenarios(scenario)
    
    // Validate key metrics
    assert.Len(t, results.Scenarios, 2)
    assert.True(t, results.Scenarios[0].FirstYearNetIncome.GreaterThan(decimal.Zero))
    
    // Validate tax calculations
    for _, year := range results.Scenarios[0].Projection {
        assert.True(t, year.FederalTax.GreaterThanOrEqual(decimal.Zero))
        assert.True(t, year.StateTax.Equal(decimal.Zero)) // PA retirement exemption
        assert.True(t, year.LocalTax.Equal(decimal.Zero)) // No EIT on retirement income
    }
}
```

#### Performance Benchmarking

```go
func BenchmarkMonteCarloSimulation(b *testing.B) {
    scenario := createBenchmarkScenario()
    simulator := NewMonteCarloSimulator(scenario, 1000, 25)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        results := simulator.RunSimulation()
        _ = results
    }
}

func BenchmarkTSPProjection(b *testing.B) {
    strategy := &FourPercentRule{}
    initialBalance := decimal.NewFromInt(500000)
    returnRate := decimal.NewFromFloat(0.05)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        projection := ProjectTSP(initialBalance, strategy, returnRate, 25)
        _ = projection
    }
}
```

### 10. Development Roadmap and Implementation Phases

#### Phase 1: Core Calculations (4-6 weeks)

- Implement basic FERS pension calculations with all multiplier rules
- Build TSP withdrawal strategies and balance tracking
- Create Social Security benefit calculations with WEP/GPO removal
- Implement federal tax calculations with 2025 brackets
- Add Pennsylvania and local tax rules

#### Phase 2: Advanced Features (3-4 weeks)

- Implement FERS COLA rules and application logic
- Add RMD calculations and compliance checking
- Build FEHB premium tracking and Medicare integration
- Create scenario comparison and analysis tools
- Add comprehensive input validation

#### Phase 3: Output and Visualization (2-3 weeks)

- Build CLI interface with flexible output options
- Create HTML report generation with embedded charts
- Implement CSV export for external analysis
- Add summary statistics and key insights generation

#### Phase 4: Monte Carlo and Advanced Analysis (3-4 weeks)

- Integrate historical market data (TSP funds, inflation, interest rates)
- Implement Monte Carlo simulation engine
- Add portfolio longevity analysis and success rate calculations
- Create sensitivity analysis for key parameters

#### Phase 5: Testing and Documentation (2-3 weeks)

- Comprehensive unit test coverage (target 80%+)
- Integration testing with real-world scenarios
- Performance optimization for large simulations
- Complete documentation and user guides

### Implementation Best Practices

#### Code Quality Guidelines

- Use table-driven tests for all financial calculations
- Implement comprehensive logging for debugging complex scenarios
- Include extensive comments explaining financial calculation rationale
- Use dependency injection for testability
- Implement circuit breakers for external data sources

#### Financial Calculation Standards

- Always use banker's rounding for monetary calculations
- Round final results to cents, maintain precision during calculations
- Validate all input ranges and handle edge cases
- Include calculation audit trails for transparency
- Cross-reference calculations with authoritative sources

#### Data Management

- Store historical data in versioned files with source attribution
- Implement data validation and outlier detection
- Provide fallback defaults for missing historical data
- Include data update mechanisms for tax tables and regulations

This comprehensive specification provides the foundation for building a sophisticated, accurate, and user-friendly FERS retirement calculator that addresses all the unique aspects of federal employee retirement planning while leveraging modern Go programming best practices.

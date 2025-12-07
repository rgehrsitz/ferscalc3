# Retirement Planning Tool Specification for Federal Employees (FERS)

This document outlines the detailed requirements for a Go-based application to assist federal employees (PersonA and PersonB) in planning their retirement. The tool will allow for scenario analysis, comparing proposed retirement plans against current net income, and will focus on accuracy in financial projections, including taxes and benefit COLA adjustments.

## 1\. Overall Design Principles

* **Language:** Go (Golang)
* **Precision:** All financial calculations (balances, percentages, tax rates, income, expenses) must use a high-precision decimal type to avoid floating-point inaccuracies. Recommend using `github.com/shopspring/decimal`.
* **Input/Output:** Support both JSON and YAML for input data. Outputs should be clear, comparative, and suitable for analysis (e.g., structured data, potentially for visualization).
* **Modularity:** The codebase should be modular, with distinct packages/modules for data parsing, financial calculations (FERS, TSP, SS, Taxes), scenario management, and output generation.
* **User-Adjustable Parameters:** Key financial parameters (inflation, COLAs, TSP returns) must be user-definable within each scenario.
* **Projection Horizon:** The model should project 25+ years, with a user-definable projection period, emphasizing cash flow changes in the immediate and near-term post-retirement.
* **Comparison:** The core output must be an "apples-to-apples" comparison of current net income (after all deductions/taxes) versus projected net retirement income for each scenario.

## 2\. Input Data Structure

The input data should be provided in either JSON or YAML format, structured to support multiple scenarios and personal details.

```yaml
# Example YAML Input Structure

personal_details:
    person_a:
    birth_date: "1975-03-15"       # YYYY-MM-DD
    hire_date: "2000-07-01"        # YYYY-MM-DD
    employment_type: "federal"
    current_salary: 120000.00
    high_3_salary: 115000.00       # If not provided, calculate from current_salary/history
    tsp_balance_traditional: 500000.00
    tsp_balance_roth: 100000.00
    tsp_contribution_percent: 0.05 # 5% of salary
    ss_benefit_fra: 3000.00        # Estimated monthly benefit at Full Retirement Age (FRA)
    ss_benefit_62: 2100.00         # Estimated monthly benefit at age 62
    ss_benefit_70: 4000.00         # Estimated monthly benefit at age 70
    fehb_premium_per_pay_period: 600.00   # Current FEHB premium per pay period (will be annualized by multiplying by 26)
    survivor_benefit_election_percent: 0.0 # 0% as requested

    person_b:
    birth_date: "1978-06-20"
    hire_date: "2003-09-10"
    employment_type: "federal"
    current_salary: 90000.00
    high_3_salary: 85000.00
    tsp_balance_traditional: 350000.00
    tsp_balance_roth: 70000.00
    tsp_contribution_percent: 0.05
    ss_benefit_fra: 2500.00
    ss_benefit_62: 1750.00
    ss_benefit_70: 3300.00
    fehb_premium_per_pay_period: 0.0 # Assumed covered under Person A's FEHB
    survivor_benefit_election_percent: 0.0

global_assumptions:
  inflation_rate: 0.03                 # 3% annual inflation
  fehb_premium_inflation: 0.05         # 5% annual FEHB premium inflation
  tsp_return_pre_retirement: 0.07      # 7% average annual return for TSP before retirement
  tsp_return_post_retirement: 0.05     # 5% average annual return for TSP after retirement
  cola_general_rate: 0.03              # General COLA rate for SS (if not using specific SS COLA projections)
  projection_years: 25                 # Number of years to project
  current_location:                    # For state/local tax calculation
    state: "Pennsylvania"
    county: "Bucks"
    municipality: "Upper Makefield Township"

scenarios:
    - name: "Scenario 1: Person A Age 57, Person B Age 54"
    person_a:
      retirement_date: "2032-03-31" # YYYY-MM-DD
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule" # "4_percent_rule", "need_based"
      tsp_withdrawal_target_monthly: null # Only for need_based, e.g., 5000.00
    person_b:
      retirement_date: "2032-03-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
      tsp_withdrawal_target_monthly: null

    - name: "Scenario 2: Person A Age 62, Person B Age 59"
    person_a:
      retirement_date: "2037-03-31"
      ss_start_age: 67 # Full Retirement Age
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: 6000.00
    person_b:
      retirement_date: "2037-03-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
      tsp_withdrawal_target_monthly: null
```

Set `employment_type: "non-federal"` whenever a spouse is not a FERS participant. Non-federal employees skip pension/TSP accrual logic but remain in tax and cash-flow projections. Provide a `fixed_retirement_income` block on the employee (or on a per-scenario override) to represent constant outside income with an optional COLA override:

```yaml
fixed_retirement_income:
    annual_amount: 24000.00
    cola_rate: 0.02  # Optional; defaults to general COLA when omitted
```

## 3\. Core Calculation Modules

The application should implement the following calculation modules:

### 3.1 Date and Age Utilities

* Functions to calculate age from birth date, years of service from hire date, and remaining service years.
* Functions to determine Full Retirement Age (FRA) for Social Security based on birth year.

### 3.2 FERS Pension (Annuity) Calculation

```go
// Represents FERS pension calculation inputs
type FERSInputs struct {
    High3Salary       decimal.Decimal
    YearsOfService    decimal.Decimal // Includes years, months, days converted to decimal
    RetirementAge     int
    MRA               int // Minimum Retirement Age
}

// CalculateFERSPension calculates the annual FERS pension.
// Takes FERSInputs and returns annual pension.
func CalculateFERSPension(inputs FERSInputs) decimal.Decimal {
    // Formula: High-3 Salary * Years of Service * Multiplier
    // Multiplier:
    // - 1.1% (0.011) if age >= 62 with 20+ years of service.
    // - 1.0% (0.010) otherwise.
    // Example: 25.5 years of service, $100,000 High-3, age 62+
    // Pension = $100,000 * 25.5 * 0.011 = $28,050
    //
    // Account for partial years of service (e.g., 25 years and 6 months = 25.5 years)
    // YearsOfService should be precisely calculated from hire_date to retirement_date.
    // Convert to decimal for calculation.
    multiplier := decimal.NewFromFloat(0.010)
    if inputs.RetirementAge >= 62 && inputs.YearsOfService.GreaterThanOrEqual(decimal.NewFromInt(20)) {
        multiplier = decimal.NewFromFloat(0.011)
    }
    return inputs.High3Salary.Mul(inputs.YearsOfService).Mul(multiplier)
}

// ApplyFERSPensionCOLA applies the FERS COLA rules.
// COLA is not applied until the annuitant reaches age 62.
// Annual COLA Rules:
// - If CPI change (inflation) is 2% or less, COLA is the actual CPI change.
// - If CPI change is between 2% and 3%, COLA is 2%.
// - If CPI change is greater than 3%, COLA is CPI change minus 1%.
func ApplyFERSPensionCOLA(currentPension decimal.Decimal, inflationRate decimal.Decimal, annuitantAge int) decimal.Decimal {
    if annuitantAge < 62 {
        return currentPension // No COLA until age 62
    }

    colaRate := decimal.NewFromFloat(0.0)
    if inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.02)) {
        colaRate = inflationRate
    } else if inflationRate.GreaterThan(decimal.NewFromFloat(0.02)) && inflationRate.LessThanOrEqual(decimal.NewFromFloat(0.03)) {
        colaRate = decimal.NewFromFloat(0.02)
    } else { // inflationRate > 0.03
        colaRate = inflationRate.Sub(decimal.NewFromFloat(0.01))
    }
    return currentPension.Mul(decimal.NewFromFloat(1.0).Add(colaRate))
}

// CalculateFERSSpecialRetirementSupplement (SRS)
// SRS is paid to FERS retirees who retire before age 62 with MRA+ service.
// It is equivalent to the Social Security benefit earned during federal service.
// Formula: Estimated SS Benefit at Age 62 * (FERS Service Years / 40)
// SRS stops at age 62. It is also subject to an earnings test (though assumed not applicable here).
func CalculateFERSSpecialRetirementSupplement(ssBenefitAt62 decimal.Decimal, fersServiceYears decimal.Decimal, currentAge int) decimal.Decimal {
    if currentAge >= 62 {
        return decimal.Zero
    }
    // Consider earnings test if user wants that complexity later.
    return ssBenefitAt62.Mul(fersServiceYears.Div(decimal.NewFromInt(40)))
}
```

### 3.3 Thrift Savings Plan (TSP)

```go
// TSPAccount represents a TSP account (Traditional or Roth)
type TSPAccount struct {
    Balance decimal.Decimal
}

// TSPWithdrawalStrategy defines the interface for withdrawal strategies.
type TSPWithdrawalStrategy interface {
    CalculateWithdrawal(currentBalance decimal.Decimal, annualExpenses decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal
}

// FourPercentRule implements the 4% rule (adjusted for inflation)
type FourPercentRule struct {
    InitialWithdrawalPercent decimal.Decimal // e.g., 0.04 for 4%
    InflationRate            decimal.Decimal
    InitialBalance           decimal.Decimal
    FirstWithdrawalAmount    decimal.Decimal // Calculated based on InitialBalance * InitialWithdrawalPercent
}

func (f *FourPercentRule) CalculateWithdrawal(currentBalance decimal.Decimal, annualExpenses decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal {
    // In subsequent years, the withdrawal amount is inflated by the inflation rate.
    // If it's the first year, calculate based on initial balance.
    // Ensure RMD is taken if greater than planned withdrawal.
    // This is simplified and needs full implementation with annual inflation.
    // For simplicity of example, let's assume `FirstWithdrawalAmount` is already set.
    withdrawal := f.FirstWithdrawalAmount.Mul(decimal.NewFromFloat(1.0).Add(f.InflationRate).Pow(decimal.NewFromInt(age - // Age at start of withdrawal)))

    // Handle RMD (Required Minimum Distribution)
    if isRMDYear && withdrawal.LessThan(rmdAmount) {
        return rmdAmount
    }
    return withdrawal
}

// NeedBasedWithdrawal implements a strategy to withdraw based on a target monthly amount.
type NeedBasedWithdrawal struct {
    TargetMonthlyWithdrawal decimal.Decimal
}

func (n *NeedBasedWithdrawal) CalculateWithdrawal(currentBalance decimal.Decimal, annualExpenses decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal {
    withdrawal := n.TargetMonthlyWithdrawal.Mul(decimal.NewFromInt(12)) // Convert to annual

    // Handle RMD
    if isRMDYear && withdrawal.LessThan(rmdAmount) {
        return rmdAmount
    }
    return withdrawal
}

// Pre-Retirement TSP Growth:
// Simulate annual contributions (user + agency match ~5%) and growth at tsp_return_pre_retirement.
func SimulateTSPGrowthPreRetirement(initialBalance decimal.Decimal, annualContributions decimal.Decimal, annualReturn decimal.Decimal, years int) decimal.Decimal {
    currentBalance := initialBalance
    for i := 0; i < years; i++ {
        currentBalance = currentBalance.Add(annualContributions).Mul(decimal.NewFromFloat(1.0).Add(annualReturn))
    }
    return currentBalance
}

// Post-Retirement TSP Growth and Withdrawals:
// Iterate year by year, applying returns and then withdrawals.
func SimulateTSPPostRetirement(initialBalance decimal.Decimal, annualReturn decimal.Decimal, strategy TSPWithdrawalStrategy, projectionYears int, birthYear int, ssRMDYear int) ([]decimal.Decimal, []decimal.Decimal) {
    // This function needs to be fleshed out to include:
    // - Annual balance updates after growth and withdrawal.
    // - RMD calculation and application.
    // - Tracking of both Traditional and Roth balances.
    // - The RMD (Required Minimum Distribution) rules:
    //   - For individuals born in 1950 or earlier: RMDs start at age 72.
    //   - For individuals born between 1951 and 1959: RMDs start at age 73 (SECURE 2.0 Act).
    //   - For individuals born in 1960 or later: RMDs start at age 75 (SECURE 2.0 Act).
    //   - Use IRS life expectancy tables (e.g., Uniform Lifetime Table) for RMD divisors.
    return nil, nil // Return annual balances and withdrawals
}

// Monte Carlo Simulation for TSP:
// This is a highly recommended feature for evaluating withdrawal sustainability.
// - Takes historical TSP fund returns (e.g., C, S, I, F, G funds).
// - Runs many (e.g., 1,000 to 10,000) simulations of market performance.
// - Each simulation uses a random sequence of historical returns.
// - Calculates TSP balance and withdrawal sustainability for each simulation.
// - Outputs: probability of success (not running out of money), median/percentile outcomes.
func RunMonteCarloSimulation(initialBalance decimal.Decimal, strategy TSPWithdrawalStrategy, historicalReturns []decimal.Decimal, numSimulations int, projectionYears int) []decimal.Decimal {
    // Placeholder - this is a complex module requiring statistical sampling
    return nil // e.g., returns array of final balances
}

// Provide historical TSP fund returns (annualized averages for Monte Carlo)
// C Fund: e.g., 10%
// S Fund: e.g., 12%
// I Fund: e.g., 6%
// F Fund: e.g., 3%
// G Fund: e.g., 2%
// Note: Actual historical data should be sourced and periodically updated.
```

### 3.4 Social Security

```go
// CalculateMonthlySSBenefitAtAge calculates the monthly SS benefit based on claiming age.
// This function needs to interpolate/extrapolate from provided FRA, 62, 70 benefits.
// Standard reductions/credits (approx. 6-7% per year early, 8% per year delayed) should be used.
func CalculateMonthlySSBenefitAtAge(baseFRA decimal.Decimal, birthDate time.Time, claimingAge int) decimal.Decimal {
    // Based on provided FRA, 62, and 70 values.
    // For claiming ages earlier than FRA: reductions apply.
    // For claiming ages later than FRA (up to 70): delayed retirement credits apply.
    // Use fractional months for precise calculation if needed.
    return baseFRA // Placeholder
}

// ApplySSCOLA applies the annual Social Security COLA.
// This can be linked to a general inflation rate or a specific SS COLA historical rate.
func ApplySSCOLA(currentBenefit decimal.Decimal, colaRate decimal.Decimal) decimal.Decimal {
    return currentBenefit.Mul(decimal.NewFromFloat(1.0).Add(colaRate))
}

// Important Regulatory Update: Social Security Fairness Act (WEP/GPO Repeal)
// The model must assume that the Windfall Elimination Provision (WEP) and
// Government Pension Offset (GPO) are **repealed effective January 1, 2025.**
// This means no reduction in SS benefits due to FERS pension or non-covered employment.
// The code should explicitly reflect this and avoid applying these offsets.

// CalculateTaxableSocialSecurity determines the federally taxable portion of SS benefits.
// Provisional Income = (AGI - deductions) + Non-taxable interest + 1/2 of Social Security benefits.
// Thresholds for Married Filing Jointly:
// - Provisional Income <= $32,000: 0% of SS benefits are taxable.
// - Provisional Income > $32,000 and <= $44,000: Up to 50% of SS benefits are taxable.
// - Provisional Income > $44,000: Up to 85% of SS benefits are taxable.
func CalculateTaxableSocialSecurity(totalSSBenefitAnnual decimal.Decimal, provisionalIncome decimal.Decimal) decimal.Decimal {
    threshold1 := decimal.NewFromInt(32000)
    threshold2 := decimal.NewFromInt(44000)

    if provisionalIncome.LessThanOrEqual(threshold1) {
        return decimal.Zero
    } else if provisionalIncome.GreaterThan(threshold1) && provisionalIncome.LessThanOrEqual(threshold2) {
        // Taxable amount is the lesser of:
        // 1. 50% of (Provisional Income - Threshold 1)
        // 2. 50% of Total SS Benefit
        taxablePart1 := provisionalIncome.Sub(threshold1).Mul(decimal.NewFromFloat(0.5))
        taxablePart2 := totalSSBenefitAnnual.Mul(decimal.NewFromFloat(0.5))
        return decimal.Min(taxablePart1, taxablePart2)
    } else { // Provisional Income > Threshold 2
        // Taxable amount is the lesser of:
        // 1. 85% of (Provisional Income - Threshold 2) + Lesser of (50% of Threshold 2 - Threshold 1) or 50% of SS
        // 2. 85% of Total SS Benefit
        // This calculation is more complex and should follow IRS Publication 915 precisely.
        // A common simplification for the 85% bracket:
        // Lesser of:
        // a) 0.85 * totalSSBenefitAnnual
        // b) 0.85 * (provisionalIncome - threshold2) + 0.50 * (threshold2 - threshold1)
        taxableAmountA := totalSSBenefitAnnual.Mul(decimal.NewFromFloat(0.85))
        taxableAmountB := provisionalIncome.Sub(threshold2).Mul(decimal.NewFromFloat(0.85)).Add(
            decimal.NewFromFloat(0.5).Mul(threshold2.Sub(threshold1)),
        )
        return decimal.Min(taxableAmountA, taxableAmountB)
    }
}
```

### 3.5 Federal Employee Health Benefits (FEHB)

* **Pre-Retirement:** FEHB premiums are paid pre-tax (reducing taxable income).
* **Post-Retirement:** FEHB premiums are paid after-tax. This distinction is crucial for an "apples-to-apples" comparison of net income.
* The model should project FEHB premiums annually using the `fehb_premium_inflation` rate.

#### 3.6 Comprehensive Tax Modeling

The tax module must calculate federal, state (Pennsylvania), and local (Upper Makefield Township, Bucks County, PA) income taxes.

```go
// TaxIncome represents various income components for tax calculation.
type TaxIncome struct {
    Salary                decimal.Decimal
    FERSPension           decimal.Decimal
    TSPWithdrawalsTrad    decimal.Decimal
    TaxableSSBenefits     decimal.Decimal
    OtherTaxableIncome    decimal.Decimal // Placeholder for other income
}

// CalculateFederalIncomeTax calculates annual federal income tax.
// - Input: TaxIncome, filing status (Married Filing Jointly).
// - Apply standard deductions (and additional for age 65+).
// - Use progressive tax brackets (e.g., 2025 estimated brackets for Married Filing Jointly).
// - Example brackets (these need to be sourced and potentially updated annually):
//   - 10%: $0 - $23,200
//   - 12%: $23,201 - $94,300
//   - 22%: $94,301 - $201,050
//   - etc.
func CalculateFederalIncomeTax(income TaxIncome, filingStatus string, age int) decimal.Decimal {
    // Implement tax bracket logic, standard/itemized deductions, etc.
    return decimal.Zero // Placeholder
}

// CalculatePennsylvaniaStateIncomeTax calculates annual PA state income tax.
// - PA has a flat tax rate (currently 3.07%).
// - Key Exclusions: PA does NOT tax FERS pensions, TSP withdrawals, or Social Security benefits.
// - Only earned income (salary) is typically taxed.
func CalculatePennsylvaniaStateIncomeTax(income TaxIncome) decimal.Decimal {
    paTaxRate := decimal.NewFromFloat(0.0307)
    // Only salary is taxable for PA state income tax in this scenario.
    taxableIncome := income.Salary
    return taxableIncome.Mul(paTaxRate)
}

// CalculateLocalIncomeTax calculates annual local income tax for Upper Makefield Township, PA.
// - This is an Earned Income Tax (EIT).
// - Upper Makefield Township EIT is typically 1.00% (0.5% resident, 0.5% non-resident).
// - EIT applies to "earned income" (salaries, wages, net profits from business), NOT retirement income (pensions, SS, TSP).
func CalculateLocalIncomeTax(income TaxIncome) decimal.Decimal {
    localTaxRate := decimal.NewFromFloat(0.010) // 1.00%
    // Only salary is taxable for local EIT.
    taxableIncome := income.Salary
    return taxableIncome.Mul(localTaxRate)
}
```

## 4\. Output and Visualization

The tool should provide structured data output (e.g., CSV, JSON, or a custom structured text format) that can be easily consumed for analysis and potential visualization.

* **Annual Cash Flow Projections:** For each year of the projection period for each scenario:

  * **Pre-Retirement (Current Situation):**
    * Gross Salary
    * TSP Contributions (pre-tax vs. Roth)
    * FEHB Premiums (pre-tax)
    * Federal Income Tax
    * State Income Tax
    * Local Income Tax
    * FICA (Social Security & Medicare) Tax
    * Net Income (After all deductions and taxes)
  * **Post-Retirement (Scenario):**
    * FERS Pension (annual, with COLA)
    * FERS Special Retirement Supplement (if applicable)
    * Social Security Benefits (annual, with COLA, taxable portion identified)
    * TSP Withdrawals (Traditional and Roth, annual)
    * Total Gross Retirement Income
    * FEHB Premiums (after-tax)
    * Federal Income Tax
    * State Income Tax
    * Local Income Tax (should be zero post-retirement for Person A and Person B)
    * Net Retirement Income (After all deductions and taxes)
    * TSP Traditional Balance (Year-end)
    * TSP Roth Balance (Year-end)

* **Summary Tables/Metrics for each Scenario:**

  * Years until retirement (for each spouse)
  * Projected FERS Pension (initial annual)
  * Projected Social Security Benefit (initial annual)
  * Initial Annual TSP Withdrawal
  * Net Income in Year 1 of Retirement vs. Last Year of Work (the "apples-to-apples" comparison)
  * TSP Longevity (e.g., "TSP funds last until age X")
  * Total Projected Net Income over the full projection period.

* **Comparison View:** A direct side-by-side comparison of key metrics across all analyzed scenarios.

## 5\. Historical Data and Research

The coding agent should include or provide references to the following historical data for user parameterization or Monte Carlo simulation:

* **FERS COLA History:** While the formula is defined, historical CPI data can be used to validate or project.
* **TSP Fund Historical Returns:** Provide approximate historical average annual returns for TSP C, S, I, F, and G funds. (e.g., C Fund \~10%, S Fund \~12%, I Fund \~6%, F Fund \~3%, G Fund \~2% as general estimates for user guidance, but emphasize the need for actual data if Monte Carlo is implemented rigorously).
* **Inflation Rates:** Historical CPI data.
* **Social Security COLA History:** For context.

## 6\. Key Considerations and Assumptions

* **No Debt:** Assume no mortgage, car, or other loans, simplifying expense tracking.
  * **FEHB Continuity:** Assume Person A's FEHB coverage continues into retirement for both spouses.
* **Survivor Benefit:** Assume 0% survivor benefit election, meaning no pension reduction for this purpose.
* **FICA Taxes:** FICA taxes (Social Security & Medicare) apply to earned income (salary) only, not retirement income. This should be accounted for in the "current situation" net income.
* **Medicare Part B:** Not explicitly modeled as FEHB is maintained. This can be a future enhancement if needed.
* **Regulatory Updates:**
  * **Social Security Fairness Act (WEP/GPO Repeal):** Crucially, assume WEP and GPO are repealed as of January 1, 2025. This simplifies Social Security benefit calculations.
  * **SECURE 2.0 Act:** Implement the updated RMD ages: 73 for those born 1951-1959, and 75 for those born 1960 or later.
* **High-3 Salary Calculation:** If `high_3_salary` is not provided, the model should be able to estimate it from `current_salary` and a simple growth assumption, or prompt for more historical salary data if accuracy is critical.
* **TSP Agency Match:** Assume the standard 5% agency match on TSP contributions if the user contributes at least 5%. This should be factored into pre-retirement TSP growth.

-----

This specification provides a detailed roadmap for the coding agent, incorporating precise financial calculations, regulatory nuances, and a clear structure for implementation.

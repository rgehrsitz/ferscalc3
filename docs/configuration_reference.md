# Configuration Reference

Complete field-by-field reference for the FERS Retirement Planning Calculator configuration file.

Configuration files can be written in **YAML** (`.yaml` / `.yml`) or **JSON** (`.json`). Both formats support identical fields. YAML is recommended for human-authored configs due to its readability and comment support.

**Date format:** All date fields use ISO 8601: `"YYYY-MM-DDTHH:MM:SSZ"` (e.g., `"1965-02-25T00:00:00Z"`).

**Decimal values:** Can be specified as quoted strings (`"0.025"`) or bare numbers (`0.025`). Strings are recommended in YAML for financial precision.

---

## Table of Contents

1. [Top-Level Structure](#top-level-structure)
2. [Personal Details](#personal-details)
3. [TSP Allocation](#tsp-allocation)
4. [TSP Lifecycle Fund](#tsp-lifecycle-fund)
5. [Fixed Retirement Income](#fixed-retirement-income)
6. [Global Assumptions](#global-assumptions)
7. [Location](#location)
8. [Federal Rules](#federal-rules)
9. [Monte Carlo Settings](#monte-carlo-settings)
10. [TSP Statistical Models](#tsp-statistical-models)
11. [Scenarios](#scenarios)
12. [Retirement Scenario](#retirement-scenario)
13. [TSP Withdrawal Strategies](#tsp-withdrawal-strategies)
14. [TSP Withdrawal Ordering](#tsp-withdrawal-ordering)
15. [Fixed Annuity Options](#fixed-annuity-options)
16. [Mortality Modeling](#mortality-modeling)
17. [Validation Rules Summary](#validation-rules-summary)
18. [Complete Minimal Example](#complete-minimal-example)
19. [Changelog](#changelog)

---

## Top-Level Structure

```yaml
personal_details:    # Employee/participant data (required)
  person_a: { ... }  # Primary participant (required)
  person_b: { ... }  # Spouse/second participant (optional)

global_assumptions:  # Economic and projection parameters (required)
  ...

scenarios:           # Retirement scenario comparisons (required, at least one)
  - name: "..."
    person_a: { ... }
    person_b: { ... }
    mortality: { ... }  # optional
```

---

## Personal Details

Each person (`person_a`, `person_b`) supports the following fields:

### Required Fields

| Field | YAML Key | Type | Description |
|-------|----------|------|-------------|
| Name | `name` | string | Display name for reports |
| Birth Date | `birth_date` | date | Date of birth (ISO 8601) |
| Hire Date | `hire_date` | date | Service Computation Date (SCD). This is when creditable federal service began. |
| Current Salary | `current_salary` | decimal | Current annual base salary |
| High-3 Salary | `high_3_salary` | decimal | Average of the highest 3 consecutive years of salary. Used in pension calculation. |
| TSP Traditional Balance | `tsp_balance_traditional` | decimal | Current Traditional TSP balance |
| TSP Roth Balance | `tsp_balance_roth` | decimal | Current Roth TSP balance |
| TSP Contribution % | `tsp_contribution_percent` | decimal | Employee contribution as a decimal (e.g., `0.15` = 15%). Range: 0.0 - 1.0 |
| SS Benefit at FRA | `ss_benefit_fra` | decimal | Estimated **monthly** Social Security benefit at Full Retirement Age. Obtain from your SSA statement. |
| SS Benefit at 62 | `ss_benefit_62` | decimal | Estimated **monthly** SS benefit if claiming at age 62 |
| SS Benefit at 70 | `ss_benefit_70` | decimal | Estimated **monthly** SS benefit if claiming at age 70 |
| FEHB Premium | `fehb_premium_per_pay_period` | decimal | Bi-weekly FEHB health insurance premium. Set to `0` if not enrolled. Will be multiplied by 26 pay periods. |
| Survivor Benefit Election | `survivor_benefit_election_percent` | decimal | FERS survivor annuity election. `0.0` = no survivor benefit, up to `1.0`. Common values: `0.25` (partial) or `0.50` (maximum). |

### Optional Fields

| Field | YAML Key | Type | Default | Description |
|-------|----------|------|---------|-------------|
| Employment Type | `employment_type` | string | `"federal"` | Set to `"non-federal"` for civilian spouses. Non-federal employees generate no FERS pension. |
| Sick Leave Hours | `sick_leave_hours` | decimal | `0` | Unused sick leave hours at retirement. Converted to service credit using the OPM standard: **2,087 hours = 1 year** (per 5 CFR 630.301). Check your LES for current balance. |
| Pay Plan/Grade | `pay_plan_grade` | string | - | GS grade or pay plan (e.g., `"GS-13"`, `"NM-05"`). Display only. |
| SSN Last 4 | `ssn_last4` | string | - | Last 4 digits of SSN. Display only; not used in calculations. |
| TSP Allocation | `tsp_allocation` | object | - | Manual TSP fund allocation. See [TSP Allocation](#tsp-allocation). |
| TSP Lifecycle Fund | `tsp_lifecycle_fund` | object | - | Lifecycle fund selection. See [TSP Lifecycle Fund](#tsp-lifecycle-fund). |
| Fixed Retirement Income | `fixed_retirement_income` | object | - | Non-FERS income stream (e.g., military pension, private pension). See [Fixed Retirement Income](#fixed-retirement-income). |

> **Note:** `ss_benefit_62`, `ss_benefit_fra`, and `ss_benefit_70` must be in ascending order. The calculator validates that 62 <= FRA <= 70 benefits.

---

## TSP Allocation

Manually specify the percentage invested in each TSP fund. **Required for Monte Carlo simulations** to properly model market variability. Lifecycle funds produce identical Monte Carlo paths.

```yaml
tsp_allocation:
  c_fund: "0.60"   # S&P 500 Index (Large Cap Stock)
  s_fund: "0.20"   # Small Cap Stock Index (Russell 2000)
  i_fund: "0.10"   # International Stock Index (MSCI World ex-US)
  f_fund: "0.10"   # Fixed Income Index (Bloomberg US Aggregate)
  g_fund: "0.00"   # Government Securities (guaranteed return)
```

| Fund | YAML Key | Description | Risk Level |
|------|----------|-------------|------------|
| C Fund | `c_fund` | S&P 500 Large Cap Stock Index | High |
| S Fund | `s_fund` | Small Cap Stock Index (Russell 2000) | High |
| I Fund | `i_fund` | International Stock Index | High |
| F Fund | `f_fund` | Fixed Income Bond Index | Low-Moderate |
| G Fund | `g_fund` | Government Securities (guaranteed) | Very Low |

**Validation:** All five fund percentages must sum to `1.0` (100%), with a tolerance of 1%.

> **Tip:** Use `tsp_allocation` instead of `tsp_lifecycle_fund` when running Monte Carlo simulations. Lifecycle funds use a single blended return rate, which eliminates the fund-level variability that makes Monte Carlo analysis valuable.

---

## TSP Lifecycle Fund

Alternative to manual allocation. Specify a lifecycle target fund and the calculator uses its blended return profile.

```yaml
tsp_lifecycle_fund:
  fund_name: "L2030"
```

**Valid fund names:** `"L2025"`, `"L2030"`, `"L2035"`, `"L2040"`, `"L2045"`, `"L2050"`, `"L2055"`, `"L2060"`, `"L2065"`, `"L Income"`

> **Important:** Only specify one of `tsp_allocation` or `tsp_lifecycle_fund`, not both.

---

## Fixed Retirement Income

Use this for any non-FERS, non-Social Security income stream that begins at retirement:
- Military pension
- Private-sector pension
- Non-federal spouse's retirement income
- Rental income or other fixed annuities

Can be specified at the **employee level** (applies to all scenarios) or at the **scenario level** (per-scenario override).

```yaml
fixed_retirement_income:
  annual_amount: "24000"        # Annual income amount (required)
  cola_rate: "0.02"             # Annual COLA rate (optional; defaults to global cola_general_rate)
```

| Field | YAML Key | Type | Required | Description |
|-------|----------|------|----------|-------------|
| Annual Amount | `annual_amount` | decimal | Yes | Gross annual income from this source |
| COLA Rate | `cola_rate` | decimal | No | Annual cost-of-living adjustment. Omit to use the global `cola_general_rate`. Set to `0` for no COLA. |

---

## Global Assumptions

Economic parameters that apply across all scenarios.

```yaml
global_assumptions:
  inflation_rate: "0.025"
  fehb_premium_inflation: "0.065"
  tsp_return_pre_retirement: "0.055"
  tsp_return_post_retirement: "0.045"
  cola_general_rate: "0.025"
  projection_years: 25
  current_location:
    state: "Pennsylvania"
```

### Fields

| Field | YAML Key | Type | Default | Valid Range | Description |
|-------|----------|------|---------|-------------|-------------|
| Inflation Rate | `inflation_rate` | decimal | `0.025` | >= -0.10 | Annual CPI inflation assumption. Used for income projections and bracket indexing. |
| FEHB Premium Inflation | `fehb_premium_inflation` | decimal | `0.065` | >= 0.0 | Annual FEHB health premium increase rate. See [Medicare guidance](#medicare-premium-inflation-guidance) for recommended values. |
| TSP Return (Pre-Retirement) | `tsp_return_pre_retirement` | decimal | `0.055` | >= -1.0 | Expected annual TSP return before retirement. Used with lifecycle funds or as fallback. |
| TSP Return (Post-Retirement) | `tsp_return_post_retirement` | decimal | `0.045` | >= -1.0 | Expected annual TSP return after retirement. Typically lower due to conservative rebalancing. |
| COLA General Rate | `cola_general_rate` | decimal | `0.025` | >= 0.0 | Applied to FERS pension and Social Security. Subject to FERS COLA rules (see below). |
| Federal Bracket Inflation | `federal_bracket_inflation_rate` | decimal | Same as `inflation_rate` | - | Rate at which federal tax brackets are indexed. Omit to inherit from `inflation_rate`. |
| Projection Years | `projection_years` | int | - | 1 - 50 | Number of years to project. Typically 25-30 for retirement planning. |
| Projection Base Year | `projection_base_year` | int | Current year | - | Starting year for the projection. Usually left at default. |
| Location | `current_location` | object | - | - | See [Location](#location). Required for state/local tax calculations. |
| Monte Carlo Settings | `monte_carlo_settings` | object | - | - | See [Monte Carlo Settings](#monte-carlo-settings). |
| Federal Rules | `federal_rules` | object | 2025 defaults | - | See [Federal Rules](#federal-rules). Override tax brackets, Medicare thresholds, etc. |
| TSP Statistical Models | `tsp_statistical_models` | object | - | - | See [TSP Statistical Models](#tsp-statistical-models). |

### FERS COLA Rules

The calculator applies COLA to FERS pensions following OPM regulations (5 CFR 842.403):

- **Unreduced annuitants** (MRA+30, age 60 with 20+ years, age 62 with 5+ years, special provisions): COLA applies immediately in the first year after retirement.
- **Reduced annuitants** (MRA+10 early retirement with penalty): COLA is deferred until age 62.
- **COLA calculation** (when eligible):
  - CPI increase <= 2%: Full CPI adjustment
  - CPI increase 2-3%: Capped at 2%
  - CPI increase > 3%: CPI minus 1%
- **COLA floor:** COLA can never be negative. In deflationary years, pension remains unchanged.

### Medicare Premium Inflation Guidance

| Scenario | Rate | Notes |
|----------|------|-------|
| Moderate / Baseline | **5.5%** | Matches ~5.5% compound annual growth for Part B premiums 2005-2024. |
| Conservative / Safer | **6.5%** | Captures volatility (e.g., 2022 +14.5%). Good for stress-testing. |
| Aggressive / Optimistic | **3.5-4%** | Only if you expect Medicare cost growth near general CPI. Most planners consider this risky. |

---

## Location

Required for state and local tax calculations.

```yaml
current_location:
  state: "Pennsylvania"
  county: "Bucks"
  municipality: "Upper Makefield Township"
```

| Field | YAML Key | Required | Description |
|-------|----------|----------|-------------|
| State | `state` | Yes | Full name or two-letter abbreviation (e.g., `"PA"` or `"Pennsylvania"`). Determines state income tax rules. |
| County | `county` | No | County name. Used for local tax rates in some states. |
| Municipality | `municipality` | No | Township/city name. Used for local Earned Income Tax (EIT) rates. |

> **Currently supported states:** Pennsylvania (full support with retirement income exemptions), New Jersey (basic support). Other states apply federal-only calculations.

---

## Federal Rules

Override default 2025 federal tax rules, FERS parameters, Medicare thresholds, and FICA configuration. All sub-sections are optional; omitted values use built-in 2025 defaults.

### FERS Rules

```yaml
federal_rules:
  fers_rules:
    tsp_matching_rate: "0.05"
    tsp_matching_threshold: "0.05"
    srs_earnings_limit: 23400
```

| Field | YAML Key | Default | Description |
|-------|----------|---------|-------------|
| TSP Matching Rate | `tsp_matching_rate` | `0.05` | Agency matching percentage |
| TSP Matching Threshold | `tsp_matching_threshold` | `0.05` | Employee contribution threshold for full match |
| SRS Earnings Limit | `srs_earnings_limit` | `23400` | Special Retirement Supplement earnings test limit (2025 value) |

### Federal Tax Configuration

```yaml
federal_rules:
  federal_tax_config:
    standard_deduction_mfj: 30000
    standard_deduction_single: 15000
    additional_standard_deduction_65_plus: 1600
    tax_brackets_2025:
      - min: "0"
        max: "23850"
        rate: "0.10"
      - min: "23850"
        max: "96950"
        rate: "0.12"
      # ... additional brackets
```

**2025 MFJ brackets** (IRS Rev. Proc. 2024-40, built-in defaults):

| Rate | Min | Max |
|------|-----|-----|
| 10% | $0 | $23,850 |
| 12% | $23,850 | $96,950 |
| 22% | $96,950 | $206,700 |
| 24% | $206,700 | $394,600 |
| 32% | $394,600 | $501,050 |
| 35% | $501,050 | $751,600 |
| 37% | $751,600+ | |

### FICA Tax Configuration

```yaml
federal_rules:
  fica_tax_config:
    social_security_wage_base: 176100
    social_security_rate: "0.062"
    medicare_rate: "0.0145"
    additional_medicare_rate: "0.009"
    high_income_threshold_mfj: 250000
```

### Medicare Configuration

```yaml
federal_rules:
  medicare_config:
    base_premium_2025: "185.00"
    premium_inflation_rate: "0.055"
    irmaa_thresholds:
      - income_threshold_single: "103000"
        income_threshold_joint: "206000"
        monthly_surcharge: "69.90"
      - income_threshold_single: "129000"
        income_threshold_joint: "258000"
        monthly_surcharge: "174.70"
```

Medicare IRMAA (Income-Related Monthly Adjustment Amount) uses MAGI from **two years prior** to determine premium surcharges. The calculator automatically applies this lookback.

### Social Security Tax Thresholds

```yaml
federal_rules:
  social_security_tax_thresholds:
    married_filing_jointly:
      threshold_1: 32000
      threshold_2: 44000
    single:
      threshold_1: 25000
      threshold_2: 34000
```

These provisional income thresholds determine what percentage of Social Security benefits are taxable (0%, 50%, or 85%).

### FEHB Configuration

```yaml
federal_rules:
  fehb_config:
    pay_periods_per_year: 26
    retirement_calculation_method: "same_as_active"
    retirement_premium_multiplier: "1.0"
```

---

## Monte Carlo Settings

Parameters for stochastic simulation. Only relevant when running `monte-carlo` commands.

```yaml
monte_carlo_settings:
  tsp_return_variability: "0.15"
  inflation_variability: "0.02"
  cola_variability: "0.02"
  fehb_variability: "0.05"
  max_reasonable_income: 5000000
```

| Field | YAML Key | Default | Description |
|-------|----------|---------|-------------|
| TSP Return Variability | `tsp_return_variability` | `0.15` | Standard deviation of TSP return shocks (15%) |
| Inflation Variability | `inflation_variability` | `0.02` | Standard deviation of inflation shocks |
| COLA Variability | `cola_variability` | `0.02` | Standard deviation of COLA shocks |
| FEHB Variability | `fehb_variability` | `0.05` | Standard deviation of FEHB premium shocks |
| Max Reasonable Income | `max_reasonable_income` | `5000000` | Cap on annual income to prevent runaway simulations |

### Stress Tests

Define deterministic market stress scenarios:

```yaml
monte_carlo_settings:
  stress_tests:
    severe_recession:
      name: "2008-Style Crisis"
      description: "Three-year severe downturn"
      repeat: false
      years:
        - year: 1
          label: "Crisis Year"
          tsp_returns:
            c_fund: -0.37
            s_fund: -0.36
            i_fund: -0.42
            f_fund: 0.05
            g_fund: 0.03
          inflation: 0.038
          cola: 0.058
          fehb: 0.08
        - year: 2
          label: "Recovery"
          tsp_returns:
            c_fund: 0.26
            s_fund: 0.28
            i_fund: 0.32
            f_fund: 0.06
            g_fund: 0.03
          inflation: 0.027
          cola: 0.0
          fehb: 0.06
```

---

## TSP Statistical Models

Per-fund historical return characteristics for Monte Carlo simulation. Override to use custom data or different time periods.

```yaml
tsp_statistical_models:
  c_fund:
    mean: 0.1125
    standard_dev: 0.1744
    data_source: "TSP.gov 1988-2024"
    last_updated: "2025-01-15"
  s_fund:
    mean: 0.1117
    standard_dev: 0.1933
  i_fund:
    mean: 0.0634
    standard_dev: 0.1863
  f_fund:
    mean: 0.0532
    standard_dev: 0.0565
  g_fund:
    mean: 0.0493
    standard_dev: 0.0165
```

| Field | Description |
|-------|-------------|
| `mean` | Historical mean annual return for the fund |
| `standard_dev` | Standard deviation of annual returns |
| `data_source` | Label describing the data source (informational) |
| `last_updated` | Date the statistics were last refreshed (informational) |

---

## Scenarios

Each scenario models a "what-if" retirement plan. You can define multiple scenarios to compare different retirement dates, SS claiming ages, and withdrawal strategies side by side.

```yaml
scenarios:
  - name: "Early Retirement 2025"        # Required: descriptive label
    person_a:                              # Required: primary participant
      employee_name: "person_a"
      retirement_date: "2025-12-31T00:00:00Z"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    person_b:                              # Optional: spouse/second participant
      employee_name: "person_b"
      retirement_date: "2025-12-31T00:00:00Z"
      ss_start_age: 67
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: "3000"
    mortality:                             # Optional: mortality modeling
      person_a:
        death_age: 90
```

| Field | YAML Key | Required | Description |
|-------|----------|----------|-------------|
| Name | `name` | Yes | Scenario label shown in reports |
| Person A | `person_a` | Yes | See [Retirement Scenario](#retirement-scenario) |
| Person B | `person_b` | No | See [Retirement Scenario](#retirement-scenario) |
| Mortality | `mortality` | No | See [Mortality Modeling](#mortality-modeling) |

---

## Retirement Scenario

Each participant within a scenario has these fields:

### Required Fields

| Field | YAML Key | Type | Description |
|-------|----------|------|-------------|
| Employee Name | `employee_name` | string | Must match `"person_a"` or `"person_b"` from `personal_details` |
| Retirement Date | `retirement_date` | date | Planned retirement date (ISO 8601) |
| SS Start Age | `ss_start_age` | int | Age to begin claiming Social Security. Range: **62-70**. Note: this is a whole-year value; fractional claiming ages are not supported. |
| TSP Strategy | `tsp_withdrawal_strategy` | string | Withdrawal strategy name. See [TSP Withdrawal Strategies](#tsp-withdrawal-strategies). |

### Optional Fields

| Field | YAML Key | Type | Default | Description |
|-------|----------|------|---------|-------------|
| TSP Withdrawal Target | `tsp_withdrawal_target_monthly` | decimal | - | **Required for `need_based` strategy.** Monthly withdrawal target. |
| TSP Withdrawal Rate | `tsp_withdrawal_rate` | decimal | - | **Required for `variable_percentage` and `floor_ceiling` strategies.** Annual withdrawal rate as decimal (e.g., `0.04` = 4%). Range: 0.0 - 0.20. |
| TSP Withdrawal Floor | `tsp_withdrawal_floor` | decimal | - | **Optional for `floor_ceiling` strategy.** Minimum annual withdrawal amount. |
| TSP Withdrawal Ceiling | `tsp_withdrawal_ceiling` | decimal | - | **Optional for `floor_ceiling` strategy.** Maximum annual withdrawal amount. |
| TSP Withdrawal Ordering | `tsp_withdrawal_ordering` | string | `"traditional_first"` | Which TSP account to draw from first (after RMD). See [TSP Withdrawal Ordering](#tsp-withdrawal-ordering). |
| Fixed Retirement Income | `fixed_retirement_income` | object | - | Per-scenario override. See [Fixed Retirement Income](#fixed-retirement-income). |
| Annuity Options | (multiple) | - | - | **Required for `fixed_annuity` strategy.** See [Fixed Annuity Options](#fixed-annuity-options). |

---

## TSP Withdrawal Strategies

Five withdrawal strategies are available. Each requires different optional fields.

### 1. `4_percent_rule`

The classic safe withdrawal rate approach. Withdraws 4% of the initial balance in the first year of retirement, then adjusts that dollar amount for inflation each subsequent year.

```yaml
tsp_withdrawal_strategy: "4_percent_rule"
# No additional fields required
```

### 2. `need_based`

Withdraws a fixed monthly amount regardless of balance. Useful when targeting a specific income level.

```yaml
tsp_withdrawal_strategy: "need_based"
tsp_withdrawal_target_monthly: "3000"   # Required: monthly target
```

### 3. `variable_percentage`

Withdraws a fixed percentage of the **current balance** each year. Naturally adjusts to market conditions: higher withdrawals when balances are up, lower when down.

```yaml
tsp_withdrawal_strategy: "variable_percentage"
tsp_withdrawal_rate: "0.04"             # Required: 4% of current balance
```

### 4. `floor_ceiling`

A guardrails approach. Withdraws a percentage of the current balance, but clamps the result between a minimum floor and maximum ceiling. Prevents over- or under-spending.

```yaml
tsp_withdrawal_strategy: "floor_ceiling"
tsp_withdrawal_rate: "0.05"             # Required: base percentage
tsp_withdrawal_floor: "30000"           # Optional: minimum annual withdrawal
tsp_withdrawal_ceiling: "80000"         # Optional: maximum annual withdrawal
```

**Validation:** If both `tsp_withdrawal_floor` and `tsp_withdrawal_ceiling` are specified, the floor must be less than the ceiling.

### 5. `fixed_annuity`

Converts part or all of the TSP balance into an annuity with guaranteed payments. Simulates purchasing a commercial annuity at retirement.

```yaml
tsp_withdrawal_strategy: "fixed_annuity"
annuity_payout_rate: "0.055"            # Required: annual payout rate (5.5%)
annuity_premium_percent: "0.50"         # Optional: % of TSP to annuitize (50%)
annuity_cola_rate: "0.02"               # Optional: annual COLA on payments
annuity_survivor_percent: "1.0"         # Optional: survivor payout (100%)
annuity_guaranteed_years: 10            # Optional: guaranteed payment period
```

See [Fixed Annuity Options](#fixed-annuity-options) for details.

---

## TSP Withdrawal Ordering

Controls which TSP account is drawn from first **after** Required Minimum Distributions (RMDs) are satisfied. RMDs always come from the Traditional TSP account first, as required by IRS rules.

```yaml
tsp_withdrawal_ordering: "traditional_first"   # default
# or
tsp_withdrawal_ordering: "roth_first"
```

| Value | Behavior | Tax Implication |
|-------|----------|-----------------|
| `"traditional_first"` | **Default.** After satisfying RMD from Traditional, additional withdrawals come from Traditional first, then Roth when Traditional is depleted. | Higher taxable income in early retirement years. Preserves Roth for tax-free growth and legacy. Depletes the taxable account faster, reducing future RMDs. |
| `"roth_first"` | After satisfying RMD from Traditional, additional withdrawals come from Roth first, then Traditional. | Lower taxable income early on. Preserves Traditional for potential Roth conversion strategies or future tax planning. |

**When to use each:**

- **`traditional_first`** (recommended default): Best for most retirees. Depleting the Traditional balance first reduces future RMD obligations and shifts more of your long-term portfolio into tax-free Roth growth. This is typically more tax-efficient over a 25-30 year retirement.

- **`roth_first`**: May be beneficial if you plan to do strategic Roth conversions during low-income years, or if you expect to be in a significantly higher tax bracket later (e.g., large pension + full Social Security + RMDs stacking up).

---

## Fixed Annuity Options

When using the `fixed_annuity` strategy, these fields configure the annuity:

| Field | YAML Key | Type | Required | Valid Range | Description |
|-------|----------|------|----------|-------------|-------------|
| Payout Rate | `annuity_payout_rate` | decimal | Yes | 0.0 - 0.20 | Annual payout as a percentage of the annuitized amount (e.g., `0.055` = 5.5%). |
| Premium Percent | `annuity_premium_percent` | decimal | No | 0.0 - 1.0 | Fraction of TSP balance used to purchase the annuity. Default: `1.0` (100%). Set to `0.5` to annuitize half and keep the rest invested. |
| COLA Rate | `annuity_cola_rate` | decimal | No | - | Annual COLA increase on annuity payments. `0` for level payments. |
| Survivor Percent | `annuity_survivor_percent` | decimal | No | 0.0 - 1.0 | Percentage of annuity that continues to a surviving spouse. `1.0` = 100%, `0.5` = 50%, `0` = life-only. |
| Guaranteed Years | `annuity_guaranteed_years` | int | No | - | Minimum payment period (e.g., `10` guarantees 10 years of payments even if the annuitant dies). |

---

## Mortality Modeling

Model the financial impact of one participant predeceasing the other. Defines death timing and survivor assumptions.

```yaml
mortality:
  person_a:
    death_age: 90                          # Die at age 90
  person_b:
    death_date: "2055-06-15T00:00:00Z"     # Or use an exact date
  assumptions:
    survivor_spending_factor: "0.80"
    tsp_spousal_transfer: "merge"
    filing_status_switch: "next_year"
```

### Mortality Spec (per person)

| Field | YAML Key | Type | Description |
|-------|----------|------|-------------|
| Death Age | `death_age` | int | Age at death. Mutually exclusive with `death_date`. |
| Death Date | `death_date` | date | Exact date of death (ISO 8601). Mutually exclusive with `death_age`. |

> **Validation:** Specify either `death_age` or `death_date` for a person, not both.

### Mortality Assumptions

| Field | YAML Key | Type | Valid Range | Description |
|-------|----------|------|-------------|-------------|
| Survivor Spending Factor | `survivor_spending_factor` | decimal | 0.4 - 1.0 | Fraction of income the survivor needs. `0.80` means the survivor's expenses are 80% of the couple's. Applied to pension and TSP withdrawals **before** tax calculations. |
| TSP Spousal Transfer | `tsp_spousal_transfer` | string | `"merge"` or `"separate"` | What happens to the deceased's TSP. `"merge"` transfers balances to the survivor. `"separate"` keeps them distinct. |
| Filing Status Switch | `filing_status_switch` | string | `"next_year"` or `"immediate"` | When the survivor switches from Married Filing Jointly to Single. `"next_year"` reflects the IRS rule that the surviving spouse can file jointly for the year of death. |

---

## Validation Rules Summary

Quick reference of all input constraints enforced by the configuration validator.

### Employee Validation
- `birth_date` must be before `hire_date`
- `current_salary` and `high_3_salary` must be positive
- `tsp_balance_traditional` and `tsp_balance_roth` must be non-negative
- `tsp_contribution_percent` must be between 0.0 and 1.0
- Social Security benefits must increase: `ss_benefit_62` <= `ss_benefit_fra` <= `ss_benefit_70`
- `survivor_benefit_election_percent` must be between 0.0 and 1.0
- If `tsp_allocation` is provided, fund percentages must sum to 1.0 (within 1% tolerance)

### Global Assumptions Validation
- `inflation_rate` >= -0.10 (allows extreme deflation scenarios)
- `fehb_premium_inflation` >= 0.0
- `tsp_return_pre_retirement` and `tsp_return_post_retirement` >= -1.0
- `cola_general_rate` >= 0.0
- `projection_years` must be between 1 and 50
- `current_location.state` is required

### Scenario Validation
- `name` is required
- `ss_start_age` must be between 62 and 70
- `tsp_withdrawal_strategy` must be one of: `4_percent_rule`, `need_based`, `variable_percentage`, `fixed_annuity`, `floor_ceiling`
- `tsp_withdrawal_rate` (when required) must be between 0.0 and 0.20 (0-20%)
- `tsp_withdrawal_ordering` must be `"traditional_first"` or `"roth_first"` (if specified)
- For `need_based`: `tsp_withdrawal_target_monthly` is required
- For `variable_percentage`: `tsp_withdrawal_rate` is required
- For `floor_ceiling`: `tsp_withdrawal_rate` is required; if both floor and ceiling are specified, floor must be less than ceiling
- For `fixed_annuity`: `annuity_payout_rate` is required (range: 0.0-0.20)
- Mortality: specify either `death_date` or `death_age`, not both
- `survivor_spending_factor`: 0.4 - 1.0
- `tsp_spousal_transfer`: `"merge"` or `"separate"`
- `filing_status_switch`: `"next_year"` or `"immediate"`

---

## Complete Minimal Example

A working two-person, two-scenario configuration with only required fields:

```yaml
personal_details:
  person_a:
    name: "Alice"
    birth_date: "1965-06-15T00:00:00Z"
    hire_date: "1990-08-01T00:00:00Z"
    current_salary: "120000"
    high_3_salary: "118000"
    tsp_balance_traditional: "500000"
    tsp_balance_roth: "50000"
    tsp_contribution_percent: "0.15"
    ss_benefit_62: "2200"
    ss_benefit_fra: "3100"
    ss_benefit_70: "3850"
    fehb_premium_per_pay_period: "350"
    survivor_benefit_election_percent: "0.50"

  person_b:
    name: "Bob"
    birth_date: "1967-03-20T00:00:00Z"
    hire_date: "1992-01-15T00:00:00Z"
    current_salary: "95000"
    high_3_salary: "93000"
    tsp_balance_traditional: "350000"
    tsp_balance_roth: "40000"
    tsp_contribution_percent: "0.10"
    ss_benefit_62: "1800"
    ss_benefit_fra: "2600"
    ss_benefit_70: "3200"
    fehb_premium_per_pay_period: "0"
    survivor_benefit_election_percent: "0.0"

global_assumptions:
  inflation_rate: "0.025"
  fehb_premium_inflation: "0.06"
  tsp_return_pre_retirement: "0.055"
  tsp_return_post_retirement: "0.045"
  cola_general_rate: "0.025"
  projection_years: 25
  current_location:
    state: "Pennsylvania"

scenarios:
  - name: "Both Retire Early"
    person_a:
      employee_name: "person_a"
      retirement_date: "2027-06-30T00:00:00Z"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    person_b:
      employee_name: "person_b"
      retirement_date: "2027-06-30T00:00:00Z"
      ss_start_age: 67
      tsp_withdrawal_strategy: "variable_percentage"
      tsp_withdrawal_rate: "0.04"

  - name: "Alice Delays to 67"
    person_a:
      employee_name: "person_a"
      retirement_date: "2032-07-01T00:00:00Z"
      ss_start_age: 67
      tsp_withdrawal_strategy: "floor_ceiling"
      tsp_withdrawal_rate: "0.05"
      tsp_withdrawal_floor: "25000"
      tsp_withdrawal_ceiling: "75000"
      tsp_withdrawal_ordering: "roth_first"
    person_b:
      employee_name: "person_b"
      retirement_date: "2029-04-01T00:00:00Z"
      ss_start_age: 62
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: "2500"
    mortality:
      person_a:
        death_age: 88
      assumptions:
        survivor_spending_factor: "0.75"
        tsp_spousal_transfer: "merge"
        filing_status_switch: "next_year"
```

---

## Changelog

### 2025 Red-Team Review Updates

These fields and behaviors were added or corrected as part of a comprehensive code review:

| Change | Type | Description |
|--------|------|-------------|
| `tsp_withdrawal_ordering` | **New field** | Controls whether Traditional or Roth TSP is drawn first (after RMD). Default: `"traditional_first"`. Previously hardcoded as Roth-first. |
| `floor_ceiling` strategy | **New validation** | The `floor_ceiling` withdrawal strategy is now fully validated. Requires `tsp_withdrawal_rate`; validates floor < ceiling when both specified. |
| Sick leave conversion | **Formula fix** | `sick_leave_hours` now uses the OPM standard of **2,087 hours = 1 year** (per 5 CFR 630.301), replacing the previous 8 hours/day / 365.25 days formula. |
| FERS COLA eligibility | **Logic fix** | COLA deferral until age 62 now only applies to **reduced** annuitants (MRA+10 early retirees). Unreduced annuitants receive COLA immediately. |
| FERS COLA floor | **Logic fix** | COLA can never be negative. Deflationary CPI changes leave pension unchanged (per 5 CFR 842.403). |
| Tax brackets | **Data fix** | 2025 MFJ brackets corrected to match IRS Rev. Proc. 2024-40. Additional senior standard deduction updated from $1,550 to $1,600. |
| SS wage base | **Data fix** | Social Security wage base display updated from $168,600 to $176,100 (2025 value). |
| Survivor spending factor | **Logic fix** | Now applied **before** tax calculations, so taxes reflect the survivor's reduced income rather than the couple's full income. |
| SS claiming age | **Documentation** | Added code-level documentation noting that `ss_start_age` is integer-only, which introduces ~1-2% approximation for mid-year claiming. |

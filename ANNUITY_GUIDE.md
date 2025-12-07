# Fixed Annuity Feature - User Guide

## Overview

The FERS Calculator now supports modeling **Fixed Immediate Annuities** as an alternative TSP withdrawal strategy. This allows you to directly compare the guaranteed income from an annuity contract against traditional TSP withdrawal strategies like the 4% Rule.

## What is a Fixed Immediate Annuity?

A fixed immediate annuity is an insurance product where you convert a lump sum (your TSP balance) into guaranteed lifetime income payments. Key features:

- **Guaranteed lifetime income**: Payments continue for life, regardless of market performance
- **No market risk**: Your income is fixed and not affected by stock market volatility
- **Optional COLA**: Some annuities offer cost-of-living adjustments
- **Survivor benefits**: Payments can continue to your spouse after your death
- **Guaranteed period**: Minimum payment period (e.g., "10 years certain")

## Configuration Options

To use the fixed annuity strategy in your retirement scenario, set `tsp_withdrawal_strategy: "fixed_annuity"` and configure these parameters:

### Required Parameters

- **`annuity_payout_rate`**: Annual payout as a percentage of your premium (e.g., `"0.055"` for 5.5%)
  - Range: 0.01 to 0.20 (1% to 20%)
  - Typical rates: 5.0% to 6.5% depending on age and product

### Optional Parameters (with defaults)

- **`annuity_premium_percent`**: Percentage of TSP to convert to annuity
  - Default: `"1.0"` (100% of TSP)
  - Range: 0.01 to 1.0 (1% to 100%)
  - Example: `"0.5"` converts only 50% of TSP to annuity

- **`annuity_cola_rate`**: Annual cost-of-living adjustment
  - Default: `"0.0"` (no COLA, fixed payment)
  - Range: 0.0 to 0.10 (0% to 10%)
  - Note: COLA riders typically reduce initial payout rate

- **`annuity_survivor_percent`**: Percentage paid to survivor
  - Default: `"1.0"` (100% to survivor, "joint and survivor" option)
  - Range: 0.0 to 1.0 (0% to 100%)
  - Common options: `"1.0"` (100%), `"0.5"` (50%), `"0.0"` (single life)

- **`annuity_guaranteed_years`**: Minimum guaranteed payment period
  - Default: `10` years
  - Common options: 10, 15, or 20 years certain

## Example Scenarios

### Scenario 1: 100% TSP to Fixed Annuity (No COLA)

```yaml
person_a:
  employee_name: "person_a"
  retirement_date: "2025-12-31T00:00:00Z"
  ss_start_age: 62
  tsp_withdrawal_strategy: "fixed_annuity"
  annuity_premium_percent: "1.0"      # Convert 100% of TSP
  annuity_payout_rate: "0.055"        # 5.5% annual payout
  annuity_cola_rate: "0.0"            # No COLA (fixed payment)
  annuity_survivor_percent: "1.0"     # 100% to survivor
  annuity_guaranteed_years: 10        # 10 years certain
```

**For $1,966,168.86 TSP balance:**
- Annual payment: $108,139.29
- Monthly payment: $9,011.61
- Payment stays the same every year (no COLA)

### Scenario 2: Annuity with 2% COLA

```yaml
person_a:
  employee_name: "person_a"
  retirement_date: "2025-12-31T00:00:00Z"
  ss_start_age: 62
  tsp_withdrawal_strategy: "fixed_annuity"
  annuity_premium_percent: "1.0"
  annuity_payout_rate: "0.050"        # Lower rate due to COLA rider
  annuity_cola_rate: "0.02"           # 2% annual increase
  annuity_survivor_percent: "1.0"
  annuity_guaranteed_years: 10
```

**For $1,966,168.86 TSP balance:**
- Year 1 payment: $98,308.44
- Year 2 payment: $100,274.61 (2% increase)
- Year 10 payment: $116,458.67 (compounded growth)

### Scenario 3: Hybrid Approach (50% Annuity, 50% Remains in TSP)

```yaml
person_a:
  employee_name: "person_a"
  retirement_date: "2025-12-31T00:00:00Z"
  ss_start_age: 62
  tsp_withdrawal_strategy: "fixed_annuity"
  annuity_premium_percent: "0.5"      # Only 50% to annuity
  annuity_payout_rate: "0.055"
  annuity_cola_rate: "0.0"
  annuity_survivor_percent: "1.0"
  annuity_guaranteed_years: 10
```

**For $1,966,168.86 TSP balance:**
- Annuity portion: $983,084.43 (50%)
- Remaining in TSP: $983,084.43 (50%)
- Annuity annual payment: $54,069.64
- Plus: Flexibility to withdraw from remaining TSP as needed

## Running Comparisons

### Compare TSP 4% Rule vs. Fixed Annuity

1. Create scenarios with different withdrawal strategies:
   - Scenario 1: `tsp_withdrawal_strategy: "4_percent_rule"`
   - Scenario 2: `tsp_withdrawal_strategy: "fixed_annuity"`

2. Run the calculator:
   ```bash
   fers-calc calculate annuity_comparison_config.yaml -f html -o comparison.html
   ```

3. Review the HTML report to compare:
   - First year net income
   - Year 5 and Year 10 net income
   - Total lifetime income (present value)
   - TSP longevity (how long money lasts)

### Key Comparison Metrics

**TSP 4% Rule:**
- ✅ Maintains TSP balance that can grow with markets
- ✅ Flexibility to adjust withdrawals
- ✅ Can leave legacy to heirs
- ⚠️ Subject to market volatility
- ⚠️ Could run out of money if markets underperform

**Fixed Annuity:**
- ✅ Guaranteed lifetime income (no longevity risk)
- ✅ No market risk - payments don't change with market crashes
- ✅ Simplified tax planning (predictable income)
- ⚠️ No access to lump sum (irreversible decision)
- ⚠️ No growth potential beyond COLA
- ⚠️ Reduced or no legacy for heirs

## Important Notes

### Tax Considerations

- Annuity payments from Traditional TSP are fully taxable as ordinary income
- The calculator models annuity payments as TSP withdrawals for tax purposes
- All existing tax calculations (federal, state, IRMAA, etc.) apply

### RMD Compliance

- Annuity payments from qualified retirement accounts (like TSP) satisfy Required Minimum Distribution (RMD) requirements
- The calculator does not apply additional RMD rules to annuity payments
- If you do a partial annuity (e.g., 50%), the remaining TSP balance would still have RMD requirements

### Inflation Protection

- Fixed annuities without COLA lose purchasing power over time
- 2% inflation means your Year 20 payment buys ~33% less than Year 1
- COLA riders help but typically reduce initial payout rate by 1-2 percentage points

### What the Calculator Models

✅ **Included:**
- Guaranteed annual payments based on payout rate
- Optional COLA adjustments
- Survivor benefit percentages
- Integration with all existing tax calculations
- Comparison with other withdrawal strategies

❌ **Not Included** (simplified assumptions):
- Partial year pro-rating (pays full annual amount in retirement year)
- Death benefit calculations beyond guaranteed period
- State insurance guarantee limits
- Specific insurance company ratings
- Annuity contract fees or surrender charges

## Recommended Analysis Approach

1. **Baseline**: Run your current plan with 4% rule
2. **Full Annuity**: Model converting 100% of TSP to annuity
3. **Hybrid**: Model 50/50 split for balance of security and flexibility
4. **COLA Comparison**: Compare fixed vs. COLA annuities

Compare the scenarios focusing on:
- First 5 years of income (cash flow stability)
- 10-20 year outlook (inflation impact)
- Total lifetime income (present value)
- Legacy considerations (remaining TSP balance)

## Real-World Next Steps

If the annuity option looks favorable in the calculator:

1. **Get actual quotes** from multiple insurance companies
2. **Verify payout rates** - calculator uses your estimates
3. **Review insurance company ratings** (A.M. Best, Moody's, etc.)
4. **Understand EXACT contract terms** - this is a permanent decision
5. **Consult with a financial advisor** - calculator is for illustration only
6. **Consider split strategy** - don't put all eggs in one basket

## Sample Configuration File

See `annuity_comparison_config.yaml` for a complete example with 4 scenarios:
1. Traditional TSP 4% Rule
2. 100% Fixed Annuity (no COLA)
3. Fixed Annuity with 2% COLA  
4. Hybrid 50/50 approach

Run it with:
```bash
fers-calc calculate annuity_comparison_config.yaml -f html -o report.html
```

---

**Disclaimer**: This calculator is for educational and planning purposes only. Actual annuity products vary significantly in terms, rates, fees, and features. Always consult with qualified financial and tax professionals before making irreversible financial decisions.

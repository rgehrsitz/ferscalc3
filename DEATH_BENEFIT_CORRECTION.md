# Death Benefit Correction Summary

## What Was Wrong

The original `FG_COMPARISON_ANALYSIS.md` contained **incorrect assumptions** about F&G annuity death benefits:

### Incorrect Statements (Now Fixed):
- ❌ "Premature Death Risk: HIGH (heirs may get little/nothing after guarantee period)"
- ❌ "If you die in year 11, heirs likely get $0 (after 10-year guarantee)"
- ❌ "No access to the $2M ever again"

## What Is Actually True

Based on F&G Safe Income Advantage product documentation:

### Death Benefits
**"Your account value is paid as a lump sum death benefit to the beneficiary or beneficiaries you name in your contract."**

- Heirs receive remaining **account value** as lump sum
- Account value = Premium - Cumulative Withdrawals + Credited Interest
- The 6.16% annual payout depletes account value faster than conservative TSP withdrawals
- After 15-20 years, significantly less remains for heirs compared to TSP approach

### The "10-Year" Confusion

Your advisor mentioned "children would get their inheritance over ten years" - this refers to **IRS rules**, not the annuity contract:

1. **IRS 10-Year Rule**: Non-spouse beneficiaries can spread inherited retirement account distributions over 10 years
   - Provides tax benefits (avoid single large tax hit)
   - Applies to BOTH TSP inheritance AND annuity death benefits
   - This is federal tax law, not an annuity feature

2. **Annuity 10-Year Guarantee**: Separate feature ensuring minimum 10 years of income payments
   - If you die in year 5, beneficiaries continue receiving payments for 5 more years
   - If you live past year 10, payments continue for life
   - Protects against early death scenario

These are **two different things** - one is a tax rule, one is an insurance feature.

## Corrected Comparison

### TSP 4% Rule - Legacy Perspective
- After 20 years of 4% withdrawals with 5% market returns: **~$2M+ still remains**
- Full balance passes to heirs as lump sum
- Can use IRS 10-year distribution rule for tax spreading

### F&G Annuity - Legacy Perspective  
- After 20 years of 6.16% withdrawals: **Account value significantly depleted**
- Remaining account value (if any) passes to heirs as lump sum
- Can use IRS 10-year distribution rule for tax spreading
- During first 10 years, guaranteed period ensures heirs get remaining payments if you die early

## The Real Tradeoff

**TSP Approach:**
- Lower initial income ($6,667/month vs $10,267/month)
- Preserves principal better (4% withdrawal vs 6.16%)
- Likely leaves substantial inheritance ($1.5M - $2M+ after 20 years)

**F&G Annuity Approach:**
- Higher guaranteed income ($10,267/month)
- Depletes principal faster (6.16% > typical growth rates)
- Leaves smaller inheritance (remaining account value after years of withdrawals)

Both approaches provide death benefits to heirs. The difference is **how much remains**.

## Files Updated

- ✅ `FG_COMPARISON_ANALYSIS.md` - Corrected death benefit and inheritance sections
- ✅ Added "Understanding Death Benefits vs Guaranteed Period" section
- ✅ Clarified IRS 10-year rule vs annuity 10-year guarantee
- ✅ Updated risk analysis to reflect accurate legacy comparison

## Source Documentation

Death benefit confirmation from:
- `docs/annuity/SafeIncomeAdvantageBrochureadv2780.md` (lines 540-543)
- `docs/annuity/FG Safe Income Advantage Consumer Overview.md` (line 191)
- `docs/annuity/ADV2006 Performance Pro (CB)-Standard.md` (lines 201, 509-514)

# Chart Improvements Summary

## Break-Even Line Explanation

The **break-even line** appears on the "Cumulative Net Income Comparison" chart and shows:

### What It Represents
- **The crossover point** where total lifetime income becomes equal between Scenario 1 and Scenario 2
- Compares cumulative (total) net income over time, not annual income
- Shows when one scenario "catches up" to the other in total earnings

### How to Read It
- **Vertical dashed line**: Marks the exact month/year of the crossover
- **Label**: Shows the break-even date and cumulative amount at that point
- **Before break-even**: One scenario has earned more total lifetime income
- **After break-even**: The other scenario has earned more total lifetime income

### Example: F&G Annuity vs TSP 4% Rule
- **Early years**: F&G annuity earns more annually ($123K vs $80K), so cumulative total is higher
- **Later years**: TSP 4% Rule increases with inflation while annuity stays flat
- **Break-even point**: Where TSP cumulative total catches up to annuity cumulative total
- **After break-even**: TSP continues pulling ahead in total lifetime earnings

### Calculation Method
The break-even is calculated using linear interpolation:
1. Compute cumulative net income for each scenario year-by-year
2. Find where cumulative totals cross (sign change in difference)
3. Interpolate within that year to find the exact month
4. Mark on chart with vertical line and label

## New Combined Chart

### "Combined: TSP Balance & Net Income Over Time"

A new full-width chart has been added at the top of the Visual Analysis section that shows **both metrics together**:

### Features
- **Dual Y-Axis**:
  - Left axis (darker): TSP Balance in millions ($M)
  - Right axis (teal): Annual Net Income in thousands ($K)

- **Visual Distinction**:
  - TSP Balance: **Solid lines**
  - Net Income: **Dashed lines**

- **Color Coding**:
  - Each scenario uses the same color for both metrics
  - Example: Scenario 1 (blue) shows blue solid line for TSP and blue dashed line for income

### Why This Chart Is Useful

**See The Tradeoff Visually:**
- F&G Annuity: High income (dashed line) but **TSP depletes to zero** (solid line drops)
- TSP 4% Rule: Lower income initially but **balance preserved** (solid line stays high)

**Key Insights at a Glance:**
1. **Income vs Preservation**: Higher withdrawals = more income but less balance remaining
2. **Legacy Planning**: See exactly when TSP balance hits zero for each scenario
3. **Flexibility**: Higher balance = more flexibility for emergencies or changes
4. **Inheritance**: Solid lines show what's left for heirs at any point

### Reading the Chart

**Example Interpretation:**
- Year 2030: F&G shows $120K income (dashed) but only $1.2M TSP left (solid)
- Year 2030: TSP 4% shows $90K income (dashed) but still $2.0M TSP (solid)
- **Tradeoff**: $30K more annual income vs $800K more preserved wealth

## Files Modified

- ✅ `internal/output/templates/report.html.tmpl`:
  - Added combined chart section at top of Visual Analysis
  - Created dual y-axis Chart.js implementation
  - Solid lines for TSP Balance, dashed for Net Income
  - Formatted y-axes: $M for balance, $K for income
  - Positioned legend at bottom for clarity

## How to Use

Simply regenerate any HTML report:
```bash
.\fers-calc.exe calculate fg_annuity_vs_tsp_comparison.yaml -f html -o report.html
```

The new combined chart will automatically appear in all HTML reports going forward.

## Benefits

1. **Single View**: See both critical metrics without switching between charts
2. **Direct Comparison**: Understand income vs preservation tradeoff immediately
3. **Color Consistency**: Same scenario = same color across all charts
4. **Visual Clarity**: Solid vs dashed makes it easy to distinguish metrics
5. **Scaled Appropriately**: Dual axes prevent one metric from dwarfing the other

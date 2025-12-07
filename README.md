# FERS Retirement Planning Calculator

A production-ready retirement planning suite for Federal Employees Retirement System (FERS) participants, written in Go. The project now includes a memory-optimized Monte Carlo engine with deterministic stress-scenario sweeps and overlay reporting, making it practical to analyze hundreds of offsets on commodity hardware.

## Highlights

- **Complete FERS Modeling**: Accurate pension math (1.0% / 1.1% multipliers), FEHB premiums, survivor options, COLA rules, and dual-scenario comparisons.
- **TSP + Tax Stack**: Manual TSP allocations, Roth vs. Traditional withdrawal ordering, Pennsylvania + federal tax engines, Social Security integration with 2025 WEP/GPO repeal.
- **Scenario Fan-out**: Run multiple deterministic scenarios or batch stress cases with a single CLI call.
- **Monte Carlo Engine**: Parallel historical/statistical simulations, percentile tables, TSP depletion stats.
- **Stress Sweeps** *(new)*: Deterministic offset sweep for each stress scenario, generating overlay HTML reports with worst-offset highlighting while keeping only summary + median data per offset.
- **Batch Outputs**: HTML dashboards, CSV summaries, JSON payloads for automation.

## Recent Regulatory Compliance

- **Social Security Fairness Act (2025)**: WEP/GPO repeal implementation
- **SECURE 2.0 Act**: Updated RMD ages (73 for 1951-1959, 75 for 1960+)
- **2025 Tax Brackets**: Current federal tax calculations
- **Pennsylvania Tax Rules**: State-specific retirement income exemptions
- **SRS Modeling Note**: The Special Retirement Supplement assumes zero earned income after the retirement date. Because the earnings test depends on wages post-retirement, users planning to keep working should manually adjust projections to reflect potential SRS reductions.

## Project Status

- ✅ CLI wiring for batch stress scenarios and deterministic sweeps.
- ✅ Memory-optimized sweep execution (summary + overlay data retained, full simulation payloads discarded per offset).
- ✅ Overlay HTML template with chart.js visualizations and worst-offset badges.
- 🚧 Additional docs and examples for custom stress libraries (see `data/stress/periods.yaml`).
- 🚧 Reference configs (`dr.yaml`) are still evolving—treat as examples.

## Installation

1. Install prerequisites:
   - Go 1.21+
   - Git
2. Clone and build:

   ```bash
   git clone https://github.com/example/rpgo.git
   cd rpgo
   go mod tidy
   go build -o fers-calc ./cmd/fers-calc
   ```

## Usage

### Quick Start

1. **Generate an example configuration**:

   ```bash
   ./fers-calc example config.yaml
   ```

2. **Run calculations**:

   ```bash
   ./fers-calc calculate config.yaml
   ```

3. **Generate HTML report**:

   ```bash
   ./fers-calc calculate config.yaml --format html > report.html
   ```

4. **Run Monte Carlo stress sweep (HTML overlay)**:

   ```bash
   go run ./cmd/fers-calc/main.go monte-carlo dr.yaml \
     --stress-all \
     --format html \
     --output reports/mc.html \
     --stress-sweep
   ```

   **Tip**: Use manual `tsp_allocation` blocks for Monte Carlo variability; lifecycle funds yield identical paths.

### Command Line Options

To help you understand the core commands, here's a breakdown:

- **`./fers-calc calculate [input-file]`**: Runs a single, deterministic retirement projection based on the fixed assumptions in your input configuration file. This provides a detailed report for one specific set of conditions.

- **`./fers-calc historical monte-carlo [data-path] [flags]`**: Executes simple portfolio Monte Carlo simulations. This command is focused on assessing the sustainability of a specific investment balance with a defined withdrawal strategy, using historical (or statistical) market data. It does not model the full FERS retirement system.

- **`./fers-calc monte-carlo [config-file] [data-path]`**: Runs comprehensive FERS Monte Carlo simulations. This command integrates your full retirement configuration (pension, Social Security, TSP, taxes, FEHB) with historical market data to run thousands of scenarios, providing a probabilistic assessment of your complete retirement plan's success.

```bash
# Basic calculation
./fers-calc calculate [input-file]

# With options
./fers-calc calculate [input-file] --format html --verbose --debug > report.html

# Validate configuration
./fers-calc validate [input-file]

# Generate example
./fers-calc example [output-file]

# Break-even analysis
./fers-calc break-even [input-file]

# Historical data management
./fers-calc historical load ./data
./fers-calc historical stats ./data
./fers-calc historical query ./data 2020 C

# Simple Portfolio Monte Carlo simulations
./fers-calc historical monte-carlo ./data --simulations 1000 --balance 1000000 --withdrawal 40000
./fers-calc historical monte-carlo ./data --strategy guardrails --years 30

# Comprehensive FERS Monte Carlo simulations
./fers-calc monte-carlo config.yaml ./data --simulations 1000
./fers-calc monte-carlo config.yaml ./data --simulations 5000 --seed 12345 --debug
```

#### Logging and Debug Mode

- Use `--debug` on CLI commands (calculate, break-even, monte-carlo) to enable detailed debug logs.
- Debug logs are generated via an internal Logger interface; the CLI wires a simple logger that prints level-prefixed lines (DEBUG/INFO/WARN/ERROR).
- When `--debug` is off, a no-op logger is used to keep output clean.

#### Output formats and aliases

Supported `--format` values:

- `console`, `console-lite`, `csv`, `detailed-csv`, `html`, `json`, `all`

Aliases map to canonical names:

- `console-verbose` → `console`; `verbose` → `console`
- `csv-detailed` → `detailed-csv`; `csv-summary` → `csv`
- `html-report` → `html`; `json-pretty` → `json`

Reports are output to stdout by default. Redirect to files as needed (e.g., `> report.html`).

### Output Formats

- `console`: Formatted text output (default)
- `html`: Interactive HTML report with charts and visualizations
- `json`: Structured JSON data
- `csv`: Comma-separated values for spreadsheet analysis

## Configuration File Format

The calculator uses YAML configuration files. Here's an example structure:

```yaml
personal_details:
  person_a:
    name: "Person A"
    birth_date: "1963-06-15"
    hire_date: "1985-03-20"
    employment_type: "federal"  # use "non-federal" for civilian spouses
    current_salary: 95000
    high_3_salary: 93000
    tsp_balance_traditional: 450000
    tsp_balance_roth: 50000
    tsp_contribution_percent: 0.15
    ss_benefit_fra: 2400
    ss_benefit_62: 1680
    ss_benefit_70: 2976
    fehb_premium_monthly: 875
    survivor_benefit_election_percent: 0.0
    
    # TSP allocation (required for Monte Carlo variability)
    tsp_allocation:
      c_fund: "0.60"  # 60% C Fund (Large Cap Stock Index)
      s_fund: "0.20"  # 20% S Fund (Small Cap Stock Index)
      i_fund: "0.10"  # 10% I Fund (International Stock Index)
      f_fund: "0.10"  # 10% F Fund (Fixed Income Index)
      g_fund: "0.00"  # 0% G Fund (Government Securities)

  person_b:
    name: "Person B"
    birth_date: "1965-08-22"
    hire_date: "1988-07-10"
    employment_type: "federal"
    current_salary: 87000
    high_3_salary: 85000
    tsp_balance_traditional: 380000
    tsp_balance_roth: 45000
    tsp_contribution_percent: 0.12
    ss_benefit_fra: 2200
    ss_benefit_62: 1540
    ss_benefit_70: 2728
    fehb_premium_monthly: 0.0
    survivor_benefit_election_percent: 0.0
    
    # TSP allocation (required for Monte Carlo variability)
    tsp_allocation:
      c_fund: "0.40"  # 40% C Fund (Large Cap Stock Index)
      s_fund: "0.10"  # 10% S Fund (Small Cap Stock Index)
      i_fund: "0.10"  # 10% I Fund (International Stock Index)
      f_fund: "0.30"  # 30% F Fund (Fixed Income Index)
      g_fund: "0.10"  # 10% G Fund (Government Securities)

global_assumptions:
  inflation_rate: 0.025
  fehb_premium_inflation: 0.065
  tsp_return_pre_retirement: 0.055
  tsp_return_post_retirement: 0.045
  cola_general_rate: 0.025
  projection_years: 25
  current_location:
    state: "Pennsylvania"
    county: "Bucks"
    municipality: "Upper Makefield Township"

scenarios:
  - name: "Early Retirement 2025"
    person_a:
      employee_name: "person_a"
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    person_b:
      employee_name: "person_b"
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"

  - name: "Delayed Retirement 2028"
    person_a:
      employee_name: "person_a"
      retirement_date: "2028-12-31"
      ss_start_age: 67
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: 3000
    person_b:
      employee_name: "person_b"
      retirement_date: "2028-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"

### Medicare premium inflation guidance

| Scenario | Rate | Notes |
| --- | --- | --- |
| Moderate / Baseline | **5.5%** | Matches the ~5.5% compound annual growth rate for Part B premiums from 2005–2024. |
| Conservative / Safer | **6.5%** | Captures volatility seen in years like 2022 (+14.5%). Useful for stress-testing. |
| Aggressive / Optimistic | **3.5–4%** | Only use if you expect Medicare cost growth to fall near general CPI; most planners consider this risky. |

Set `federal_rules.medicare_config.premium_inflation_rate` to one of these values to tailor projections. If omitted, the calculator falls back to 5.5%.

## Configuration File Format

The calculator uses YAML configuration files. Here's an example structure:

```yaml
personal_details:
  person_a:
    employment_type: "non-federal"
    fixed_retirement_income:
      annual_amount: "24000"
      cola_rate: "0.02"   # Optional; defaults to global COLA when omitted

scenarios:
  - name: "Spouse Fixed Income"
    person_b:
      fixed_retirement_income:
        annual_amount: "30000"  # override for this scenario only

federal_rules:
  medicare_config:
    base_premium_2025: "185.00"           # 2025 base Part B premium
    premium_inflation_rate: "0.055"       # Recommended default (see table below)
    irmaa_thresholds:
      - income_threshold_single: "103000"  # First IRMAA tier (single)
        income_threshold_joint: "206000"   # First IRMAA tier (MFJ)
        monthly_surcharge: "69.90"         # Additional monthly premium
```

## Calculation Details

### FERS Pension

- **Formula**: High-3 Salary × Years of Service × Multiplier
- **Multipliers**:
  - Standard: 1.0% per year of service
  - Enhanced: 1.1% per year if retiring at age 62+ with 20+ years service
- **COLA Rules**:
  - No COLA until age 62
  - CPI ≤ 2%: Full CPI increase
  - CPI 2-3%: Capped at 2%
  - CPI > 3%: CPI minus 1%

### TSP Configuration

#### TSP Allocations vs Lifecycle Funds

For **deterministic calculations**, both approaches work equivalently:

- **Manual TSP Allocation**: Specify exact percentages for each fund (C, S, I, F, G)
- **TSP Lifecycle Fund**: Use predefined lifecycle funds (L2030, L2035, L2040, L Income)

For **Monte Carlo simulations**, use **manual TSP allocations** for proper market variability:

```yaml
# Recommended for Monte Carlo
tsp_allocation:
  c_fund: "0.60"  # 60% C Fund
  s_fund: "0.20"  # 20% S Fund
  i_fund: "0.10"  # 10% I Fund (International Stock Index)
  f_fund: "0.10"  # 10% F Fund (Fixed Income Index)
  g_fund: "0.00"  # 0% G Fund (Government Securities)

# Avoid for Monte Carlo (produces identical results)
# tsp_lifecycle_fund:
#   fund_name: "L2030"
```

#### TSP Fund Types

- **C Fund**: S&P 500 Index (Large Cap Stock)
- **S Fund**: Small Cap Stock Index (Russell 2000)
- **I Fund**: International Stock Index (MSCI World ex-US)
- **F Fund**: Fixed Income Index (Bloomberg US Aggregate)
- **G Fund**: Government Securities (Guaranteed return)

### TSP Withdrawal Strategies

- **4% Rule**: Initial 4% withdrawal, adjusted for inflation annually
- **Need-Based**: Withdraw based on target monthly income
- **RMD Compliance**: Automatic Required Minimum Distribution calculations
- **Traditional vs Roth**: Optimized withdrawal order (Roth first, then Traditional)

### Monte Carlo Analysis

#### Simple Portfolio Monte Carlo

- **Historical Data**: Real TSP fund returns, inflation, and COLA data (1990-2023)
- **Withdrawal Strategies**: Fixed amount, percentage, inflation-adjusted, guardrails
- **Risk Assessment**: Success rates, percentile analysis, drawdown tracking
- **Asset Allocation**: Customizable TSP fund allocations (C, S, I, F, G funds)
- **Parallel Processing**: Efficient simulation execution for 1000+ scenarios

#### Comprehensive FERS Monte Carlo

- **Full FERS Integration**: Models all retirement components (pension, SS, TSP, taxes, FEHB)
- **Market Variability**: Historical or statistical market condition generation
- **Income Sustainability**: Success rates based on complete retirement income
- **TSP Longevity**: Tracks when TSP balances deplete
- **Tax Implications**: Includes all federal, state, and local taxes
- **Healthcare Costs**: Models FEHB premium increases over time

#### Monte Carlo Examples

#### Conservative 4% Rule (25 years)

```bash
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000 \
  --strategy fixed_amount
```

Result: 99% success rate, median ending balance $6.6M

### Comprehensive FERS Analysis

```bash
./fers-calc monte-carlo config.yaml ./data \
  --simulations 1000
```

*Result: 100% success rate, median net income $234,681, low risk assessment

High-Precision FERS Analysis

```bash
./fers-calc monte-carlo config.yaml ./data \
  --simulations 5000 \
  --seed 12345
```

*Result: Reproducible results with comprehensive risk analysis

Aggressive 6% Rule with Guardrails (Simple Portfolio)

```bash
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 30000 \
  --strategy guardrails \
  --years 30
```

Result: 82% success rate, high risk assessment

Inflation-Adjusted Strategy (Simple Portfolio)

```bash
./fers-calc historical monte-carlo ./data \
  --simulations 500 \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted \
  --years 30
```

Result: 84% success rate, moderate risk assessment

### Social Security

- **2025 WEP/GPO Repeal**: No benefit reductions for federal employees
- **Claiming Ages**: 62-70 with proper benefit adjustments
- **Taxation**: Up to 85% taxable based on provisional income

### Tax Calculations

- **Federal**: 2025 tax brackets with standard deductions
- **Pennsylvania**: 3.07% flat rate, retirement income exempt
- **Local**: Earned Income Tax (EIT) only on wages
- **FICA**: Social Security and Medicare taxes on earned income only

## Project Structure

```text
rpgo/
├── cmd/cli/                 # Command line interface
├── data/                   # Historical financial data
│   ├── tsp-returns/        # TSP fund historical returns
│   ├── inflation/          # CPI-U inflation rates
│   └── cola/               # Social Security COLA rates
├── internal/
│   ├── domain/             # Core domain models
│   ├── calculation/        # Calculation engines
│   ├── config/         # Medicare Part B premium configuration - 2025 values
    # Source: Centers for Medicare & Medicaid Services (CMS)
    medicare_config:
      base_premium_2025: "185.00"           # 2025 base Part B premium
      premium_inflation_rate: "0.055"       # Recommended default (see table below)
      irmaa_thresholds:
        - income_threshold_single: "103000"  # First IRMAA tier (single)
          income_threshold_joint: "206000"   # First IRMAA tier (MFJ)
          monthly_surcharge: "69.90"         # Additional monthly premium
├── pkg/
│   ├── decimal/            # Financial precision utilities
│   └── dateutil/           # Date calculation utilities
├── test/                   # Test files and data
└── docs/                   # Documentation
```

## Testing

Run the test suite:

```bash
go test ./...
```

Run specific test packages:

```bash
go test ./internal/calculation
go test ./internal/config
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This calculator is for educational and planning purposes only. It should not be considered as financial advice. Please consult with qualified financial professionals for personalized retirement planning advice. The calculations are based on current regulations and may change over time.

## Support

For issues, questions, or contributions, please use the GitHub issue tracker or contact the maintainers.

## Roadmap

- [x] Monte Carlo simulation for TSP returns
- [x] Historical data integration
- [x] Interactive HTML reports with charts and visualizations
- [ ] TSP lifecycle fund support for Monte Carlo simulations
- [ ] Enhanced withdrawal strategies (floor-ceiling, bond tent)
- [ ] Web interface
- [ ] Additional state tax support
- [ ] Medicare Part B premium calculations
- [ ] Survivor benefit optimization
- [ ] Export to financial planning software

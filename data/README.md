# Historical Data Infrastructure

This directory contains the historical financial data used for Monte Carlo simulations and historical scenario analysis in the FERS Retirement Calculator.

## Directory Structure

```
data/
├── README.md                 # This file
├── tsp-returns/             # TSP fund historical returns
│   ├── c-fund-annual.csv    # C Fund (S&P 500) returns
│   ├── s-fund-annual.csv    # S Fund (small cap) returns
│   ├── i-fund-annual.csv    # I Fund (international) returns
│   ├── f-fund-annual.csv    # F Fund (bonds) returns
│   └── g-fund-annual.csv    # G Fund (government securities) returns
├── inflation/               # Inflation data
│   └── cpi-annual.csv       # CPI-U annual inflation rates
└── cola/                    # Social Security COLA data
    └── ss-cola-annual.csv   # Social Security COLA rates
```

## Data Sources

### TSP Fund Returns
- **Source**: TSP.gov historical performance data
- **Period**: 1990-2023 (34 years)
- **Format**: Annual returns as decimal values (e.g., 0.181 = 18.1%)
- **Funds**:
  - **C Fund**: S&P 500 Index (large cap stocks)
  - **S Fund**: Dow Jones U.S. Completion Total Stock Market Index (small cap stocks)
  - **I Fund**: MSCI EAFE Index (international stocks)
  - **F Fund**: Bloomberg U.S. Aggregate Bond Index (bonds)
  - **G Fund**: U.S. Treasury securities (government securities)

### Inflation Data
- **Source**: Bureau of Labor Statistics (BLS.gov)
- **Period**: 1990-2023 (34 years)
- **Format**: Annual CPI-U inflation rates as decimal values
- **Note**: Negative values indicate deflation (e.g., 2009: -0.4%)

### Social Security COLA
- **Source**: Social Security Administration (SSA.gov)
- **Period**: 1990-2023 (34 years)
- **Format**: Annual COLA rates as decimal values
- **Note**: Zero values indicate years with no COLA increase

## Data Quality

The historical data includes:
- **33 years of complete data** (1990-2022)
- **Statistical validation** for outliers and data consistency
- **Missing year detection** and reporting
- **Source attribution** for all datasets

### Statistical Summary (1990-2022)

| Fund | Mean Return | Std Dev | Min | Max |
|------|-------------|---------|-----|-----|
| C Fund | 11.25% | 17.44% | -36.7% | 37.4% |
| S Fund | 11.17% | 19.33% | -38.1% | 46.1% |
| I Fund | 6.34% | 18.63% | -42.3% | 38.9% |
| F Fund | 5.32% | 5.65% | -13.4% | 18.7% |
| G Fund | 4.93% | 1.65% | 3.4% | 8.9% |
| Inflation | 2.59% | 1.37% | -0.4% | 6.5% |
| COLA | 2.56% | 1.82% | 0.0% | 8.7% |

## Usage

### CLI Commands

```bash
# Load and validate historical data
./fers-calc historical load ./data

# Display statistical summaries
./fers-calc historical stats ./data

# Query specific data points
./fers-calc historical query ./data 2020 C
./fers-calc historical query ./data 2020 inflation
./fers-calc historical query ./data 2020 cola
```

### Programmatic Usage

```go
// Create historical data manager
hdm := calculation.NewHistoricalDataManager("./data")

// Load all data
if err := hdm.LoadAllData(); err != nil {
    log.Fatal(err)
}

// Get specific returns
cReturn, err := hdm.GetTSPReturn("C", 2020)
inflation, err := hdm.GetInflationRate(2020)
cola, err := hdm.GetCOLARate(2020)

// Validate data quality
issues, err := hdm.ValidateDataQuality()
```

## Data Format

All CSV files follow the same format:

```csv
Year,Return
1990,0.061
1991,0.031
...
```

- **Year**: 4-digit year (e.g., 1990)
- **Return**: Decimal value representing the rate (e.g., 0.061 = 6.1%)

## Updating Data

To update the historical data:

1. **Download new data** from official sources:
   - TSP.gov for fund returns
   - BLS.gov for inflation rates
   - SSA.gov for COLA rates

2. **Add new rows** to the appropriate CSV files
3. **Validate data quality** using the CLI commands
4. **Update this README** with new statistical summaries

## Monte Carlo Integration

This historical data infrastructure provides the foundation for:

- **Historical scenario replay**: Test retirement plans against actual historical market conditions
- **Monte Carlo simulations**: Random sampling from historical returns for probabilistic analysis
- **Sequence of returns risk**: Analyze the impact of market timing on retirement outcomes
- **Correlation analysis**: Study relationships between different asset classes

## Data Validation

The system automatically validates:

- **Data completeness**: Checks for missing years
- **Outlier detection**: Flags extreme returns (>100% or <-50%)
- **Consistency**: Ensures all funds have the same year range
- **Format validation**: Verifies CSV structure and data types

## Future Enhancements

Planned improvements:

- **Monthly data**: Higher frequency data for more granular analysis
- **Additional funds**: Support for lifecycle funds (L funds)
- **Real-time updates**: Automated data fetching from official sources
- **Extended history**: Data going back to TSP inception (1987)
- **International data**: Additional international market indices 
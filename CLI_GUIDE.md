# FERS Calculator CLI Guide

A powerful command-line tool for FERS retirement planning with scriptable workflows.

## Quick Start

### Build the CLI

```bash
make build
```

Or manually:

```bash
go build -o fers-calc ./cmd/fers-calc
```

## Commands

### 1. Calculate Scenarios

Run deterministic retirement projections:

```bash
# Basic usage
./fers-calc calculate config.yaml

# Generate HTML report
./fers-calc calculate config.yaml --format html --output report.html

# Generate CSV for Excel
./fers-calc calculate config.yaml --format csv --output results.csv

# Verbose console output
./fers-calc calculate config.yaml --format verbose
```

**Available formats:**
- `console` - Simple console output (default)
- `verbose` - Detailed console output
- `html` - Interactive HTML report with charts
- `json` - JSON export
- `csv` - Summary CSV
- `detailed-csv` - Detailed CSV with all years

### 2. Monte Carlo Simulations

Run probabilistic analysis with historical market data:

```bash
# Run 1000 simulations (default)
./fers-calc monte-carlo config.yaml

# Run more simulations for better accuracy
./fers-calc monte-carlo config.yaml --runs 10000

# Generate HTML report
./fers-calc monte-carlo config.yaml --format html --output mc_report.html

# Use statistical mode (no historical data needed)
./fers-calc monte-carlo config.yaml --statistical
```

**Flags:**
- `-r, --runs` - Number of simulations (default: 1000)
- `-d, --data-path` - Path to historical data directory (default: ./data)
- `-f, --format` - Output format: html, csv (default: html)
- `-s, --statistical` - Use statistical distributions instead of historical data

### 3. Break-Even Analysis

Compare two scenarios to find the break-even point:

```bash
# Compare first two scenarios in config
./fers-calc break-even config.yaml

# Compare specific scenarios by name
./fers-calc break-even config.yaml \
  --scenario1 "Retire at 62" \
  --scenario2 "Retire at 65"

# Verbose output with year-by-year comparison
./fers-calc break-even config.yaml --verbose
```

### 4. Version Information

```bash
./fers-calc version
```

## Global Flags

Available for all commands:

- `--verbose` - Enable verbose output
- `--quiet` - Suppress non-error output
- `-h, --help` - Show help

## Examples

### Complete Workflow

```bash
# 1. Prepare a configuration (edit YAML manually or start from example_config.yaml)
cp example_config.yaml myplan.yaml
vim myplan.yaml

# 2. Run basic calculations
./fers-calc calculate myplan.yaml --format verbose

# 3. Generate HTML report
./fers-calc calculate myplan.yaml --format html

# 4. Run Monte Carlo analysis
./fers-calc monte-carlo myplan.yaml --runs 5000

# 5. Compare scenarios
./fers-calc break-even myplan.yaml
```

### Automated Reports

```bash
#!/bin/bash
# Generate all reports

CONFIG="retirement-plan.yaml"
DATE=$(date +%Y%m%d)

./fers-calc calculate $CONFIG \
  --format html \
  --output reports/${DATE}-deterministic.html

./fers-calc monte-carlo $CONFIG \
  --runs 10000 \
  --format html \
  --output reports/${DATE}-montecarlo.html

./fers-calc calculate $CONFIG \
  --format csv \
  --output reports/${DATE}-data.csv

echo "Reports generated in reports/"
```

### CI/CD Integration

```bash
# Run in non-interactive mode for automation
./fers-calc calculate config.yaml --quiet --format json > results.json

# Exit code indicates success/failure
if [ $? -eq 0 ]; then
  echo "Calculations succeeded"
else
  echo "Calculations failed"
  exit 1
fi
```

## Configuration Files

The CLI uses YAML configuration files. Example structure:

```yaml
personal_details:
  person_a:
    name: "Robert"
    birth_date: "1969-01-15"
    hire_date: "2000-06-01"
    current_salary: 150000
    high_3_salary: 155000
    # ... more fields
  person_b:
    # ... similar structure

global_assumptions:
  inflation_rate: 0.025
  tsp_return_pre_retirement: 0.055
  tsp_return_post_retirement: 0.04
  projection_years: 25

scenarios:
  - name: "Retire at 62"
    person_a:
      retirement_date: "2031-01-31"
      ss_start_age: 67
      tsp_withdrawal_strategy: "four_percent_rule"
    person_b:
      retirement_date: "2033-03-31"
      ss_start_age: 67
      tsp_withdrawal_strategy: "four_percent_rule"
```

See `example_config.yaml` for a complete example.

## Building from Source

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Build

```bash
# Using Make
make build

# Or with Go directly
go build -o fers-calc ./cmd/fers-calc
```

### Install to $GOPATH/bin

```bash
make install
# or
go install ./cmd/fers-calc
```

### Cross-compile for Multiple Platforms

```bash
make build-all
```

Generates binaries for:
- Linux (amd64)
- macOS (amd64, arm64)
- Windows (amd64)

## Development

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Run Linters

```bash
make vet
```

### Clean Build Artifacts

```bash
make clean
```

## Troubleshooting

### "No config file found"

Ensure you specify a config file or have `config.yaml` in the current directory:

```bash
./fers-calc calculate path/to/config.yaml
```

### Monte Carlo "Failed to load historical data"

The tool will automatically fall back to statistical mode. To use historical data:

1. Ensure `data/` directory exists
2. Populate with TSP fund returns (see `data/README.md`)
3. Or use `--statistical` flag explicitly

### "Command not found: fers-calc"

Either run from the build directory (`./fers-calc`) or install to PATH:

```bash
make install
# Then use from anywhere:
fers-calc calculate config.yaml
```

## Advanced Usage

### Custom Historical Data Path

```bash
./fers-calc monte-carlo config.yaml --data-path /path/to/historical/data
```

### Combining Flags

```bash
./fers-calc calculate config.yaml \
  --format html \
  --output report.html \
  --verbose
```

### Scripting with JSON Output

```bash
# Generate JSON and process with jq
./fers-calc calculate config.yaml --format json | \
  jq '.scenarios[] | {name, first_year_net_income}'
```

## Next Steps

1. Copy `example_config.yaml` and tailor it to your scenario.
2. Run calculations: `./fers-calc calculate config.yaml`
3. Generate reports in HTML format for easy sharing
4. Run Monte Carlo simulations to understand risk

For more details on configuration options, see the main `README.md`.

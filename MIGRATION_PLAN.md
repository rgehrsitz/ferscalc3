# Migration Plan: Bring ferex Calculation Engine Goodness into ferscalc3

## Overview

This plan brings three distinct improvements from `ferex` into `ferscalc3`:

1. **IRS Simplified Method exclusion** — a tax-free portion of the FERS pension for employees who made after-tax contributions
2. **Calculation audit trail** — step-by-step breakdown of pension calculation with OPM references
3. **Regression test suite** — 9 deterministic FERS/SS/TSP scenarios (excluding CSRS) with expected CSV outputs

CSRS and GUI code are explicitly excluded.

No architectural changes are required. Each item is additive and self-contained.

---

## What Is NOT Being Migrated

- CSRS calculations (`csrs_calc.go`, `survivor.go` CSRS paths)
- Any GUI / Wails / Svelte frontend code
- `ferex`'s `date_utils.go` (ferscalc3 has a superior `pkg/dateutil/` package)
- `ferex`'s `cola.go` (ferscalc3's `ApplyFERSPensionCOLA` in `fers.go` is more correct)
- `ferex`'s `social_security.go` (ferscalc3's version is more complete: decimal precision, WEP/GPO)
- `ferex`'s `tax.go` (ferscalc3's tax stack is strictly superior: IRMAA, multi-state, decimal)
- `ferex`'s `monte_carlo.go` (ferscalc3's Monte Carlo engine is strictly superior)
- `ferex`'s `retirement_flow.go` orchestration (ferscalc3's `engine.go` / `projection.go` replaces this)

---

## Item 1: IRS Simplified Method Exclusion

### What it is

Federal employees make after-tax contributions to FERS throughout their career. When they retire, a portion of each monthly pension payment is a **tax-free return of those contributions**, calculated per IRS Publication 721 / Publication 575 using the "Simplified Method." This reduces the taxable portion of the pension dollar-for-dollar.

### Why ferscalc3 needs it

ferscalc3's `taxes.go` calculates federal tax against the full gross pension amount with no exclusion. This overstates taxable income and thus federal tax liability for any employee who has made after-tax contributions. The current TODO comment in `taxes.go` acknowledges this gap.

### Source

`/c/code/ferex/ferex1-main/backend/calculation/irs_simplified_method.go`

The source file is 56 lines and has no dependencies outside the standard library. The logic: divide `totalContributions` by `numExpectedMonthlyPayments` (from IRS Pub 575 Table 1, keyed on retirement age and whether a survivor is elected) to get the monthly exclusion. Multiply by 12 for the annual figure.

The IRS tables encoded in ferex cover ages in bands (< 50, 50–54, 55–60, 61–65, 66–69, 70+) for both single-life and joint-and-survivor annuities. These are stable IRS tables that do not change year-to-year.

### Implementation Steps

#### Step 1.1 — Create the new file

Create `/c/code/ferscalc3/internal/calculation/irs_simplified_method.go`.

Port the logic from ferex directly. Convert `float64` to `decimal.Decimal` to match ferscalc3's precision standard. The function signature should be:

```go
package calculation

import "github.com/shopspring/decimal"

// CalculateIRSSimplifiedMethodExclusion computes the annual tax-free portion of a pension
// using the IRS Simplified Method (IRS Publication 721, Table 1).
//
// Parameters:
//   - totalContributions: employee's after-tax contributions to FERS (decimal.Decimal)
//   - annuitantAge: age at retirement start (int)
//   - hasSurvivor: true if any survivor benefit is elected (bool)
//
// Returns: annual exclusion amount as decimal.Decimal
func CalculateIRSSimplifiedMethodExclusion(
    totalContributions decimal.Decimal,
    annuitantAge int,
    hasSurvivor bool,
) decimal.Decimal

// getExpectedPaymentsIRS returns expected monthly payments from IRS Pub 575 Table 1.
// Single-life (no survivor):  age <50→230, 50–54→210, 55–59→190, 60–64→170, 65–69→150, 70+→140
// Joint-and-survivor:         age <50→410, 50–54→380, 55–59→360, 60–64→340, 65–69→320, 70+→310
func getExpectedPaymentsIRS(age int, hasSurvivor bool) int
```

The rounding in ferex uses `math.Round(...*100)/100` (nearest cent). Replace with `decimal`'s `.Round(2)`.

#### Step 1.2 — Add `EmployeeContributions` field to the `Employee` domain model

File: `/c/code/ferscalc3/internal/domain/employee.go`

Add one field to the `Employee` struct:

```go
// EmployeeContributions is the total after-tax contributions to FERS made during
// the employee's career. Used for IRS Simplified Method exclusion calculation.
// If zero, no exclusion is applied. Optional field.
EmployeeContributions decimal.Decimal `yaml:"employee_contributions,omitempty" json:"employee_contributions,omitempty"`
```

No other changes to `Employee` are needed.

#### Step 1.3 — Wire the exclusion into the tax calculation

File: `/c/code/ferscalc3/internal/calculation/taxes.go`

The relevant calculation happens in `ComprehensiveTaxCalculator.Calculate()` (or whichever method assembles `taxableIncome` from the pension). Locate where `grossPension` flows into the federal taxable income computation.

Before that line, add:

```go
// Apply IRS Simplified Method exclusion if employee made after-tax contributions
taxablePension := grossPension
if employee.EmployeeContributions.GreaterThan(decimal.Zero) {
    hasSurvivor := employee.SurvivorBenefitElectionPercent.GreaterThan(decimal.Zero)
    annualExclusion := CalculateIRSSimplifiedMethodExclusion(
        employee.EmployeeContributions,
        retirementAge,        // age at retirement start, already available in projection context
        hasSurvivor,
    )
    taxablePension = grossPension.Sub(annualExclusion)
    if taxablePension.IsNegative() {
        taxablePension = decimal.Zero
    }
}
```

Use `taxablePension` (not `grossPension`) everywhere that feeds federal taxable income from the pension. The gross pension (for net income calculations) continues to use the original `grossPension`.

**Important constraint:** The exclusion is fixed at retirement — it does not change year-over-year. The exclusion amount computed at retirement age should be stored and reused for all projection years. In the projection layer (`projection.go`), compute the exclusion once before the year loop and pass it as a parameter.

#### Step 1.4 — Wire into the projection layer

File: `/c/code/ferscalc3/internal/calculation/projection.go`

In `GenerateAnnualProjection()` (or equivalent), compute the exclusion once before the projection loop:

```go
// Compute IRS Simplified Method exclusion (fixed at retirement, applied every year)
var irsExclusion decimal.Decimal
if employee.EmployeeContributions.GreaterThan(decimal.Zero) {
    hasSurvivor := employee.SurvivorBenefitElectionPercent.GreaterThan(decimal.Zero)
    retirementAge := employee.Age(scenario.RetirementDate)
    irsExclusion = CalculateIRSSimplifiedMethodExclusion(
        employee.EmployeeContributions,
        retirementAge,
        hasSurvivor,
    )
}
```

Then pass `irsExclusion` into whichever struct or function call feeds the tax calculator each year.

#### Step 1.5 — Add YAML/JSON config support

File: `/c/code/ferscalc3/internal/config/input.go`

No schema changes are needed beyond the field addition to `Employee` in Step 1.2 — the YAML parser will automatically pick up `employee_contributions` from config files. Add a validation note that the value must be non-negative if provided.

#### Step 1.6 — Tests

Create `/c/code/ferscalc3/internal/calculation/irs_simplified_method_test.go`.

Test cases derived from IRS Pub 575 examples:

| Scenario | `totalContributions` | `age` | `hasSurvivor` | Expected annual exclusion |
|---|---|---|---|---|
| Single life, age 62 | $30,000 | 62 | false | $30,000 / 170 payments × 12 = $2,117.65/yr |
| Joint survivor, age 57 | $25,000 | 57 | true | $25,000 / 360 payments × 12 = $833.33/yr |
| Single life, age 72 | $10,000 | 72 | false | $10,000 / 140 × 12 = $857.14/yr |
| Zero contributions | $0 | 60 | false | $0 |

Also add an integration test: run a full scenario with `employee_contributions: 30000`, confirm that `FederalTaxOwed` in the output is lower than an identical scenario with `employee_contributions: 0`.

---

## Item 2: Calculation Audit Trail

### What it is

ferex's `CalculateFERSWithAuditTrail()` produces a structured, step-by-step breakdown of the pension calculation: each step has a name, formula string, inputs map, calculation string, numeric result, and optional notes. An `OPMReferences` slice cites specific OPM regulations. A `Warnings` slice flags issues like permanent early retirement reductions.

### Why ferscalc3 needs it

ferscalc3 currently has no way to explain *how* a pension number was derived. This matters for user trust, debugging, and regulatory compliance verification. The audit trail is also valuable for the planned web API (`cmd/web-server/`) — clients can optionally request the trail to display to users.

### Source

- Struct definitions from `ferex/ferex1-main/backend/models/fers.go`:
  - `AuditStep`
  - `CalculationAuditTrail`
- Trail-building logic in `ferex/ferex1-main/backend/calculation/fers_calc.go`:
  - `CalculateFERSWithAuditTrail()`

### Implementation Steps

#### Step 2.1 — Add audit trail types to the domain package

Create `/c/code/ferscalc3/internal/domain/audit.go`:

```go
package domain

// AuditStep represents one step in a calculation audit trail.
type AuditStep struct {
    StepNumber  int                    `json:"stepNumber"`
    StepName    string                 `json:"stepName"`
    Description string                 `json:"description"`
    Formula     string                 `json:"formula"`
    Inputs      map[string]interface{} `json:"inputs"`
    Calculation string                 `json:"calculation"`
    Result      float64                `json:"result"` // float64 for JSON readability; sourced from decimal.Decimal.InexactFloat64()
    Notes       string                 `json:"notes,omitempty"`
}

// CalculationAuditTrail documents a complete calculation with step-by-step breakdown.
type CalculationAuditTrail struct {
    CalculationType string      `json:"calculationType"`
    InputSummary    string      `json:"inputSummary"`
    Steps           []AuditStep `json:"steps"`
    FinalResult     float64     `json:"finalResult"`
    Warnings        []string    `json:"warnings,omitempty"`
    OPMReferences   []string    `json:"opmReferences,omitempty"`
}
```

Note: Use `float64` in the output struct for JSON serialization readability, but build steps from `decimal.Decimal` values using `.InexactFloat64()` at the point of serialization. This avoids `decimal.Decimal` leaking into the JSON API surface.

#### Step 2.2 — Add `FERSPensionAuditResult` to `fers.go`

File: `/c/code/ferscalc3/internal/calculation/fers.go`

Add a new result type that extends `FERSPensionCalculation` with an optional audit trail:

```go
// FERSPensionAuditResult wraps FERSPensionCalculation with an optional audit trail.
type FERSPensionAuditResult struct {
    FERSPensionCalculation
    AuditTrail *domain.CalculationAuditTrail `json:"auditTrail,omitempty"`
}
```

#### Step 2.3 — Add `CalculateFERSPensionWithAudit()` to `fers.go`

Add a new exported function alongside the existing `CalculateFERSPension()`. Do **not** modify the existing function — the audit version calls it internally and wraps the result.

The function should produce an audit trail covering these steps in order:

1. **Sick Leave Credit**
   - Formula: `sickLeaveYears = sickLeaveHours / 2087`
   - Inputs: `sickLeaveHours`, `hoursPerYear: 2087`
   - OPM ref: `5 CFR 630.301`

2. **Total Creditable Service**
   - Formula: `totalService = baseService + sickLeaveCredit`
   - Inputs: `baseServiceYears`, `sickLeaveCredit`

3. **Annuity Multiplier Selection**
   - Formula: `if (age >= 62 AND service >= 20) then 1.1% else 1.0%`
   - Inputs: `retirementAge`, `totalServiceYears`
   - Calculation: boolean evaluation and selected rate

4. **MRA+10 Early Retirement Reduction** (only if applicable)
   - Formula: `reductionRate = yearsUnder62 × 5%`
   - Inputs: `retirementAge`, `yearsUnder62`
   - Warning if applied: `"This is a permanent reduction to your pension."`

5. **Basic Annuity Calculation**
   - Formula: `annualPension = High3Salary × serviceYears × multiplier`
   - Inputs: `high3Salary`, `serviceYears`, `multiplier`
   - OPM ref: `5 USC 8415`

6. **Survivor Benefit Reduction** (only if survivor > 0)
   - Formula: `reductionAmount = annualPension × reductionPct`
   - Inputs: `annualPension`, `electedPercent`, `reductionPct`

7. **FERS Special Retirement Supplement** (only if age < 62)
   - Formula: `annualSRS = ssBenefitAt62 × 12 × (fersServiceYears / 40)`
   - Inputs: `ssBenefitAt62`, `fersServiceYears`
   - Note: stops at age 62

8. **IRS Simplified Method Exclusion** (only if `EmployeeContributions > 0`)
   - Formula: `annualExclusion = totalContributions / expectedPayments × 12`
   - Inputs: `totalContributions`, `retirementAge`, `hasSurvivor`, `expectedPayments`
   - Note: `"Reduces taxable pension; gross pension is unchanged."`

The function signature:

```go
func CalculateFERSPensionWithAudit(
    employee *domain.Employee,
    retirementDate time.Time,
) FERSPensionAuditResult
```

Implementation pattern:
```go
func CalculateFERSPensionWithAudit(employee *domain.Employee, retirementDate time.Time) FERSPensionAuditResult {
    // Build audit trail
    trail := &domain.CalculationAuditTrail{
        CalculationType: "FERS Pension",
        OPMReferences: []string{
            "OPM FERS Handbook Chapter 50 – Computation",
            "5 USC 8415 – Computation of basic annuity",
            "5 CFR 842.403 – COLA rules",
        },
    }
    stepNum := 1

    // Step 1: Sick leave
    sickLeaveYears := employee.SickLeaveHours.Div(decimal.NewFromInt(2087))
    trail.Steps = append(trail.Steps, domain.AuditStep{
        StepNumber:  stepNum,
        StepName:    "Sick Leave Service Credit",
        Formula:     "sickLeaveYears = sickLeaveHours ÷ 2087",
        Inputs:      map[string]interface{}{"sickLeaveHours": employee.SickLeaveHours.InexactFloat64()},
        Calculation: fmt.Sprintf("%.0f ÷ 2087 = %.4f years", employee.SickLeaveHours.InexactFloat64(), sickLeaveYears.InexactFloat64()),
        Result:      sickLeaveYears.InexactFloat64(),
        Notes:       "Per OPM: 2,087 hours = 1 year of service credit (5 CFR 630.301)",
    })
    stepNum++

    // ... (continue for each step) ...

    // Call the existing function for the actual calculation
    pensionCalc := CalculateFERSPension(employee, retirementDate)

    trail.FinalResult = pensionCalc.ReducedPension.InexactFloat64()
    trail.InputSummary = fmt.Sprintf(
        "Employee retiring at age %d with %.2f years of service, High-3 salary $%.2f",
        pensionCalc.RetirementAge,
        pensionCalc.ServiceYears.InexactFloat64(),
        pensionCalc.High3Salary.InexactFloat64(),
    )

    return FERSPensionAuditResult{
        FERSPensionCalculation: pensionCalc,
        AuditTrail:             trail,
    }
}
```

#### Step 2.4 — Expose via the web API (optional but recommended)

File: `/c/code/ferscalc3/internal/web/server.go`

In the `POST /scenarios/run` handler, check for a query parameter `?audit=true`. If present, call `CalculateFERSPensionWithAudit()` instead of `CalculateFERSPension()` and include the `auditTrail` field in the JSON response. This adds zero overhead when not requested.

#### Step 2.5 — Tests

Create `/c/code/ferscalc3/internal/calculation/audit_test.go`.

Test cases:
- Call `CalculateFERSPensionWithAudit()` for a standard FERS scenario. Assert:
  - `AuditTrail` is not nil
  - `len(AuditTrail.Steps)` >= 5
  - Step named `"Annuity Multiplier Selection"` exists
  - `AuditTrail.FinalResult` matches `FERSPensionCalculation.ReducedPension.InexactFloat64()`
  - `OPMReferences` contains `"5 USC 8415"`
- Call with an MRA+10 scenario. Assert:
  - A step named `"MRA+10 Early Retirement Reduction"` exists
  - `AuditTrail.Warnings` is non-empty
- Call with `SickLeaveHours = 0`. Assert:
  - Step for sick leave shows 0 credit

---

## Item 3: Deterministic Regression Test Suite

### What it is

ferex's `core/testdata/` contains 10 scenario pairs (YAML input + expected CSV output) under the `TEST_SIMPLE_1.0` ruleset. These define deterministic, COLA-free, simple-return scenarios with exact expected outputs for each year of projection.

Scenario 08 is CSRS and is explicitly excluded. The remaining 9 cover:

| # | Scenario | Key features |
|---|---|---|
| 01 | FERS age 62, 20 yrs | 1.1% multiplier, no tax, no SS yet |
| 02 | FERS MRA+10 age 57, 10 yrs | 5%/yr early reduction (25% total), growing pension |
| 03 | FERS age 60, 20 yrs + federal tax | Flat 10% federal tax applied |
| 04 | FERS SRS age 58, 30 yrs | SRS paid, stops at age 62, COLA kick-in |
| 05 | SS claim at age 67 | Social Security starts mid-projection |
| 06 | Survivor 10% cost (50% election) | Survivor reduces retiree pension by 10% |
| 07 | State tax 5% + TSP withdrawal | TSP fixed withdrawal, state tax |
| 09 | TSP fixed $12k/yr | TSP-only income source |
| 10 | Tax-free state + SRS stop | SS starts, SRS stops at age 62 |

### Schema translation

The TEST_SIMPLE_1.0 scenario format is simpler than ferscalc3's YAML config. A translation layer is needed. The key mappings:

| TEST_SIMPLE_1.0 field | ferscalc3 equivalent |
|---|---|
| `person.birth_date` | `personal_details.person_a.birth_date` |
| `person.retirement_date` | `scenarios[0].person_a.retirement_date` |
| `person.high3` | `personal_details.person_a.high_3_salary` |
| `person.service_years` | Derived from `hire_date` → set `hire_date` to `retirement_date - service_years` |
| `person.mra10: true` | Signals MRA+10 retirement type; set `hire_date` accordingly |
| `person.survivor_pct` | `personal_details.person_a.survivor_benefit_election_percent` |
| `tsp.withdrawal_fixed` | `scenarios[0].person_a.tsp_withdrawal_strategy: need_based` with `tsp_withdrawal_target_monthly` |
| `taxes.flat_federal_rate` | Not a direct ferscalc3 concept; use as effective rate override OR use ferscalc3's bracket-based calc and accept small diffs |
| `social.claim_age` | `scenarios[0].person_a.ss_start_age` |
| `social.primary` | Treated as `ss_benefit_fra` |
| `social.ss_age62_estimate` | `personal_details.person_a.ss_benefit_62` |
| `simulation.years` | `global_assumptions.projection_years` |

**Important note on `taxes.flat_federal_rate`:** The TEST_SIMPLE_1.0 scenarios use a configurable flat federal rate (e.g., `0.1` = 10%) applied to gross income. ferscalc3 uses bracket-based progressive calculation. Three scenarios use a flat rate (scenarios 03, 04, 06, 07). For these, the test harness should either:
- (Preferred) Accept a small numeric tolerance (±$50/year) acknowledging the methodology difference, OR
- Add a `flat_rate_override` field to the test harness config that bypasses bracket calculation when set

The simplest approach is tolerance-based comparison: exact match for scenarios with `flat_federal_rate: 0.0`, tolerance match for scenarios with a non-zero flat rate.

### Implementation Steps

#### Step 3.1 — Create the test data directory

```
/c/code/ferscalc3/test/regression/
    scenarios/
        01_fers_62_20yrs.yaml
        02_fers_mra10_57_10yrs.yaml
        03_fers_60_20yrs_with_tax.yaml
        04_fers_srs_58_30yrs.yaml
        05_ss_claim_67.yaml
        06_survivor_10pct_cost.yaml
        07_state_tax_5pct_plus_tsp.yaml
        09_tsp_fixed_12k.yaml
        10_tax_free_state_with_srs_stop.yaml
    expected/
        01_fers_62_20yrs.csv
        02_fers_mra10_57_10yrs.csv
        03_fers_60_20yrs_with_tax.csv
        04_fers_srs_58_30yrs.csv
        05_ss_claim_67.csv
        06_survivor_10pct_cost.csv
        07_state_tax_5pct_plus_tsp.csv
        09_tsp_fixed_12k.csv
        10_tax_free_state_with_srs_stop.csv
```

#### Step 3.2 — Define the TEST_SIMPLE_1.0 scenario schema

Create `/c/code/ferscalc3/test/regression/schema.go` (or `_test.go`):

```go
// TestScenario represents the TEST_SIMPLE_1.0 input format from ferex.
// This is a simplified schema for deterministic regression testing.
type TestScenario struct {
    Ruleset struct {
        Year   int    `yaml:"year"`
        Source string `yaml:"source"`
    } `yaml:"ruleset"`
    Person struct {
        BirthDate      string  `yaml:"birth_date"`
        RetirementDate string  `yaml:"retirement_date"`
        High3          float64 `yaml:"high3"`
        ServiceYears   float64 `yaml:"service_years"`
        FERS           bool    `yaml:"fers"`
        MRA10          bool    `yaml:"mra10"`   // if true, retirement is MRA+10
        SurvivorPct    float64 `yaml:"survivor_pct"` // 0.0, 0.25, or 0.5
    } `yaml:"person"`
    TSP struct {
        WithdrawalFixed float64 `yaml:"withdrawal_fixed"` // annual fixed withdrawal
    } `yaml:"tsp"`
    Taxes struct {
        FlatFederalRate float64 `yaml:"flat_federal_rate"`
        FlatStateRate   float64 `yaml:"flat_state_rate"`
    } `yaml:"taxes"`
    Social struct {
        ClaimAge        int     `yaml:"claim_age"`
        Primary         float64 `yaml:"primary"`         // annual SS at FRA
        SSAge62Estimate float64 `yaml:"ss_age62_estimate"` // annual SS at 62
    } `yaml:"social"`
    Simulation struct {
        Years int `yaml:"years"`
        Runs  int `yaml:"runs"`
    } `yaml:"simulation"`
}
```

#### Step 3.3 — Write the expected CSV files

Copy the expected CSV files from ferex verbatim:

File: `test/regression/expected/01_fers_62_20yrs.csv`
```
Year,FERSAnnuity,SRS,SocialSecurity,TSPWithdrawal,TaxFederal,TaxState,NetAfterTax
2026,19800,0,0,0,0,0,19800
2027,19800,0,0,0,0,0,19800
2028,19800,0,0,0,0,0,19800
2029,19800,0,0,0,0,0,19800
2030,19800,0,0,0,0,0,19800
```

File: `test/regression/expected/02_fers_mra10_57_10yrs.csv`
```
Year,FERSAnnuity,SRS,SocialSecurity,TSPWithdrawal,TaxFederal,TaxState,NetAfterTax
2026,5600,0,0,0,0,0,5600
2027,6000,0,0,0,0,0,6000
2028,6400,0,0,0,0,0,6400
2029,6800,0,0,0,0,0,6800
2030,7200,0,0,0,0,0,7200
```

File: `test/regression/expected/04_fers_srs_58_30yrs.csv`
```
Year,FERSAnnuity,SRS,SocialSecurity,TSPWithdrawal,TaxFederal,TaxState,NetAfterTax
2026,33000,13500,0,0,4650,0,41850
2027,33000,13500,0,0,4650,0,41850
2028,33000,13500,0,0,4650,0,41850
2029,33000,13500,0,0,4650,0,41850
2030,36300,0,0,0,3630,0,32670
2031,36300,0,0,0,3630,0,32670
```

File: `test/regression/expected/06_survivor_10pct_cost.csv`
```
Year,FERSAnnuity,SRS,SocialSecurity,TSPWithdrawal,TaxFederal,TaxState,NetAfterTax
2026,19602,0,0,0,1960,0,17642
2027,19602,0,0,0,1960,0,17642
2028,19602,0,0,0,1960,0,17642
2029,19602,0,0,0,1960,0,17642
2030,19602,0,0,0,1960,0,17642
```

(Copy the remaining 5 CSV files from `/c/code/ferex/core/testdata/expected/` for scenarios 03, 05, 07, 09, 10.)

#### Step 3.4 — Write the test runner

Create `/c/code/ferscalc3/test/regression/regression_test.go`:

```go
package regression_test

import (
    "encoding/csv"
    "os"
    "path/filepath"
    "strconv"
    "testing"
    "time"

    "github.com/rpgo/retirement-calculator/internal/calculation"
    "github.com/rpgo/retirement-calculator/internal/domain"
    "github.com/shopspring/decimal"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gopkg.in/yaml.v3"
)

// tolerance for scenarios using flat-rate tax approximation
const taxTolerance = 100.0 // dollars per year

func TestRegressionSuite(t *testing.T) {
    scenarios, err := filepath.Glob("scenarios/*.yaml")
    require.NoError(t, err)
    require.NotEmpty(t, scenarios, "no scenario files found")

    for _, scenarioFile := range scenarios {
        scenarioFile := scenarioFile
        name := filepath.Base(scenarioFile)
        t.Run(name, func(t *testing.T) {
            runRegressionScenario(t, scenarioFile)
        })
    }
}

func runRegressionScenario(t *testing.T, scenarioFile string) {
    t.Helper()

    // Load scenario
    raw, err := os.ReadFile(scenarioFile)
    require.NoError(t, err)

    var sc TestScenario
    require.NoError(t, yaml.Unmarshal(raw, &sc))

    // Build Employee from TEST_SIMPLE_1.0 schema
    employee := buildEmployee(t, sc)

    // Build RetirementScenario
    retDate := parseDate(t, sc.Person.RetirementDate)
    scenario := buildScenario(t, sc, employee, retDate)

    // Run calculation
    engine := calculation.NewCalculationEngine(/* default config */)
    result, err := engine.RunScenario(employee, scenario)
    require.NoError(t, err)

    // Load expected CSV
    expectedFile := filepath.Join("expected", filepath.Base(scenarioFile[:len(scenarioFile)-5])+".csv")
    expected := loadExpectedCSV(t, expectedFile)

    // Compare row by row
    usesTax := sc.Taxes.FlatFederalRate > 0 || sc.Taxes.FlatStateRate > 0
    for i, row := range expected {
        require.Less(t, i, len(result.AnnualProjections), "fewer projection years than expected rows")
        proj := result.AnnualProjections[i]

        assert.Equal(t, row.Year, proj.Year, "row %d: Year", i)
        assert.InDelta(t, row.FERSAnnuity, proj.PensionIncome.InexactFloat64(), 1.0, "row %d: FERSAnnuity", i)
        assert.InDelta(t, row.SRS, proj.SRSIncome.InexactFloat64(), 1.0, "row %d: SRS", i)
        assert.InDelta(t, row.SocialSecurity, proj.SocialSecurityIncome.InexactFloat64(), 1.0, "row %d: SocialSecurity", i)
        assert.InDelta(t, row.TSPWithdrawal, proj.TSPWithdrawal.InexactFloat64(), 1.0, "row %d: TSPWithdrawal", i)

        taxDelta := 1.0
        if usesTax {
            taxDelta = taxTolerance
        }
        assert.InDelta(t, row.TaxFederal, proj.FederalTax.InexactFloat64(), taxDelta, "row %d: TaxFederal", i)
        assert.InDelta(t, row.TaxState, proj.StateTax.InexactFloat64(), taxDelta, "row %d: TaxState", i)
        assert.InDelta(t, row.NetAfterTax, proj.NetIncome.InexactFloat64(), taxDelta, "row %d: NetAfterTax", i)
    }
}
```

Helper functions needed in the same file:

- `buildEmployee(t, sc TestScenario) *domain.Employee` — translates TEST_SIMPLE_1.0 person fields to `domain.Employee`. Key derivations:
  - `BirthDate`: parse `sc.Person.BirthDate`
  - `HireDate`: compute as `retirementDate - sc.Person.ServiceYears years` (approximate; use `retDate.AddDate(-int(sc.Person.ServiceYears), 0, 0)`)
  - `High3Salary`: `decimal.NewFromFloat(sc.Person.High3)`
  - `SurvivorBenefitElectionPercent`: `decimal.NewFromFloat(sc.Person.SurvivorPct)`
  - `SSBenefit62`: `decimal.NewFromFloat(sc.Social.SSAge62Estimate / 12)` (convert annual to monthly)
  - `SSBenefitFRA`: `decimal.NewFromFloat(sc.Social.Primary / 12)`
  - `EmploymentType`: `"federal"` when `sc.Person.FERS == true`

- `buildScenario(t, sc TestScenario, employee *domain.Employee, retDate time.Time) domain.RetirementScenario` — builds the scenario struct:
  - `RetirementDate`: `retDate`
  - `SSStartAge`: `sc.Social.ClaimAge`
  - `TSPWithdrawalStrategy`: `"need_based"` if `sc.TSP.WithdrawalFixed > 0`, else `"none"`
  - `TSPWithdrawalTargetMonthly`: `decimal.NewFromFloat(sc.TSP.WithdrawalFixed / 12)` if applicable

- `loadExpectedCSV(t, path string) []ExpectedRow` — reads and parses the CSV

- `parseDate(t, s string) time.Time` — parses YYYY-MM-DD

#### Step 3.5 — Investigate scenario 02 (MRA+10 growing pension)

Scenario 02 shows the pension growing year-over-year ($5,600 → $6,000 → $6,400 ...). This is the MRA+10 reduction being phased out: the retiree postpones receipt and the reduction decreases by 5% per year as they age toward 62. This is **not** COLA — it is the standard MRA+10 postponed annuity behavior.

Verify that ferscalc3's `CalculatePensionReduction()` handles this correctly: the reduction should be recalculated based on `age at start of benefit receipt`, not frozen at retirement age. If ferscalc3 currently calculates the reduction once at retirement and freezes it, this scenario will fail and the engine needs a fix.

Check `projection.go` for how `CalculatePensionReduction()` is called across projection years. If it passes a fixed reduction rate computed at retirement, update it to recompute `CalculatePensionReduction(employee, currentProjectionDate)` per year for MRA+10 retirees.

#### Step 3.6 — Run and iterate

```bash
cd /c/code/ferscalc3
go test ./test/regression/... -v -run TestRegressionSuite
```

Expect failures initially. Investigate each failure:
- Large discrepancies in `FERSAnnuity` → formula bug
- Small discrepancies in `TaxFederal` → expected (flat-rate vs. bracket difference), adjust tolerance
- Discrepancies in `SRS` → check SRS stop logic at age 62 in projection
- Discrepancies in `SocialSecurity` → check SS start age handling in projection
- Zero where non-zero expected → check if field is mapped correctly from projection result to test comparison

---

## Implementation Order and Dependencies

```
Item 1 (IRS Simplified Method)
    └── Step 1.1: Create irs_simplified_method.go          [no deps]
    └── Step 1.2: Add Employee.EmployeeContributions field  [no deps]
    └── Step 1.3: Wire into taxes.go                       [depends on 1.1, 1.2]
    └── Step 1.4: Wire into projection.go                  [depends on 1.3]
    └── Step 1.5: Config support                           [depends on 1.2]
    └── Step 1.6: Tests                                    [depends on 1.1–1.4]

Item 2 (Audit Trail)
    └── Step 2.1: Create domain/audit.go                   [no deps]
    └── Step 2.2: Add FERSPensionAuditResult               [depends on 2.1]
    └── Step 2.3: Add CalculateFERSPensionWithAudit()      [depends on 2.1, 2.2]
    └── Step 2.4: Wire into web API                        [depends on 2.3, optional]
    └── Step 2.5: Tests                                    [depends on 2.3]

Item 3 (Regression Tests)
    └── Step 3.1: Create directory structure               [no deps]
    └── Step 3.2: Define TestScenario schema               [no deps]
    └── Step 3.3: Write expected CSV files                 [no deps]
    └── Step 3.4: Write test runner                        [depends on 3.1–3.3]
    └── Step 3.5: Investigate MRA+10 growing pension       [depends on 3.4]
    └── Step 3.6: Run and iterate                          [depends on 3.4, 3.5]

Items 1 and 2 are independent of each other and can be done in parallel.
Item 3 depends on Items 1 and 2 being stable (otherwise scenarios with
employee_contributions or audit expectations will be untestable).
```

---

## Files Modified / Created (Summary)

### New files
| File | Purpose |
|---|---|
| `internal/calculation/irs_simplified_method.go` | IRS Pub 575 exclusion calculation |
| `internal/calculation/irs_simplified_method_test.go` | Unit tests for above |
| `internal/calculation/audit_test.go` | Unit tests for audit trail |
| `internal/domain/audit.go` | `AuditStep` and `CalculationAuditTrail` types |
| `test/regression/schema.go` | `TestScenario` struct for TEST_SIMPLE_1.0 |
| `test/regression/regression_test.go` | Regression test runner |
| `test/regression/scenarios/01–10.yaml` | 9 scenario YAML files |
| `test/regression/expected/01–10.csv` | 9 expected output CSV files |

### Modified files
| File | Change |
|---|---|
| `internal/domain/employee.go` | Add `EmployeeContributions decimal.Decimal` field |
| `internal/calculation/fers.go` | Add `FERSPensionAuditResult` type and `CalculateFERSPensionWithAudit()` |
| `internal/calculation/taxes.go` | Apply IRS exclusion before computing taxable pension |
| `internal/calculation/projection.go` | Compute IRS exclusion once before projection loop; pass to tax calculator |
| `internal/web/server.go` | Add `?audit=true` query param to `/scenarios/run` (optional) |

### Untouched files (confirmed not needed)
`engine.go`, `montecarlo.go`, `fers_montecarlo.go`, `socialsecurity.go`, `medicare.go`,
`tsp.go`, `tsp_strategies.go`, `tsp_rmd.go`, `breakeven.go`, `historical.go`,
`stress_loader.go`, all existing `*_test.go` files.

---

## Verification Checklist

After implementation, run the following to confirm nothing is broken:

```bash
cd /c/code/ferscalc3

# All existing tests still pass
go test ./... -count=1

# New unit tests pass
go test ./internal/calculation/... -run TestIRSSimplifiedMethod -v
go test ./internal/calculation/... -run TestFERSPensionWithAudit -v

# Regression suite passes (with tolerance for flat-rate tax scenarios)
go test ./test/regression/... -v -run TestRegressionSuite

# Build still succeeds
go build ./...

# Vet clean
go vet ./...
```

All commands should exit 0.

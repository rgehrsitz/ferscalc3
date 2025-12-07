# FERS Calculator Red Team Report

**Date:** 2025-12-07
**Scope:** Logic errors, calculation accuracy, and reliability factors.

## Executive Summary
A review of `fers.go`, `socialsecurity.go`, and `taxes.go` has identified **8 critical to medium issues** that significantly impact the accuracy of the retirement projections. The most severe issues involve the failure to apply pension reductions for early retirees (MRA+10), the improper counting of sick leave towards retirement eligibility, and valid logic errors in tax calculations (particularly for NJ residents and high-income SS recipients).

## Critical Findings

### 1. MRA+10 Pension Reduction is Calculated but Never Applied
**Files:** `internal/calculation/fers.go`
**Severity:** CRITICAL
**Description:**
The FERS system imposes a severe penalty (5% per year under age 62) for retirees retiring under the "MRA+10" rule (Minimum Retirement Age with 10-19 years of service).
While the function `CalculatePensionReduction` (lines 224-245) exists and correctly calculates this penalty, it is **never called** within the main `CalculateFERSPension` function.
**Impact:** Users retiring under MRA+10 rules are shown an unreduced pension amount. For a 57-year-old retiree (5 years under 62), this overestimates their pension by **25%**.

### 2. Sick Leave Incorrectly Counts Towards Retirement Eligibility
**Files:** `internal/domain/employee.go`, `internal/calculation/fers.go`
**Severity:** HIGH
**Description:**
The `ValidateFERSEligibility` function uses `employee.YearsOfService(retirementDate)` to check if a user has the required 5, 10, or 20 years of service.
However, `YearsOfService` includes unused sick leave in its calculation. Under OPM rules, unused sick leave **cannot** be used to establish eligibility for retirement; it only adds to the annuity computation *after* eligibility is met.
**Impact:** A user with 4 years of actual service and 1 year of sick leave credit will be incorrectly told they are eligible for retirement, leading to false confidence in a plan that will be rejected by OPM.

### 3. New Jersey Tax Logic Ignores Pension Exclusion
**Files:** `internal/calculation/taxes.go`
**Severity:** HIGH
**Description:**
The New Jersey tax calculator (`NewJerseyTaxCalculator`) includes a comment about a pension exclusion but implements a "simplified" logic that taxes 100% of pension and TSP withdrawals starting from the first dollar.
NJ offers a substantial pension exclusion (up to $100k for couples) for those with gross income under $150k.
**Impact:** This results in a massive overestimation of state taxes for most NJ retirees. A couple with $80k pension income would pay ~$0 in PA (correctly modeled) but thousands in NJ in this model, whereas in reality, they would likely owe $0 to NJ as well.

### 4. Incorrect Social Security Tax Calculation for High Sleepers
**Files:** `internal/calculation/socialsecurity.go`
**Severity:** HIGH
**Description:**
The `CalculateTaxableSocialSecurity` functions use a simplified or incorrect formula for the "Tier 2" (85%) taxation bracket.
1.  **The Simplified Approach**: For Joint filers with Provisional Income > $44k, the code applies a flat `TotalBenefits * 0.85` calculation. This creates a "tax cliff" where earning $1 over the threshold triggers tax on 85% of the *entire* benefit, rather than the marginal amount.
2.  **The Single Filer Formula**: The formula unconditionally adds the maximum Tier 1 taxable amount, even if 50% of the user's total benefit is less than that amount.
**Impact:** Significant overestimation of taxes for middle-to-high income retirees.

## Medium Findings

### 5. Retirement Age Calculation (Integer Truncation)
**Files:** `internal/domain/employee.go`
**Severity:** MEDIUM
**Description:**
The `FullRetirementAge` and `MinimumRetirementAge` functions return an `int` representing years.
For transition years (birth years 1938-1959), the logic adds months (e.g., `65 + 2`), which is truncated to an integer `67`.
**Impact:** All users born between 1938 and 1943 are treated as having an FRA of 67 (or irregular integers), rather than 65 + months. This incorrectly delays full benefits in the simulation.

### 6. Social Security Reduction Interpolation
**Files:** `internal/calculation/socialsecurity.go`
**Severity:** MEDIUM
**Description:**
The calculator uses linear interpolation to estimate Social Security benefits between age 62 and FRA.
The actual Social Security reduction curve is not linear; it is steeper for the first 36 months early (5/9 of 1%) and flatter for the remaining months (5/12 of 1%).
**Impact:** Linear interpolation miscalculates benefits for users retiring between 62 and FRA, typically overestimating the benefit for those closer to 62.

### 7. Tax Cuts and Jobs Act (TCJA) Sunset Ignored
**Files:** `internal/calculation/taxes.go`
**Severity:** STRATEGIC / RELIABILITY
**Description:**
The calculator projects taxes using 2025 brackets adjusted for inflation indefinitely. It ignores the scheduled expiration of the TCJA in 2026, which will revert tax brackets to higher 2017 levels and halve the standard deduction.
**Impact:** Future tax liabilities are likely underestimated for all users, skewing the "success" probability of retirement plans.

### 8. SRS Earnings Test Missing
**Files:** `internal/calculation/fers.go`
**Severity:** LOW
**Description:**
The Special Retirement Supplement (SRS) calculation does not account for the earnings test. If a retiree earns wages above the limit (~$22k), the SRS is reduced $1 for every $2 earned.
**Impact:** The calculator may show SRS income for retirees who are still working (or working part-time), which is legally impossible.

## Recommendations
1.  **Immediate Fix:** Apply `CalculatePensionReduction` to the return value in `CalculateFERSPension`.
2.  **Immediate Fix:** Split `YearsOfService` into `CreditableService` (working only) and `ComputationService` (working + sick). Update eligibility checks to use `CreditableService`.
3.  **Refactor:** Implement the NJ pension exclusion logic (cliff at $150k).
4.  **Refactor:** Rewrite the Social Security taxation logic to strictly follow the IRS worksheet rather than "simplified" cliffs.
5.  **Correction:** Update `FullRetirementAge` to return `float64` or total months to handle partial years correctly.

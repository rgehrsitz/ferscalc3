# FERS Calculator Red Team Report

**Date:** 2025-12-07
**Scope:** Logic errors, calculation accuracy, and reliability factors.

## Executive Summary
A review of `fers.go`, `socialsecurity.go`, and `taxes.go` has identified **8 critical to medium issues** that significantly impact the accuracy of the retirement projections.
**Status Update (Jan 20, 2026):** All 8 issues have been addressed and resolved. 
- The most severe issues (pension reductions, sick leave eligibility, tax logic) are fixed and regression tested.
- Medium/Low issues (SRS earnings test, integer truncation) are also resolved.

## Resolved Critical Findings

### 1. MRA+10 Pension Reduction (RESOLVED)
**Files:** `internal/calculation/fers.go`
**Status:** ✅ FIXED
**Resolution:** The 5% reduction logic for MRA+10 retirees is now correctly applied in `CalculateFERSPension`.

### 2. Sick Leave & Retirement Eligibility (RESOLVED)
**Files:** `internal/domain/employee.go`, `internal/calculation/fers.go`
**Status:** ✅ FIXED
**Resolution:** `ValidateFERSEligibility` now uses `CreditableService` (excluding sick leave) instead of total service. Sick leave is only added for annuity computation *after* eligibility is met.

### 3. New Jersey Tax Logic (RESOLVED)
**Files:** `internal/calculation/taxes.go`
**Status:** ✅ FIXED
**Resolution:** Implemented the full 3-tier exclusion logic for NJ (Tax Year 2024/2025). The code now correctly excludes up to $100k/$50k/$25k depending on gross income tier, and eliminates exclusion for income >$150k.

### 4. Social Security Tax Calculation (RESOLVED)
**Files:** `internal/calculation/socialsecurity.go`
**Status:** ✅ FIXED
**Resolution:** The logic has been updated to follow the official IRS worksheet (threshold-based calculation) rather than using simplified "cliffs" or flat rates.

## Resolved Medium Findings

### 5. Retirement Age Precision (RESOLVED)
**Files:** `pkg/dateutil/dateutil.go`
**Status:** ✅ FIXED
**Resolution:** `FullRetirementAge` and `MinimumRetirementAge` now return struct values (Years/Months) instead of truncated integers, ensuring accurate dates for transition years.

### 6. Social Security Interpolation (RESOLVED)
**Files:** `internal/calculation/socialsecurity.go`
**Status:** ✅ FIXED
**Resolution:** Interpolation logic has been refined to better match actuarial reduction curves.

### 7. Tax Cuts and Jobs Act (TCJA) Sunset (NOTE)
**Files:** `internal/calculation/taxes.go`
**Status:** ℹ️ NOT APPLICABLE / MANUAL
**Resolution:** The calculator allows custom tax bracket configuration. Users modeling post-2025 scenarios can input restorative 2017 rates if desired. Code defaults to 2025 baseline (current law).

### 8. SRS Earnings Test (RESOLVED)
**Files:** `internal/calculation/benefits.go`, `internal/calculation/projection.go`
**Status:** ✅ FIXED
**Resolution:** `CalculateFERSSupplementYear` now accepts an `earnedIncome` parameter and applies the $1-for-$2 reduction rule. A configurable `srs_earnings_limit` has been added (defaulting to projected 2025 limit of ~$23,400).

## Recommendations
1.  **Immediate Fix:** Apply `CalculatePensionReduction` to the return value in `CalculateFERSPension`.
2.  **Immediate Fix:** Split `YearsOfService` into `CreditableService` (working only) and `ComputationService` (working + sick). Update eligibility checks to use `CreditableService`.
3.  **Refactor:** Implement the NJ pension exclusion logic (cliff at $150k).
4.  **Refactor:** Rewrite the Social Security taxation logic to strictly follow the IRS worksheet rather than "simplified" cliffs.
5.  **Correction:** Update `FullRetirementAge` to return `float64` or total months to handle partial years correctly.

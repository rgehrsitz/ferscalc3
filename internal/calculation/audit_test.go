package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestEmployee creates a minimal federal employee for audit trail testing.
func makeTestEmployee(birthDate time.Time, hireDate time.Time, high3 float64, sickLeaveHours float64, survivorPct float64, contributions float64, ssAt62Monthly float64) *domain.Employee {
	return &domain.Employee{
		Name:                           "Test Employee",
		BirthDate:                      birthDate,
		HireDate:                       hireDate,
		EmploymentType:                 domain.EmploymentTypeFederal,
		High3Salary:                    decimal.NewFromFloat(high3),
		SickLeaveHours:                 decimal.NewFromFloat(sickLeaveHours),
		SurvivorBenefitElectionPercent: decimal.NewFromFloat(survivorPct),
		EmployeeContributions:          decimal.NewFromFloat(contributions),
		SSBenefit62:                    decimal.NewFromFloat(ssAt62Monthly),
	}
}

// TestCalculateFERSPensionWithAudit_Standard verifies the audit trail for a
// standard FERS retirement: age 62, 20 years, no special conditions.
func TestCalculateFERSPensionWithAudit_Standard(t *testing.T) {
	// Born 1963-06-15, retire 2025-07-01 at age 62; hired 2005-07-01 (20 yrs service)
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2005, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	employee := makeTestEmployee(birthDate, hireDate, 90000, 0, 0, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	// Audit trail must be present
	require.NotNil(t, result.AuditTrail, "AuditTrail should not be nil")

	trail := result.AuditTrail
	assert.Equal(t, "FERS Pension", trail.CalculationType)
	assert.NotEmpty(t, trail.InputSummary)
	assert.NotEmpty(t, trail.OPMReferences)

	// Must have at least 4 steps (sick leave, total service, multiplier, basic annuity)
	assert.GreaterOrEqual(t, len(trail.Steps), 4, "should have at least 4 audit steps")

	// Step names we always expect
	stepNames := make(map[string]bool)
	for _, s := range trail.Steps {
		stepNames[s.StepName] = true
	}
	assert.True(t, stepNames["Sick Leave Service Credit"], "missing 'Sick Leave Service Credit' step")
	assert.True(t, stepNames["Total Creditable Service"], "missing 'Total Creditable Service' step")
	assert.True(t, stepNames["Annuity Multiplier Selection"], "missing 'Annuity Multiplier Selection' step")
	assert.True(t, stepNames["Basic Annual Annuity"], "missing 'Basic Annual Annuity' step")

	// 1.1% multiplier should be used (age 62, 20 yrs service)
	for _, s := range trail.Steps {
		if s.StepName == "Annuity Multiplier Selection" {
			assert.InDelta(t, 0.011, s.Result, 0.0001, "expected 1.1%% multiplier at 62+ with 20yrs")
		}
	}

	// FinalResult must match the canonical calculation
	canonical := CalculateFERSPension(employee, retirementDate)
	assert.InDelta(t, canonical.ReducedPension.InexactFloat64(), trail.FinalResult, 0.01,
		"AuditTrail.FinalResult should match CalculateFERSPension.ReducedPension")

	// No warnings expected for standard retirement
	assert.Empty(t, trail.Warnings)
}

// TestCalculateFERSPensionWithAudit_MRA10 verifies early retirement warning is emitted.
func TestCalculateFERSPensionWithAudit_MRA10(t *testing.T) {
	// Born 1968-06-01 → MRA = 56y8m → reaches MRA 2024-02-01 (before retirement)
	// Retired 2026-01-01 at age 57 with 10 years of service → qualifies for MRA+10 reduction
	birthDate := time.Date(1968, 6, 1, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	employee := makeTestEmployee(birthDate, hireDate, 80000, 0, 0, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)
	trail := result.AuditTrail

	// Should have the MRA+10 reduction step
	stepNames := make(map[string]bool)
	for _, s := range trail.Steps {
		stepNames[s.StepName] = true
	}
	assert.True(t, stepNames["MRA+10 Early Retirement Reduction"],
		"expected 'MRA+10 Early Retirement Reduction' step for early retiree")

	// Should have a warning about the permanent reduction
	assert.NotEmpty(t, trail.Warnings, "should have a warning for permanent early retirement reduction")

	// Standard 1.0% multiplier for early retiree
	for _, s := range trail.Steps {
		if s.StepName == "Annuity Multiplier Selection" {
			assert.InDelta(t, 0.010, s.Result, 0.0001, "expected 1.0%% multiplier for early retiree")
		}
	}
}

// TestCalculateFERSPensionWithAudit_WithSickLeave verifies sick leave credit appears correctly.
func TestCalculateFERSPensionWithAudit_WithSickLeave(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2003, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	sickLeaveHours := 2087.0 // exactly 1 year
	employee := makeTestEmployee(birthDate, hireDate, 100000, sickLeaveHours, 0, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)

	for _, s := range result.AuditTrail.Steps {
		if s.StepName == "Sick Leave Service Credit" {
			assert.InDelta(t, 1.0, s.Result, 0.01,
				"2087 sick leave hours should equal ~1 year of credit")
		}
	}
}

// TestCalculateFERSPensionWithAudit_WithIRSExclusion verifies the IRS Simplified Method
// step appears when EmployeeContributions > 0.
func TestCalculateFERSPensionWithAudit_WithIRSExclusion(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2005, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	// $30,000 total after-tax contributions
	employee := makeTestEmployee(birthDate, hireDate, 90000, 0, 0, 30000, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)

	stepNames := make(map[string]bool)
	for _, s := range result.AuditTrail.Steps {
		stepNames[s.StepName] = true
	}
	assert.True(t, stepNames["IRS Simplified Method Exclusion"],
		"expected 'IRS Simplified Method Exclusion' step when contributions > 0")

	// The step result should be a positive dollar amount (the annual exclusion)
	for _, s := range result.AuditTrail.Steps {
		if s.StepName == "IRS Simplified Method Exclusion" {
			assert.Greater(t, s.Result, 0.0, "annual exclusion should be positive")
		}
	}
}

// TestCalculateFERSPensionWithAudit_NoIRSExclusionWhenZeroContributions verifies the
// IRS Simplified Method step is omitted when there are no employee contributions.
func TestCalculateFERSPensionWithAudit_NoIRSExclusionWhenZeroContributions(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2005, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	employee := makeTestEmployee(birthDate, hireDate, 90000, 0, 0, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)
	for _, s := range result.AuditTrail.Steps {
		assert.NotEqual(t, "IRS Simplified Method Exclusion", s.StepName,
			"should not emit IRS exclusion step when contributions == 0")
	}
}

// TestCalculateFERSPensionWithAudit_WithSurvivor verifies the Survivor Benefit Cost
// step is included when a survivor benefit is elected.
func TestCalculateFERSPensionWithAudit_WithSurvivor(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2003, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	// 50% survivor election
	employee := makeTestEmployee(birthDate, hireDate, 90000, 0, 0.50, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)

	stepNames := make(map[string]bool)
	for _, s := range result.AuditTrail.Steps {
		stepNames[s.StepName] = true
	}
	assert.True(t, stepNames["Survivor Benefit Cost"],
		"expected 'Survivor Benefit Cost' step when survivor benefit is elected")
}

// TestCalculateFERSPensionWithAudit_OPMReferences verifies that known OPM citations are present.
func TestCalculateFERSPensionWithAudit_OPMReferences(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2005, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	employee := makeTestEmployee(birthDate, hireDate, 90000, 0, 0, 0, 0)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)
	refs := result.AuditTrail.OPMReferences

	// At minimum, expect 5 USC 8415 and a CFR citation
	found8415 := false
	for _, ref := range refs {
		if ref == "5 USC 8415 – Computation of basic annuity" {
			found8415 = true
		}
	}
	assert.True(t, found8415, "OPMReferences should include '5 USC 8415'")
	assert.GreaterOrEqual(t, len(refs), 2, "should have at least 2 OPM references")
}

// TestAuditStepNumbers verifies steps are numbered sequentially starting from 1.
func TestAuditStepNumbers(t *testing.T) {
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(2005, 7, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	employee := makeTestEmployee(birthDate, hireDate, 90000, 2087, 0.50, 20000, 1200)
	result := CalculateFERSPensionWithAudit(employee, retirementDate)

	require.NotNil(t, result.AuditTrail)
	steps := result.AuditTrail.Steps
	require.NotEmpty(t, steps)

	assert.Equal(t, 1, steps[0].StepNumber, "first step should be numbered 1")
	for i := 1; i < len(steps); i++ {
		assert.Equal(t, steps[i-1].StepNumber+1, steps[i].StepNumber,
			"steps should be numbered sequentially")
	}
}

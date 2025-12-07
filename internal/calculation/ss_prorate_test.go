package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// Test that Social Security benefit is prorated in the first year the person reaches SS start age
func TestSSProrate_FirstYearBirthdayMidYear(t *testing.T) {
	ce := NewCalculationEngine()

	// PersonA turns 62 on July 1, 2025 (birthday mid-year)
	personA := domain.Employee{
		BirthDate:     time.Date(1963, 7, 1, 0, 0, 0, 0, time.UTC),
		HireDate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary: decimal.Zero,
		SSBenefitFRA:  decimal.NewFromInt(1800),
	}
	personB := domain.Employee{
		Name:      "PersonB",
		BirthDate: time.Date(1965, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:  time.Date(1992, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Set retirement after birthday so SS prorating logic applies in-projection and retirement-year adjustment is used
	rs := domain.RetirementScenario{EmployeeName: "person_a", RetirementDate: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	ds := domain.RetirementScenario{EmployeeName: "person_b", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	scenario := domain.Scenario{Name: "ss-prorate", PersonA: rs, PersonB: ds}

	cfg := &domain.Configuration{
		GlobalAssumptions: domain.GlobalAssumptions{ProjectionYears: 3, COLAGeneralRate: decimal.Zero},
	}

	proj := ce.GenerateAnnualProjection(&personA, &personB, &scenario, &cfg.GlobalAssumptions, cfg.GlobalAssumptions.FederalRules)
	if len(proj) < 1 {
		t.Fatalf("expected projection rows")
	}

	// Year 0 is 2025; PersonA turns 62 on July 1, so SS should be prorated (less than full annual benefit)
	row := proj[0]
	full := CalculateSSBenefitForYear(&personA, rs.SSStartAge, 0, decimal.Zero)
	// Compute expected prorated fraction: days after birthday / days in year
	yearEnd := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	birthdayThisYear := time.Date(2025, personA.BirthDate.Month(), personA.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
	daysAfter := yearEnd.Sub(birthdayThisYear).Hours() / 24.0
	daysInYear := float64(dateutil.DaysInYear(2025))
	frac := daysAfter / daysInYear
	expected := full.Mul(decimal.NewFromFloat(frac))

	// Allow small rounding tolerance
	diff := row.SSBenefitPersonA.Sub(expected).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		t.Fatalf("prorated SS mismatch; expected %s, got %s (diff %s)", expected.StringFixed(2), row.SSBenefitPersonA.StringFixed(2), diff.StringFixed(2))
	}
}

package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func TestProjectionBaseYearOverride(t *testing.T) {
	personA := &domain.Employee{
		BirthDate:      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.Zero,
		High3Salary:    decimal.Zero,
		EmploymentType: domain.EmploymentTypeFederal,
	}
	personB := &domain.Employee{
		BirthDate:      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.Zero,
		High3Salary:    decimal.Zero,
		EmploymentType: domain.EmploymentTypeFederal,
	}

	scenario := &domain.Scenario{
		PersonA: domain.RetirementScenario{
			RetirementDate:        time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
		PersonB: domain.RetirementScenario{
			RetirementDate:        time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
	}

	assumptions := &domain.GlobalAssumptions{
		ProjectionYears:    1,
		ProjectionBaseYear: 2030,
		CurrentLocation:    domain.Location{State: "Pennsylvania"},
	}

	engine := NewCalculationEngineWithConfig(*assumptions)
	projection := engine.GenerateAnnualProjection(personA, personB, scenario, assumptions, assumptions.FederalRules)

	if len(projection) != 1 {
		t.Fatalf("expected 1 projection year, got %d", len(projection))
	}
	if projection[0].Date.Year() != 2030 {
		t.Fatalf("projection base year = %d, expected 2030", projection[0].Date.Year())
	}
}

package calculation

import (
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func TestProjection_MedicareMAGIIncludesWages(t *testing.T) {
	personA := &domain.Employee{
		BirthDate:      time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.NewFromInt(50000),
		High3Salary:    decimal.NewFromInt(50000),
		EmploymentType: domain.EmploymentTypeFederal,
	}
	personB := &domain.Employee{
		BirthDate:      time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.Zero,
		High3Salary:    decimal.Zero,
		EmploymentType: domain.EmploymentTypeFederal,
	}

	scenario := &domain.Scenario{
		PersonA: domain.RetirementScenario{
			RetirementDate:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
		PersonB: domain.RetirementScenario{
			RetirementDate:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
	}

	assumptions := &domain.GlobalAssumptions{
		InflationRate:           decimal.Zero,
		FEHBPremiumInflation:    decimal.Zero,
		TSPReturnPreRetirement:  decimal.Zero,
		TSPReturnPostRetirement: decimal.Zero,
		COLAGeneralRate:         decimal.Zero,
		ProjectionYears:         1,
		CurrentLocation:         domain.Location{State: "Pennsylvania"},
		FederalRules: domain.FederalRules{
			MedicareConfig: domain.MedicareConfig{
				BasePremium2025:      decimal.NewFromInt(100),
				PremiumInflationRate: decimal.Zero,
				IRMAAThresholds: []domain.MedicareIRMAAThreshold{
					{
						IncomeThresholdSingle: decimal.NewFromInt(20000),
						IncomeThresholdJoint:  decimal.NewFromInt(40000),
						MonthlySurcharge:      decimal.NewFromInt(50),
					},
				},
			},
		},
	}

	engine := NewCalculationEngineWithConfig(*assumptions)
	projection := engine.GenerateAnnualProjection(personA, personB, scenario, assumptions, assumptions.FederalRules)

	if len(projection) != 1 {
		t.Fatalf("expected 1 projection year, got %d", len(projection))
	}

	expectedAnnualPerPerson := decimal.NewFromInt(150).Mul(decimal.NewFromInt(12)) // 100 base + 50 surcharge
	expectedTotal := expectedAnnualPerPerson.Mul(decimal.NewFromInt(2))

	if !projection[0].MedicarePremium.Equal(expectedTotal) {
		t.Fatalf("medicare premium = %s, expected %s", projection[0].MedicarePremium.StringFixed(2), expectedTotal.StringFixed(2))
	}
}

func TestProjection_MedicareUsesSingleFilingStatusAfterDeath(t *testing.T) {
	personA := &domain.Employee{
		BirthDate:      time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.NewFromInt(50000),
		High3Salary:    decimal.NewFromInt(50000),
		EmploymentType: domain.EmploymentTypeFederal,
	}
	personB := &domain.Employee{
		BirthDate:      time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
		HireDate:       time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:  decimal.Zero,
		High3Salary:    decimal.Zero,
		EmploymentType: domain.EmploymentTypeFederal,
	}

	deathDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	scenario := &domain.Scenario{
		PersonA: domain.RetirementScenario{
			RetirementDate:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
		PersonB: domain.RetirementScenario{
			RetirementDate:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
		Mortality: &domain.ScenarioMortality{
			PersonB: &domain.MortalitySpec{DeathDate: &deathDate},
			Assumptions: &domain.MortalityAssumptions{
				FilingStatusSwitch: "immediate",
			},
		},
	}

	assumptions := &domain.GlobalAssumptions{
		InflationRate:           decimal.Zero,
		FEHBPremiumInflation:    decimal.Zero,
		TSPReturnPreRetirement:  decimal.Zero,
		TSPReturnPostRetirement: decimal.Zero,
		COLAGeneralRate:         decimal.Zero,
		ProjectionYears:         1,
		CurrentLocation:         domain.Location{State: "Pennsylvania"},
		FederalRules: domain.FederalRules{
			MedicareConfig: domain.MedicareConfig{
				BasePremium2025:      decimal.NewFromInt(100),
				PremiumInflationRate: decimal.Zero,
				IRMAAThresholds: []domain.MedicareIRMAAThreshold{
					{
						IncomeThresholdSingle: decimal.NewFromInt(30000),
						IncomeThresholdJoint:  decimal.NewFromInt(60000),
						MonthlySurcharge:      decimal.NewFromInt(50),
					},
				},
			},
		},
	}

	engine := NewCalculationEngineWithConfig(*assumptions)
	projection := engine.GenerateAnnualProjection(personA, personB, scenario, assumptions, assumptions.FederalRules)

	if len(projection) != 1 {
		t.Fatalf("expected 1 projection year, got %d", len(projection))
	}

	expectedAnnualPerPerson := decimal.NewFromInt(150).Mul(decimal.NewFromInt(12)) // single threshold hit
	if !projection[0].MedicarePremiumPersonA.Equal(expectedAnnualPerPerson) {
		t.Fatalf("person A medicare premium = %s, expected %s", projection[0].MedicarePremiumPersonA.StringFixed(2), expectedAnnualPerPerson.StringFixed(2))
	}
}

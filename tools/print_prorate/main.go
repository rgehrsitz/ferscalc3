package main

import (
	"fmt"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

func main() {
	ce := calculation.NewCalculationEngine()

	// SS scenario
	personA := domain.Employee{
		Name:          "PersonA",
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
	rs := domain.RetirementScenario{EmployeeName: "person_a", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}

	ds := domain.RetirementScenario{EmployeeName: "person_b", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	scenario := domain.Scenario{Name: "ss-prorate", PersonA: rs, PersonB: ds}

	proj := ce.GenerateAnnualProjection(&personA, &personB, &scenario, &domain.GlobalAssumptions{ProjectionYears: 3, COLAGeneralRate: decimal.Zero}, domain.FederalRules{})
	fmt.Println("SS Scenario projection row 0:")
	fmt.Printf("SSBenefitPersonA: %s\n", proj[0].SSBenefitPersonA.StringFixed(2))
	full := calculation.CalculateSSBenefitForYear(&personA, rs.SSStartAge, 0, decimal.Zero)
	fmt.Printf("Full-year SS (calc): %s\n", full.StringFixed(2))

	// RMD scenario
	personA2 := domain.Employee{
		Name:                  "PersonA",
		BirthDate:             time.Date(1953, 7, 1, 0, 0, 0, 0, time.UTC),
		HireDate:              time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC),
		CurrentSalary:         decimal.Zero,
		TSPBalanceTraditional: decimal.NewFromInt(500000),
	}
	ds2 := domain.RetirementScenario{EmployeeName: "person_a", RetirementDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), SSStartAge: 62, TSPWithdrawalStrategy: "4_percent_rule"}
	scenario2 := domain.Scenario{Name: "rmd-prorate", PersonA: ds2, PersonB: ds}
	proj2 := ce.GenerateAnnualProjection(&personA2, &personB, &scenario2, &domain.GlobalAssumptions{ProjectionYears: 3, COLAGeneralRate: decimal.Zero}, domain.FederalRules{})
	fmt.Println("RMD Scenario projection row 0:")
	fmt.Printf("TSPWithdrawalPersonA: %s\n", proj2[0].TSPWithdrawalPersonA.StringFixed(2))
	fullRMD := calculation.CalculateRMD(personA2.TSPBalanceTraditional, personA2.BirthDate.Year(), dateutil.GetRMDAge(personA2.BirthDate.Year()))
	fmt.Printf("Full RMD: %s\n", fullRMD.StringFixed(2))
}

// convenience wrapper because dateutil.GetRMDAge is in another package and unexported helper used in tests; create a small helper in calculation

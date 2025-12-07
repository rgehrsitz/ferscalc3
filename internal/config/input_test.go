package config

import (
	"os"
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInputParser(t *testing.T) {
	parser := NewInputParser()
	assert.NotNil(t, parser)
}

func TestLoadFromFile_Success(t *testing.T) {
	// Create a temporary test file with minimal, well-formed YAML (spaces only)
	testConfig := "personal_details:\n" +
		"  person_a:\n" +
		"    name: \"PersonA\"\n" +
		"    birth_date: \"1963-06-15T00:00:00Z\"\n" +
		"    hire_date: \"1985-03-20T00:00:00Z\"\n" +
		"    current_salary: 95000\n" +
		"    ss_benefit_62: 1680\n" +
		"    ss_benefit_fra: 2400\n" +
		"    ss_benefit_70: 2976\n" +
		"    high_3_salary: 93000\n" +
		"  person_b:\n" +
		"    name: \"PersonB\"\n" +
		"    birth_date: \"1965-08-22T00:00:00Z\"\n" +
		"    hire_date: \"1988-07-10T00:00:00Z\"\n" +
		"    current_salary: 85000\n" +
		"    ss_benefit_62: 1400\n" +
		"    ss_benefit_fra: 2000\n" +
		"    ss_benefit_70: 2480\n" +
		"    high_3_salary: 83000\n\n" +
		"global_assumptions:\n" +
		"  inflation_rate: 0.025\n" +
		"  projection_years: 30\n" +
		"  current_location:\n" +
		"    state: \"PA\"\n\n" +
		"scenarios:\n" +
		"  - name: \"Standard Retirement\"\n" +
		"    person_a:\n" +
		"      employee_name: \"person_a\"\n" +
		"      retirement_date: \"2025-12-31T00:00:00Z\"\n" +
		"      ss_start_age: 67\n" +
		"      tsp_withdrawal_strategy: \"4_percent_rule\"\n" +
		"    person_b:\n" +
		"      employee_name: \"person_b\"\n" +
		"      retirement_date: \"2025-12-31T00:00:00Z\"\n" +
		"      ss_start_age: 67\n" +
		"      tsp_withdrawal_strategy: \"4_percent_rule\"\n"

	tmpfile, err := os.CreateTemp("", "test_config_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(testConfig))
	require.NoError(t, err)
	tmpfile.Close()

	parser := NewInputParser()
	config, err := parser.LoadFromFile(tmpfile.Name())

	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Len(t, config.PersonalDetails, 2)
	assert.Contains(t, config.PersonalDetails, "person_a")
	assert.Contains(t, config.PersonalDetails, "person_b")
	assert.Len(t, config.Scenarios, 1)
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	parser := NewInputParser()
	config, err := parser.LoadFromFile("nonexistent_file.yaml")

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	// Create a temporary test file with invalid YAML
	testConfig := `
personal_details:
	person_a:
		name: "PersonA"
		birth_date: "invalid-date"
		current_salary: "not-a-number"
`

	tmpfile, err := os.CreateTemp("", "test_config_*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(testConfig))
	require.NoError(t, err)
	tmpfile.Close()

	parser := NewInputParser()
	config, err := parser.LoadFromFile(tmpfile.Name())

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestValidateConfiguration_Success(t *testing.T) {
	parser := NewInputParser()
	config := createValidTestConfiguration()

	err := parser.ValidateConfiguration(config)
	assert.NoError(t, err)
}

func TestValidateConfiguration_NoPersonalDetails(t *testing.T) {
	parser := NewInputParser()
	config := &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{},
		Scenarios:       []domain.Scenario{},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no personal details provided")
}

func TestValidateConfiguration_MissingPersonA(t *testing.T) {
	parser := NewInputParser()
	config := &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"person_b": createValidEmployee("person_b", "1965-08-22", "1988-07-10"),
		},
		Scenarios: []domain.Scenario{},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "person_a employee details are required")
}

func TestValidateConfiguration_MissingPersonB(t *testing.T) {
	parser := NewInputParser()
	config := &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"person_a": createValidEmployee("person_a", "1963-06-15", "1985-03-20"),
		},
		Scenarios: []domain.Scenario{},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "person_b employee details are required")
}

func TestValidateConfiguration_NoScenarios(t *testing.T) {
	parser := NewInputParser()
	config := &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"person_a": createValidEmployee("person_a", "1963-06-15", "1985-03-20"),
			"person_b": createValidEmployee("person_b", "1965-08-22", "1988-07-10"),
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.04),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.05),
			COLAGeneralRate:         decimal.NewFromFloat(0.02),
			ProjectionYears:         30,
			CurrentLocation: domain.Location{
				State:        "PA",
				County:       "Bucks",
				Municipality: "Upper Makefield",
			},
		},
		Scenarios: []domain.Scenario{},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no scenarios provided")
}

func TestValidateEmployee_Success(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("person_a", "1963-06-15", "1985-03-20")

	err := parser.validateEmployee("person_a", &employee)
	assert.NoError(t, err)
}

func TestValidateEmployee_ZeroBirthDate(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("person_a", "1963-06-15", "1985-03-20")
	employee.BirthDate = time.Time{}

	err := parser.validateEmployee("person_a", &employee)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "birth date is required")
}

func TestValidateEmployee_ZeroHireDate(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("person_a", "1963-06-15", "1985-03-20")
	employee.HireDate = time.Time{}

	err := parser.validateEmployee("person_a", &employee)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hire date is required")
}

func TestValidateEmployee_ZeroSalary(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("person_a", "1963-06-15", "1985-03-20")
	employee.CurrentSalary = decimal.Zero

	err := parser.validateEmployee("person_a", &employee)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "current salary must be positive")
}

func TestValidateEmployee_NegativeSalary(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("person_a", "1963-06-15", "1985-03-20")
	employee.CurrentSalary = decimal.NewFromInt(-1000)

	err := parser.validateEmployee("person_a", &employee)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "current salary must be positive")
}

func TestValidateEmployee_NegativeTSPBalance(t *testing.T) {
	parser := NewInputParser()
	employee := createValidEmployee("PersonA", "1963-06-15", "1985-03-20")
	employee.TSPBalanceTraditional = decimal.NewFromInt(-1000)

	err := parser.validateEmployee("person_a", &employee)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP traditional balance cannot be negative")
}

func TestValidateGlobalAssumptions_Success(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(0.025),
		FEHBPremiumInflation:    decimal.NewFromFloat(0.04),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.05),
		COLAGeneralRate:         decimal.NewFromFloat(0.02),
		ProjectionYears:         30,
		CurrentLocation: domain.Location{
			State:        "PA",
			County:       "Bucks",
			Municipality: "Upper Makefield",
		},
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.NoError(t, err)
}

func TestValidateGlobalAssumptions_ExtremeDeflation(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		InflationRate: decimal.NewFromFloat(-0.15), // -15%
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inflation rate cannot be less than -10%")
}

func TestValidateGlobalAssumptions_NegativeFEHBInflation(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		FEHBPremiumInflation: decimal.NewFromFloat(-0.01),
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FEHB premium inflation cannot be negative")
}

func TestValidateGlobalAssumptions_ExtremeTSPReturn(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		TSPReturnPreRetirement: decimal.NewFromFloat(-1.5), // -150%
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP return pre-retirement cannot be less than -100%")
}

func TestValidateGlobalAssumptions_InvalidProjectionYears(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		ProjectionYears: 0,
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "projection years must be between 1 and 50")

	assumptions.ProjectionYears = 60
	err = parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "projection years must be between 1 and 50")
}

func TestValidateGlobalAssumptions_MissingState(t *testing.T) {
	parser := NewInputParser()
	assumptions := domain.GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(0.025),
		FEHBPremiumInflation:    decimal.NewFromFloat(0.04),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.05),
		COLAGeneralRate:         decimal.NewFromFloat(0.02),
		ProjectionYears:         30,
		CurrentLocation: domain.Location{
			County:       "Bucks",
			Municipality: "Upper Makefield",
		},
	}

	err := parser.validateGlobalAssumptions(&assumptions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state is required")
}

func TestValidateScenario_Success(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.Scenario{
		Name: "Test Scenario",
		PersonA: domain.RetirementScenario{
			EmployeeName:          "person_a",
			RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
		PersonB: domain.RetirementScenario{
			EmployeeName:          "person_b",
			RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			SSStartAge:            67,
			TSPWithdrawalStrategy: "4_percent_rule",
		},
	}

	err := parser.validateScenario(0, &scenario)
	assert.NoError(t, err)
}

func TestValidateScenario_EmptyName(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.Scenario{
		Name: "",
	}

	err := parser.validateScenario(0, &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scenario name is required")
}

func TestValidateRetirementScenario_Success(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:          "person_a",
		RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:            67,
		TSPWithdrawalStrategy: "4_percent_rule",
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.NoError(t, err)
}

func TestValidateRetirementScenario_EmptyEmployeeName(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName: "",
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "employee name is required")
}

func TestValidateRetirementScenario_ZeroRetirementDate(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:   "person_a",
		RetirementDate: time.Time{},
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retirement date is required")
}

func TestValidateRetirementScenario_InvalidSSStartAge(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:   "person_a",
		RetirementDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:     60, // Too young
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "social security start age must be between 62 and 70")

	scenario.SSStartAge = 75 // Too old
	err = parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "social security start age must be between 62 and 70")
}

func TestValidateRetirementScenario_InvalidTSPStrategy(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:          "person_a",
		RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:            67,
		TSPWithdrawalStrategy: "invalid_strategy",
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal strategy must be")
}

func TestValidateRetirementScenario_NeedBasedWithoutTarget(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:          "person_a",
		RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:            67,
		TSPWithdrawalStrategy: "need_based",
		// Missing TSPWithdrawalTargetMonthly
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal target monthly is required for need_based strategy")
}

func TestValidateRetirementScenario_VariablePercentageWithoutRate(t *testing.T) {
	parser := NewInputParser()
	scenario := domain.RetirementScenario{
		EmployeeName:          "person_a",
		RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:            67,
		TSPWithdrawalStrategy: "variable_percentage",
		// Missing TSPWithdrawalRate
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal rate is required for variable_percentage strategy")
}

func TestValidateRetirementScenario_InvalidWithdrawalTarget(t *testing.T) {
	parser := NewInputParser()
	target := decimal.Zero
	scenario := domain.RetirementScenario{
		EmployeeName:               "person_a",
		RetirementDate:             time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:                 67,
		TSPWithdrawalStrategy:      "need_based",
		TSPWithdrawalTargetMonthly: &target,
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal target monthly must be positive")
}

func TestValidateRetirementScenario_InvalidWithdrawalRate(t *testing.T) {
	parser := NewInputParser()
	rate := decimal.NewFromFloat(-0.01)
	scenario := domain.RetirementScenario{
		EmployeeName:          "person_a",
		RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		SSStartAge:            67,
		TSPWithdrawalStrategy: "variable_percentage",
		TSPWithdrawalRate:     &rate,
	}

	err := parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal rate must be between 0 and 20%")

	rate = decimal.NewFromFloat(0.25) // 25%
	scenario.TSPWithdrawalRate = &rate
	err = parser.validateRetirementScenario("person_a", &scenario)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TSP withdrawal rate must be between 0 and 20%")
}

func TestCreateExampleConfiguration(t *testing.T) {
	parser := NewInputParser()
	config := parser.CreateExampleConfiguration()

	assert.NotNil(t, config)
	assert.Len(t, config.PersonalDetails, 2)
	assert.Contains(t, config.PersonalDetails, "person_a")
	assert.Contains(t, config.PersonalDetails, "person_b")
	assert.Len(t, config.Scenarios, 2) // The example creates 2 scenarios

	// Validate the example configuration
	err := parser.ValidateConfiguration(config)
	assert.NoError(t, err)
}

// Helper functions

func createValidEmployee(name, birthDate, hireDate string) domain.Employee {
	birth, _ := time.Parse("2006-01-02", birthDate)
	hire, _ := time.Parse("2006-01-02", hireDate)

	return domain.Employee{
		Name:                           name,
		BirthDate:                      birth,
		HireDate:                       hire,
		CurrentSalary:                  decimal.NewFromInt(95000),
		High3Salary:                    decimal.NewFromInt(93000),
		TSPBalanceTraditional:          decimal.NewFromInt(450000),
		TSPBalanceRoth:                 decimal.NewFromInt(50000),
		TSPContributionPercent:         decimal.NewFromFloat(0.15),
		SSBenefitFRA:                   decimal.NewFromInt(2400),
		SSBenefit62:                    decimal.NewFromInt(1680),
		SSBenefit70:                    decimal.NewFromInt(2976), // Must be > FRA
		FEHBPremiumPerPayPeriod:        decimal.NewFromInt(488),
		SurvivorBenefitElectionPercent: decimal.NewFromFloat(0.25),
	}
}

func createValidTestConfiguration() *domain.Configuration {
	return &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"person_a": createValidEmployee("PersonA", "1963-06-15", "1985-03-20"),
			"person_b": createValidEmployee("PersonB", "1965-08-22", "1988-07-10"),
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.04),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.05),
			COLAGeneralRate:         decimal.NewFromFloat(0.02),
			ProjectionYears:         30,
			CurrentLocation: domain.Location{
				State:        "PA",
				County:       "Bucks",
				Municipality: "Upper Makefield",
			},
		},
		Scenarios: []domain.Scenario{
			{
				Name: "Standard Retirement",
				PersonA: domain.RetirementScenario{
					EmployeeName:          "person_a",
					RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
					SSStartAge:            67,
					TSPWithdrawalStrategy: "4_percent_rule",
				},
				PersonB: domain.RetirementScenario{
					EmployeeName:          "person_b",
					RetirementDate:        time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
					SSStartAge:            67,
					TSPWithdrawalStrategy: "4_percent_rule",
				},
			},
		},
	}
}

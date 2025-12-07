package config

import (
	"fmt"
	"os"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// InputParser handles parsing of input configuration files
type InputParser struct{}

// NewInputParser creates a new input parser
func NewInputParser() *InputParser {
	return &InputParser{}
}

// LoadFromFile loads configuration from a YAML or JSON file
func (ip *InputParser) LoadFromFile(filename string) (*domain.Configuration, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var config domain.Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Apply sane defaults for optional nested configs not always present in files
	ip.applyDefaults(&config)

	// Legacy key mapping has been removed; configuration should use
	// 'person_a' and 'person_b' keys exclusively.

	// Validate the configuration
	if err := ip.ValidateConfiguration(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// ValidateConfiguration validates the loaded configuration
func (ip *InputParser) ValidateConfiguration(config *domain.Configuration) error {
	// Validate personal details
	if len(config.PersonalDetails) == 0 {
		return fmt.Errorf("no personal details provided")
	}

	// Check for required employees
	if _, exists := config.PersonalDetails["person_a"]; !exists {
		return fmt.Errorf("person_a employee details are required")
	}
	if _, exists := config.PersonalDetails["person_b"]; !exists {
		return fmt.Errorf("person_b employee details are required")
	}

	// Validate each employee
	for name, employee := range config.PersonalDetails {
		if err := ip.validateEmployee(name, &employee); err != nil {
			return fmt.Errorf("employee %s validation failed: %w", name, err)
		}
	}

	// Validate global assumptions
	if err := ip.validateGlobalAssumptions(&config.GlobalAssumptions); err != nil {
		return fmt.Errorf("global assumptions validation failed: %w", err)
	}

	// Validate scenarios
	if len(config.Scenarios) == 0 {
		return fmt.Errorf("no scenarios provided")
	}

	for i, scenario := range config.Scenarios {
		if err := ip.validateScenario(i, &scenario); err != nil {
			return fmt.Errorf("scenario %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateEmployee validates a single employee's data
func (ip *InputParser) validateEmployee(_ string, employee *domain.Employee) error {
	// Validate required fields
	if employee.BirthDate.IsZero() {
		return fmt.Errorf("birth date is required")
	}
	if employee.HireDate.IsZero() {
		return fmt.Errorf("hire date is required")
	}
	if employee.CurrentSalary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("current salary must be positive")
	}
	if employee.High3Salary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("high 3 salary must be positive")
	}
	if employee.TSPBalanceTraditional.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP traditional balance cannot be negative")
	}
	if employee.TSPBalanceRoth.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP Roth balance cannot be negative")
	}
	if employee.TSPContributionPercent.LessThan(decimal.Zero) || employee.TSPContributionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("TSP contribution percent must be between 0 and 1")
	}
	if employee.SSBenefitFRA.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at FRA must be positive")
	}
	if employee.SSBenefit62.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 62 must be positive")
	}
	if employee.SSBenefit70.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 70 must be positive")
	}
	if employee.FEHBPremiumPerPayPeriod.LessThan(decimal.Zero) {
		return fmt.Errorf("FEHB premium per pay period cannot be negative")
	}
	if employee.SurvivorBenefitElectionPercent.LessThan(decimal.Zero) || employee.SurvivorBenefitElectionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("survivor benefit election percent must be between 0 and 1")
	}

	// Validate date logic
	if employee.BirthDate.After(employee.HireDate) {
		return fmt.Errorf("birth date cannot be after hire date")
	}

	// Validate Social Security benefit progression
	if employee.SSBenefit62.GreaterThan(employee.SSBenefitFRA) {
		return fmt.Errorf("SS benefit at 62 cannot be greater than at FRA")
	}
	if employee.SSBenefitFRA.GreaterThan(employee.SSBenefit70) {
		return fmt.Errorf("SS benefit at FRA cannot be greater than at 70")
	}

	return nil
}

// validateGlobalAssumptions validates global assumptions
func (ip *InputParser) validateGlobalAssumptions(assumptions *domain.GlobalAssumptions) error {
	if assumptions.InflationRate.LessThan(decimal.NewFromFloat(-0.10)) {
		return fmt.Errorf("inflation rate cannot be less than -10%% (extreme deflation)")
	}
	if assumptions.FEHBPremiumInflation.LessThan(decimal.Zero) {
		return fmt.Errorf("FEHB premium inflation cannot be negative")
	}
	if assumptions.TSPReturnPreRetirement.LessThan(decimal.NewFromFloat(-1.0)) {
		return fmt.Errorf("TSP return pre-retirement cannot be less than -100%%")
	}
	if assumptions.TSPReturnPostRetirement.LessThan(decimal.NewFromFloat(-1.0)) {
		return fmt.Errorf("TSP return post-retirement cannot be less than -100%%")
	}
	if assumptions.COLAGeneralRate.LessThan(decimal.Zero) {
		return fmt.Errorf("COLA general rate cannot be negative")
	}
	if assumptions.ProjectionYears <= 0 || assumptions.ProjectionYears > 50 {
		return fmt.Errorf("projection years must be between 1 and 50")
	}

	// Validate location
	if assumptions.CurrentLocation.State == "" {
		return fmt.Errorf("state is required")
	}

	return nil
}

// validateScenario validates a single scenario
func (ip *InputParser) validateScenario(_ int, scenario *domain.Scenario) error {
	if scenario.Name == "" {
		return fmt.Errorf("scenario name is required")
	}

	// Validate PersonA scenario
	if err := ip.validateRetirementScenario("person_a", &scenario.PersonA); err != nil {
		return fmt.Errorf("person_a scenario validation failed: %w", err)
	}

	// Validate PersonB scenario
	if err := ip.validateRetirementScenario("person_b", &scenario.PersonB); err != nil {
		return fmt.Errorf("person_b scenario validation failed: %w", err)
	}

	// Validate optional mortality block
	if scenario.Mortality != nil {
		if scenario.Mortality.PersonA != nil {
			if scenario.Mortality.PersonA.DeathDate != nil && scenario.Mortality.PersonA.DeathAge != nil {
				return fmt.Errorf("mortality.person_a: specify either death_date or death_age, not both")
			}
		}
		if scenario.Mortality.PersonB != nil {
			if scenario.Mortality.PersonB.DeathDate != nil && scenario.Mortality.PersonB.DeathAge != nil {
				return fmt.Errorf("mortality.person_b: specify either death_date or death_age, not both")
			}
		}
		if scenario.Mortality.Assumptions != nil {
			if !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() && (scenario.Mortality.Assumptions.SurvivorSpendingFactor.LessThan(decimal.NewFromFloat(0.4)) || scenario.Mortality.Assumptions.SurvivorSpendingFactor.GreaterThan(decimal.NewFromFloat(1.0))) {
				return fmt.Errorf("mortality.assumptions.survivor_spending_factor must be between 0.4 and 1.0")
			}
			if scenario.Mortality.Assumptions.TSPSpousalTransfer != "" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "merge" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "separate" {
				return fmt.Errorf("mortality.assumptions.tsp_spousal_transfer must be 'merge' or 'separate'")
			}
			if scenario.Mortality.Assumptions.FilingStatusSwitch != "" && scenario.Mortality.Assumptions.FilingStatusSwitch != "next_year" && scenario.Mortality.Assumptions.FilingStatusSwitch != "immediate" {
				return fmt.Errorf("mortality.assumptions.filing_status_switch must be 'next_year' or 'immediate'")
			}
		}
	}

	return nil
}

// validateRetirementScenario validates a retirement scenario for an employee
func (ip *InputParser) validateRetirementScenario(_ string, scenario *domain.RetirementScenario) error {
	if scenario.EmployeeName == "" {
		return fmt.Errorf("employee name is required")
	}
	if scenario.RetirementDate.IsZero() {
		return fmt.Errorf("retirement date is required")
	}
	if scenario.SSStartAge < 62 || scenario.SSStartAge > 70 {
		return fmt.Errorf("social security start age must be between 62 and 70")
	}
	if scenario.TSPWithdrawalStrategy != "4_percent_rule" &&
		scenario.TSPWithdrawalStrategy != "need_based" &&
		scenario.TSPWithdrawalStrategy != "variable_percentage" &&
		scenario.TSPWithdrawalStrategy != "fixed_annuity" {
		return fmt.Errorf("TSP withdrawal strategy must be '4_percent_rule', 'need_based', 'variable_percentage', or 'fixed_annuity'")
	}
	if scenario.TSPWithdrawalStrategy == "need_based" && scenario.TSPWithdrawalTargetMonthly == nil {
		return fmt.Errorf("TSP withdrawal target monthly is required for need_based strategy")
	}
	if scenario.TSPWithdrawalStrategy == "variable_percentage" && scenario.TSPWithdrawalRate == nil {
		return fmt.Errorf("TSP withdrawal rate is required for variable_percentage strategy")
	}
	if scenario.TSPWithdrawalStrategy == "fixed_annuity" {
		// Validate annuity-specific fields
		if scenario.AnnuityPayoutRate == nil {
			return fmt.Errorf("annuity_payout_rate is required for fixed_annuity strategy")
		}
		if scenario.AnnuityPayoutRate.LessThanOrEqual(decimal.Zero) || scenario.AnnuityPayoutRate.GreaterThan(decimal.NewFromFloat(0.20)) {
			return fmt.Errorf("annuity_payout_rate must be between 0 and 0.20 (0%% and 20%%)")
		}
		if scenario.AnnuityPremiumPercent != nil {
			if scenario.AnnuityPremiumPercent.LessThanOrEqual(decimal.Zero) || scenario.AnnuityPremiumPercent.GreaterThan(decimal.NewFromInt(1)) {
				return fmt.Errorf("annuity_premium_percent must be between 0 and 1.0 (0%% and 100%%)")
			}
		}
		if scenario.AnnuityCOLARate != nil {
			if scenario.AnnuityCOLARate.LessThan(decimal.Zero) || scenario.AnnuityCOLARate.GreaterThan(decimal.NewFromFloat(0.10)) {
				return fmt.Errorf("annuity_cola_rate must be between 0 and 0.10 (0%% and 10%%)")
			}
		}
		if scenario.AnnuitySurvivorPercent != nil {
			if scenario.AnnuitySurvivorPercent.LessThan(decimal.Zero) || scenario.AnnuitySurvivorPercent.GreaterThan(decimal.NewFromInt(1)) {
				return fmt.Errorf("annuity_survivor_percent must be between 0 and 1.0 (0%% and 100%%)")
			}
		}
	}
	if scenario.TSPWithdrawalTargetMonthly != nil && scenario.TSPWithdrawalTargetMonthly.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("TSP withdrawal target monthly must be positive")
	}
	if scenario.TSPWithdrawalRate != nil && (scenario.TSPWithdrawalRate.LessThan(decimal.Zero) || scenario.TSPWithdrawalRate.GreaterThan(decimal.NewFromFloat(0.2))) {
		return fmt.Errorf("TSP withdrawal rate must be between 0 and 20%%")
	}

	return nil
}

// CreateExampleConfiguration creates an example configuration file
func (ip *InputParser) CreateExampleConfiguration() *domain.Configuration {
	personABirthDate, _ := time.Parse("2006-01-02", "1963-06-15")
	personAHireDate, _ := time.Parse("2006-01-02", "1985-03-20")
	personBBirthDate, _ := time.Parse("2006-01-02", "1965-08-22")
	personBHireDate, _ := time.Parse("2006-01-02", "1988-07-10")

	personARetirementDate, _ := time.Parse("2006-01-02", "2025-12-31")
	personBRetirementDate, _ := time.Parse("2006-01-02", "2025-12-31")

	return &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"person_a": {
				Name:                           "PersonA",
				BirthDate:                      personABirthDate,
				HireDate:                       personAHireDate,
				CurrentSalary:                  decimal.NewFromInt(95000),
				High3Salary:                    decimal.NewFromInt(93000),
				TSPBalanceTraditional:          decimal.NewFromInt(450000),
				TSPBalanceRoth:                 decimal.NewFromInt(50000),
				TSPContributionPercent:         decimal.NewFromFloat(0.15),
				SSBenefitFRA:                   decimal.NewFromInt(2400),
				SSBenefit62:                    decimal.NewFromInt(1680),
				SSBenefit70:                    decimal.NewFromInt(2976),
				FEHBPremiumPerPayPeriod:        decimal.NewFromInt(875),
				SurvivorBenefitElectionPercent: decimal.Zero,
			},
			"person_b": {
				Name:                           "PersonB",
				BirthDate:                      personBBirthDate,
				HireDate:                       personBHireDate,
				CurrentSalary:                  decimal.NewFromInt(87000),
				High3Salary:                    decimal.NewFromInt(85000),
				TSPBalanceTraditional:          decimal.NewFromInt(380000),
				TSPBalanceRoth:                 decimal.NewFromInt(45000),
				TSPContributionPercent:         decimal.NewFromFloat(0.12),
				SSBenefitFRA:                   decimal.NewFromInt(2200),
				SSBenefit62:                    decimal.NewFromInt(1540),
				SSBenefit70:                    decimal.NewFromInt(2728),
				FEHBPremiumPerPayPeriod:        decimal.Zero, // Covered under PersonA FEHB
				SurvivorBenefitElectionPercent: decimal.Zero,
			},
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.065),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.055),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.045),
			COLAGeneralRate:         decimal.NewFromFloat(0.025),
			ProjectionYears:         25,
			CurrentLocation: domain.Location{
				State:        "Pennsylvania",
				County:       "Bucks",
				Municipality: "Upper Makefield Township",
			},
		},
		Scenarios: []domain.Scenario{
			{
				Name: "Early Retirement 2025",
				PersonA: domain.RetirementScenario{
					EmployeeName:               "person_a",
					RetirementDate:             personARetirementDate,
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
				PersonB: domain.RetirementScenario{
					EmployeeName:               "person_b",
					RetirementDate:             personBRetirementDate,
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
			},
			{
				Name: "Delayed Retirement 2028",
				PersonA: domain.RetirementScenario{
					EmployeeName:               "person_a",
					RetirementDate:             personARetirementDate.AddDate(3, 0, 0),
					SSStartAge:                 67,
					TSPWithdrawalStrategy:      "need_based",
					TSPWithdrawalTargetMonthly: &[]decimal.Decimal{decimal.NewFromInt(3000)}[0],
				},
				PersonB: domain.RetirementScenario{
					EmployeeName:               "person_b",
					RetirementDate:             personBRetirementDate.AddDate(3, 0, 0),
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
			},
		},
	}
}

// applyDefaults populates default values for missing optional fields
func (ip *InputParser) applyDefaults(cfg *domain.Configuration) {
	// FEHB config defaults if not provided
	if cfg.GlobalAssumptions.FederalRules.FEHBConfig.PayPeriodsPerYear == 0 {
		cfg.GlobalAssumptions.FederalRules.FEHBConfig.PayPeriodsPerYear = 26
	}
	if cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementCalculationMethod == "" {
		cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementCalculationMethod = "same_as_active"
	}
	if cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementPremiumMultiplier.IsZero() {
		cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementPremiumMultiplier = decimal.NewFromFloat(1.0)
	}
}

package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// Employee represents a federal employee with all necessary information for retirement planning
type Employee struct {
	Name                           string          `yaml:"name" json:"name"`
	BirthDate                      time.Time       `yaml:"birth_date" json:"birth_date"`
	HireDate                       time.Time       `yaml:"hire_date" json:"hire_date"`
	EmploymentType                 string          `yaml:"employment_type,omitempty" json:"employment_type,omitempty"`
	CurrentSalary                  decimal.Decimal `yaml:"current_salary" json:"current_salary"`
	High3Salary                    decimal.Decimal `yaml:"high_3_salary" json:"high_3_salary"`
	TSPBalanceTraditional          decimal.Decimal `yaml:"tsp_balance_traditional" json:"tsp_balance_traditional"`
	TSPBalanceRoth                 decimal.Decimal `yaml:"tsp_balance_roth" json:"tsp_balance_roth"`
	TSPContributionPercent         decimal.Decimal `yaml:"tsp_contribution_percent" json:"tsp_contribution_percent"`
	SSBenefitFRA                   decimal.Decimal `yaml:"ss_benefit_fra" json:"ss_benefit_fra"` // Monthly at Full Retirement Age
	SSBenefit62                    decimal.Decimal `yaml:"ss_benefit_62" json:"ss_benefit_62"`   // Monthly at age 62
	SSBenefit70                    decimal.Decimal `yaml:"ss_benefit_70" json:"ss_benefit_70"`   // Monthly at age 70
	FEHBPremiumPerPayPeriod        decimal.Decimal `yaml:"fehb_premium_per_pay_period" json:"fehb_premium_per_pay_period"`
	SurvivorBenefitElectionPercent decimal.Decimal `yaml:"survivor_benefit_election_percent" json:"survivor_benefit_election_percent"`

	// Sick Leave Credit (for pension calculation)
	SickLeaveHours decimal.Decimal `yaml:"sick_leave_hours,omitempty" json:"sick_leave_hours,omitempty"`

	// TSP Asset Allocation (optional - uses default allocation if not specified)
	TSPAllocation *TSPAllocation `yaml:"tsp_allocation,omitempty" json:"tsp_allocation,omitempty"`

	// TSP Lifecycle Fund (optional - overrides tsp_allocation if specified)
	// If specified, allocation will change over time based on age
	TSPLifecycleFund *TSPLifecycleFund `yaml:"tsp_lifecycle_fund,omitempty" json:"tsp_lifecycle_fund,omitempty"`

	// Optional fields for additional context (not used in calculations)
	PayPlanGrade string `yaml:"pay_plan_grade,omitempty" json:"pay_plan_grade,omitempty"`
	SSNLast4     string `yaml:"ssn_last4,omitempty" json:"ssn_last4,omitempty"`

	// Fixed retirement income for non-federal spouses (or overrides for federal employees)
	FixedRetirementIncome *FixedRetirementIncome `yaml:"fixed_retirement_income,omitempty" json:"fixed_retirement_income,omitempty"`
}

const (
	EmploymentTypeFederal    = "federal"
	EmploymentTypeNonFederal = "non-federal"
)

// RetirementScenario represents a specific retirement scenario for an employee
type RetirementScenario struct {
	EmployeeName               string                 `yaml:"employee_name" json:"employee_name"`
	RetirementDate             time.Time              `yaml:"retirement_date" json:"retirement_date"`
	SSStartAge                 int                    `yaml:"ss_start_age" json:"ss_start_age"`
	TSPWithdrawalStrategy      string                 `yaml:"tsp_withdrawal_strategy" json:"tsp_withdrawal_strategy"`
	TSPWithdrawalTargetMonthly *decimal.Decimal       `yaml:"tsp_withdrawal_target_monthly,omitempty" json:"tsp_withdrawal_target_monthly,omitempty"`
	TSPWithdrawalRate          *decimal.Decimal       `yaml:"tsp_withdrawal_rate,omitempty" json:"tsp_withdrawal_rate,omitempty"`
	TSPWithdrawalCeiling       *decimal.Decimal       `yaml:"tsp_withdrawal_ceiling,omitempty" json:"tsp_withdrawal_ceiling,omitempty"`
	TSPWithdrawalFloor         *decimal.Decimal       `yaml:"tsp_withdrawal_floor,omitempty" json:"tsp_withdrawal_floor,omitempty"`
	FixedRetirementIncome      *FixedRetirementIncome `yaml:"fixed_retirement_income,omitempty" json:"fixed_retirement_income,omitempty"`

	// Annuity-specific configuration (used when tsp_withdrawal_strategy is "fixed_annuity")
	AnnuityPremiumPercent  *decimal.Decimal `yaml:"annuity_premium_percent,omitempty" json:"annuity_premium_percent,omitempty"`   // Percentage of TSP to convert (e.g., 1.0 for 100%, 0.5 for 50%)
	AnnuityPayoutRate      *decimal.Decimal `yaml:"annuity_payout_rate,omitempty" json:"annuity_payout_rate,omitempty"`           // Annual payout rate (e.g., 0.055 for 5.5%)
	AnnuityCOLARate        *decimal.Decimal `yaml:"annuity_cola_rate,omitempty" json:"annuity_cola_rate,omitempty"`               // Annual COLA adjustment (e.g., 0.02 for 2%, 0 for none)
	AnnuitySurvivorPercent *decimal.Decimal `yaml:"annuity_survivor_percent,omitempty" json:"annuity_survivor_percent,omitempty"` // Survivor payout (1.0 = 100%, 0.5 = 50%, 0 = none)
	AnnuityGuaranteedYears *int             `yaml:"annuity_guaranteed_years,omitempty" json:"annuity_guaranteed_years,omitempty"` // Guaranteed payment period (e.g., 10)
}

// UnmarshalYAML implements custom YAML unmarshaling for RetirementScenario
func (rs *RetirementScenario) UnmarshalYAML(value *yaml.Node) error {
	// Define a temporary struct with string fields for parsing
	type Alias struct {
		EmployeeName               string                 `yaml:"employee_name"`
		RetirementDate             time.Time              `yaml:"retirement_date"`
		SSStartAge                 int                    `yaml:"ss_start_age"`
		TSPWithdrawalStrategy      string                 `yaml:"tsp_withdrawal_strategy"`
		TSPWithdrawalTargetMonthly *string                `yaml:"tsp_withdrawal_target_monthly,omitempty"`
		TSPWithdrawalRate          *string                `yaml:"tsp_withdrawal_rate,omitempty"`
		TSPWithdrawalCeiling       *string                `yaml:"tsp_withdrawal_ceiling,omitempty"`
		TSPWithdrawalFloor         *string                `yaml:"tsp_withdrawal_floor,omitempty"`
		FixedRetirementIncome      *FixedRetirementIncome `yaml:"fixed_retirement_income,omitempty"`
		AnnuityPremiumPercent      *string                `yaml:"annuity_premium_percent,omitempty"`
		AnnuityPayoutRate          *string                `yaml:"annuity_payout_rate,omitempty"`
		AnnuityCOLARate            *string                `yaml:"annuity_cola_rate,omitempty"`
		AnnuitySurvivorPercent     *string                `yaml:"annuity_survivor_percent,omitempty"`
		AnnuityGuaranteedYears     *int                   `yaml:"annuity_guaranteed_years,omitempty"`
	}

	var aux Alias
	if err := value.Decode(&aux); err != nil {
		return err
	}

	// Copy non-decimal fields
	rs.EmployeeName = aux.EmployeeName
	rs.RetirementDate = aux.RetirementDate
	rs.SSStartAge = aux.SSStartAge
	rs.TSPWithdrawalStrategy = aux.TSPWithdrawalStrategy
	rs.FixedRetirementIncome = aux.FixedRetirementIncome
	rs.AnnuityGuaranteedYears = aux.AnnuityGuaranteedYears

	// Convert string decimal fields to *decimal.Decimal
	if aux.TSPWithdrawalTargetMonthly != nil {
		val, err := decimal.NewFromString(*aux.TSPWithdrawalTargetMonthly)
		if err != nil {
			return err
		}
		rs.TSPWithdrawalTargetMonthly = &val
	}

	if aux.TSPWithdrawalRate != nil {
		val, err := decimal.NewFromString(*aux.TSPWithdrawalRate)
		if err != nil {
			return err
		}
		rs.TSPWithdrawalRate = &val
	}

	if aux.TSPWithdrawalCeiling != nil {
		val, err := decimal.NewFromString(*aux.TSPWithdrawalCeiling)
		if err != nil {
			return err
		}
		rs.TSPWithdrawalCeiling = &val
	}

	if aux.TSPWithdrawalFloor != nil {
		val, err := decimal.NewFromString(*aux.TSPWithdrawalFloor)
		if err != nil {
			return err
		}
		rs.TSPWithdrawalFloor = &val
	}

	// Convert annuity decimal fields
	if aux.AnnuityPremiumPercent != nil {
		val, err := decimal.NewFromString(*aux.AnnuityPremiumPercent)
		if err != nil {
			return err
		}
		rs.AnnuityPremiumPercent = &val
	}

	if aux.AnnuityPayoutRate != nil {
		val, err := decimal.NewFromString(*aux.AnnuityPayoutRate)
		if err != nil {
			return err
		}
		rs.AnnuityPayoutRate = &val
	}

	if aux.AnnuityCOLARate != nil {
		val, err := decimal.NewFromString(*aux.AnnuityCOLARate)
		if err != nil {
			return err
		}
		rs.AnnuityCOLARate = &val
	}

	if aux.AnnuitySurvivorPercent != nil {
		val, err := decimal.NewFromString(*aux.AnnuitySurvivorPercent)
		if err != nil {
			return err
		}
		rs.AnnuitySurvivorPercent = &val
	}

	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for RetirementScenario.
func (rs *RetirementScenario) UnmarshalJSON(data []byte) error {
	type Alias struct {
		EmployeeName               string                 `json:"employee_name"`
		RetirementDate             time.Time              `json:"retirement_date"`
		SSStartAge                 int                    `json:"ss_start_age"`
		TSPWithdrawalStrategy      string                 `json:"tsp_withdrawal_strategy"`
		TSPWithdrawalTargetMonthly json.RawMessage        `json:"tsp_withdrawal_target_monthly,omitempty"`
		TSPWithdrawalRate          json.RawMessage        `json:"tsp_withdrawal_rate,omitempty"`
		TSPWithdrawalCeiling       json.RawMessage        `json:"tsp_withdrawal_ceiling,omitempty"`
		TSPWithdrawalFloor         json.RawMessage        `json:"tsp_withdrawal_floor,omitempty"`
		FixedRetirementIncome      *FixedRetirementIncome `json:"fixed_retirement_income,omitempty"`
		AnnuityPremiumPercent      json.RawMessage        `json:"annuity_premium_percent,omitempty"`
		AnnuityPayoutRate          json.RawMessage        `json:"annuity_payout_rate,omitempty"`
		AnnuityCOLARate            json.RawMessage        `json:"annuity_cola_rate,omitempty"`
		AnnuitySurvivorPercent     json.RawMessage        `json:"annuity_survivor_percent,omitempty"`
		AnnuityGuaranteedYears     *int                   `json:"annuity_guaranteed_years,omitempty"`
	}

	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	rs.EmployeeName = aux.EmployeeName
	rs.RetirementDate = aux.RetirementDate
	rs.SSStartAge = aux.SSStartAge
	rs.TSPWithdrawalStrategy = aux.TSPWithdrawalStrategy
	rs.FixedRetirementIncome = aux.FixedRetirementIncome
	rs.AnnuityGuaranteedYears = aux.AnnuityGuaranteedYears

	if val, err := parseOptionalDecimal(aux.TSPWithdrawalTargetMonthly); err != nil {
		return err
	} else if val != nil {
		rs.TSPWithdrawalTargetMonthly = val
	}

	if val, err := parseOptionalDecimal(aux.TSPWithdrawalRate); err != nil {
		return err
	} else if val != nil {
		rs.TSPWithdrawalRate = val
	}

	if val, err := parseOptionalDecimal(aux.TSPWithdrawalCeiling); err != nil {
		return err
	} else if val != nil {
		rs.TSPWithdrawalCeiling = val
	}

	if val, err := parseOptionalDecimal(aux.TSPWithdrawalFloor); err != nil {
		return err
	} else if val != nil {
		rs.TSPWithdrawalFloor = val
	}

	if val, err := parseOptionalDecimal(aux.AnnuityPremiumPercent); err != nil {
		return err
	} else if val != nil {
		rs.AnnuityPremiumPercent = val
	}

	if val, err := parseOptionalDecimal(aux.AnnuityPayoutRate); err != nil {
		return err
	} else if val != nil {
		rs.AnnuityPayoutRate = val
	}

	if val, err := parseOptionalDecimal(aux.AnnuityCOLARate); err != nil {
		return err
	} else if val != nil {
		rs.AnnuityCOLARate = val
	}

	if val, err := parseOptionalDecimal(aux.AnnuitySurvivorPercent); err != nil {
		return err
	} else if val != nil {
		rs.AnnuitySurvivorPercent = val
	}

	return nil
}

func parseOptionalDecimal(raw json.RawMessage) (*decimal.Decimal, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	str := strings.TrimSpace(string(raw))
	if str == "" || str == "null" {
		return nil, nil
	}
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = strings.Trim(str, "\"")
	}
	val, err := decimal.NewFromString(str)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// Scenario represents a complete retirement scenario for both employees
type Scenario struct {
	Name      string             `yaml:"name" json:"name"`
	PersonA   RetirementScenario `yaml:"person_a" json:"person_a"`
	PersonB   RetirementScenario `yaml:"person_b" json:"person_b"`
	Mortality *ScenarioMortality `yaml:"mortality,omitempty" json:"mortality,omitempty"`
}

// FixedRetirementIncome represents a constant (optionally COLA-adjusted) annual amount
type FixedRetirementIncome struct {
	AnnualAmount decimal.Decimal  `yaml:"annual_amount" json:"annual_amount"`
	COLARate     *decimal.Decimal `yaml:"cola_rate,omitempty" json:"cola_rate,omitempty"`
}

// ScenarioMortality groups mortality specifications and assumptions for a scenario
type ScenarioMortality struct {
	PersonA     *MortalitySpec        `yaml:"person_a,omitempty" json:"person_a,omitempty"`
	PersonB     *MortalitySpec        `yaml:"person_b,omitempty" json:"person_b,omitempty"`
	Assumptions *MortalityAssumptions `yaml:"assumptions,omitempty" json:"assumptions,omitempty"`
}

// MortalitySpec defines a deterministic death event by date or by age (one may be supplied)
type MortalitySpec struct {
	DeathDate *time.Time `yaml:"death_date,omitempty" json:"death_date,omitempty"`
	DeathAge  *int       `yaml:"death_age,omitempty" json:"death_age,omitempty"`
}

// MortalityAssumptions defines how to treat finances after a death event (Phase 1 limited subset)
type MortalityAssumptions struct {
	SurvivorSpendingFactor decimal.Decimal `yaml:"survivor_spending_factor" json:"survivor_spending_factor"`
	TSPSpousalTransfer     string          `yaml:"tsp_spousal_transfer" json:"tsp_spousal_transfer"` // merge|separate (Phase 1 supports only merge & separate=ignore merge)
	FilingStatusSwitch     string          `yaml:"filing_status_switch" json:"filing_status_switch"` // next_year|immediate (not yet applied in Phase 1)
}

// GlobalAssumptions contains all the global parameters for calculations
type GlobalAssumptions struct {
	InflationRate               decimal.Decimal `yaml:"inflation_rate" json:"inflation_rate"`
	FEHBPremiumInflation        decimal.Decimal `yaml:"fehb_premium_inflation" json:"fehb_premium_inflation"`
	TSPReturnPreRetirement      decimal.Decimal `yaml:"tsp_return_pre_retirement" json:"tsp_return_pre_retirement"`
	TSPReturnPostRetirement     decimal.Decimal `yaml:"tsp_return_post_retirement" json:"tsp_return_post_retirement"`
	COLAGeneralRate             decimal.Decimal `yaml:"cola_general_rate" json:"cola_general_rate"`
	FederalBracketInflationRate decimal.Decimal `yaml:"federal_bracket_inflation_rate" json:"federal_bracket_inflation_rate"`
	ProjectionYears             int             `yaml:"projection_years" json:"projection_years"`
	ProjectionBaseYear          int             `yaml:"projection_base_year" json:"projection_base_year"`
	CurrentLocation             Location        `yaml:"current_location" json:"current_location"`

	// Monte Carlo Configuration
	MonteCarloSettings MonteCarloSettings `yaml:"monte_carlo_settings" json:"monte_carlo_settings"`

	// Federal Rules and Limits (updated annually)
	FederalRules FederalRules `yaml:"federal_rules" json:"federal_rules"`

	// TSP Statistical Models (calculated from historical data, but configurable)
	TSPStatisticalModels TSPStatisticalModels `yaml:"tsp_statistical_models" json:"tsp_statistical_models"`
}

// GenerateAssumptions creates dynamic assumptions list from actual config values
func (ga *GlobalAssumptions) GenerateAssumptions() []string {
	bracketRate := ga.FederalBracketInflationRate
	if bracketRate.IsZero() {
		bracketRate = ga.InflationRate
	}
	return []string{
		fmt.Sprintf("General COLA (FERS pension & SS): %.1f%% annually", ga.COLAGeneralRate.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		fmt.Sprintf("FEHB premium inflation: %.1f%% annually", ga.FEHBPremiumInflation.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		fmt.Sprintf("TSP growth pre-retirement: %.1f%% annually", ga.TSPReturnPreRetirement.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		fmt.Sprintf("TSP growth post-retirement: %.1f%% annually", ga.TSPReturnPostRetirement.Mul(decimal.NewFromInt(100)).InexactFloat64()),
		"Social Security wage base indexing: ~5% annually (2025 est: $168,600)",
		fmt.Sprintf("Tax brackets & deductions: indexed independently at %.1f%% annually from 2025 baseline", bracketRate.Mul(decimal.NewFromInt(100)).InexactFloat64()),
	}
}

// Location represents the geographic location for tax calculations
type Location struct {
	State        string `yaml:"state" json:"state"`
	County       string `yaml:"county" json:"county"`
	Municipality string `yaml:"municipality" json:"municipality"`
}

// MonteCarloSettings contains Monte Carlo simulation parameters
type MonteCarloSettings struct {
	// Variability parameters for statistical generation
	TSPReturnVariability decimal.Decimal `yaml:"tsp_return_variability" json:"tsp_return_variability"` // Default: 0.15 (15% std dev)
	InflationVariability decimal.Decimal `yaml:"inflation_variability" json:"inflation_variability"`   // Default: 0.02 (2% std dev)
	COLAVariability      decimal.Decimal `yaml:"cola_variability" json:"cola_variability"`             // Default: 0.02 (2% std dev)
	FEHBVariability      decimal.Decimal `yaml:"fehb_variability" json:"fehb_variability"`             // Default: 0.05 (5% std dev)

	// Income limits and caps
	MaxReasonableIncome decimal.Decimal `yaml:"max_reasonable_income" json:"max_reasonable_income"` // Default: 5000000 ($5M annual cap)

	// Default TSP asset allocation (used when individual allocations not specified)
	DefaultTSPAllocation TSPAllocation `yaml:"default_tsp_allocation" json:"default_tsp_allocation"`

	// Optional named stress scenarios for deterministic Monte Carlo runs
	StressTests map[string]StressScenario `yaml:"stress_tests" json:"stress_tests"`
}

// StressScenario provides deterministic multi-year market conditions for stress testing
type StressScenario struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Repeat      bool         `yaml:"repeat" json:"repeat"`
	Years       []StressYear `yaml:"years" json:"years"`
}

// StressYear defines market conditions for a single year inside a stress scenario
type StressYear struct {
	Year       int                        `yaml:"year" json:"year"`
	Label      string                     `yaml:"label" json:"label"`
	TSPReturns map[string]decimal.Decimal `yaml:"tsp_returns" json:"tsp_returns"`
	Inflation  *decimal.Decimal           `yaml:"inflation" json:"inflation"`
	COLA       *decimal.Decimal           `yaml:"cola" json:"cola"`
	FEHB       *decimal.Decimal           `yaml:"fehb" json:"fehb"`
}

// TSPAllocation represents asset allocation across TSP funds
type TSPAllocation struct {
	CFund decimal.Decimal `yaml:"c_fund" json:"c_fund"` // Default: 0.60 (60% - Large Cap Stock Index)
	SFund decimal.Decimal `yaml:"s_fund" json:"s_fund"` // Default: 0.20 (20% - Small Cap Stock Index)
	IFund decimal.Decimal `yaml:"i_fund" json:"i_fund"` // Default: 0.10 (10% - International Stock Index)
	FFund decimal.Decimal `yaml:"f_fund" json:"f_fund"` // Default: 0.10 (10% - Fixed Income Index)
	GFund decimal.Decimal `yaml:"g_fund" json:"g_fund"` // Default: 0.00 (0% - Government Securities)
}

// TSPLifecycleFund represents a TSP Lifecycle Fund with age-based allocation changes
type TSPLifecycleFund struct {
	FundName       string                              `yaml:"fund_name" json:"fund_name"`             // e.g., "L2030", "L2035", "L2040", "L Income"
	AllocationData map[string][]TSPAllocationDataPoint `yaml:"allocation_data" json:"allocation_data"` // Quarterly allocation data
}

// TSPAllocationDataPoint represents allocation at a specific date
type TSPAllocationDataPoint struct {
	Date       string        `yaml:"date" json:"date"` // Format: "YYYY-MM-DD"
	Allocation TSPAllocation `yaml:"allocation" json:"allocation"`
}

// FederalRules contains federal rules and limits that change annually
type FederalRules struct {
	// Social Security taxation thresholds (2025 values, updated annually)
	SocialSecurityTaxThresholds SocialSecurityTaxThresholds `yaml:"social_security_tax_thresholds" json:"social_security_tax_thresholds"`

	// Social Security benefit calculation rules (rarely change, but configurable)
	SocialSecurityRules SocialSecurityRules `yaml:"social_security_rules" json:"social_security_rules"`

	// FERS rules and matching rates
	FERSRules FERSRules `yaml:"fers_rules" json:"fers_rules"`

	// Federal tax configuration (updated annually)
	FederalTaxConfig FederalTaxConfig `yaml:"federal_tax_config" json:"federal_tax_config"`

	// State and local tax configuration
	StateLocalTaxConfig StateLocalTaxConfig `yaml:"state_local_tax_config" json:"state_local_tax_config"`

	// FICA tax configuration (updated annually)
	FICATaxConfig FICATaxConfig `yaml:"fica_tax_config" json:"fica_tax_config"`

	// Medicare configuration (updated annually)
	MedicareConfig MedicareConfig `yaml:"medicare_config" json:"medicare_config"`

	// FEHB configuration
	FEHBConfig FEHBConfig `yaml:"fehb_config" json:"fehb_config"`
}

// SocialSecurityTaxThresholds contains income thresholds for SS taxation (updated annually)
type SocialSecurityTaxThresholds struct {
	// 2025 thresholds for determining taxable portion of Social Security benefits
	MarriedFilingJointly struct {
		Threshold1 decimal.Decimal `yaml:"threshold_1" json:"threshold_1"` // Default: 32000 (50% taxation begins)
		Threshold2 decimal.Decimal `yaml:"threshold_2" json:"threshold_2"` // Default: 44000 (85% taxation begins)
	} `yaml:"married_filing_jointly" json:"married_filing_jointly"`

	Single struct {
		Threshold1 decimal.Decimal `yaml:"threshold_1" json:"threshold_1"` // Default: 25000 (50% taxation begins)
		Threshold2 decimal.Decimal `yaml:"threshold_2" json:"threshold_2"` // Default: 34000 (85% taxation begins)
	} `yaml:"single" json:"single"`
}

// SocialSecurityRules contains benefit calculation rules
type SocialSecurityRules struct {
	// Early retirement reduction: 5/9 of 1% per month for first 36 months, 5/12 of 1% thereafter
	EarlyRetirementReduction struct {
		First36MonthsRate    decimal.Decimal `yaml:"first_36_months_rate" json:"first_36_months_rate"`     // Default: 0.0055556 (5/9 of 1%)
		AdditionalMonthsRate decimal.Decimal `yaml:"additional_months_rate" json:"additional_months_rate"` // Default: 0.0041667 (5/12 of 1%)
	} `yaml:"early_retirement_reduction" json:"early_retirement_reduction"`

	// Delayed retirement credit: 2/3 of 1% per month (8% per year)
	DelayedRetirementCredit decimal.Decimal `yaml:"delayed_retirement_credit" json:"delayed_retirement_credit"` // Default: 0.0066667 (2/3 of 1%)
}

// FERSRules contains FERS-specific rules and matching rates
type FERSRules struct {
	// TSP matching rates
	TSPMatchingRate      decimal.Decimal `yaml:"tsp_matching_rate" json:"tsp_matching_rate"`           // Default: 0.05 (5% maximum match)
	TSPMatchingThreshold decimal.Decimal `yaml:"tsp_matching_threshold" json:"tsp_matching_threshold"` // Default: 0.05 (5% contribution required for full match)

	// Special Retirement Supplement (SRS) earnings test limit
	SRSEarningsLimit decimal.Decimal `yaml:"srs_earnings_limit" json:"srs_earnings_limit"` // Default: 23400 (2025 projected)
}

// FederalTaxConfig contains federal income tax configuration (updated annually)
type FederalTaxConfig struct {
	// Standard deduction amounts
	StandardDeductionMFJ        decimal.Decimal `yaml:"standard_deduction_mfj" json:"standard_deduction_mfj"`                               // Default: 30000 (2025 MFJ)
	StandardDeductionSingle     decimal.Decimal `yaml:"standard_deduction_single" json:"standard_deduction_single"`                         // Default: 15000 (2025 Single)
	AdditionalStandardDeduction decimal.Decimal `yaml:"additional_standard_deduction_65_plus" json:"additional_standard_deduction_65_plus"` // Default: 1550 (per person 65+)

	// Tax brackets for 2025 (updated annually)
	TaxBrackets2025       []TaxBracket `yaml:"tax_brackets_2025" json:"tax_brackets_2025"`
	TaxBrackets2025Single []TaxBracket `yaml:"tax_brackets_2025_single" json:"tax_brackets_2025_single"`
}

// TaxBracket represents a federal tax bracket
type TaxBracket struct {
	Min  decimal.Decimal `yaml:"min" json:"min"`   // Minimum income for bracket
	Max  decimal.Decimal `yaml:"max" json:"max"`   // Maximum income for bracket (use 999999999 for top bracket)
	Rate decimal.Decimal `yaml:"rate" json:"rate"` // Tax rate for this bracket
}

// StateLocalTaxConfig contains state and local tax configuration
type StateLocalTaxConfig struct {
	// Pennsylvania state tax (flat rate)
	PennsylvaniaRate decimal.Decimal `yaml:"pennsylvania_rate" json:"pennsylvania_rate"` // Default: 0.0307 (3.07%)

	// Upper Makefield Township EIT (local tax)
	UpperMakefieldEITRate decimal.Decimal `yaml:"upper_makefield_eit_rate" json:"upper_makefield_eit_rate"` // Default: 0.01 (1% on earned income)

	// New Jersey state tax (simplified rate for now, or use brackets if implementing full logic)
	// NJ has progressive brackets, but we can allow a configurable effective rate or top rate here.
	// For now, let's just add the field.
	NewJerseyRate decimal.Decimal `yaml:"new_jersey_rate,omitempty" json:"new_jersey_rate,omitempty"`
}

// FICATaxConfig contains FICA tax configuration (updated annually)
type FICATaxConfig struct {
	// Tax year the configuration applies to
	Year int `yaml:"year" json:"year"`

	// Social Security tax
	SocialSecurityWageBase decimal.Decimal `yaml:"social_security_wage_base" json:"social_security_wage_base"` // Default: 176100 (2025)
	SocialSecurityRate     decimal.Decimal `yaml:"social_security_rate" json:"social_security_rate"`           // Default: 0.062 (6.2%)

	// Medicare tax
	MedicareRate decimal.Decimal `yaml:"medicare_rate" json:"medicare_rate"` // Default: 0.0145 (1.45%)

	// Additional Medicare tax (for high earners)
	AdditionalMedicareRate decimal.Decimal `yaml:"additional_medicare_rate" json:"additional_medicare_rate"`   // Default: 0.009 (0.9%)
	HighIncomeThresholdMFJ decimal.Decimal `yaml:"high_income_threshold_mfj" json:"high_income_threshold_mfj"` // Default: 250000 (MFJ)
}

// MedicareConfig contains Medicare Part B premium configuration (updated annually)
type MedicareConfig struct {
	// Base Part B premium
	BasePremium2025 decimal.Decimal `yaml:"base_premium_2025" json:"base_premium_2025"` // Default: 185.00 (2025)

	// Annual premium inflation rate applied when projecting future costs
	PremiumInflationRate decimal.Decimal `yaml:"premium_inflation_rate" json:"premium_inflation_rate"` // Default: 0.055 (5.5%)

	// IRMAA (Income-Related Monthly Adjustment Amount) thresholds
	IRMAAThresholds []MedicareIRMAAThreshold `yaml:"irmaa_thresholds" json:"irmaa_thresholds"`
}

// MedicareIRMAAThreshold represents an IRMAA income threshold and corresponding surcharge
type MedicareIRMAAThreshold struct {
	IncomeThresholdSingle decimal.Decimal `yaml:"income_threshold_single" json:"income_threshold_single"` // For single filers
	IncomeThresholdJoint  decimal.Decimal `yaml:"income_threshold_joint" json:"income_threshold_joint"`   // For married filing jointly
	MonthlySurcharge      decimal.Decimal `yaml:"monthly_surcharge" json:"monthly_surcharge"`             // Additional monthly premium per person
}

// FEHBConfig contains FEHB (Federal Employees Health Benefits) configuration
type FEHBConfig struct {
	// Pay periods per year (typically 26 for bi-weekly pay)
	PayPeriodsPerYear int `yaml:"pay_periods_per_year" json:"pay_periods_per_year"` // Default: 26

	// Retirement premium calculation method
	// Options: "same_as_active", "reduced_rate", "custom_multiplier"
	RetirementCalculationMethod string `yaml:"retirement_calculation_method" json:"retirement_calculation_method"` // Default: "same_as_active"

	// Custom multiplier for retirement premiums (if using custom_multiplier method)
	RetirementPremiumMultiplier decimal.Decimal `yaml:"retirement_premium_multiplier" json:"retirement_premium_multiplier"` // Default: 1.0
}

// TSPStatisticalModels contains statistical parameters for each TSP fund
// These are calculated from historical data but can be overridden
type TSPStatisticalModels struct {
	CFund TSPFundStats `yaml:"c_fund" json:"c_fund"` // Large Cap Stock Index
	SFund TSPFundStats `yaml:"s_fund" json:"s_fund"` // Small Cap Stock Index
	IFund TSPFundStats `yaml:"i_fund" json:"i_fund"` // International Stock Index
	FFund TSPFundStats `yaml:"f_fund" json:"f_fund"` // Fixed Income Index
	GFund TSPFundStats `yaml:"g_fund" json:"g_fund"` // Government Securities
}

// TSPFundStats contains statistical parameters for a TSP fund
type TSPFundStats struct {
	Mean        decimal.Decimal `yaml:"mean" json:"mean"`                 // Historical mean return
	StandardDev decimal.Decimal `yaml:"standard_dev" json:"standard_dev"` // Historical standard deviation
	DataSource  string          `yaml:"data_source" json:"data_source"`   // Source of the data (e.g., "TSP.gov 1988-2024")
	LastUpdated string          `yaml:"last_updated" json:"last_updated"` // When these stats were calculated
}

// Configuration represents the complete input configuration
type Configuration struct {
	PersonalDetails   map[string]Employee `yaml:"personal_details" json:"personal_details"`
	GlobalAssumptions GlobalAssumptions   `yaml:"global_assumptions" json:"global_assumptions"`
	Scenarios         []Scenario          `yaml:"scenarios" json:"scenarios"`
}

// Age calculates the age of the employee at a given date
func (e *Employee) Age(atDate time.Time) int {
	age := atDate.Year() - e.BirthDate.Year()
	if atDate.YearDay() < e.BirthDate.YearDay() {
		age--
	}
	return age
}

// CreditableService calculates service used for eligibility (excludes sick leave)
func (e *Employee) CreditableService(atDate time.Time) decimal.Decimal {
	// Calculate basic service time from hire date to retirement/calculation date
	serviceDuration := atDate.Sub(e.HireDate)
	years := decimal.NewFromFloat(serviceDuration.Hours() / 24 / 365.25)
	return years.Round(4)
}

// YearsOfService calculates the years of service at a given date, including sick leave credit
func (e *Employee) YearsOfService(atDate time.Time) decimal.Decimal {
	years := e.CreditableService(atDate)
	// Add sick leave credit if available
	// FERS Rule: Unused sick leave at retirement counts toward service computation
	// 1 day of sick leave = 1 day of service credit (8 hours = 1 day)
	if e.SickLeaveHours.GreaterThan(decimal.Zero) {
		sickLeaveDays := e.SickLeaveHours.Div(decimal.NewFromInt(8))
		sickLeaveYears := sickLeaveDays.Div(decimal.NewFromFloat(365.25))
		years = years.Add(sickLeaveYears)
	}

	return years.Round(4) // Round to 4 decimal places for precision
}

// FullRetirementAge calculates the Social Security Full Retirement Age based on birth year
func (e *Employee) FullRetirementAge() dateutil.RetirementAge {
	return dateutil.FullRetirementAge(e.BirthDate)
}

// MinimumRetirementAge calculates the FERS Minimum Retirement Age
func (e *Employee) MinimumRetirementAge() dateutil.RetirementAge {
	return dateutil.MinimumRetirementAge(e.BirthDate)
}

// TotalTSPBalance returns the combined traditional and Roth TSP balance
func (e *Employee) TotalTSPBalance() decimal.Decimal {
	return e.TSPBalanceTraditional.Add(e.TSPBalanceRoth)
}

// EmploymentCategory returns a normalized employment type string
func (e *Employee) EmploymentCategory() string {
	if e == nil {
		return EmploymentTypeFederal
	}
	typeLower := strings.ToLower(strings.TrimSpace(e.EmploymentType))
	switch typeLower {
	case "", EmploymentTypeFederal, "fed", "federal employee":
		return EmploymentTypeFederal
	case EmploymentTypeNonFederal, "nonfederal", "private", "spouse":
		return EmploymentTypeNonFederal
	default:
		return typeLower
	}
}

// HasTSPAccount indicates whether the employee can contribute to the TSP
func (e *Employee) HasTSPAccount() bool {
	return e.EmploymentCategory() == EmploymentTypeFederal
}

// AnnualTSPContribution calculates the annual TSP contribution amount
func (e *Employee) AnnualTSPContribution() decimal.Decimal {
	if !e.HasTSPAccount() {
		return decimal.Zero
	}
	return e.CurrentSalary.Mul(e.TSPContributionPercent)
}

// AgencyMatch calculates the annual agency match (5% of salary if contributing at least 5%)
func (e *Employee) AgencyMatch() decimal.Decimal {
	if !e.HasTSPAccount() {
		return decimal.Zero
	}
	if e.TSPContributionPercent.GreaterThanOrEqual(decimal.NewFromFloat(0.05)) {
		return e.CurrentSalary.Mul(decimal.NewFromFloat(0.05))
	}
	return decimal.Zero
}

// TotalAnnualTSPContribution returns the combined employee and agency contributions
func (e *Employee) TotalAnnualTSPContribution() decimal.Decimal {
	return e.AnnualTSPContribution().Add(e.AgencyMatch())
}

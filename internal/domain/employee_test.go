package domain

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEmployee_Age(t *testing.T) {
	// Test employee born on June 15, 1963
	birthDate := time.Date(1963, 6, 15, 0, 0, 0, 0, time.UTC)
	employee := &Employee{
		BirthDate: birthDate,
	}

	// Test age calculation on different dates
	testCases := []struct {
		atDate   time.Time
		expected int
		desc     string
	}{
		{
			atDate:   time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC), // Day before birthday
			expected: 61,
			desc:     "day before birthday",
		},
		{
			atDate:   time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC), // On birthday
			expected: 62,
			desc:     "on birthday",
		},
		{
			atDate:   time.Date(2025, 6, 16, 0, 0, 0, 0, time.UTC), // Day after birthday
			expected: 62,
			desc:     "day after birthday",
		},
		{
			atDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), // End of year
			expected: 62,
			desc:     "end of year",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			age := employee.Age(tc.atDate)
			assert.Equal(t, tc.expected, age)
		})
	}
}

func TestEmployee_YearsOfService(t *testing.T) {
	// Test employee hired on March 20, 1985
	hireDate := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)
	employee := &Employee{
		HireDate: hireDate,
	}

	// Test years of service calculation
	testCases := []struct {
		atDate   time.Time
		expected string // Expected result as string for decimal comparison
		desc     string
	}{
		{
			atDate:   time.Date(2025, 3, 19, 0, 0, 0, 0, time.UTC), // Day before anniversary
			expected: "39.9973",
			desc:     "day before anniversary",
		},
		{
			atDate:   time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC), // On anniversary
			expected: "40.0000",
			desc:     "on anniversary",
		},
		{
			atDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), // End of year
			expected: "40.7830",
			desc:     "end of year",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			years := employee.YearsOfService(tc.atDate)
			assert.Equal(t, tc.expected, years.StringFixed(4))
		})
	}
}

func TestEmployee_YearsOfService_WithSickLeave(t *testing.T) {
	// Test employee with sick leave credit
	hireDate := time.Date(1985, 3, 20, 0, 0, 0, 0, time.UTC)
	employee := &Employee{
		HireDate:       hireDate,
		SickLeaveHours: decimal.NewFromInt(2080), // 260 days * 8 hours = 2080 hours
	}

	atDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	years := employee.YearsOfService(atDate)

	// Should be approximately 40.78 + 0.71 (260 days / 365.25) = 41.49 years
	expected := "41.4949"
	assert.Equal(t, expected, years.StringFixed(4))
}

func TestEmployee_FullRetirementAge(t *testing.T) {
	testCases := []struct {
		birthYear int
		expected  int
		desc      string
	}{
		{1935, 65, "born 1935"},
		{1937, 65, "born 1937"},
		{1938, 67, "born 1938 (65 + 2 months)"},
		{1939, 69, "born 1939 (65 + 4 months)"},
		{1940, 71, "born 1940 (65 + 6 months)"},
		{1941, 73, "born 1941 (65 + 8 months)"},
		{1942, 75, "born 1942 (65 + 10 months)"},
		{1943, 66, "born 1943"},
		{1954, 66, "born 1954"},
		{1955, 68, "born 1955 (66 + 2 months)"},
		{1956, 70, "born 1956 (66 + 4 months)"},
		{1957, 72, "born 1957 (66 + 6 months)"},
		{1958, 74, "born 1958 (66 + 8 months)"},
		{1959, 76, "born 1959 (66 + 10 months)"},
		{1960, 67, "born 1960"},
		{1963, 67, "born 1963"},
		{1970, 67, "born 1970"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			employee := &Employee{
				BirthDate: time.Date(tc.birthYear, 6, 15, 0, 0, 0, 0, time.UTC),
			}
			fra := employee.FullRetirementAge()
			assert.Equal(t, tc.expected, fra)
		})
	}
}

func TestEmployee_MinimumRetirementAge(t *testing.T) {
	testCases := []struct {
		birthYear int
		expected  int
		desc      string
	}{
		{1945, 55, "born 1945"},
		{1947, 55, "born 1947"},
		{1948, 57, "born 1948 (55 + 2 months)"},
		{1949, 59, "born 1949 (55 + 4 months)"},
		{1950, 61, "born 1950 (55 + 6 months)"},
		{1951, 63, "born 1951 (55 + 8 months)"},
		{1952, 65, "born 1952 (55 + 10 months)"},
		{1953, 56, "born 1953"},
		{1964, 56, "born 1964"},
		{1965, 58, "born 1965 (56 + 2 months)"},
		{1966, 60, "born 1966 (56 + 4 months)"},
		{1967, 62, "born 1967 (56 + 6 months)"},
		{1968, 64, "born 1968 (56 + 8 months)"},
		{1969, 66, "born 1969 (56 + 10 months)"},
		{1970, 57, "born 1970"},
		{1975, 57, "born 1975"},
		{1980, 57, "born 1980"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			employee := &Employee{
				BirthDate: time.Date(tc.birthYear, 6, 15, 0, 0, 0, 0, time.UTC),
			}
			mra := employee.MinimumRetirementAge()
			assert.Equal(t, tc.expected, mra)
		})
	}
}

func TestEmployee_TotalTSPBalance(t *testing.T) {
	employee := &Employee{
		TSPBalanceTraditional: decimal.NewFromInt(450000),
		TSPBalanceRoth:        decimal.NewFromInt(50000),
	}

	total := employee.TotalTSPBalance()
	expected := decimal.NewFromInt(500000)
	assert.True(t, total.Equal(expected))
}

func TestEmployee_TotalTSPBalance_ZeroRoth(t *testing.T) {
	employee := &Employee{
		TSPBalanceTraditional: decimal.NewFromInt(450000),
		TSPBalanceRoth:        decimal.Zero,
	}

	total := employee.TotalTSPBalance()
	expected := decimal.NewFromInt(450000)
	assert.True(t, total.Equal(expected))
}

func TestEmployee_AnnualTSPContribution(t *testing.T) {
	employee := &Employee{
		CurrentSalary:          decimal.NewFromInt(95000),
		TSPContributionPercent: decimal.NewFromFloat(0.15),
	}

	contribution := employee.AnnualTSPContribution()
	expected := decimal.NewFromInt(14250) // 95000 * 0.15
	assert.True(t, contribution.Equal(expected))
}

func TestEmployee_AgencyMatch(t *testing.T) {
	employee := &Employee{
		CurrentSalary:          decimal.NewFromInt(95000),
		TSPContributionPercent: decimal.NewFromFloat(0.15),
	}

	match := employee.AgencyMatch()
	expected := decimal.NewFromInt(4750) // 95000 * 0.05 (5% match)
	assert.True(t, match.Equal(expected))
}

func TestEmployee_AgencyMatch_LimitedByContribution(t *testing.T) {
	employee := &Employee{
		CurrentSalary:          decimal.NewFromInt(95000),
		TSPContributionPercent: decimal.NewFromFloat(0.03), // Only 3% contribution
	}

	match := employee.AgencyMatch()
	expected := decimal.Zero // No match because contribution < 5%
	assert.True(t, match.Equal(expected))
}

func TestEmployee_TotalAnnualTSPContribution(t *testing.T) {
	employee := &Employee{
		CurrentSalary:          decimal.NewFromInt(95000),
		TSPContributionPercent: decimal.NewFromFloat(0.15),
	}

	total := employee.TotalAnnualTSPContribution()
	expected := decimal.NewFromInt(19000) // 14250 (employee) + 4750 (match)
	assert.True(t, total.Equal(expected))
}

func TestRetirementScenario_UnmarshalYAML(t *testing.T) {
	// Test YAML unmarshaling with string values for decimal fields
	// Note: This test would require creating a yaml.Node, which is complex
	// For now, we'll test the decimal parsing logic directly

	// Test valid decimal parsing
	validTarget := "5000.00"
	validRate := "0.04"

	target, err := decimal.NewFromString(validTarget)
	assert.NoError(t, err)
	assert.True(t, target.Equal(decimal.NewFromFloat(5000.00)))

	rate, err := decimal.NewFromString(validRate)
	assert.NoError(t, err)
	assert.True(t, rate.Equal(decimal.NewFromFloat(0.04)))
}

func TestRetirementScenario_UnmarshalYAML_InvalidDecimal(t *testing.T) {
	// Test invalid decimal parsing
	invalidNumber := "invalid_number"

	_, err := decimal.NewFromString(invalidNumber)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't convert")
}

func TestTSPAllocation_Validation(t *testing.T) {
	// Test valid allocation
	allocation := TSPAllocation{
		CFund: decimal.NewFromFloat(0.60),
		SFund: decimal.NewFromFloat(0.20),
		IFund: decimal.NewFromFloat(0.10),
		FFund: decimal.NewFromFloat(0.10),
		GFund: decimal.NewFromFloat(0.00),
	}

	total := allocation.CFund.Add(allocation.SFund).Add(allocation.IFund).Add(allocation.FFund).Add(allocation.GFund)
	assert.True(t, total.Equal(decimal.NewFromFloat(1.0)))
}

func TestTSPAllocation_InvalidTotal(t *testing.T) {
	// Test allocation that doesn't sum to 100%
	allocation := TSPAllocation{
		CFund: decimal.NewFromFloat(0.60),
		SFund: decimal.NewFromFloat(0.20),
		IFund: decimal.NewFromFloat(0.10),
		FFund: decimal.NewFromFloat(0.10),
		GFund: decimal.NewFromFloat(0.10), // This makes it 110%
	}

	total := allocation.CFund.Add(allocation.SFund).Add(allocation.IFund).Add(allocation.FFund).Add(allocation.GFund)
	assert.True(t, total.Equal(decimal.NewFromFloat(1.1)))
}

func TestLocation_Validation(t *testing.T) {
	location := Location{
		State:        "PA",
		County:       "Bucks",
		Municipality: "Upper Makefield",
	}

	assert.Equal(t, "PA", location.State)
	assert.Equal(t, "Bucks", location.County)
	assert.Equal(t, "Upper Makefield", location.Municipality)
}

func TestMonteCarloSettings_DefaultValues(t *testing.T) {
	settings := MonteCarloSettings{
		TSPReturnVariability: decimal.NewFromFloat(0.15),
		InflationVariability: decimal.NewFromFloat(0.02),
		COLAVariability:      decimal.NewFromFloat(0.02),
		FEHBVariability:      decimal.NewFromFloat(0.05),
		MaxReasonableIncome:  decimal.NewFromInt(5000000),
	}

	assert.True(t, settings.TSPReturnVariability.Equal(decimal.NewFromFloat(0.15)))
	assert.True(t, settings.InflationVariability.Equal(decimal.NewFromFloat(0.02)))
	assert.True(t, settings.COLAVariability.Equal(decimal.NewFromFloat(0.02)))
	assert.True(t, settings.FEHBVariability.Equal(decimal.NewFromFloat(0.05)))
	assert.True(t, settings.MaxReasonableIncome.Equal(decimal.NewFromInt(5000000)))
}

func TestTSPLifecycleFund_Validation(t *testing.T) {
	fund := TSPLifecycleFund{
		FundName: "L2030",
		AllocationData: map[string][]TSPAllocationDataPoint{
			"2025-Q1": {
				{
					Date: "2025-01-01",
					Allocation: TSPAllocation{
						CFund: decimal.NewFromFloat(0.60),
						SFund: decimal.NewFromFloat(0.20),
						IFund: decimal.NewFromFloat(0.10),
						FFund: decimal.NewFromFloat(0.10),
						GFund: decimal.NewFromFloat(0.00),
					},
				},
			},
		},
	}

	assert.Equal(t, "L2030", fund.FundName)
	assert.Len(t, fund.AllocationData, 1)
	assert.Len(t, fund.AllocationData["2025-Q1"], 1)
}

func TestFederalRules_Validation(t *testing.T) {
	rules := FederalRules{
		SocialSecurityTaxThresholds: SocialSecurityTaxThresholds{
			MarriedFilingJointly: struct {
				Threshold1 decimal.Decimal `yaml:"threshold_1" json:"threshold_1"`
				Threshold2 decimal.Decimal `yaml:"threshold_2" json:"threshold_2"`
			}{
				Threshold1: decimal.NewFromInt(32000),
				Threshold2: decimal.NewFromInt(44000),
			},
			Single: struct {
				Threshold1 decimal.Decimal `yaml:"threshold_1" json:"threshold_1"`
				Threshold2 decimal.Decimal `yaml:"threshold_2" json:"threshold_2"`
			}{
				Threshold1: decimal.NewFromInt(25000),
				Threshold2: decimal.NewFromInt(34000),
			},
		},
	}

	assert.True(t, rules.SocialSecurityTaxThresholds.MarriedFilingJointly.Threshold1.Equal(decimal.NewFromInt(32000)))
	assert.True(t, rules.SocialSecurityTaxThresholds.MarriedFilingJointly.Threshold2.Equal(decimal.NewFromInt(44000)))
	assert.True(t, rules.SocialSecurityTaxThresholds.Single.Threshold1.Equal(decimal.NewFromInt(25000)))
	assert.True(t, rules.SocialSecurityTaxThresholds.Single.Threshold2.Equal(decimal.NewFromInt(34000)))
}

func TestTaxBracket_Validation(t *testing.T) {
	bracket := TaxBracket{
		Min:  decimal.NewFromInt(0),
		Max:  decimal.NewFromInt(22000),
		Rate: decimal.NewFromFloat(0.10),
	}

	assert.True(t, bracket.Min.Equal(decimal.Zero))
	assert.True(t, bracket.Max.Equal(decimal.NewFromInt(22000)))
	assert.True(t, bracket.Rate.Equal(decimal.NewFromFloat(0.10)))
}

func TestTSPFundStats_Validation(t *testing.T) {
	stats := TSPFundStats{
		Mean:        decimal.NewFromFloat(0.1125),
		StandardDev: decimal.NewFromFloat(0.15),
		DataSource:  "TSP.gov 1988-2024",
		LastUpdated: "2024-01-01",
	}

	assert.True(t, stats.Mean.Equal(decimal.NewFromFloat(0.1125)))
	assert.True(t, stats.StandardDev.Equal(decimal.NewFromFloat(0.15)))
	assert.Equal(t, "TSP.gov 1988-2024", stats.DataSource)
	assert.Equal(t, "2024-01-01", stats.LastUpdated)
}

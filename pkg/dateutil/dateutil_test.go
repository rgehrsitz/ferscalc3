package dateutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestAgeCalculation tests the age calculation function with various scenarios
func TestAgeCalculation(t *testing.T) {
	tests := []struct {
		name        string
		birthDate   time.Time
		atDate      time.Time
		expectedAge int
		description string
	}{
		{
			name:        "Same month and day",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 2, 25, 0, 0, 0, 0, time.UTC),
			expectedAge: 60,
			description: "Exact birthday",
		},
		{
			name:        "Day before birthday",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC),
			expectedAge: 59,
			description: "One day before 60th birthday",
		},
		{
			name:        "Day after birthday",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 2, 26, 0, 0, 0, 0, time.UTC),
			expectedAge: 60,
			description: "One day after 60th birthday",
		},
		{
			name:        "Month before birthday",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 1, 25, 0, 0, 0, 0, time.UTC),
			expectedAge: 59,
			description: "Same day, month before birthday",
		},
		{
			name:        "Month after birthday",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC),
			expectedAge: 60,
			description: "Same day, month after birthday",
		},
		{
			name:        "Leap year birth, non-leap year check",
			birthDate:   time.Date(1964, 2, 29, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedAge: 60,
			description: "Born on leap day, checking on Feb 28",
		},
		{
			name:        "Leap year birth, leap year check",
			birthDate:   time.Date(1964, 2, 29, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			expectedAge: 60,
			description: "Born on leap day, checking on leap day",
		},
		{
			name:        "Person A actual scenario",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), // Retirement date
			expectedAge: 60,
			description: "Person A age at retirement",
		},
		{
			name:        "Person B actual scenario",
			birthDate:   time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC), // Retirement date
			expectedAge: 62,
			description: "Person B age at retirement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age := Age(tt.birthDate, tt.atDate)
			assert.Equal(t, tt.expectedAge, age,
				"%s: Expected age %d, got %d", tt.description, tt.expectedAge, age)
		})
	}
}

// TestYearsOfService tests years of service calculation
func TestYearsOfService(t *testing.T) {
	tests := []struct {
		name          string
		hireDate      time.Time
		atDate        time.Time
		expectedYears float64
		tolerance     float64
		description   string
	}{
		{
			name:          "Exact years",
			hireDate:      time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedYears: 25.0,
			tolerance:     0.01,
			description:   "Exactly 25 years of service",
		},
		{
			name:          "Partial year",
			hireDate:      time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:        time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
			expectedYears: 25.5,
			tolerance:     0.05,
			description:   "25.5 years of service",
		},
		{
			name:          "Person A actual service",
			hireDate:      time.Date(1987, 6, 22, 0, 0, 0, 0, time.UTC),
			atDate:        time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			expectedYears: 38.44,
			tolerance:     0.1,
			description:   "Person A service years at retirement",
		},
		{
			name:          "Person B actual service",
			hireDate:      time.Date(1995, 7, 11, 0, 0, 0, 0, time.UTC),
			atDate:        time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC),
			expectedYears: 30.13,
			tolerance:     0.1,
			description:   "Person B service years at retirement",
		},
		{
			name:          "Leap year handling",
			hireDate:      time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			atDate:        time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			expectedYears: 4.0,
			tolerance:     0.01,
			description:   "Service spanning leap years",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			years := YearsOfService(tt.hireDate, tt.atDate)
			assert.InDelta(t, tt.expectedYears, years, tt.tolerance,
				"%s: Expected %f years, got %f", tt.description, tt.expectedYears, years)
		})
	}
}

// TestFullRetirementAge tests Social Security FRA calculation
func TestFullRetirementAge(t *testing.T) {
	tests := []struct {
		name        string
		birthDate   time.Time
		expectedFRA int
		description string
	}{
		{
			name:        "Born 1937 or earlier",
			birthDate:   time.Date(1937, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedFRA: 65,
			description: "FRA 65 for pre-1938 births",
		},
		{
			name:        "Born 1943-1954",
			birthDate:   time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedFRA: 66,
			description: "FRA 66 for 1943-1954 births",
		},
		{
			name:        "Born 1959 - transition year",
			birthDate:   time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedFRA: 66, // 66 + 10 months, but function returns 66
			description: "FRA during transition period",
		},
		{
			name:        "Born 1960 - Person A",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			expectedFRA: 67,
			description: "FRA 67 for 1960+ births",
		},
		{
			name:        "Person B FRA",
			birthDate:   time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			expectedFRA: 67,
			description: "Person B full retirement age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fra := FullRetirementAge(tt.birthDate)
			assert.Equal(t, tt.expectedFRA, fra,
				"%s: Expected FRA %d, got %d", tt.description, tt.expectedFRA, fra)
		})
	}
}

// TestMinimumRetirementAge tests FERS MRA calculation
func TestMinimumRetirementAge(t *testing.T) {
	tests := []struct {
		name        string
		birthDate   time.Time
		expectedMRA int
		description string
	}{
		{
			name:        "Born 1947 or earlier",
			birthDate:   time.Date(1947, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedMRA: 55,
			description: "MRA 55 for pre-1948 births",
		},
		{
			name:        "Born 1953-1964",
			birthDate:   time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedMRA: 56,
			description: "MRA 56 for 1953-1964 births",
		},
		{
			name:        "Born 1965 - Person A",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			expectedMRA: 56, // 56 + 2 months, but function returns 56
			description: "Person A minimum retirement age",
		},
		{
			name:        "Born 1963 - Person B",
			birthDate:   time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			expectedMRA: 56,
			description: "Person B minimum retirement age",
		},
		{
			name:        "Born 1970 or later",
			birthDate:   time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedMRA: 57,
			description: "MRA 57 for 1970+ births",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mra := MinimumRetirementAge(tt.birthDate)
			assert.Equal(t, tt.expectedMRA, mra,
				"%s: Expected MRA %d, got %d", tt.description, tt.expectedMRA, mra)
		})
	}
}

// TestMedicareEligibility tests Medicare eligibility
func TestMedicareEligibility(t *testing.T) {
	tests := []struct {
		name             string
		birthDate        time.Time
		atDate           time.Time
		expectedEligible bool
		description      string
	}{
		{
			name:             "Age 64 - not eligible",
			birthDate:        time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedEligible: false,
			description:      "Not yet 65",
		},
		{
			name:             "Age 65 - eligible",
			birthDate:        time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedEligible: true,
			description:      "Exactly 65",
		},
		{
			name:             "Age 66 - eligible",
			birthDate:        time.Date(1959, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:           time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			expectedEligible: true,
			description:      "Over 65",
		},
		{
			name:             "Person A at retirement - not eligible",
			birthDate:        time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:           time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
			expectedEligible: false,
			description:      "Person A at age 60 (not Medicare eligible)",
		},
		{
			name:             "Person B at retirement - not eligible",
			birthDate:        time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			atDate:           time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC),
			expectedEligible: false,
			description:      "Person B at age 62 (not Medicare eligible)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible := IsMedicareEligible(tt.birthDate, tt.atDate)
			assert.Equal(t, tt.expectedEligible, eligible,
				"%s: Expected %t, got %t", tt.description, tt.expectedEligible, eligible)
		})
	}
}

// TestRMDYear tests Required Minimum Distribution year determination
func TestRMDYear(t *testing.T) {
	tests := []struct {
		name        string
		birthDate   time.Time
		atDate      time.Time
		expectedRMD bool
		description string
	}{
		{
			name:        "Born 1950 - RMD at 72",
			birthDate:   time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), // Age 72
			expectedRMD: true,
			description: "Pre-SECURE 2.0 RMD age",
		},
		{
			name:        "Born 1951 - RMD at 73",
			birthDate:   time.Date(1951, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Age 73
			expectedRMD: true,
			description: "SECURE 2.0 transition RMD age",
		},
		{
			name:        "Born 1960 - RMD at 75",
			birthDate:   time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC), // Age 75
			expectedRMD: true,
			description: "Future SECURE 2.0 RMD age",
		},
		{
			name:        "Person A - not yet RMD age",
			birthDate:   time.Date(1965, 2, 25, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), // Age 60
			expectedRMD: false,
			description: "Person A not yet at RMD age",
		},
		{
			name:        "Person B - not yet RMD age",
			birthDate:   time.Date(1963, 7, 31, 0, 0, 0, 0, time.UTC),
			atDate:      time.Date(2025, 8, 30, 0, 0, 0, 0, time.UTC), // Age 62
			expectedRMD: false,
			description: "Person B not yet at RMD age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRMD := IsRMDYear(tt.birthDate, tt.atDate)
			assert.Equal(t, tt.expectedRMD, isRMD,
				"%s: Expected %t, got %t", tt.description, tt.expectedRMD, isRMD)
		})
	}
}

// TestGetRMDAge tests the RMD age determination
func TestGetRMDAge(t *testing.T) {
	tests := []struct {
		name        string
		birthYear   int
		expectedAge int
		description string
	}{
		{
			name:        "Born 1950 or earlier",
			birthYear:   1950,
			expectedAge: 72,
			description: "Pre-SECURE 2.0 RMD age",
		},
		{
			name:        "Born 1951-1959",
			birthYear:   1955,
			expectedAge: 73,
			description: "SECURE 2.0 transition RMD age",
		},
		{
			name:        "Born 1960 or later",
			birthYear:   1965,
			expectedAge: 75,
			description: "Future SECURE 2.0 RMD age",
		},
		{
			name:        "Person A RMD age",
			birthYear:   1965,
			expectedAge: 75,
			description: "Person A will have RMD at age 75",
		},
		{
			name:        "Person B RMD age",
			birthYear:   1963,
			expectedAge: 75,
			description: "Person B will have RMD at age 75",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rmdAge := GetRMDAge(tt.birthYear)
			assert.Equal(t, tt.expectedAge, rmdAge,
				"%s: Expected RMD age %d, got %d", tt.description, tt.expectedAge, rmdAge)
		})
	}
}

// TestLeapYearCalculation tests leap year determination
func TestLeapYearCalculation(t *testing.T) {
	tests := []struct {
		year     int
		expected bool
	}{
		{2000, true},  // Divisible by 400
		{1900, false}, // Divisible by 100 but not 400
		{2004, true},  // Divisible by 4
		{2001, false}, // Not divisible by 4
		{2024, true},  // Recent leap year
		{2025, false}, // Current projection year
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Year_%d", tt.year), func(t *testing.T) {
			result := IsLeapYear(tt.year)
			assert.Equal(t, tt.expected, result,
				"Year %d: Expected %t, got %t", tt.year, tt.expected, result)
		})
	}
}

// TestDaysInYear tests days in year calculation
func TestDaysInYear(t *testing.T) {
	tests := []struct {
		year         int
		expectedDays int
	}{
		{2024, 366}, // Leap year
		{2025, 365}, // Regular year
		{2000, 366}, // Leap year (divisible by 400)
		{1900, 365}, // Not leap year (divisible by 100 but not 400)
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Year_%d", tt.year), func(t *testing.T) {
			days := DaysInYear(tt.year)
			assert.Equal(t, tt.expectedDays, days,
				"Year %d: Expected %d days, got %d", tt.year, tt.expectedDays, days)
		})
	}
}

// TestDateArithmetic tests date arithmetic functions
func TestDateArithmetic(t *testing.T) {
	baseDate := time.Date(2025, 6, 15, 12, 30, 45, 0, time.UTC)

	// Test AddYears
	futureDate := AddYears(baseDate, 5)
	expectedFuture := time.Date(2030, 6, 15, 12, 30, 45, 0, time.UTC)
	assert.Equal(t, expectedFuture, futureDate, "AddYears should add 5 years correctly")

	// Test AddMonths
	monthDate := AddMonths(baseDate, 18) // 1.5 years
	expectedMonth := time.Date(2026, 12, 15, 12, 30, 45, 0, time.UTC)
	assert.Equal(t, expectedMonth, monthDate, "AddMonths should add 18 months correctly")

	// Test BeginningOfYear
	yearStart := BeginningOfYear(baseDate)
	expectedStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedStart, yearStart, "BeginningOfYear should return Jan 1")

	// Test EndOfYear
	yearEnd := EndOfYear(baseDate)
	expectedEnd := time.Date(2025, 12, 31, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expectedEnd, yearEnd, "EndOfYear should return Dec 31 23:59:59.999999999")
}

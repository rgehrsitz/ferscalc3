package dateutil

import (
	"time"
)

// Age calculates the age at a given date
func Age(birthDate, atDate time.Time) int {
	age := atDate.Year() - birthDate.Year()
	if atDate.Month() < birthDate.Month() || 
		(atDate.Month() == birthDate.Month() && atDate.Day() < birthDate.Day()) {
		age--
	}
	return age
}

// YearsOfService calculates the years of service at a given date
func YearsOfService(hireDate, atDate time.Time) float64 {
	serviceDuration := atDate.Sub(hireDate)
	return serviceDuration.Hours() / 24 / 365.25
}

// YearsOfServiceDecimal calculates the years of service as a decimal for precise calculations
func YearsOfServiceDecimal(hireDate, atDate time.Time) float64 {
	serviceDuration := atDate.Sub(hireDate)
	years := serviceDuration.Hours() / 24 / 365.25
	return years
}

// FullRetirementAge calculates the Social Security Full Retirement Age based on birth year
func FullRetirementAge(birthDate time.Time) int {
	birthYear := birthDate.Year()
	
	switch {
	case birthYear <= 1937:
		return 65
	case birthYear == 1938:
		return 65 // 65 years and 2 months, rounded down
	case birthYear == 1939:
		return 65 // 65 years and 4 months, rounded down
	case birthYear == 1940:
		return 65 // 65 years and 6 months, rounded down
	case birthYear == 1941:
		return 65 // 65 years and 8 months, rounded down
	case birthYear == 1942:
		return 65 // 65 years and 10 months, rounded down
	case birthYear >= 1943 && birthYear <= 1954:
		return 66
	case birthYear == 1955:
		return 66 // 66 years and 2 months, rounded down
	case birthYear == 1956:
		return 66 // 66 years and 4 months, rounded down
	case birthYear == 1957:
		return 66 // 66 years and 6 months, rounded down
	case birthYear == 1958:
		return 66 // 66 years and 8 months, rounded down
	case birthYear == 1959:
		return 66 // 66 years and 10 months, rounded down
	default: // 1960 and later
		return 67
	}
}

// MinimumRetirementAge calculates the FERS Minimum Retirement Age
func MinimumRetirementAge(birthDate time.Time) int {
	birthYear := birthDate.Year()
	
	switch {
	case birthYear <= 1947:
		return 55
	case birthYear == 1948:
		return 55 // 55 years and 2 months, rounded down
	case birthYear == 1949:
		return 55 // 55 years and 4 months, rounded down
	case birthYear == 1950:
		return 55 // 55 years and 6 months, rounded down
	case birthYear == 1951:
		return 55 // 55 years and 8 months, rounded down
	case birthYear == 1952:
		return 55 // 55 years and 10 months, rounded down
	case birthYear >= 1953 && birthYear <= 1964:
		return 56
	case birthYear == 1965:
		return 56 // 56 years and 2 months, rounded down
	case birthYear == 1966:
		return 56 // 56 years and 4 months, rounded down
	case birthYear == 1967:
		return 56 // 56 years and 6 months, rounded down
	case birthYear == 1968:
		return 56 // 56 years and 8 months, rounded down
	case birthYear == 1969:
		return 56 // 56 years and 10 months, rounded down
	case birthYear >= 1970:
		return 57
	default:
		return 57
	}
}

// IsMedicareEligible checks if a person is eligible for Medicare (age 65+)
func IsMedicareEligible(birthDate, atDate time.Time) bool {
	return Age(birthDate, atDate) >= 65
}

// IsRMDYear checks if this is a year when Required Minimum Distributions apply
func IsRMDYear(birthDate, atDate time.Time) bool {
	age := Age(birthDate, atDate)
	birthYear := birthDate.Year()
	
	// SECURE 2.0 Act RMD ages
	switch {
	case birthYear <= 1950:
		return age >= 72
	case birthYear >= 1951 && birthYear <= 1959:
		return age >= 73
	default: // 1960 and later
		return age >= 75
	}
}

// GetRMDAge returns the age when RMDs start for a given birth year
func GetRMDAge(birthYear int) int {
	switch {
	case birthYear <= 1950:
		return 72
	case birthYear >= 1951 && birthYear <= 1959:
		return 73
	default: // 1960 and later
		return 75
	}
}

// YearsUntilDate calculates the number of years between two dates
func YearsUntilDate(fromDate, toDate time.Time) float64 {
	duration := toDate.Sub(fromDate)
	return duration.Hours() / 24 / 365.25
}

// MonthsUntilDate calculates the number of months between two dates
func MonthsUntilDate(fromDate, toDate time.Time) int {
	years := YearsUntilDate(fromDate, toDate)
	return int(years * 12)
}

// IsLeapYear checks if a year is a leap year
func IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// DaysInYear returns the number of days in a given year
func DaysInYear(year int) int {
	if IsLeapYear(year) {
		return 366
	}
	return 365
}

// AddYears adds a specified number of years to a date
func AddYears(date time.Time, years int) time.Time {
	return date.AddDate(years, 0, 0)
}

// AddMonths adds a specified number of months to a date
func AddMonths(date time.Time, months int) time.Time {
	return date.AddDate(0, months, 0)
}

// EndOfYear returns the last day of the year for a given date
func EndOfYear(date time.Time) time.Time {
	return time.Date(date.Year(), 12, 31, 23, 59, 59, 999999999, date.Location())
}

// BeginningOfYear returns the first day of the year for a given date
func BeginningOfYear(date time.Time) time.Time {
	return time.Date(date.Year(), 1, 1, 0, 0, 0, 0, date.Location())
} 
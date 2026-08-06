package dateutil

import (
	"time"
)

// RetirementAge represents an age threshold expressed in years and months.
type RetirementAge struct {
	Years  int
	Months int
}

// TotalMonths returns the total number of months contained in the retirement age.
func (ra RetirementAge) TotalMonths() int {
	return ra.Years*12 + ra.Months
}

// AddTo returns the calendar date when the retirement age is reached given a birth date.
func (ra RetirementAge) AddTo(birthDate time.Time) time.Time {
	return birthDate.AddDate(ra.Years, ra.Months, 0)
}

// HasReachedRetirementAge reports whether referenceDate is on or after the date when the
// target retirement age is attained.
func HasReachedRetirementAge(birthDate, referenceDate time.Time, target RetirementAge) bool {
	targetDate := target.AddTo(birthDate)
	return !referenceDate.Before(targetDate)
}

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

// FullRetirementAge calculates the Social Security Full Retirement Age for
// old-age, wife's, or husband's benefits, per 20 CFR 404.409(a)'s own
// chart -- exact calendar-date brackets ("1/2/38 - 1/1/39"), not birth-YEAR
// brackets. A January 1st birthday genuinely falls in the PRIOR bracket
// (e.g. 1/1/1939 is still "65 + 2", not "65 + 4") per 404.102's own
// age-attainment rule ("You reach a particular age on the day before your
// birthday") -- this table's date boundaries already express that, so no
// separate day-before adjustment is needed at the call site.
//
// Any birth date OTHER than January 1st gives the identical answer this
// function always gave (a plain birthDate.Year() lookup) -- the two only
// diverge on a January 1st birthday, previously placed one bracket too
// high. Ported from the TypeScript engine's identical fix
// (rgehrsitz/ferscalc-web#404), sourced from the same regulation.
//
// Compares calendar-date components (Year/Month/Day), not the time.Time
// instant, matching this file's own Age() convention above -- safe
// regardless of birthDate's Location, since all three components are read
// from that single value rather than compared across two different Locations.
func FullRetirementAge(birthDate time.Time) RetirementAge {
	dob := birthDate.Year()*10000 + int(birthDate.Month())*100 + birthDate.Day()

	switch {
	case dob <= 19380101:
		return RetirementAge{Years: 65}
	case dob <= 19390101:
		return RetirementAge{Years: 65, Months: 2}
	case dob <= 19400101:
		return RetirementAge{Years: 65, Months: 4}
	case dob <= 19410101:
		return RetirementAge{Years: 65, Months: 6}
	case dob <= 19420101:
		return RetirementAge{Years: 65, Months: 8}
	case dob <= 19430101:
		return RetirementAge{Years: 65, Months: 10}
	case dob <= 19550101:
		return RetirementAge{Years: 66}
	case dob <= 19560101:
		return RetirementAge{Years: 66, Months: 2}
	case dob <= 19570101:
		return RetirementAge{Years: 66, Months: 4}
	case dob <= 19580101:
		return RetirementAge{Years: 66, Months: 6}
	case dob <= 19590101:
		return RetirementAge{Years: 66, Months: 8}
	case dob <= 19600101:
		return RetirementAge{Years: 66, Months: 10}
	default: // 1/2/1960 and later
		return RetirementAge{Years: 67}
	}
}

// MinimumRetirementAge calculates the FERS Minimum Retirement Age.
func MinimumRetirementAge(birthDate time.Time) RetirementAge {
	birthYear := birthDate.Year()

	switch {
	case birthYear <= 1947:
		return RetirementAge{Years: 55}
	case birthYear == 1948:
		return RetirementAge{Years: 55, Months: 2}
	case birthYear == 1949:
		return RetirementAge{Years: 55, Months: 4}
	case birthYear == 1950:
		return RetirementAge{Years: 55, Months: 6}
	case birthYear == 1951:
		return RetirementAge{Years: 55, Months: 8}
	case birthYear == 1952:
		return RetirementAge{Years: 55, Months: 10}
	case birthYear >= 1953 && birthYear <= 1964:
		return RetirementAge{Years: 56}
	case birthYear == 1965:
		return RetirementAge{Years: 56, Months: 2}
	case birthYear == 1966:
		return RetirementAge{Years: 56, Months: 4}
	case birthYear == 1967:
		return RetirementAge{Years: 56, Months: 6}
	case birthYear == 1968:
		return RetirementAge{Years: 56, Months: 8}
	case birthYear == 1969:
		return RetirementAge{Years: 56, Months: 10}
	case birthYear >= 1970:
		return RetirementAge{Years: 57}
	default:
		return RetirementAge{Years: 57}
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

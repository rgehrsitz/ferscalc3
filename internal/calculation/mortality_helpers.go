package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// deriveDeathYearIndexes returns 0-based projection year indexes for each death if within projection horizon.
func deriveDeathYearIndexes(scenario *domain.Scenario, personA, personB *domain.Employee, projectionYears int) (personAIdx *int, personBIdx *int) {
	if scenario == nil || scenario.Mortality == nil {
		return nil, nil
	}
	baseYear := ProjectionBaseYear
	if scenario.Mortality.PersonA != nil {
		if scenario.Mortality.PersonA.DeathDate != nil {
			y := scenario.Mortality.PersonA.DeathDate.Year() - baseYear
			if y >= 0 && y < projectionYears {
				personAIdx = &y
			}
		} else if scenario.Mortality.PersonA.DeathAge != nil {
			targetYear := personA.BirthDate.Year() + *scenario.Mortality.PersonA.DeathAge
			y := targetYear - baseYear
			if y >= 0 && y < projectionYears {
				personAIdx = &y
			}
		}
	}
	if scenario.Mortality.PersonB != nil {
		if scenario.Mortality.PersonB.DeathDate != nil {
			y := scenario.Mortality.PersonB.DeathDate.Year() - baseYear
			if y >= 0 && y < projectionYears {
				personBIdx = &y
			}
		} else if scenario.Mortality.PersonB.DeathAge != nil {
			targetYear := personB.BirthDate.Year() + *scenario.Mortality.PersonB.DeathAge
			y := targetYear - baseYear
			if y >= 0 && y < projectionYears {
				personBIdx = &y
			}
		}
	}
	return personAIdx, personBIdx
}

// deathFractionInYear returns fraction of year before death (0<frac<1) and true if death occurs that projection year.
// If only age-based death specified, assumes mid-year (0.5) unless override needed.
func deathFractionInYear(deathIdx *int, year int, deathDate *time.Time) (decimal.Decimal, bool) {
	if deathIdx == nil || year != *deathIdx {
		return decimal.Zero, false
	}
	if deathDate == nil {
		return decimal.NewFromFloat(0.5), true
	}
	yearStart := time.Date(deathDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	daysBefore := deathDate.Sub(yearStart).Hours() / 24.0
	daysInYear := 365.0 // ignore leap for simplicity
	frac := daysBefore / daysInYear
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	return decimal.NewFromFloat(frac), true
}

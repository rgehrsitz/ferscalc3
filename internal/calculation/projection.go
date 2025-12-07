package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

type ssRetirementProration int

const (
	ssProrationFractional ssRetirementProration = iota
	ssProrationMonthlyAfterRetirement
)

type personProjectionState struct {
	employee          *domain.Employee
	scenario          *domain.RetirementScenario
	strategy          TSPWithdrawalStrategy
	traditional       decimal.Decimal
	roth              decimal.Decimal
	retirementYear    int
	deathYearIndex    *int
	mortalitySpec     *domain.MortalitySpec
	deceased          bool
	label             string
	ssProration       ssRetirementProration
	fixedIncomeConfig *domain.FixedRetirementIncome
}

type personAnnualResult struct {
	ageStart        int
	ageEnd          int
	isRetired       bool
	workFraction    decimal.Decimal
	salary          decimal.Decimal
	pension         decimal.Decimal
	survivorPension decimal.Decimal
	socialSecurity  decimal.Decimal
	supplement      decimal.Decimal
	fixedIncome     decimal.Decimal
	rmd             decimal.Decimal
	tspWithdrawal   decimal.Decimal
}

// GenerateAnnualProjection generates annual cash flow projections for a scenario
func (ce *CalculationEngine) GenerateAnnualProjection(personA, personB *domain.Employee, scenario *domain.Scenario, assumptions *domain.GlobalAssumptions, federalRules domain.FederalRules) []domain.AnnualCashFlow {
	projection := make([]domain.AnnualCashFlow, assumptions.ProjectionYears)
	// Buffer to store estimated MAGI per projection year so we can apply the
	// SSA 2-year lag for IRMAA determinations (use MAGI from Year-2).
	magiBuffer := make([]decimal.Decimal, assumptions.ProjectionYears)

	projectionStartYear := ProjectionBaseYear

	personADeathYearIndex, personBDeathYearIndex := deriveDeathYearIndexes(scenario, personA, personB, assumptions.ProjectionYears)

	personAStrategy := ce.createTSPStrategy(&scenario.PersonA, personA.TSPBalanceTraditional.Add(personA.TSPBalanceRoth), assumptions.InflationRate)
	personBStrategy := ce.createTSPStrategy(&scenario.PersonB, personB.TSPBalanceTraditional.Add(personB.TSPBalanceRoth), assumptions.InflationRate)

	personStates := []personProjectionState{
		newPersonProjectionState(personA, &scenario.PersonA, personAStrategy, personADeathYearIndex, scenarioMortalitySpec(scenario, true), "PersonA", ssProrationFractional),
		newPersonProjectionState(personB, &scenario.PersonB, personBStrategy, personBDeathYearIndex, scenarioMortalitySpec(scenario, false), "PersonB", ssProrationMonthlyAfterRetirement),
	}

	survivorSpendingFactor := decimal.NewFromFloat(1.0)
	if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() {
		survivorSpendingFactor = scenario.Mortality.Assumptions.SurvivorSpendingFactor
	}

	for year := 0; year < assumptions.ProjectionYears; year++ {
		projectionDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(year, 0, 0)
		yearEnd := time.Date(projectionDate.Year(), 12, 31, 23, 59, 59, 0, time.UTC)

		yearResults := make([]personAnnualResult, len(personStates))

		for idx := range personStates {
			state := &personStates[idx]
			if state.deathYearIndex != nil && year >= *state.deathYearIndex {
				state.deceased = true
			}

			result := &yearResults[idx]
			result.ageStart = state.employee.Age(projectionDate)
			result.ageEnd = state.employee.Age(yearEnd)
			result.isRetired = year >= state.retirementYear
			result.workFraction = calculateWorkFraction(year, state.retirementYear, state.scenario.RetirementDate, projectionDate, result.isRetired)
			result.salary = state.employee.CurrentSalary.Mul(result.workFraction)
		}

		if shouldMergeTSP(scenario) {
			applyTSPSpousalTransfer(&personStates[0], &personStates[1])
		}

		for idx := range personStates {
			state := &personStates[idx]
			result := &yearResults[idx]

			if result.isRetired && !state.deceased {
				result.pension = calculatePensionForYear(state, year, assumptions.COLAGeneralRate, result.workFraction)
				if ce.Debug && state.label == "PersonA" && year == state.retirementYear {
					ce.logPensionDebug(state, year, result.pension)
				}
				result.supplement = calculateSupplementForYear(state, year, assumptions.InflationRate, result.workFraction)
			}

			fixedIncome := calculateFixedRetirementIncome(state, year, assumptions.COLAGeneralRate, result.workFraction)
			result.fixedIncome = fixedIncome
			result.pension = result.pension.Add(fixedIncome)

			result.socialSecurity = calculateSocialSecurityForYear(state, year, projectionDate, yearEnd, assumptions.COLAGeneralRate, result)
			result.rmd = calculateRMDForYear(state, projectionDate, yearEnd, result)
		}

		applySurvivorPensions(year, assumptions, personStates, yearResults)
		applySocialSecuritySurvivorBenefits(year, assumptions.COLAGeneralRate, personStates, yearResults)

		targetIncome := yearResults[0].pension.
			Add(yearResults[1].pension).
			Add(yearResults[0].socialSecurity).
			Add(yearResults[1].socialSecurity).
			Add(yearResults[0].supplement).
			Add(yearResults[1].supplement)

		for idx := range personStates {
			state := &personStates[idx]
			result := &yearResults[idx]

			result.tspWithdrawal = ce.calculateTSPWithdrawalForPerson(state, year, projectionDate, targetIncome, result)
			ce.updateTSPBalancesForPerson(state, projectionDate, assumptions, result)
		}

		fehbPremium := CalculateFEHBPremium(personA, year, assumptions.FEHBPremiumInflation, federalRules.FEHBConfig)

		// Estimate MAGI for this projection year and store it in the buffer.
		totalPension := yearResults[0].pension.Add(yearResults[1].pension)
		totalTSP := yearResults[0].tspWithdrawal.Add(yearResults[1].tspWithdrawal)
		totalSS := yearResults[0].socialSecurity.Add(yearResults[1].socialSecurity)
		otherIncome := totalPension.Add(totalTSP)
		taxableSS := ce.TaxCalc.CalculateSocialSecurityTaxation(totalSS, otherIncome)
		estimatedMAGI := EstimateMAGI(totalPension, totalTSP, taxableSS, decimal.Zero)
		magiBuffer[year] = estimatedMAGI

		// Use MAGI from two years prior for IRMAA determinations per SSA rules.
		var magiForIRMAA decimal.Decimal
		if year-2 >= 0 {
			magiForIRMAA = magiBuffer[year-2]
		} else {
			// Fallback: use earliest available MAGI (year 0) for first two years
			magiForIRMAA = magiBuffer[0]
		}

		// Use year-end date for Medicare eligibility checks so that
		// persons who turn 65 during the calendar year are counted
		// for that year's Medicare premiums (pro-rated handling may
		// be added later if desired).
		personAPrem, personBPrem := ce.calculateMedicarePremium(personA, personB, yearEnd, magiForIRMAA)
		medicarePremium := personAPrem.Add(personBPrem)

		taxInput := TaxCalculationInput{
			PersonA:          personA,
			PersonB:          personB,
			Scenario:         scenario,
			Year:             year,
			IsRetired:        yearResults[0].isRetired && yearResults[1].isRetired,
			Pensions:         [2]decimal.Decimal{yearResults[0].pension, yearResults[1].pension},
			SurvivorPensions: [2]decimal.Decimal{yearResults[0].survivorPension, yearResults[1].survivorPension},
			TSPWithdrawals:   [2]decimal.Decimal{yearResults[0].tspWithdrawal, yearResults[1].tspWithdrawal},
			SocialSecurity:   [2]decimal.Decimal{yearResults[0].socialSecurity, yearResults[1].socialSecurity},
			WorkingIncome:    [2]decimal.Decimal{yearResults[0].salary, yearResults[1].salary},
		}
		taxResult := ce.calculateTaxes(taxInput)
		federalTax := taxResult.FederalTax
		stateTax := taxResult.StateTax
		localTax := taxResult.LocalTax
		ficaTax := taxResult.FICATax
		taxableTotal := taxResult.TaxableIncomeTotal
		stdDedUsed := taxResult.StandardDeduction
		filingStatusUsed := taxResult.FilingStatus
		seniors65 := taxResult.Seniors

		tspContributions := decimal.Zero
		if (!yearResults[0].isRetired || !yearResults[1].isRetired) && !(personStates[0].deceased || personStates[1].deceased) {
			personAContributions := personA.TotalAnnualTSPContribution().Mul(yearResults[0].workFraction)
			personBContributions := personB.TotalAnnualTSPContribution().Mul(yearResults[1].workFraction)
			tspContributions = personAContributions.Add(personBContributions)
		}

		cashFlow := domain.AnnualCashFlow{
			Year:                     year + 1,
			Date:                     projectionDate,
			AgePersonA:               yearResults[0].ageStart,
			AgePersonB:               yearResults[1].ageStart,
			SalaryPersonA:            yearResults[0].salary,
			SalaryPersonB:            yearResults[1].salary,
			PensionPersonA:           yearResults[0].pension,
			PensionPersonB:           yearResults[1].pension,
			TSPWithdrawalPersonA:     yearResults[0].tspWithdrawal,
			TSPWithdrawalPersonB:     yearResults[1].tspWithdrawal,
			SSBenefitPersonA:         yearResults[0].socialSecurity,
			SSBenefitPersonB:         yearResults[1].socialSecurity,
			FERSSupplementPersonA:    yearResults[0].supplement,
			FERSSupplementPersonB:    yearResults[1].supplement,
			WorkFractionPersonA:      yearResults[0].workFraction,
			WorkFractionPersonB:      yearResults[1].workFraction,
			FederalTax:               federalTax,
			FederalTaxableIncome:     taxableTotal,
			FederalStandardDeduction: stdDedUsed,
			FederalFilingStatus:      filingStatusUsed,
			FederalSeniors65Plus:     seniors65,
			StateTax:                 stateTax,
			LocalTax:                 localTax,
			FICATax:                  ficaTax,
			TSPContributions:         tspContributions,
			FEHBPremium:              fehbPremium,
			MedicarePremium:          medicarePremium,
			MedicarePremiumPersonA:   personAPrem,
			MedicarePremiumPersonB:   personBPrem,
			TSPBalancePersonA:        personStates[0].traditional.Add(personStates[0].roth),
			TSPBalancePersonB:        personStates[1].traditional.Add(personStates[1].roth),
			TSPBalanceTraditional:    personStates[0].traditional.Add(personStates[1].traditional),
			TSPBalanceRoth:           personStates[0].roth.Add(personStates[1].roth),
			IsRetired:                yearResults[0].isRetired && yearResults[1].isRetired,
			// Consider Medicare eligibility based on year-end so that
			// individuals who turn 65 during the calendar year are
			// reflected as Medicare-eligible for that year.
			IsMedicareEligible: dateutil.IsMedicareEligible(personA.BirthDate, yearEnd) || dateutil.IsMedicareEligible(personB.BirthDate, yearEnd),
			IsRMDYear:          dateutil.IsRMDYear(personA.BirthDate, projectionDate) || dateutil.IsRMDYear(personB.BirthDate, projectionDate),
			RMDAmount:          yearResults[0].rmd.Add(yearResults[1].rmd),
			PersonADeceased:    personStates[0].deceased,
			PersonBDeceased:    personStates[1].deceased,
			FilingStatusSingle: false,
		}

		if scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && (personStates[0].deceased != personStates[1].deceased) {
			switch scenario.Mortality.Assumptions.FilingStatusSwitch {
			case "immediate":
				cashFlow.FilingStatusSingle = true
			case "next_year":
				if personStates[0].deathYearIndex != nil && personStates[0].deceased && year > *personStates[0].deathYearIndex {
					cashFlow.FilingStatusSingle = true
				}
				if personStates[1].deathYearIndex != nil && personStates[1].deceased && year > *personStates[1].deathYearIndex {
					cashFlow.FilingStatusSingle = true
				}
			}
		}

		cashFlow.SurvivorPensionPersonA = yearResults[0].survivorPension
		cashFlow.SurvivorPensionPersonB = yearResults[1].survivorPension

		if (personStates[0].deceased || personStates[1].deceased) && survivorSpendingFactor.LessThan(decimal.NewFromFloat(0.999)) {
			cashFlow.TSPWithdrawalPersonA = cashFlow.TSPWithdrawalPersonA.Mul(survivorSpendingFactor)
			cashFlow.TSPWithdrawalPersonB = cashFlow.TSPWithdrawalPersonB.Mul(survivorSpendingFactor)
			cashFlow.PensionPersonA = cashFlow.PensionPersonA.Mul(survivorSpendingFactor)
			cashFlow.PensionPersonB = cashFlow.PensionPersonB.Mul(survivorSpendingFactor)
		}

		cashFlow.TotalGrossIncome = cashFlow.CalculateTotalIncome()
		cashFlow.CalculateNetIncome()

		projection[year] = cashFlow
	}

	return projection
}

func newPersonProjectionState(employee *domain.Employee, scenario *domain.RetirementScenario, strategy TSPWithdrawalStrategy, deathYearIndex *int, mortalitySpec *domain.MortalitySpec, label string, ssMode ssRetirementProration) personProjectionState {
	return personProjectionState{
		employee:          employee,
		scenario:          scenario,
		strategy:          strategy,
		traditional:       employee.TSPBalanceTraditional,
		roth:              employee.TSPBalanceRoth,
		retirementYear:    scenario.RetirementDate.Year() - ProjectionBaseYear,
		deathYearIndex:    deathYearIndex,
		mortalitySpec:     mortalitySpec,
		label:             label,
		ssProration:       ssMode,
		fixedIncomeConfig: resolveFixedRetirementIncome(employee, scenario),
	}
}

func resolveFixedRetirementIncome(employee *domain.Employee, scenario *domain.RetirementScenario) *domain.FixedRetirementIncome {
	if scenario != nil && scenario.FixedRetirementIncome != nil {
		return scenario.FixedRetirementIncome
	}
	return employee.FixedRetirementIncome
}

func scenarioMortalitySpec(scenario *domain.Scenario, isPersonA bool) *domain.MortalitySpec {
	if scenario == nil || scenario.Mortality == nil {
		return nil
	}
	if isPersonA {
		return scenario.Mortality.PersonA
	}
	return scenario.Mortality.PersonB
}

func calculateWorkFraction(year, retirementYear int, retirementDate, projectionDate time.Time, isRetired bool) decimal.Decimal {
	if year == retirementYear && retirementYear >= 0 {
		yearStart := time.Date(projectionDate.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		daysWorked := retirementDate.Sub(yearStart).Hours() / 24.0
		if daysWorked < 0 {
			daysWorked = 0
		}
		daysInYear := float64(dateutil.DaysInYear(projectionDate.Year()))
		if daysInYear == 0 {
			return decimal.Zero
		}
		frac := daysWorked / daysInYear
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		return decimal.NewFromFloat(frac)
	}
	if isRetired {
		return decimal.Zero
	}
	return decimal.NewFromInt(1)
}

func calculatePensionForYear(state *personProjectionState, year int, cola decimal.Decimal, workFraction decimal.Decimal) decimal.Decimal {
	if state.employee.EmploymentCategory() != domain.EmploymentTypeFederal {
		return decimal.Zero
	}
	pension := CalculatePensionForYear(state.employee, state.scenario.RetirementDate, year-state.retirementYear, cola)
	if year == state.retirementYear && state.retirementYear >= 0 {
		pension = pension.Mul(decimal.NewFromInt(1).Sub(workFraction))
	}
	return pension
}

func calculateFixedRetirementIncome(state *personProjectionState, year int, defaultCOLA decimal.Decimal, workFraction decimal.Decimal) decimal.Decimal {
	cfg := state.fixedIncomeConfig
	if cfg == nil {
		return decimal.Zero
	}

	if year < state.retirementYear {
		return decimal.Zero
	}

	yearsSinceRetirement := year - state.retirementYear
	if yearsSinceRetirement < 0 {
		yearsSinceRetirement = 0
	}

	amount := cfg.AnnualAmount
	colaRate := defaultCOLA
	if cfg.COLARate != nil {
		colaRate = *cfg.COLARate
	}

	if yearsSinceRetirement > 0 {
		amount = applyRecurringCOLA(amount, colaRate, yearsSinceRetirement)
	}

	if year == state.retirementYear && state.retirementYear >= 0 {
		amount = amount.Mul(decimal.NewFromInt(1).Sub(workFraction))
	}

	return amount
}

func applyRecurringCOLA(amount decimal.Decimal, rate decimal.Decimal, periods int) decimal.Decimal {
	if periods <= 0 {
		return amount
	}
	multiplier := decimal.NewFromInt(1).Add(rate)
	result := amount
	for i := 0; i < periods; i++ {
		result = result.Mul(multiplier)
	}
	return result
}

func (ce *CalculationEngine) logPensionDebug(state *personProjectionState, year int, pension decimal.Decimal) {
	ce.Logger.Debugf("DEBUG: %s pension calculation for year %d", state.label, ProjectionBaseYear+year)
	ce.Logger.Debugf("  Retirement date: %s", state.scenario.RetirementDate.Format("2006-01-02"))
	ce.Logger.Debugf("  Age at retirement: %d", state.employee.Age(state.scenario.RetirementDate))
	ce.Logger.Debugf("  Years of service: %s", state.employee.YearsOfService(state.scenario.RetirementDate).StringFixed(2))
	ce.Logger.Debugf("  High-3 salary: %s", state.employee.High3Salary.StringFixed(2))

	pensionCalc := CalculateFERSPension(state.employee, state.scenario.RetirementDate)
	ce.Logger.Debugf("  Multiplier: %s", pensionCalc.Multiplier.StringFixed(4))
	ce.Logger.Debugf("  ANNUAL pension (before reduction): $%s", pensionCalc.AnnualPension.StringFixed(2))
	ce.Logger.Debugf("  Survivor election: %s", pensionCalc.SurvivorElection.StringFixed(4))
	ce.Logger.Debugf("  ANNUAL pension (final): $%s", pensionCalc.ReducedPension.StringFixed(2))
	ce.Logger.Debugf("  MONTHLY pension amount: $%s", pensionCalc.ReducedPension.Div(decimal.NewFromInt(12)).StringFixed(2))
	ce.Logger.Debugf("  Current-year cash received (partial): $%s", pension.StringFixed(2))
}

func calculateSupplementForYear(state *personProjectionState, year int, inflation decimal.Decimal, workFraction decimal.Decimal) decimal.Decimal {
	if state.employee.EmploymentCategory() != domain.EmploymentTypeFederal {
		return decimal.Zero
	}
	supplement := CalculateFERSSupplementYear(state.employee, state.scenario.RetirementDate, year-state.retirementYear, inflation)
	if year == state.retirementYear && state.retirementYear >= 0 {
		supplement = supplement.Mul(decimal.NewFromInt(1).Sub(workFraction))
	}
	return supplement
}

func calculateSocialSecurityForYear(state *personProjectionState, year int, projectionDate, yearEnd time.Time, cola decimal.Decimal, result *personAnnualResult) decimal.Decimal {
	if state.deceased || !result.isRetired {
		return decimal.Zero
	}

	ss := CalculateSSBenefitForYear(state.employee, state.scenario.SSStartAge, year, cola)

	if result.ageStart < state.scenario.SSStartAge && result.ageEnd >= state.scenario.SSStartAge {
		birthdayThisYear := time.Date(projectionDate.Year(), state.employee.BirthDate.Month(), state.employee.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
		if !(year == state.retirementYear && state.scenario.RetirementDate.Before(birthdayThisYear)) {
			daysAfter := yearEnd.Sub(birthdayThisYear).Hours() / 24.0
			daysInYear := float64(dateutil.DaysInYear(projectionDate.Year()))
			if daysInYear > 0 {
				frac := daysAfter / daysInYear
				if frac < 0 {
					frac = 0
				}
				if frac > 1 {
					frac = 1
				}
				ss = ss.Mul(decimal.NewFromFloat(frac))
			}
		}
	}

	if year == state.retirementYear && state.retirementYear >= 0 {
		ageAtRetirement := state.employee.Age(state.scenario.RetirementDate)
		if ageAtRetirement >= state.scenario.SSStartAge {
			switch state.ssProration {
			case ssProrationMonthlyAfterRetirement:
				retirementDate := state.scenario.RetirementDate
				ssStartDate := time.Date(retirementDate.Year(), retirementDate.Month()+1, 1, 0, 0, 0, 0, time.UTC)
				if ssStartDate.Year() > projectionDate.Year() {
					return decimal.Zero
				}
				monthsOfBenefits := 12 - int(ssStartDate.Month()) + 1
				if monthsOfBenefits < 0 {
					monthsOfBenefits = 0
				}
				if monthsOfBenefits > 12 {
					monthsOfBenefits = 12
				}
				birthdayThisYear := time.Date(projectionDate.Year(), state.employee.BirthDate.Month(), state.employee.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
				if retirementDate.Before(birthdayThisYear) {
					monthly := ss.Div(decimal.NewFromInt(12))
					ss = monthly.Mul(decimal.NewFromInt(int64(monthsOfBenefits)))
				} else if monthsOfBenefits <= 0 {
					ss = decimal.Zero
				}
			default:
				birthdayThisYear := time.Date(projectionDate.Year(), state.employee.BirthDate.Month(), state.employee.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
				if state.scenario.RetirementDate.Before(birthdayThisYear) {
					ss = ss.Mul(decimal.NewFromInt(1).Sub(result.workFraction))
				}
			}
		} else {
			return decimal.Zero
		}
	}

	return ss
}

func calculateRMDForYear(state *personProjectionState, projectionDate, yearEnd time.Time, result *personAnnualResult) decimal.Decimal {
	rmdAge := dateutil.GetRMDAge(state.employee.BirthDate.Year())
	if result.ageStart < rmdAge && result.ageEnd >= rmdAge {
		birthdayThisYear := time.Date(projectionDate.Year(), state.employee.BirthDate.Month(), state.employee.BirthDate.Day(), 0, 0, 0, 0, time.UTC)
		daysAfter := yearEnd.Sub(birthdayThisYear).Hours() / 24.0
		daysInYear := float64(dateutil.DaysInYear(projectionDate.Year()))
		if daysInYear == 0 {
			return decimal.Zero
		}
		frac := daysAfter / daysInYear
		if frac < 0 {
			frac = 0
		}
		if frac > 1 {
			frac = 1
		}
		fullRMD := CalculateRMD(state.traditional, state.employee.BirthDate.Year(), rmdAge)
		return fullRMD.Mul(decimal.NewFromFloat(frac))
	}
	if result.ageStart >= rmdAge {
		return CalculateRMD(state.traditional, state.employee.BirthDate.Year(), result.ageStart)
	}
	return decimal.Zero
}

func applySurvivorPensions(year int, assumptions *domain.GlobalAssumptions, states []personProjectionState, results []personAnnualResult) {
	for idx := range states {
		decedent := &states[idx]
		if decedent.deathYearIndex == nil || year < *decedent.deathYearIndex {
			continue
		}
		if decedent.employee.EmploymentCategory() != domain.EmploymentTypeFederal {
			continue
		}
		survivorIdx := 1 - idx
		survivor := &states[survivorIdx]
		if survivor.deceased || !results[idx].isRetired {
			continue
		}

		baseCalc := CalculateFERSPension(decedent.employee, decedent.scenario.RetirementDate)
		yearsSinceRet := year - decedent.retirementYear
		if yearsSinceRet < 0 {
			yearsSinceRet = 0
		}

		currentSurvivor := baseCalc.SurvivorAnnuity
		for cy := 1; cy <= yearsSinceRet; cy++ {
			projDate := decedent.scenario.RetirementDate.AddDate(cy, 0, 0)
			ageAt := decedent.employee.Age(projDate)
			currentSurvivor = ApplyFERSPensionCOLA(currentSurvivor, assumptions.COLAGeneralRate, ageAt)
		}

		var survivorAmount decimal.Decimal
		if year == *decedent.deathYearIndex {
			var deathDate *time.Time
			if decedent.mortalitySpec != nil {
				deathDate = decedent.mortalitySpec.DeathDate
			}
			frac, occurred := deathFractionInYear(decedent.deathYearIndex, year, deathDate)
			if occurred {
				survivorAmount = currentSurvivor.Mul(decimal.NewFromInt(1).Sub(frac))
			} else {
				survivorAmount = currentSurvivor
			}
		} else {
			survivorAmount = currentSurvivor
		}

		results[survivorIdx].survivorPension = survivorAmount
	}
}

func applySocialSecuritySurvivorBenefits(year int, cola decimal.Decimal, states []personProjectionState, results []personAnnualResult) {
	if states[0].deceased && !states[1].deceased {
		fra := dateutil.FullRetirementAge(states[1].employee.BirthDate)
		deceasedBenefit := CalculateSSBenefitForYear(states[0].employee, states[0].scenario.SSStartAge, year, cola)
		candidate := CalculateSurvivorSSBenefit(deceasedBenefit, results[1].ageStart, fra)
		if candidate.GreaterThan(results[1].socialSecurity) {
			results[1].socialSecurity = candidate
		}
	}
	if states[1].deceased && !states[0].deceased {
		fra := dateutil.FullRetirementAge(states[0].employee.BirthDate)
		deceasedBenefit := CalculateSSBenefitForYear(states[1].employee, states[1].scenario.SSStartAge, year, cola)
		candidate := CalculateSurvivorSSBenefit(deceasedBenefit, results[0].ageStart, fra)
		if candidate.GreaterThan(results[0].socialSecurity) {
			results[0].socialSecurity = candidate
		}
	}
}

func (ce *CalculationEngine) calculateTSPWithdrawalForPerson(state *personProjectionState, year int, projectionDate time.Time, targetIncome decimal.Decimal, result *personAnnualResult) decimal.Decimal {
	if !result.isRetired || state.deceased {
		return decimal.Zero
	}

	yearsIntoRetirement := year - state.retirementYear + 1
	currentBalance := state.traditional.Add(state.roth)

	switch state.scenario.TSPWithdrawalStrategy {
	case "4_percent_rule":
		withdrawal := state.strategy.CalculateWithdrawal(
			currentBalance,
			yearsIntoRetirement,
			decimal.Zero,
			result.ageStart,
			dateutil.IsRMDYear(state.employee.BirthDate, projectionDate),
			CalculateRMD(state.traditional, state.employee.BirthDate.Year(), result.ageStart),
		)
		if year == state.retirementYear && state.retirementYear >= 0 {
			withdrawal = withdrawal.Mul(decimal.NewFromInt(1).Sub(result.workFraction))
		}
		return withdrawal
	default:
		isRMDYear := dateutil.IsRMDYear(state.employee.BirthDate, projectionDate) || result.rmd.GreaterThan(decimal.Zero)
		withdrawal := state.strategy.CalculateWithdrawal(
			currentBalance,
			yearsIntoRetirement,
			targetIncome,
			result.ageStart,
			isRMDYear,
			result.rmd,
		)
		if year == state.retirementYear && state.retirementYear >= 0 {
			withdrawal = withdrawal.Mul(decimal.NewFromInt(1).Sub(result.workFraction))
		}
		return withdrawal
	}
}

func (ce *CalculationEngine) updateTSPBalancesForPerson(state *personProjectionState, projectionDate time.Time, assumptions *domain.GlobalAssumptions, result *personAnnualResult) {
	if result.isRetired {
		if state.employee.TSPLifecycleFund != nil || state.employee.TSPAllocation != nil {
			if result.tspWithdrawal.GreaterThan(state.traditional) {
				remaining := result.tspWithdrawal.Sub(state.traditional)
				state.traditional = decimal.Zero
				if remaining.GreaterThan(state.roth) {
					state.roth = decimal.Zero
				} else {
					state.roth = state.roth.Sub(remaining)
				}
			} else {
				state.traditional = state.traditional.Sub(result.tspWithdrawal)
			}

			allocation := ce.getTSPAllocationForEmployee(state.employee, projectionDate)
			weightedReturn := ce.calculateTSPReturnWithAllocation(allocation, projectionDate.Year())
			growthFactor := decimal.NewFromInt(1).Add(weightedReturn)
			state.traditional = state.traditional.Mul(growthFactor)
			state.roth = state.roth.Mul(growthFactor)
		} else {
			state.traditional, state.roth = ce.updateTSPBalances(state.traditional, state.roth, result.tspWithdrawal, assumptions.TSPReturnPostRetirement)
		}
		return
	}

	contributions := state.employee.TotalAnnualTSPContribution()
	if state.employee.TSPLifecycleFund != nil || state.employee.TSPAllocation != nil {
		state.traditional = ce.growTSPBalanceWithAllocation(state.employee, state.traditional, contributions, projectionDate)
		state.roth = ce.growTSPBalanceWithAllocation(state.employee, state.roth, decimal.Zero, projectionDate)
	} else {
		state.traditional = ce.growTSPBalance(state.traditional, contributions, assumptions.TSPReturnPreRetirement)
		state.roth = ce.growTSPBalance(state.roth, decimal.Zero, assumptions.TSPReturnPreRetirement)
	}
}

func shouldMergeTSP(scenario *domain.Scenario) bool {
	return scenario != nil && scenario.Mortality != nil && scenario.Mortality.Assumptions != nil && scenario.Mortality.Assumptions.TSPSpousalTransfer == "merge"
}

func applyTSPSpousalTransfer(personA, personB *personProjectionState) {
	if personA.deceased && !personB.deceased {
		if !personA.traditional.IsZero() || !personA.roth.IsZero() {
			personB.traditional = personB.traditional.Add(personA.traditional)
			personB.roth = personB.roth.Add(personA.roth)
			personA.traditional = decimal.Zero
			personA.roth = decimal.Zero
		}
	}
	if personB.deceased && !personA.deceased {
		if !personB.traditional.IsZero() || !personB.roth.IsZero() {
			personA.traditional = personA.traditional.Add(personB.traditional)
			personA.roth = personA.roth.Add(personB.roth)
			personB.traditional = decimal.Zero
			personB.roth = decimal.Zero
		}
	}
}

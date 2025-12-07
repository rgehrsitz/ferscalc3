package calculation

import (
	"fmt"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculateBreakEvenTSPWithdrawalRate calculates the TSP withdrawal percentage needed to match current net income
func (ce *CalculationEngine) CalculateBreakEvenTSPWithdrawalRate(config *domain.Configuration, scenario *domain.Scenario, targetNetIncome decimal.Decimal) (decimal.Decimal, *domain.AnnualCashFlow, error) {
	personAEmployee := config.PersonalDetails["person_a"]
	personBEmployee := config.PersonalDetails["person_b"]

	// Find the first year when both are fully retired
	projectionStartYear := ProjectionBaseYear
	personARetirementYear := scenario.PersonA.RetirementDate.Year() - projectionStartYear
	personBRetirementYear := scenario.PersonB.RetirementDate.Year() - projectionStartYear
	firstFullRetirementYear := personARetirementYear
	if personBRetirementYear > personARetirementYear {
		firstFullRetirementYear = personBRetirementYear
	}
	// Add 1 to get the first FULL year after both are retired
	firstFullRetirementYear++

	// Binary search for the correct TSP withdrawal rate
	minRate := decimal.NewFromFloat(0.001)  // 0.1%
	maxRate := decimal.NewFromFloat(0.15)   // 15%
	tolerance := decimal.NewFromFloat(1000) // Within $1,000
	maxIterations := 50

	for i := 0; i < maxIterations; i++ {
		// Calculate midpoint withdrawal rate
		testRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))

		// Create a test scenario with this withdrawal rate
		testScenario := *scenario
		testScenario.PersonA.TSPWithdrawalStrategy = "variable_percentage"
		testScenario.PersonA.TSPWithdrawalRate = &testRate
		testScenario.PersonB.TSPWithdrawalStrategy = "variable_percentage"
		testScenario.PersonB.TSPWithdrawalRate = &testRate

		// Run projection to get the first full retirement year
		projection := ce.GenerateAnnualProjection(&personAEmployee, &personBEmployee, &testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)

		// Check if we have enough projection years
		if firstFullRetirementYear >= len(projection) {
			return decimal.Zero, nil, fmt.Errorf("first full retirement year (%d) exceeds projection length (%d)", firstFullRetirementYear, len(projection))
		}

		testYear := projection[firstFullRetirementYear]
		netIncomeDiff := testYear.NetIncome.Sub(targetNetIncome)

		// Check if we're within tolerance
		if netIncomeDiff.Abs().LessThan(tolerance) {
			return testRate, &testYear, nil
		}

		// Adjust search range
		if netIncomeDiff.LessThan(decimal.Zero) {
			// Net income is too low, need higher withdrawal rate
			minRate = testRate
		} else {
			// Net income is too high, need lower withdrawal rate
			maxRate = testRate
		}

		// Check if search range is too narrow
		if maxRate.Sub(minRate).LessThan(decimal.NewFromFloat(0.0001)) {
			break
		}
	}

	// Return the best rate found
	finalRate := minRate.Add(maxRate).Div(decimal.NewFromInt(2))
	testScenario := *scenario
	testScenario.PersonA.TSPWithdrawalStrategy = "variable_percentage"
	testScenario.PersonA.TSPWithdrawalRate = &finalRate
	testScenario.PersonB.TSPWithdrawalStrategy = "variable_percentage"
	testScenario.PersonB.TSPWithdrawalRate = &finalRate

	projection := ce.GenerateAnnualProjection(&personAEmployee, &personBEmployee, &testScenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)
	finalYear := projection[firstFullRetirementYear]

	return finalRate, &finalYear, nil
}

// CalculateBreakEvenAnalysis calculates break-even TSP withdrawal rates for all scenarios
func (ce *CalculationEngine) CalculateBreakEvenAnalysis(config *domain.Configuration) (*BreakEvenAnalysis, error) {
	// Calculate current net income as the target
	personAEmployee := config.PersonalDetails["person_a"]
	personBEmployee := config.PersonalDetails["person_b"]
	targetNetIncome := ce.NetIncomeCalc.Calculate(&personAEmployee, &personBEmployee, ce.Debug)

	results := make([]BreakEvenResult, len(config.Scenarios))

	for i, scenario := range config.Scenarios {
		rate, yearData, err := ce.CalculateBreakEvenTSPWithdrawalRate(config, &scenario, targetNetIncome)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate break-even rate for scenario %s: %v", scenario.Name, err)
		}

		results[i] = BreakEvenResult{
			ScenarioName:            scenario.Name,
			BreakEvenWithdrawalRate: rate,
			ProjectedNetIncome:      yearData.NetIncome,
			ProjectedYear:           yearData.Year + (ProjectionBaseYear - 1),
			TSPWithdrawalAmount:     yearData.TSPWithdrawalPersonA.Add(yearData.TSPWithdrawalPersonB),
			TotalTSPBalance:         yearData.TotalTSPBalance(),
			CurrentVsBreakEvenDiff:  yearData.NetIncome.Sub(targetNetIncome),
		}
	}

	return &BreakEvenAnalysis{
		TargetNetIncome: targetNetIncome,
		Results:         results,
	}, nil
}

// BreakEvenAnalysis contains the results of break-even TSP withdrawal rate analysis
type BreakEvenAnalysis struct {
	TargetNetIncome decimal.Decimal   `json:"target_net_income"`
	Results         []BreakEvenResult `json:"results"`
}

// BreakEvenResult contains break-even calculation results for a single scenario
type BreakEvenResult struct {
	ScenarioName            string          `json:"scenario_name"`
	BreakEvenWithdrawalRate decimal.Decimal `json:"break_even_withdrawal_rate"`
	ProjectedNetIncome      decimal.Decimal `json:"projected_net_income"`
	ProjectedYear           int             `json:"projected_year"`
	TSPWithdrawalAmount     decimal.Decimal `json:"tsp_withdrawal_amount"`
	TotalTSPBalance         decimal.Decimal `json:"total_tsp_balance"`
	CurrentVsBreakEvenDiff  decimal.Decimal `json:"current_vs_break_even_diff"`
}

// CumulativeBreakEvenResult describes the crossover point where cumulative net income of two projections is equal
type CumulativeBreakEvenResult struct {
	// Index of the later year in the projection where the crossover occurs (1-based Year field)
	YearIndex int `json:"year_index"`

	// Calendar year (fractional) when the crossover occurs (e.g., 2030.75)
	CalendarYear float64 `json:"calendar_year"`

	// Fraction (0..1) of the year between YearIndex-1 and YearIndex where crossover happens
	Fraction decimal.Decimal `json:"fraction_of_year"`

	// Cumulative net income at the crossover (equal for both scenarios)
	CumulativeAmount decimal.Decimal `json:"cumulative_amount"`

	// Bookkeeping: previous and next integer calendar years
	PrevYear int `json:"prev_year"`
	NextYear int `json:"next_year"`

	// Explicit month and year for convenience (month: 1..12)
	BreakEvenMonth int `json:"break_even_month"`
	BreakEvenYear  int `json:"break_even_year"`
}

// CalculateCumulativeBreakEven finds the first crossover (if any) between cumulative net income
// of projection A and projection B. It returns a fractional calendar year and the cumulative amount
// at which they are equal. Projections must be aligned by index (same start year). If no crossover
// is found, returns nil, nil.
func CalculateCumulativeBreakEven(projA, projB []domain.AnnualCashFlow) (*CumulativeBreakEvenResult, error) {
	if len(projA) == 0 || len(projB) == 0 {
		return nil, fmt.Errorf("one or both projections are empty")
	}

	// Align to the minimum length
	n := len(projA)
	if len(projB) < n {
		n = len(projB)
	}

	// Start accumulation at the projection start (index 0).
	// Previously we tried to begin at the first year both were retired to avoid
	// spurious crossovers from identical pre-retirement rows. That approach
	// hid legitimate cumulative differences when one scenario retires earlier.
	// Instead, start at 0 and only ignore an exact equality at the very first
	// projection index as trivial.
	start := 0

	cumA := decimal.Zero
	cumB := decimal.Zero

	// Iterate years starting from 'start' and find first sign change in (cumA - cumB)
	var prevDiff decimal.Decimal
	for i := start; i < n; i++ {
		// For year i (0-based), compute diff after adding this year's net income
		yearNetA := projA[i].NetIncome
		yearNetB := projB[i].NetIncome

		prevDiff = cumA.Sub(cumB)

		cumA = cumA.Add(yearNetA)
		cumB = cumB.Add(yearNetB)

		currDiff := cumA.Sub(cumB)

		// Check exact equality within a small tolerance (1 cent)
		if currDiff.Abs().LessThan(decimal.NewFromFloat(0.01)) {
			// If this occurs at the very first projection index (no prior accumulation),
			// ignore as trivial and continue searching.
			if i == 0 {
				// continue searching
			} else {
				// Crossover occurs exactly at year end i
				calendarYear := float64(projA[i].Date.Year())
				// exact year-end -> month = December
				month := 12
				year := projA[i].Date.Year()
				return &CumulativeBreakEvenResult{
					YearIndex:        projA[i].Year,
					CalendarYear:     calendarYear,
					Fraction:         decimal.NewFromInt(1),
					CumulativeAmount: cumA,
					PrevYear:         projA[i].Date.Year() - 1,
					NextYear:         projA[i].Date.Year(),
					BreakEvenMonth:   month,
					BreakEvenYear:    year,
				}, nil
			}
		}

		// If sign changed between prevDiff and currDiff, we crossed inside this year
		if i > 0 && prevDiff.Mul(currDiff).LessThan(decimal.Zero) {
			// Linear interpolation within the year using the difference sequence
			// diff(t) = prevDiff + t*(currDiff - prevDiff), solve for t: t = -prevDiff/(currDiff - prevDiff)
			denom := currDiff.Sub(prevDiff)
			if denom.IsZero() {
				// Fallback: assign mid-year
				t := decimal.NewFromFloat(0.5)
				calendarYearPrev := projA[i-1].Date.Year()
				cumAprev := cumA.Sub(yearNetA)
				cumAt := cumAprev.Add(yearNetA.Mul(t))
				// compute month from fraction (1..12)
				month := int((t.InexactFloat64()) * 12)
				if month < 1 {
					month = 1
				}
				if month > 12 {
					month = 12
				}
				return &CumulativeBreakEvenResult{
					YearIndex:        projA[i].Year,
					CalendarYear:     float64(calendarYearPrev) + t.InexactFloat64(),
					Fraction:         t,
					CumulativeAmount: cumAt,
					PrevYear:         calendarYearPrev,
					NextYear:         projA[i].Date.Year(),
					BreakEvenMonth:   month,
					BreakEvenYear:    calendarYearPrev,
				}, nil
			}

			t := prevDiff.Neg().Div(denom)
			// Clamp t to [0,1]
			if t.LessThan(decimal.Zero) {
				t = decimal.Zero
			} else if t.GreaterThan(decimal.NewFromInt(1)) {
				t = decimal.NewFromInt(1)
			}

			calendarYearPrev := projA[i-1].Date.Year()
			cumAprev := cumA.Sub(yearNetA)
			cumAt := cumAprev.Add(yearNetA.Mul(t))

			// compute month from fraction (1..12)
			month := int((t.InexactFloat64()) * 12)
			if month < 1 {
				month = 1
			}
			if month > 12 {
				month = 12
			}

			return &CumulativeBreakEvenResult{
				YearIndex:        projA[i].Year,
				CalendarYear:     float64(calendarYearPrev) + t.InexactFloat64(),
				Fraction:         t,
				CumulativeAmount: cumAt,
				PrevYear:         calendarYearPrev,
				NextYear:         projA[i].Date.Year(),
				BreakEvenMonth:   month,
				BreakEvenYear:    calendarYearPrev,
			}, nil
		}
	}

	// No crossover found
	return nil, nil
}

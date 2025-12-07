package output

import (
	"sort"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// Recommendation encapsulates the selection result of the best scenario.
type Recommendation struct {
	ScenarioName       string
	FirstRetirementNet decimal.Decimal
	NetIncomeChange    decimal.Decimal
	PercentageChange   decimal.Decimal
}

// AnalyzeScenarios determines the best retirement scenario based on long-term net income.
// Compares scenarios at year 2030 (or earliest year when both spouses are fully retired in all scenarios).
// This provides an apples-to-apples comparison avoiding distortions from partial retirement years.
func AnalyzeScenarios(results *domain.ScenarioComparison) Recommendation {
	baseline := results.BaselineNetIncome
	type ranked struct {
		name   string
		income decimal.Decimal
	}
	var ranks []ranked

	// Use 2030 net income for comparison (typically 5+ years into retirement for most scenarios)
	// This avoids comparing partial retirement years and provides meaningful long-term comparison
	for i := range results.Scenarios {
		sc := &results.Scenarios[i]
		// Prefer the 2030 calendar year comparison if available
		income := sc.NetIncome2030
		if income.IsZero() {
			// Fallback: use first year where both are fully retired
			for _, y := range sc.Projection {
				if y.IsRetired && !y.PersonADeceased && !y.PersonBDeceased {
					income = y.NetIncome
					break
				}
			}
		}

		// Compute reference indices for the template comparisons:
		// - Last year both persons have salary > 0
		// - First year where either person has salary == 0 (first any retired full year)
		// - First year where both salaries == 0 (first both retired full year)
		// Determine per-person retirement years (first index where work fraction < 1)
		personARet := -1
		personBRet := -1
		firstBoth := -1
		one := decimal.NewFromInt(1)
		zero := decimal.Zero
		for idx, y := range sc.Projection {
			if personARet == -1 && y.WorkFractionPersonA.LessThan(one) {
				personARet = idx
			}
			if personBRet == -1 && y.WorkFractionPersonB.LessThan(one) {
				personBRet = idx
			}
			if firstBoth == -1 && y.WorkFractionPersonA.Equal(zero) && y.WorkFractionPersonB.Equal(zero) {
				firstBoth = idx
			}
		}

		// firstAny is the earlier of the two retirement indices
		firstAny := -1
		if personARet != -1 && personBRet != -1 {
			if personARet < personBRet {
				firstAny = personARet
			} else {
				firstAny = personBRet
			}
		} else if personARet != -1 {
			firstAny = personARet
		} else if personBRet != -1 {
			firstAny = personBRet
		}

		// lastBothEmployed is the year before the earliest retirement year (fully employed last year)
		lastBoth := -1
		if firstAny > 0 {
			lastBoth = firstAny - 1
		} else if firstAny == -1 {
			// no retirement detected in projection — default to last index
			lastBoth = len(sc.Projection) - 1
		} else {
			// retirement happens in projection year 0 — indicate prior year by using -1
			lastBoth = -1
		}
		// Leave lastBoth == -1 to indicate the prior-year (before projection start)
		if firstAny == -1 {
			firstAny = len(sc.Projection) - 1
			if firstAny < 0 {
				firstAny = 0
			}
		}
		if firstBoth == -1 {
			firstBoth = len(sc.Projection) - 1
			if firstBoth < 0 {
				firstBoth = 0
			}
		}
		sc.LastBothEmployedIndex = lastBoth
		sc.FirstAnyRetiredIndex = firstAny
		sc.FirstBothRetiredIndex = firstBoth

		// Build snapshots (cash flows) for the comparison table. If an index
		// falls outside projection (e.g., lastBoth < 0) reconstruct a prior-year
		// snapshot by scaling the first projection year to full-year amounts.
		makeSnapshot := func(idx int) domain.AnnualCashFlow {
			// If index valid, return a copy of the projection year
			if idx >= 0 && idx < len(sc.Projection) {
				return sc.Projection[idx]
			}
			// Reconstruct prior year from first projection row if available
			if len(sc.Projection) == 0 {
				return domain.AnnualCashFlow{}
			}
			base := sc.Projection[0]
			snap := base
			// Determine reconstructed calendar year (previous year)
			snap.Date = snap.Date.AddDate(-1, 0, 0)
			// Reconstruct full-year salaries by dividing by work fraction when possible
			if base.WorkFractionPersonA.GreaterThan(decimal.Zero) {
				snap.SalaryPersonA = base.SalaryPersonA.Div(base.WorkFractionPersonA)
			}
			if base.WorkFractionPersonB.GreaterThan(decimal.Zero) {
				snap.SalaryPersonB = base.SalaryPersonB.Div(base.WorkFractionPersonB)
			}
			// Reconstruct TSP contributions proportionally
			if base.WorkFractionPersonA.GreaterThan(decimal.Zero) {
				// approximate per-person contributions by scaling total contribution
				// proportionally to salary share
				snap.TSPContributions = base.TSPContributions.Mul(decimal.NewFromInt(1))
			}
			// Scale taxes and deductions proportionally to reconstructed gross income
			reconGross := snap.SalaryPersonA.Add(snap.SalaryPersonB).Add(snap.PensionPersonA).Add(snap.PensionPersonB).Add(snap.TSPWithdrawalPersonA).Add(snap.TSPWithdrawalPersonB).Add(snap.SSBenefitPersonA).Add(snap.SSBenefitPersonB).Add(snap.FERSSupplementPersonA).Add(snap.FERSSupplementPersonB)
			baseGross := base.TotalGrossIncome
			if baseGross.GreaterThan(decimal.Zero) {
				scale := reconGross.Div(baseGross)
				snap.FederalTax = base.FederalTax.Mul(scale)
				snap.StateTax = base.StateTax.Mul(scale)
				snap.LocalTax = base.LocalTax.Mul(scale)
				snap.FICATax = base.FICATax.Mul(scale)
				snap.FEHBPremium = base.FEHBPremium.Mul(scale)
				snap.MedicarePremium = base.MedicarePremium.Mul(scale)
				snap.MedicarePremiumPersonA = base.MedicarePremiumPersonA.Mul(scale)
				snap.MedicarePremiumPersonB = base.MedicarePremiumPersonB.Mul(scale)
				snap.TSPContributions = base.TSPContributions.Mul(scale)
			}
			snap.TotalGrossIncome = reconGross
			snap.NetIncome = snap.TotalGrossIncome.Sub(snap.FederalTax.Add(snap.StateTax).Add(snap.LocalTax).Add(snap.FICATax).Add(snap.TSPContributions).Add(snap.FEHBPremium).Add(snap.MedicarePremium))
			return snap
		}

		sc.LastBothCashFlow = makeSnapshot(lastBoth)
		sc.FirstAnyCashFlow = makeSnapshot(firstAny)
		sc.FirstBothCashFlow = makeSnapshot(firstBoth)

		ranks = append(ranks, ranked{sc.Name, income})
	}

	if len(ranks) == 0 {
		return Recommendation{}
	}

	sort.Slice(ranks, func(i, j int) bool { return ranks[i].income.GreaterThan(ranks[j].income) })
	best := ranks[0]

	// Calculate change vs baseline (current working income)
	delta := best.income.Sub(baseline)
	pct := decimal.Zero
	if !baseline.IsZero() {
		pct = delta.Div(baseline).Mul(decimal.NewFromInt(100))
	}

	return Recommendation{
		ScenarioName:       best.name,
		FirstRetirementNet: best.income,
		NetIncomeChange:    delta,
		PercentageChange:   pct,
	}
}

package calculation

import (
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// createTSPStrategy creates a TSP withdrawal strategy based on scenario configuration
func (ce *CalculationEngine) createTSPStrategy(scenario *domain.RetirementScenario, initialBalance decimal.Decimal, inflationRate decimal.Decimal) TSPWithdrawalStrategy {
	switch scenario.TSPWithdrawalStrategy {
	case "4_percent_rule":
		return NewFourPercentRule(initialBalance, inflationRate)
	case "need_based":
		if scenario.TSPWithdrawalTargetMonthly != nil {
			return NewNeedBasedWithdrawal(*scenario.TSPWithdrawalTargetMonthly)
		}
		// Fallback to 4% rule if target not specified
		return NewFourPercentRule(initialBalance, inflationRate)
	case "variable_percentage":
		if scenario.TSPWithdrawalRate != nil {
			return NewVariablePercentageWithdrawal(*scenario.TSPWithdrawalRate)
		}
		// Fallback to 4% rule if rate not specified
		return NewFourPercentRule(initialBalance, inflationRate)
	case "floor_ceiling":
		rate := decimal.NewFromFloat(0.04) // Default 4% initial
		if scenario.TSPWithdrawalRate != nil {
			rate = *scenario.TSPWithdrawalRate
		}

		ceiling := decimal.NewFromFloat(0.20) // Default 20% ceiling
		if scenario.TSPWithdrawalCeiling != nil {
			ceiling = *scenario.TSPWithdrawalCeiling
		}

		floor := decimal.NewFromFloat(0.15) // Default 15% floor
		if scenario.TSPWithdrawalFloor != nil {
			floor = *scenario.TSPWithdrawalFloor
		}

		return NewFloorCeilingWithdrawal(rate, ceiling, floor, inflationRate)
	case "fixed_annuity":
		// Calculate annuity premium (portion of TSP to convert)
		premiumPercent := decimal.NewFromInt(1) // Default: 100% of TSP
		if scenario.AnnuityPremiumPercent != nil {
			premiumPercent = *scenario.AnnuityPremiumPercent
		}
		premium := initialBalance.Mul(premiumPercent)

		// Get annuity payout rate (default: 5.5% annual payout)
		payoutRate := decimal.NewFromFloat(0.055)
		if scenario.AnnuityPayoutRate != nil {
			payoutRate = *scenario.AnnuityPayoutRate
		}

		// Get COLA rate (default: 0 for fixed payment)
		colaRate := decimal.Zero
		if scenario.AnnuityCOLARate != nil {
			colaRate = *scenario.AnnuityCOLARate
		}

		// Get survivor benefit percentage (default: 100%)
		survivorPercent := decimal.NewFromInt(1)
		if scenario.AnnuitySurvivorPercent != nil {
			survivorPercent = *scenario.AnnuitySurvivorPercent
		}

		// Get guaranteed years (default: 10 years certain)
		guaranteedYears := 10
		if scenario.AnnuityGuaranteedYears != nil {
			guaranteedYears = *scenario.AnnuityGuaranteedYears
		}

		return NewFixedAnnuity(premium, payoutRate, colaRate, survivorPercent, guaranteedYears)
	default:
		// Default to 4% rule
		return NewFourPercentRule(initialBalance, inflationRate)
	}
}

// updateTSPBalances updates TSP balances after withdrawal
func (ce *CalculationEngine) updateTSPBalances(traditional, roth, withdrawal, returnRate decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	// Apply growth first
	traditional = traditional.Mul(decimal.NewFromFloat(1).Add(returnRate))
	roth = roth.Mul(decimal.NewFromFloat(1).Add(returnRate))

	// Withdraw from Roth first, then traditional
	if withdrawal.LessThanOrEqual(roth) {
		roth = roth.Sub(withdrawal)
	} else {
		remainingWithdrawal := withdrawal.Sub(roth)
		roth = decimal.Zero
		traditional = traditional.Sub(remainingWithdrawal)
		if traditional.LessThan(decimal.Zero) {
			traditional = decimal.Zero
		}
	}

	// Ensure balances never go negative
	if traditional.LessThan(decimal.Zero) {
		traditional = decimal.Zero
	}
	if roth.LessThan(decimal.Zero) {
		roth = decimal.Zero
	}

	return traditional, roth
}

// growTSPBalance grows a TSP balance with contributions and returns
func (ce *CalculationEngine) growTSPBalance(balance, contribution, returnRate decimal.Decimal) decimal.Decimal {
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(returnRate))
}

// growTSPBalanceWithAllocation calculates TSP balance growth using lifecycle fund allocation data
func (ce *CalculationEngine) growTSPBalanceWithAllocation(employee *domain.Employee, balance, contribution decimal.Decimal, targetDate time.Time) decimal.Decimal {
	// Get the appropriate allocation for this date
	allocation := ce.getTSPAllocationForEmployee(employee, targetDate)

	// Calculate weighted return based on allocation
	weightedReturn := ce.calculateTSPReturnWithAllocation(allocation, targetDate.Year())

	// Apply growth with the weighted return
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(weightedReturn))
}

// getTSPAllocationForEmployee returns the TSP allocation for an employee at a specific date
func (ce *CalculationEngine) getTSPAllocationForEmployee(employee *domain.Employee, targetDate time.Time) domain.TSPAllocation {
	// If employee has a lifecycle fund specified, use that
	if employee.TSPLifecycleFund != nil {
		allocation, err := ce.LifecycleFundLoader.GetAllocationAtDate(employee.TSPLifecycleFund.FundName, targetDate)
		if err == nil && allocation != nil {
			return *allocation
		}
		// Fall back to default if lifecycle fund lookup fails
	}

	// If employee has a specific allocation, use that
	if employee.TSPAllocation != nil {
		return *employee.TSPAllocation
	}

	// Use default allocation from global assumptions
	// This would need to be passed in from the configuration
	// For now, return a conservative default
	return domain.TSPAllocation{
		CFund: decimal.NewFromFloat(0.60),
		SFund: decimal.NewFromFloat(0.20),
		IFund: decimal.NewFromFloat(0.10),
		FFund: decimal.NewFromFloat(0.10),
		GFund: decimal.NewFromFloat(0.00),
	}
}

// calculateTSPReturnWithAllocation calculates TSP return using specific allocation and statistical models
func (ce *CalculationEngine) calculateTSPReturnWithAllocation(allocation domain.TSPAllocation, year int) decimal.Decimal {
	// Initialize return values
	var cFundReturn, sFundReturn, iFundReturn, fFundReturn, gFundReturn decimal.Decimal

	// Check if we have Monte Carlo fund returns available (higher priority than historical data)
	usingMonteCarlo := len(ce.MonteCarloFundReturns) > 0

	if usingMonteCarlo {
		// Get returns for this specific year
		// Note: Monte Carlo years are mapped to calendar years (e.g., 2025, 2026...)
		yearReturns, hasYearData := ce.MonteCarloFundReturns[year]

		if hasYearData {
			// Use Monte Carlo generated fund returns for this year
			if cReturn, exists := yearReturns["C"]; exists {
				cFundReturn = cReturn
			} else {
				cFundReturn = ce.getFallbackReturn("C", year)
			}

			if sReturn, exists := yearReturns["S"]; exists {
				sFundReturn = sReturn
			} else {
				sFundReturn = ce.getFallbackReturn("S", year)
			}

			if iReturn, exists := yearReturns["I"]; exists {
				iFundReturn = iReturn
			} else {
				iFundReturn = ce.getFallbackReturn("I", year)
			}

			if fReturn, exists := yearReturns["F"]; exists {
				fFundReturn = fReturn
			} else {
				fFundReturn = ce.getFallbackReturn("F", year)
			}

			if gReturn, exists := yearReturns["G"]; exists {
				gFundReturn = gReturn
			} else {
				gFundReturn = ce.getFallbackReturn("G", year)
			}
		} else {
			// Fallback if year data missing (shouldn't happen in proper simulation)
			cFundReturn = ce.getFallbackReturn("C", year)
			sFundReturn = ce.getFallbackReturn("S", year)
			iFundReturn = ce.getFallbackReturn("I", year)
			fFundReturn = ce.getFallbackReturn("F", year)
			gFundReturn = ce.getFallbackReturn("G", year)
		}
	} else {
		// Use historical data or statistical models for all funds
		cFundReturn = ce.getFallbackReturn("C", year)
		sFundReturn = ce.getFallbackReturn("S", year)
		iFundReturn = ce.getFallbackReturn("I", year)
		fFundReturn = ce.getFallbackReturn("F", year)
		gFundReturn = ce.getFallbackReturn("G", year)
	}

	// Weighted return calculation using actual allocation
	weightedReturn := decimal.Zero

	// C Fund (Large Cap)
	weightedReturn = weightedReturn.Add(allocation.CFund.Mul(cFundReturn))

	// S Fund (Small Cap)
	weightedReturn = weightedReturn.Add(allocation.SFund.Mul(sFundReturn))

	// I Fund (International)
	weightedReturn = weightedReturn.Add(allocation.IFund.Mul(iFundReturn))

	// F Fund (Bonds)
	weightedReturn = weightedReturn.Add(allocation.FFund.Mul(fFundReturn))

	// G Fund (Government)
	weightedReturn = weightedReturn.Add(allocation.GFund.Mul(gFundReturn))

	return weightedReturn
}

// getFallbackReturn gets historical or statistical fallback return for a fund
func (ce *CalculationEngine) getFallbackReturn(fund string, year int) decimal.Decimal {
	// Try historical data first
	if ce.HistoricalData != nil && ce.HistoricalData.IsLoaded {
		if returnRate, err := ce.HistoricalData.GetTSPReturn(fund, year); err == nil {
			return returnRate
		}
	}

	// Fallback to statistical models
	switch fund {
	case "C":
		return decimal.NewFromFloat(0.1125) // 11.25% historical mean
	case "S":
		return decimal.NewFromFloat(0.1117) // 11.17% historical mean
	case "I":
		return decimal.NewFromFloat(0.0634) // 6.34% historical mean
	case "F":
		return decimal.NewFromFloat(0.0532) // 5.32% historical mean
	case "G":
		return decimal.NewFromFloat(0.0493) // 4.93% historical mean
	default:
		return decimal.NewFromFloat(0.08) // 8% default
	}
}

// SimulateTSPGrowthPreRetirement simulates TSP growth before retirement
func SimulateTSPGrowthPreRetirement(initialBalance decimal.Decimal, annualContributions decimal.Decimal, annualReturn decimal.Decimal, years int) decimal.Decimal {
	currentBalance := initialBalance
	for i := 0; i < years; i++ {
		currentBalance = currentBalance.Add(annualContributions).Mul(decimal.NewFromFloat(1.0).Add(annualReturn))
	}
	return currentBalance
}

// ProjectTSP projects TSP balances and withdrawals over multiple years
func ProjectTSP(initialBalance decimal.Decimal, strategy TSPWithdrawalStrategy, returnRate decimal.Decimal, years int, birthYear int, targetIncome []decimal.Decimal) []domain.TSPProjection {
	projections := make([]domain.TSPProjection, years)
	currentBalance := initialBalance
	rmdCalc := NewRMDCalculator(birthYear)

	for year := 1; year <= years; year++ {
		// Calculate growth
		growth := currentBalance.Mul(returnRate)

		// Determine if this is an RMD year
		age := birthYear + year - 1
		isRMDYear := age >= rmdCalc.GetRMDAge()
		rmdAmount := rmdCalc.CalculateRMD(currentBalance, age)

		// Calculate withdrawal
		var targetIncomeForYear decimal.Decimal
		if year <= len(targetIncome) {
			targetIncomeForYear = targetIncome[year-1]
		}

		withdrawal := strategy.CalculateWithdrawal(currentBalance, year, targetIncomeForYear, age, isRMDYear, rmdAmount)

		// Ensure withdrawal doesn't exceed balance plus growth
		if withdrawal.GreaterThan(currentBalance.Add(growth)) {
			withdrawal = currentBalance.Add(growth)
		}

		// Calculate ending balance
		endingBalance := currentBalance.Add(growth).Sub(withdrawal)

		projections[year-1] = domain.TSPProjection{
			Year:             year,
			BeginningBalance: currentBalance,
			Growth:           growth,
			Withdrawal:       withdrawal,
			RMD:              rmdAmount,
			EndingBalance:    endingBalance,
		}

		currentBalance = endingBalance
	}

	return projections
}

// ProjectTSPWithTraditionalRoth projects TSP balances separately for Traditional and Roth accounts
func ProjectTSPWithTraditionalRoth(initialTraditional decimal.Decimal, initialRoth decimal.Decimal, strategy TSPWithdrawalStrategy, returnRate decimal.Decimal, years int, birthYear int, targetIncome []decimal.Decimal) ([]decimal.Decimal, []decimal.Decimal, []decimal.Decimal) {
	traditionalBalances := make([]decimal.Decimal, years)
	rothBalances := make([]decimal.Decimal, years)
	withdrawals := make([]decimal.Decimal, years)

	currentTraditional := initialTraditional
	currentRoth := initialRoth
	rmdCalc := NewRMDCalculator(birthYear)

	for year := 1; year <= years; year++ {
		// Calculate growth for both accounts
		traditionalGrowth := currentTraditional.Mul(returnRate)
		rothGrowth := currentRoth.Mul(returnRate)

		// Determine if this is an RMD year (only affects Traditional)
		age := birthYear + year - 1
		isRMDYear := age >= rmdCalc.GetRMDAge()
		rmdAmount := rmdCalc.CalculateRMD(currentTraditional, age)

		// Calculate withdrawal
		var targetIncomeForYear decimal.Decimal
		if year <= len(targetIncome) {
			targetIncomeForYear = targetIncome[year-1]
		}

		totalWithdrawal := strategy.CalculateWithdrawal(currentTraditional.Add(currentRoth), year, targetIncomeForYear, age, isRMDYear, rmdAmount)

		// Prioritize Roth withdrawals first (no RMD requirement)
		var rothWithdrawal, traditionalWithdrawal decimal.Decimal

		if isRMDYear && rmdAmount.GreaterThan(decimal.Zero) {
			// Must take RMD from Traditional first
			traditionalWithdrawal = rmdAmount
			remainingWithdrawal := totalWithdrawal.Sub(rmdAmount)

			if remainingWithdrawal.GreaterThan(decimal.Zero) {
				// Take remaining from Roth
				if remainingWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
					rothWithdrawal = currentRoth.Add(rothGrowth)
				} else {
					rothWithdrawal = remainingWithdrawal
				}
			}
		} else {
			// Take from Roth first, then Traditional
			if totalWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
				rothWithdrawal = currentRoth.Add(rothGrowth)
				traditionalWithdrawal = totalWithdrawal.Sub(rothWithdrawal)
			} else {
				rothWithdrawal = totalWithdrawal
			}
		}

		// Ensure withdrawals don't exceed balances
		if traditionalWithdrawal.GreaterThan(currentTraditional.Add(traditionalGrowth)) {
			traditionalWithdrawal = currentTraditional.Add(traditionalGrowth)
		}
		if rothWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
			rothWithdrawal = currentRoth.Add(rothGrowth)
		}

		// Update balances
		currentTraditional = currentTraditional.Add(traditionalGrowth).Sub(traditionalWithdrawal)
		currentRoth = currentRoth.Add(rothGrowth).Sub(rothWithdrawal)

		// Store results
		traditionalBalances[year-1] = currentTraditional
		rothBalances[year-1] = currentRoth
		withdrawals[year-1] = traditionalWithdrawal.Add(rothWithdrawal)
	}

	return traditionalBalances, rothBalances, withdrawals
}

package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// ConsoleVerboseFormatter renders the original detailed console report via the pluggable interface.
type ConsoleVerboseFormatter struct{}

func (c ConsoleVerboseFormatter) Name() string { return "console" }

func (c ConsoleVerboseFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "=================================================================================")
	fmt.Fprintln(&buf, "DETAILED FERS RETIREMENT INCOME ANALYSIS")
	fmt.Fprintln(&buf, "=================================================================================")
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "KEY ASSUMPTIONS:")
	assumptions := results.Assumptions
	if len(assumptions) == 0 {
		assumptions = DefaultAssumptions
	}
	for _, a := range assumptions {
		fmt.Fprintf(&buf, "• %s\n", a)
	}
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "CURRENT NET INCOME BREAKDOWN (Pre-Retirement)")
	fmt.Fprintln(&buf, "=============================================")
	fmt.Fprintf(&buf, "Combined Gross Salary: %s\n", FormatCurrency(decimal.NewFromFloat(367399.00)))
	fmt.Fprintf(&buf, "Combined Net Income:  %s\n", FormatCurrency(results.BaselineNetIncome))
	fmt.Fprintf(&buf, "Monthly Net Income:   %s\n", FormatCurrency(results.BaselineNetIncome.Div(decimal.NewFromInt(12))))
	fmt.Fprintln(&buf)

	// Detailed comparison (condensed from original GenerateDetailedComparison)
	writeDetailedComparison(&buf, results)

	for i, scenario := range results.Scenarios {
		fmt.Fprintf(&buf, "SCENARIO %d: %s\n", i+1, scenario.Name)
		fmt.Fprintln(&buf, strings.Repeat("=", 50))
		// first retirement year
		var firstRetirementYear domain.AnnualCashFlow
		var firstRetirementYearIndex int
		found := false
		for yIdx, y := range scenario.Projection {
			if y.IsRetired {
				firstRetirementYear = y
				firstRetirementYearIndex = yIdx
				found = true
				break
			}
		}
		if found {
			actualYear := 2025 + firstRetirementYearIndex
			fmt.Fprintf(&buf, "FIRST RETIREMENT YEAR (%d) INCOME BREAKDOWN:\n", actualYear)
			fmt.Fprintln(&buf, "(Note: Amounts shown are current-year cash received - may be partial year)")
			fmt.Fprintln(&buf, "----------------------------------------")
			fmt.Fprintln(&buf, "INCOME SOURCES:")
			fmt.Fprintf(&buf, "  PersonA Salary:        %s\n", FormatCurrency(firstRetirementYear.SalaryPersonA))
			fmt.Fprintf(&buf, "  PersonB Salary:          %s\n", FormatCurrency(firstRetirementYear.SalaryPersonB))
			fmt.Fprintf(&buf, "  PersonA FERS Pension:  %s\n", FormatCurrency(firstRetirementYear.PensionPersonA))
			fmt.Fprintf(&buf, "  PersonB FERS Pension:    %s\n", FormatCurrency(firstRetirementYear.PensionPersonB))
			fmt.Fprintf(&buf, "  PersonA TSP Withdrawal: %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalPersonA))
			fmt.Fprintf(&buf, "  PersonB TSP Withdrawal:   %s\n", FormatCurrency(firstRetirementYear.TSPWithdrawalPersonB))
			fmt.Fprintf(&buf, "  PersonA Social Security: %s\n", FormatCurrency(firstRetirementYear.SSBenefitPersonA))
			fmt.Fprintf(&buf, "  PersonB Social Security:   %s\n", FormatCurrency(firstRetirementYear.SSBenefitPersonB))
			fmt.Fprintf(&buf, "  PersonA FERS SRS:       %s\n", FormatCurrency(firstRetirementYear.FERSSupplementPersonA))
			fmt.Fprintf(&buf, "  PersonB FERS SRS:         %s\n", FormatCurrency(firstRetirementYear.FERSSupplementPersonB))
			fmt.Fprintf(&buf, "  TOTAL GROSS INCOME:      %s\n", FormatCurrency(firstRetirementYear.TotalGrossIncome))
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "DEDUCTIONS & TAXES:")
			fmt.Fprintf(&buf, "  Federal Tax:            %s\n", FormatCurrency(firstRetirementYear.FederalTax))
			fmt.Fprintf(&buf, "  State Tax:              %s\n", FormatCurrency(firstRetirementYear.StateTax))
			fmt.Fprintf(&buf, "  Local Tax:              %s\n", FormatCurrency(firstRetirementYear.LocalTax))
			fmt.Fprintf(&buf, "  FICA Tax:               %s\n", FormatCurrency(firstRetirementYear.FICATax))
			fmt.Fprintf(&buf, "  TSP Contributions:      %s\n", FormatCurrency(firstRetirementYear.TSPContributions))
			fmt.Fprintf(&buf, "  FEHB Premium:           %s\n", FormatCurrency(firstRetirementYear.FEHBPremium))
			fmt.Fprintf(&buf, "  Medicare Premium:       %s\n", FormatCurrency(firstRetirementYear.MedicarePremium))
			fmt.Fprintf(&buf, "  TOTAL DEDUCTIONS:       %s\n", FormatCurrency(firstRetirementYear.CalculateTotalDeductions()))
			fmt.Fprintln(&buf)
			fmt.Fprintln(&buf, "NET INCOME COMPARISON:")
			fmt.Fprintln(&buf, "----------------------")
			fmt.Fprintf(&buf, "  Current Net Income:     %s\n", FormatCurrency(results.BaselineNetIncome))
			fmt.Fprintf(&buf, "  Retirement Net Income:  %s\n", FormatCurrency(firstRetirementYear.NetIncome))
			change := firstRetirementYear.NetIncome.Sub(results.BaselineNetIncome)
			percentageChange := change.Div(results.BaselineNetIncome).Mul(decimal.NewFromInt(100))
			if change.GreaterThan(decimal.Zero) {
				fmt.Fprintf(&buf, "  CHANGE: +%s (+%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			} else {
				fmt.Fprintf(&buf, "  CHANGE: %s (%s)\n", FormatCurrency(change), FormatPercentage(percentageChange))
			}
			monthlyChange := change.Div(decimal.NewFromInt(12))
			if monthlyChange.GreaterThan(decimal.Zero) {
				fmt.Fprintf(&buf, "  Monthly Change: +%s\n", FormatCurrency(monthlyChange))
			} else {
				fmt.Fprintf(&buf, "  Monthly Change: %s\n", FormatCurrency(monthlyChange))
			}
			fmt.Fprintln(&buf, "RETIREMENT STATUS:")
			fmt.Fprintf(&buf, "  Is Retired:             %t\n", firstRetirementYear.IsRetired)
			fmt.Fprintf(&buf, "  Medicare Eligible:      %t\n", firstRetirementYear.IsMedicareEligible)
			fmt.Fprintf(&buf, "  RMD Year:               %t\n", firstRetirementYear.IsRMDYear)
			fmt.Fprintf(&buf, "  PersonA Age:           %d\n", firstRetirementYear.AgePersonA)
			fmt.Fprintf(&buf, "  PersonB Age:             %d\n", firstRetirementYear.AgePersonB)
			fmt.Fprintln(&buf)
		}

		// long term projection summary
		fmt.Fprintln(&buf, "LONG-TERM PROJECTION:")
		fmt.Fprintln(&buf, "---------------------")
		fmt.Fprintf(&buf, "  Year 5 Net Income:       %s\n", FormatCurrency(scenario.Year5NetIncome))
		fmt.Fprintf(&buf, "  Year 10 Net Income:      %s\n", FormatCurrency(scenario.Year10NetIncome))
		fmt.Fprintf(&buf, "  TSP Longevity:           %d years\n", scenario.TSPLongevity)
		fmt.Fprintf(&buf, "  Total Lifetime Income:   %s\n", FormatCurrency(scenario.TotalLifetimeIncome))
		// Optional: Health cost visibility at Year 5/10 if projection depth allows
		if found {
			y5Idx := firstRetirementYearIndex + 4  // 1-based => +4 for Year 5
			y10Idx := firstRetirementYearIndex + 9 // +9 for Year 10
			haveY5 := y5Idx >= 0 && y5Idx < len(scenario.Projection)
			haveY10 := y10Idx >= 0 && y10Idx < len(scenario.Projection)
			if haveY5 || haveY10 {
				fmt.Fprintln(&buf, "  Health Costs:")
				if haveY5 {
					p5 := scenario.Projection[y5Idx]
					fmt.Fprintf(&buf, "    FEHB (Year 5):         %s\n", FormatCurrency(p5.FEHBPremium))
					fmt.Fprintf(&buf, "    Medicare (Year 5):     %s\n", FormatCurrency(p5.MedicarePremium))
				}
				if haveY10 {
					p10 := scenario.Projection[y10Idx]
					fmt.Fprintf(&buf, "    FEHB (Year 10):        %s\n", FormatCurrency(p10.FEHBPremium))
					fmt.Fprintf(&buf, "    Medicare (Year 10):    %s\n", FormatCurrency(p10.MedicarePremium))
				}
			}
		}
		fmt.Fprintln(&buf)
		fmt.Fprintln(&buf)
	}

	// Recommendation section using existing AnalyzeScenarios logic
	rec := AnalyzeScenarios(results)
	if rec.ScenarioName != "" {
		fmt.Fprintln(&buf, "SUMMARY & RECOMMENDATIONS")
		fmt.Fprintln(&buf, "=========================")
		fmt.Fprintf(&buf, "Best scenario: %s\n", rec.ScenarioName)
		fmt.Fprintf(&buf, "Take-Home Income Change: %s (%s)\n", FormatCurrency(rec.NetIncomeChange), FormatPercentage(rec.PercentageChange))
		fmt.Fprintf(&buf, "Monthly Change: %s\n", FormatCurrency(rec.NetIncomeChange.Div(decimal.NewFromInt(12))))
	}

	return buf.Bytes(), nil
}

// writeDetailedComparison migrates the original GenerateDetailedComparison output (condensed)
func writeDetailedComparison(buf *bytes.Buffer, results *domain.ScenarioComparison) {
	fmt.Fprintln(buf, "=================================================================================")
	fmt.Fprintln(buf, "DETAILED INCOME VALIDATION: WORKING vs RETIREMENT")
	fmt.Fprintln(buf, "=================================================================================")
	for i, scenario := range results.Scenarios {
		var firstRetirementYear *domain.AnnualCashFlow
		for _, y := range scenario.Projection {
			if y.IsRetired {
				firstRetirementYear = &y
				break
			}
		}
		if firstRetirementYear == nil {
			continue
		}
		// Use the actual scenario name for clarity instead of hard-coded titles
		title := fmt.Sprintf("SCENARIO %d: %s", i+1, scenario.Name)
		fmt.Fprintf(buf, "\n%s\n", title)
		fmt.Fprintln(buf, strings.Repeat("=", len(title)))
		fmt.Fprintln(buf)
		fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", "COMPONENT", "WORKING", "RETIREMENT", "DIFFERENCE")
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		workingGross := decimal.NewFromFloat(367399.00)
		workingNet := results.BaselineNetIncome
		fmt.Fprintln(buf, "INCOME SOURCES:")
		cmpLine(buf, "  Salary (PersonA + PersonB)", workingGross, firstRetirementYear.SalaryPersonA.Add(firstRetirementYear.SalaryPersonB))
		cmpLine(buf, "  FERS Pension", decimal.Zero, firstRetirementYear.PensionPersonA.Add(firstRetirementYear.PensionPersonB))
		cmpLine(buf, "  TSP Withdrawals", decimal.Zero, firstRetirementYear.TSPWithdrawalPersonA.Add(firstRetirementYear.TSPWithdrawalPersonB))
		cmpLine(buf, "  Social Security", decimal.Zero, firstRetirementYear.SSBenefitPersonA.Add(firstRetirementYear.SSBenefitPersonB))
		cmpLine(buf, "  FERS Supplement", decimal.Zero, firstRetirementYear.FERSSupplementPersonA.Add(firstRetirementYear.FERSSupplementPersonB))
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		cmpLine(buf, "TOTAL GROSS INCOME", workingGross, firstRetirementYear.TotalGrossIncome)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "DEDUCTIONS & TAXES:")
		workingFederal := decimal.NewFromFloat(67060.18)
		workingState := decimal.NewFromFloat(11279.15)
		workingLocal := decimal.NewFromFloat(3673.99)
		workingFICA := decimal.NewFromFloat(16837.08)
		workingTSP := decimal.NewFromFloat(69812.52)
		workingFEHB := decimal.NewFromFloat(12700.74)
		cmpLine(buf, "  Federal Tax", workingFederal, firstRetirementYear.FederalTax)
		cmpLine(buf, "  State Tax", workingState, firstRetirementYear.StateTax)
		cmpLine(buf, "  Local Tax", workingLocal, firstRetirementYear.LocalTax)
		cmpLine(buf, "  FICA Tax", workingFICA, firstRetirementYear.FICATax)
		cmpLine(buf, "  TSP Contributions", workingTSP, firstRetirementYear.TSPContributions)
		cmpLine(buf, "  FEHB Premium", workingFEHB, firstRetirementYear.FEHBPremium)
		cmpLine(buf, "  Medicare Premium", decimal.Zero, firstRetirementYear.MedicarePremium)
		fmt.Fprintln(buf, strings.Repeat("-", 80))
		workingTotalDeductions := workingFederal.Add(workingState).Add(workingLocal).Add(workingFICA).Add(workingTSP).Add(workingFEHB)
		retirementTotalDeductions := firstRetirementYear.FederalTax.Add(firstRetirementYear.StateTax).Add(firstRetirementYear.LocalTax).Add(firstRetirementYear.FICATax).Add(firstRetirementYear.TSPContributions).Add(firstRetirementYear.FEHBPremium).Add(firstRetirementYear.MedicarePremium)
		cmpLine(buf, "TOTAL DEDUCTIONS", workingTotalDeductions, retirementTotalDeductions)
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, strings.Repeat("=", 80))
		cmpLine(buf, "NET TAKE-HOME INCOME", workingNet, firstRetirementYear.NetIncome)
		netDiff := firstRetirementYear.NetIncome.Sub(workingNet)
		percentChange := netDiff.Div(workingNet).Mul(decimal.NewFromInt(100))
		fmt.Fprintln(buf)
		fmt.Fprintln(buf, "KEY INSIGHTS:")
		fmt.Fprintf(buf, "• Working income is reduced by $%.2f in TSP contributions\n", workingTSP.InexactFloat64())
		fmt.Fprintf(buf, "• Working income is reduced by $%.2f in FICA taxes\n", workingFICA.InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in pension income\n", firstRetirementYear.PensionPersonA.Add(firstRetirementYear.PensionPersonB).InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in TSP withdrawals\n", firstRetirementYear.TSPWithdrawalPersonA.Add(firstRetirementYear.TSPWithdrawalPersonB).InexactFloat64())
		fmt.Fprintf(buf, "• Retirement adds $%.2f in Social Security\n", firstRetirementYear.SSBenefitPersonA.Add(firstRetirementYear.SSBenefitPersonB).InexactFloat64())
		if firstRetirementYear.FERSSupplementPersonA.Add(firstRetirementYear.FERSSupplementPersonB).GreaterThan(decimal.Zero) {
			fmt.Fprintf(buf, "• Retirement adds $%.2f in FERS supplement\n", firstRetirementYear.FERSSupplementPersonA.Add(firstRetirementYear.FERSSupplementPersonB).InexactFloat64())
		}
		fmt.Fprintf(buf, "\nNet Effect: %s (%s)\n", FormatCurrency(netDiff), FormatPercentage(percentChange))
		fmt.Fprintln(buf)
	}
}

func cmpLine(buf *bytes.Buffer, label string, working, retirement decimal.Decimal) {
	diff := retirement.Sub(working)
	fmt.Fprintf(buf, "%-35s %15s %15s %15s\n", label, FormatCurrency(working), FormatCurrency(retirement), FormatCurrency(diff))
}

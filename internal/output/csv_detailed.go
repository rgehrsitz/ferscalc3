package output

import (
	"bytes"
	"encoding/csv"
	"sort"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

// CSVDetailedExporter provides raw annual projection detail per scenario/year.
// Columns minimal placeholder until full extraction refactor.
type CSVDetailedExporter struct{}

func (c CSVDetailedExporter) Name() string { return "detailed-csv" }

func (c CSVDetailedExporter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)

	header := []string{
		"Scenario",
		"RelativeYear",
		"CalendarYear",
		"AgePersonA",
		"AgePersonB",
		"SalaryPersonA",
		"SalaryPersonB",
		"PensionPersonA",
		"PensionPersonB",
		"SurvivorPensionPersonA",
		"SurvivorPensionPersonB",
		"TSPWithdrawalPersonA",
		"TSPWithdrawalPersonB",
		"SSBenefitPersonA",
		"SSBenefitPersonB",
		"FERSSupplementPersonA",
		"FERSSupplementPersonB",
		"TotalGrossIncome",
		"FederalTaxableIncome",
		"FederalStandardDeduction",
		"FederalTax",
		"StateTax",
		"LocalTax",
		"FICATax",
		"FEHBPremium",
		"MedicarePremium",
		"MedicarePremiumPersonA",
		"MedicarePremiumPersonB",
		"TSPContributions",
		"NetIncome",
		"TSPBalancePersonA",
		"TSPBalancePersonB",
		"TSPBalanceTraditional",
		"TSPBalanceRoth",
		"WorkFractionPersonA",
		"WorkFractionPersonB",
		"IsRetired",
		"IsMedicareEligible",
		"IsRMDYear",
		"RMDAmount",
		"PersonADeceased",
		"PersonBDeceased",
		"FilingStatusSingle",
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}

	scenarios := append([]domain.ScenarioSummary(nil), results.Scenarios...)
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Name < scenarios[j].Name })

	for _, sc := range scenarios {
		for _, yr := range sc.Projection {
			row := []string{
				sc.Name,
				intToString(yr.Year),
				intToString(yr.Date.Year()),
				intToString(yr.AgePersonA),
				intToString(yr.AgePersonB),
				yr.SalaryPersonA.StringFixed(2),
				yr.SalaryPersonB.StringFixed(2),
				yr.PensionPersonA.StringFixed(2),
				yr.PensionPersonB.StringFixed(2),
				yr.SurvivorPensionPersonA.StringFixed(2),
				yr.SurvivorPensionPersonB.StringFixed(2),
				yr.TSPWithdrawalPersonA.StringFixed(2),
				yr.TSPWithdrawalPersonB.StringFixed(2),
				yr.SSBenefitPersonA.StringFixed(2),
				yr.SSBenefitPersonB.StringFixed(2),
				yr.FERSSupplementPersonA.StringFixed(2),
				yr.FERSSupplementPersonB.StringFixed(2),
				yr.TotalGrossIncome.StringFixed(2),
				yr.FederalTaxableIncome.StringFixed(2),
				yr.FederalStandardDeduction.StringFixed(2),
				yr.FederalTax.StringFixed(2),
				yr.StateTax.StringFixed(2),
				yr.LocalTax.StringFixed(2),
				yr.FICATax.StringFixed(2),
				yr.FEHBPremium.StringFixed(2),
				yr.MedicarePremium.StringFixed(2),
				yr.MedicarePremiumPersonA.StringFixed(2),
				yr.MedicarePremiumPersonB.StringFixed(2),
				yr.TSPContributions.StringFixed(2),
				yr.NetIncome.StringFixed(2),
				yr.TSPBalancePersonA.StringFixed(2),
				yr.TSPBalancePersonB.StringFixed(2),
				yr.TSPBalanceTraditional.StringFixed(2),
				yr.TSPBalanceRoth.StringFixed(2),
				yr.WorkFractionPersonA.StringFixed(2),
				yr.WorkFractionPersonB.StringFixed(2),
				boolToString(yr.IsRetired),
				boolToString(yr.IsMedicareEligible),
				boolToString(yr.IsRMDYear),
				yr.RMDAmount.StringFixed(2),
				boolToString(yr.PersonADeceased),
				boolToString(yr.PersonBDeceased),
				boolToString(yr.FilingStatusSingle),
			}
			if err := w.Write(row); err != nil {
				return nil, err
			}
		}
	}

	w.Flush()
	return buf.Bytes(), nil
}

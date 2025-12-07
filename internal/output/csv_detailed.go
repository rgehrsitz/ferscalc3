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
	// Expanded columns to make health premium impacts visible over time
	header := []string{"Scenario", "Year", "ActualYear", "AgePersonA", "AgePersonB", "PensionPersonA", "PensionPersonB", "SSBenefitPersonA", "SSBenefitPersonB", "NetIncome", "TotalGrossIncome", "TSPBalance", "FEHBPremium", "MedicarePremium", "IsRetired"}
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
				yr.PensionPersonA.StringFixed(2),
				yr.PensionPersonB.StringFixed(2),
				yr.SSBenefitPersonA.StringFixed(2),
				yr.SSBenefitPersonB.StringFixed(2),
				yr.NetIncome.StringFixed(2),
				yr.TotalGrossIncome.StringFixed(2),
				yr.TotalTSPBalance().StringFixed(2),
				yr.FEHBPremium.StringFixed(2),
				yr.MedicarePremium.StringFixed(2),
				boolToString(yr.IsRetired),
			}
			if err := w.Write(row); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), nil
}

package output

import (
	"bytes"
	"encoding/csv"
	"sort"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

// CSVSummarizer implements the simple summary CSV output (one row per scenario).
type CSVSummarizer struct{}

func (c CSVSummarizer) Name() string { return "csv" }

func (c CSVSummarizer) Format(results *domain.ScenarioComparison) ([]byte, error) {
	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	header := []string{"Scenario", "FirstYearNetIncome", "Year5NetIncome", "Year10NetIncome", "TSPLongevity", "TotalLifetimeIncomePV", "InitialTSPBalance", "FinalTSPBalance", "NetIncome2030", "NetIncome2035", "NetIncome2040", "PreRetirementNet2030", "PreRetirementNet2035", "PreRetirementNet2040"}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	scenarios := append([]domain.ScenarioSummary(nil), results.Scenarios...)
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Name < scenarios[j].Name })
	for _, sc := range scenarios {
		row := []string{
			sc.Name,
			sc.FirstYearNetIncome.StringFixed(2),
			sc.Year5NetIncome.StringFixed(2),
			sc.Year10NetIncome.StringFixed(2),
			intToString(sc.TSPLongevity),
			sc.TotalLifetimeIncome.StringFixed(2),
			sc.InitialTSPBalance.StringFixed(2),
			sc.FinalTSPBalance.StringFixed(2),
			sc.NetIncome2030.StringFixed(2),
			sc.NetIncome2035.StringFixed(2),
			sc.NetIncome2040.StringFixed(2),
			sc.PreRetirementNet2030.StringFixed(2),
			sc.PreRetirementNet2035.StringFixed(2),
			sc.PreRetirementNet2040.StringFixed(2),
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), nil
}

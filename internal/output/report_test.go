package output_test

import (
	"testing"

	stddec "github.com/shopspring/decimal"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/internal/output"
)

func TestFormatters(t *testing.T) {
	if got := output.FormatCurrency(stddec.NewFromFloat(123.45)); got != "$123.45" {
		t.Fatalf("FormatCurrency = %q", got)
	}
	if got := output.FormatPercentage(stddec.NewFromFloat(12.34)); got != "12.34%" {
		t.Fatalf("FormatPercentage = %q", got)
	}
}

func TestSaveConfiguration(t *testing.T) {
	cfg := &domain.Configuration{}
	dir := t.TempDir()
	path := dir + "/config.json"
	if err := output.SaveConfiguration(cfg, path); err != nil {
		t.Fatalf("SaveConfiguration error: %v", err)
	}
}

func TestReportGenerator_JSON_CSV(t *testing.T) {
	sc := &domain.ScenarioComparison{
		BaselineNetIncome: stddec.NewFromInt(0),
		Scenarios: []domain.ScenarioSummary{
			{
				Name:                "Baseline",
				FirstYearNetIncome:  stddec.NewFromInt(0),
				Year5NetIncome:      stddec.NewFromInt(0),
				Year10NetIncome:     stddec.NewFromInt(0),
				TotalLifetimeIncome: stddec.NewFromInt(0),
				TSPLongevity:        0,
				SuccessRate:         stddec.NewFromInt(0),
				InitialTSPBalance:   stddec.NewFromInt(0),
				FinalTSPBalance:     stddec.NewFromInt(0),
			},
		},
	}

	if err := output.GenerateReport(sc, "json"); err != nil {
		t.Fatalf("GenerateReport json error: %v", err)
	}
	if err := output.GenerateReport(sc, "csv"); err != nil {
		t.Fatalf("GenerateReport csv error: %v", err)
	}
}

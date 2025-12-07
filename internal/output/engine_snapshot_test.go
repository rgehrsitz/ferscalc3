package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
)

// TestEngineSnapshot produces a deterministic snapshot of core scenario metrics.
func TestEngineSnapshot(t *testing.T) {
	// Fix time and seed for determinism
	calculation.SetNowFunc(func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) })
	calculation.SetSeedFunc(func() int64 { return 12345 })

	parser := config.NewInputParser()
	cfg, err := parser.LoadFromFile("../../example_config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	eng := calculation.NewCalculationEngine()
	res, err := eng.RunScenarios(cfg)
	if err != nil {
		t.Fatalf("run scenarios: %v", err)
	}

	// Trim to stable summary fields only
	type scenario struct {
		Name      string `json:"name"`
		First     string `json:"first_year_net"`
		Y5        string `json:"year5"`
		Y10       string `json:"year10"`
		Longevity int    `json:"tsp_longevity"`
	}
	var out struct {
		Baseline  string     `json:"baseline_net_income"`
		Scenarios []scenario `json:"scenarios"`
	}
	out.Baseline = res.BaselineNetIncome.StringFixed(2)
	for _, sc := range res.Scenarios {
		out.Scenarios = append(out.Scenarios, scenario{
			Name:      sc.Name,
			First:     sc.FirstYearNetIncome.StringFixed(2),
			Y5:        sc.Year5NetIncome.StringFixed(2),
			Y10:       sc.Year10NetIncome.StringFixed(2),
			Longevity: sc.TSPLongevity,
		})
	}
	data, _ := json.MarshalIndent(out, "", "  ")

	goldenPath := filepath.Join("testdata", "engine_snapshot.golden.json")
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	if update {
		if err := os.WriteFile(goldenPath, data, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(golden) == "" {
		t.Fatalf("empty golden snapshot")
	}
	if string(golden) != string(data) {
		t.Fatalf("engine snapshot drift; run UPDATE_GOLDEN=1 to accept\n--- have ---\n%s\n--- want ---\n%s", string(data), string(golden))
	}
}

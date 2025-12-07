package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

func buildTestComparison() *domain.ScenarioComparison {
	cf := func(net int64, retired bool) domain.AnnualCashFlow {
		return domain.AnnualCashFlow{Year: 1, Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), NetIncome: decimal.NewFromInt(net), IsRetired: retired}
	}
	return &domain.ScenarioComparison{
		BaselineNetIncome: decimal.NewFromInt(100000),
		Scenarios: []domain.ScenarioSummary{
			{Name: "A", FirstYearNetIncome: decimal.NewFromInt(95000), Year5NetIncome: decimal.NewFromInt(96000), Year10NetIncome: decimal.NewFromInt(97000), TSPLongevity: 25, TotalLifetimeIncome: decimal.NewFromInt(1500000), Projection: []domain.AnnualCashFlow{cf(95000, true)}},
			{Name: "B", FirstYearNetIncome: decimal.NewFromInt(105000), Year5NetIncome: decimal.NewFromInt(106000), Year10NetIncome: decimal.NewFromInt(107000), TSPLongevity: 30, TotalLifetimeIncome: decimal.NewFromInt(1600000), Projection: []domain.AnnualCashFlow{cf(105000, true)}},
		},
	}
}

func TestConsoleLiteFormatter(t *testing.T) {
	f := ConsoleFormatter{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(out)
	if !strings.Contains(content, "Recommended: B") {
		t.Fatalf("expected recommendation for B, got: %s", content)
	}
}

func TestConsoleVerboseFormatter(t *testing.T) {
	f := ConsoleVerboseFormatter{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(out)
	if !strings.Contains(content, "DETAILED FERS RETIREMENT INCOME ANALYSIS") {
		t.Fatalf("expected verbose heading, got: %s", content[:120])
	}
}

func TestCSVSummarizerDeterministicOrder(t *testing.T) {
	f := CSVSummarizer{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header+2 rows), got %d", len(lines))
	}
	// Validate first data row starts with scenario A and second with B
	if !strings.HasPrefix(lines[1], "A,") || !strings.HasPrefix(lines[2], "B,") {
		t.Fatalf("rows not sorted deterministically: %v", lines)
	}
}

// Golden snapshot tests (prefix-based) ensure key headers remain stable.
func TestGoldenSnapshots(t *testing.T) {
	cases := []struct {
		name      string
		golden    string
		formatter Formatter
	}{
		{"console_verbose", "console_verbose.golden", ConsoleVerboseFormatter{}},
		{"console_lite", "console_lite.golden", ConsoleFormatter{}},
		{"csv_summary", "csv_summary.golden", CSVSummarizer{}},
		{"csv_detailed", "csv_detailed.golden", CSVDetailedExporter{}},
		{"html", "html_prefix.golden", HTMLFormatter{}},
	}

	cmp := buildTestComparison()
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	for _, tc := range cases {
		out, err := tc.formatter.Format(cmp)
		if err != nil {
			t.Fatalf("%s: format error: %v", tc.name, err)
		}
		goldenPath := filepath.Join("testdata", tc.golden)
		if update {
			// only first line to keep golden small & stable
			line := firstLine(string(out)) + "\n"
			if err := os.WriteFile(goldenPath, []byte(line), 0644); err != nil {
				t.Fatalf("%s: update golden failed: %v", tc.name, err)
			}
		}
		data, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("%s: read golden: %v", tc.name, err)
		}
		if !strings.HasPrefix(string(out), strings.TrimSpace(string(data))) {
			t.Fatalf("%s: output does not match golden prefix %q", tc.name, strings.TrimSpace(string(data)))
		}
	}
}

// Full snapshot (entire output) for verbose console using fixture comparison.
func TestFullVerboseConsoleGolden(t *testing.T) {
	f := ConsoleVerboseFormatter{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("format error: %v", err)
	}
	goldenPath := filepath.Join("testdata", "full", "console_verbose.full.golden")
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	if update {
		if err := os.WriteFile(goldenPath, out, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(data) == "(placeholder will be auto-updated with UPDATE_GOLDEN)\n" && !update {
		t.Skip("placeholder golden present; run with UPDATE_GOLDEN=1 to create initial snapshot")
	}
	if string(out) != string(data) {
		t.Fatalf("full verbose console output changed; run UPDATE_GOLDEN=1 to accept\n--- have ---\n%s\n--- want ---\n%s", truncate(string(out), 400), truncate(string(data), 400))
	}
}

func TestHTMLFormatterBasic(t *testing.T) {
	f := HTMLFormatter{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("html format error: %v", err)
	}
	content := string(out)
	if !strings.Contains(content, "Scenario Summary") {
		t.Fatalf("expected Scenario Summary section in HTML output")
	}
}

func TestHTMLAssumptionsSectionPresent(t *testing.T) {
	f := HTMLFormatter{}
	out, err := f.Format(buildTestComparison())
	if err != nil {
		t.Fatalf("html format error: %v", err)
	}
	content := string(out)
	// Small golden-like check: section header and at least one assumption bullet
	if !strings.Contains(content, "Key Assumptions") {
		t.Fatalf("expected Key Assumptions section in HTML output")
	}
	// Check one known default assumption phrase appears
	found := false
	for _, a := range DefaultAssumptions {
		if strings.Contains(content, a) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one default assumption to be rendered in HTML")
	}
}

func TestHTMLShowsLifetimeAndSuccessRate(t *testing.T) {
	// Arrange
	cmp := buildTestComparison()
	// inject success rate to make sure percentage renders
	for i := range cmp.Scenarios {
		// 87.5% as an example
		cmp.Scenarios[i].SuccessRate = decimal.NewFromFloat(87.5)
	}
	f := HTMLFormatter{}
	// Act
	out, err := f.Format(cmp)
	if err != nil {
		t.Fatalf("html format error: %v", err)
	}
	content := string(out)
	// Assert currency for lifetime income appears
	if !strings.Contains(content, "$1,500,000.00") && !strings.Contains(content, "$1500000.00") {
		t.Fatalf("expected formatted Total Lifetime Income in HTML, got: %s", content)
	}
	// Assert percentage formatting appears
	if !strings.Contains(content, "87.50%") {
		t.Fatalf("expected formatted Success Rate percentage in HTML")
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func TestFormatterAliasResolution(t *testing.T) {
	f := GetFormatterByName("console-verbose")
	if f == nil {
		t.Fatalf("alias console-verbose did not resolve to a formatter")
	}
	if f.Name() != "console" {
		t.Fatalf("alias resolved to %q, want 'console'", f.Name())
	}
}

func TestUnknownFormatErrorIncludesSuggestions(t *testing.T) {
	err := GenerateReport(&domain.ScenarioComparison{}, "definitely-not-a-format")
	if err == nil {
		t.Fatalf("expected error for unknown format")
	}
	msg := err.Error()
	if !strings.Contains(msg, "unsupported report format") || !strings.Contains(msg, "Try one of:") {
		t.Fatalf("error message missing suggestions: %s", msg)
	}
}

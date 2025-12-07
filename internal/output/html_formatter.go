package output

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"

	calc "github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/domain"
)

// HTMLFormatter produces an HTML report (current implementation ports legacy static HTML).
type HTMLFormatter struct{}

func (h HTMLFormatter) Name() string { return "html" }

//go:embed templates/report.html.tmpl
var htmlTemplateSource string

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"curr":   FormatCurrency,
	"pct":    FormatPercentage,
	"minus1": func(i int) int { return i - 1 },
	"add":    func(i, j int) int { return i + j },
	"slice": func(items []domain.ScenarioSummary, start int) []domain.ScenarioSummary {
		if start >= len(items) {
			return []domain.ScenarioSummary{}
		}
		return items[start:]
	},
	"json": func(v interface{}) template.JS {
		b, _ := json.Marshal(v)
		return template.JS(b)
	},
}).Parse(htmlTemplateSource))

func (h HTMLFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer
	rec := AnalyzeScenarios(results)

	// Use assumptions from results if available, otherwise fall back to defaults
	assumptions := results.Assumptions
	if len(assumptions) == 0 {
		assumptions = DefaultAssumptions
	}

	// Compute server-side cumulative break-even for scenarios 1 vs 2 if available
	var serverBreakEven *calc.CumulativeBreakEvenResult
	if len(results.Scenarios) >= 2 {
		projA := results.Scenarios[0].Projection
		projB := results.Scenarios[1].Projection
		if be, err := calc.CalculateCumulativeBreakEven(projA, projB); err == nil && be != nil {
			serverBreakEven = be
		}
	}

	data := struct {
		*domain.ScenarioComparison
		Recommendation Recommendation
		Assumptions    []string
		BreakEven      *calc.CumulativeBreakEvenResult
	}{results, rec, assumptions, serverBreakEven}
	if err := htmlTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

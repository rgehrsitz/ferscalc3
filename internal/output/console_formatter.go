package output

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// ConsoleFormatter provides a concise console style summary via the formatter interface.
type ConsoleFormatter struct{}

func (c ConsoleFormatter) Name() string { return "console-lite" }

func (c ConsoleFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "RETIREMENT SCENARIO SUMMARY")
	fmt.Fprintln(&buf, "================================")
	fmt.Fprintf(&buf, "Current Net Income: %s\n", FormatCurrency(results.BaselineNetIncome))
	fmt.Fprintln(&buf)
	scenarios := append([]domain.ScenarioSummary(nil), results.Scenarios...)
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].Name < scenarios[j].Name })
	for _, sc := range scenarios {
		var retiredNet decimal.Decimal
		for _, y := range sc.Projection {
			if y.IsRetired {
				retiredNet = y.NetIncome
				break
			}
		}
		fmt.Fprintf(&buf, "%s: FirstYear=%s Year5=%s Year10=%s Longevity=%d\n",
			sc.Name,
			FormatCurrency(sc.FirstYearNetIncome),
			FormatCurrency(sc.Year5NetIncome),
			FormatCurrency(sc.Year10NetIncome),
			sc.TSPLongevity,
		)
		fmt.Fprintf(&buf, "  FirstRetiredNet=%s LifetimePV=%s\n", FormatCurrency(retiredNet), FormatCurrency(sc.TotalLifetimeIncome))
	}
	rec := AnalyzeScenarios(results)
	if rec.ScenarioName != "" {
		fmt.Fprintln(&buf)
		fmt.Fprintf(&buf, "Recommended: %s (Δ %s / %s)\n", rec.ScenarioName, FormatCurrency(rec.NetIncomeChange), FormatPercentage(rec.PercentageChange))
	}
	return buf.Bytes(), nil
}

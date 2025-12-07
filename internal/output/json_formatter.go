package output

import (
	"encoding/json"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

// JSONFormatter serializes the scenario comparison as pretty-printed JSON.
type JSONFormatter struct{}

func (j JSONFormatter) Name() string { return "json" }

func (j JSONFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	return json.MarshalIndent(results, "", "  ")
}

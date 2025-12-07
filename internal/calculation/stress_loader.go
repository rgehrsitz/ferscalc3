package calculation

import (
	"fmt"
	"os"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// stressLibrary is the on-disk representation of shared stress scenarios.
type stressLibrary struct {
	Scenarios map[string]domain.StressScenario `yaml:"scenarios"`
}

// LoadStressScenarioLibrary loads stress scenarios from a YAML file.
func LoadStressScenarioLibrary(path string) (map[string]*domain.StressScenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read stress scenario file %s: %w", path, err)
	}

	var lib stressLibrary
	if err := yaml.Unmarshal(data, &lib); err != nil {
		return nil, fmt.Errorf("parse stress scenario file %s: %w", path, err)
	}

	result := make(map[string]*domain.StressScenario)
	for key, scenario := range lib.Scenarios {
		sc := scenario // copy to ensure pointer stability
		if sc.Name == "" {
			sc.Name = prettifyScenarioName(key)
		}
		result[strings.ToLower(key)] = &sc
	}
	return result, nil
}

func prettifyScenarioName(key string) string {
	key = strings.ReplaceAll(key, "_", " ")
	return cases.Title(language.English).String(key)
}

// MergeStressScenarioSources merges inline configuration scenarios with those loaded from file.
// Inline definitions take precedence over shared library entries.
func MergeStressScenarioSources(inline map[string]domain.StressScenario, library map[string]*domain.StressScenario) map[string]*domain.StressScenario {
	combined := make(map[string]*domain.StressScenario)

	for key, scenario := range library {
		if scenario == nil {
			continue
		}
		combined[key] = scenario
	}

	for key, scenario := range inline {
		sc := scenario
		nameKey := strings.ToLower(key)
		if sc.Name == "" {
			sc.Name = prettifyScenarioName(key)
		}
		combined[nameKey] = &sc
	}

	return combined
}

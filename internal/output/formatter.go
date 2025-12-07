package output

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

// Formatter defines a pluggable output formatter that returns a byte slice.
// Implementations should be pure (no side effects besides deterministic formatting).
type Formatter interface {
	Format(results *domain.ScenarioComparison) ([]byte, error)
	// Name returns a short identifier for logging / debugging.
	Name() string
}

// FormatterFunc adapter to allow ordinary functions to act as a Formatter.
type FormatterFunc struct {
	ID string
	F  func(*domain.ScenarioComparison) ([]byte, error)
}

func (ff FormatterFunc) Format(r *domain.ScenarioComparison) ([]byte, error) { return ff.F(r) }
func (ff FormatterFunc) Name() string                                        { return ff.ID }

// WriteFormatted runs a formatter and writes output to timestamped file with extension.
func WriteFormatted(f Formatter, results *domain.ScenarioComparison, ext string) (string, error) {
	data, err := f.Format(results)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("retirement_report_%s.%s", time.Now().Format("20060102_150405"), ext)
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return "", err
	}
	return filename, nil
}

// builtInFormatters stores available formatters (extended incrementally).
var builtInFormatters = []Formatter{
	ConsoleVerboseFormatter{},
	CSVSummarizer{},
	CSVDetailedExporter{},
	ConsoleFormatter{},
	HTMLFormatter{},
	JSONFormatter{},
}

// GetFormatterByName fetches a registered formatter.
func GetFormatterByName(name string) Formatter {
	n := NormalizeFormatName(name)
	for _, f := range builtInFormatters {
		if f.Name() == name {
			return f
		}
	}
	// try normalized name
	for _, f := range builtInFormatters {
		if f.Name() == n {
			return f
		}
	}
	return nil
}

// aliasMap provides user-friendly synonyms for format names.
var aliasMap = map[string]string{
	"console-verbose": "console",
	"verbose":         "console",
	"csv-detailed":    "detailed-csv",
	"csv-summary":     "csv",
	"html-report":     "html",
	"json-pretty":     "json",
}

// NormalizeFormatName lowers and resolves aliases.
func NormalizeFormatName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if mapped, ok := aliasMap[n]; ok {
		return mapped
	}
	return n
}

// AvailableFormatterNames returns the canonical formatter names.
func AvailableFormatterNames() []string {
	names := make([]string, 0, len(builtInFormatters))
	for _, f := range builtInFormatters {
		names = append(names, f.Name())
	}
	sort.Strings(names)
	return names
}

// AvailableFormatAliases returns the supported alias keys.
func AvailableFormatAliases() []string {
	keys := make([]string, 0, len(aliasMap))
	for k := range aliasMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

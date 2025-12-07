package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"gopkg.in/yaml.v3"
)

// ReportGenerator retained for deprecated Console path only.
type ReportGenerator struct{}

// GenerateReport prefers registered formatters; falls back to legacy generators for json/csv variants.
func GenerateReport(results *domain.ScenarioComparison, format string) error {
	if f := GetFormatterByName(format); f != nil {
		ext := format
		if format == "console-lite" {
			ext = "txt"
		}
		if strings.Contains(format, "csv") {
			ext = "csv"
		}
		_, err := WriteFormatted(f, results, ext)
		return err
	}
	switch format {
	case "json":
		// use formatter
		_, err := WriteFormatted(JSONFormatter{}, results, "json")
		return err
	case "csv":
		// use formatter
		_, err := WriteFormatted(CSVSummarizer{}, results, "csv")
		return err
	case "detailed-csv":
		// use formatter
		_, err := WriteFormatted(CSVDetailedExporter{}, results, "csv")
		return err
	case "all":
		if _, err := WriteFormatted(ConsoleVerboseFormatter{}, results, "txt"); err != nil {
			return err
		}
		if _, err := WriteFormatted(CSVDetailedExporter{}, results, "csv"); err != nil {
			return err
		}
		return nil
	default:
		// enrich error with available formatters and aliases
		return fmt.Errorf("%w: %q. Try one of: %s (aliases: %s)", ErrUnsupportedFormat, format, strings.Join(AvailableFormatterNames(), ", "), strings.Join(AvailableFormatAliases(), ", "))
	}
}

// Deprecated: use formatter "console".
func (rg *ReportGenerator) GenerateConsoleReport(results *domain.ScenarioComparison) error {
	if f := GetFormatterByName("console"); f != nil {
		_, err := WriteFormatted(f, results, "txt")
		return err
	}
	return ErrUnsupportedFormat
}

// Legacy csv/json generators removed in favor of formatters.

func SaveConfiguration(config *domain.Configuration, filename string) error {
	b, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, b, 0644)
}

// Deprecated stub (handled by verbose console formatter now).
func (rg *ReportGenerator) GenerateDetailedComparison(*domain.ScenarioComparison) {}

// Legacy detailed CSV generator removed; handled by CSVDetailedExporter formatter.

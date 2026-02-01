package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/internal/output"
	"github.com/spf13/cobra"
)

// calculateCmd represents the calculate command
var calculateCmd = &cobra.Command{
	Use:   "calculate [input-file]",
	Short: "Calculate retirement scenarios using deterministic projections",
	Long: `Calculate retirement scenarios based on the provided configuration file.
This command runs deterministic calculations for all scenarios defined in your
configuration and outputs the results in your preferred format.

Examples:
  fers-calc calculate config.yaml
  fers-calc calculate config.yaml --format html
  fers-calc calculate config.yaml --format csv --output report.csv
  fers-calc calculate config.yaml --format verbose --debug`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		// Get flags
		format, _ := cmd.Flags().GetString("format")
		outputFile, _ := cmd.Flags().GetString("output")
		debug, _ := cmd.Flags().GetBool("debug")
		quiet, _ := cmd.Flags().GetBool("quiet")

		if !quiet {
			fmt.Fprintf(os.Stderr, "Loading configuration from %s...\n", inputFile)
		}

		// Parse input configuration
		parser := config.NewInputParser()
		var cfg *domain.Configuration
		var err error
		if strings.EqualFold(filepath.Ext(inputFile), ".json") {
			cfg, err = parser.LoadFromJSONFile(inputFile)
		} else {
			cfg, err = parser.LoadFromFile(inputFile)
		}
		if err != nil {
			return fmt.Errorf("error loading configuration: %w", err)
		}

		// Validate configuration
		if err := parser.ValidateConfiguration(cfg); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "Configuration loaded successfully. Running calculations...\n")
		}

		// Create calculation engine
		engine := calculation.NewCalculationEngineWithConfigAndInflation(cfg.GlobalAssumptions.FederalRules, cfg.GlobalAssumptions.InflationRate)
		if debug {
			engine.Debug = true
		}

		// Run scenarios
		results, err := engine.RunScenariosWithContext(cmd.Context(), cfg)
		if err != nil {
			return fmt.Errorf("calculation failed: %w", err)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "Calculations completed. Generating %s output...\n", format)
		}

		// Generate output
		if outputFile != "" {
			// Output to file
			if err := generateReportToFile(results, format, outputFile); err != nil {
				return fmt.Errorf("error generating report: %w", err)
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "Report written to %s\n", outputFile)
			}
		} else {
			// Output to stdout
			if err := output.GenerateReport(results, format); err != nil {
				return fmt.Errorf("error generating report: %w", err)
			}
		}
		return nil
	},
}

func generateReportToFile(results *domain.ScenarioComparison, format, filename string) error {
	// Get the appropriate formatter
	formatter := output.GetFormatterByName(format)
	if formatter == nil {
		return fmt.Errorf("unsupported format: %s", format)
	}

	// Format the data
	data, err := formatter.Format(results)
	if err != nil {
		return fmt.Errorf("formatting failed: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(calculateCmd)

	// Format flags
	calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, verbose, html, json, csv, detailed-csv)")
	calculateCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")

	// Calculation flags
	calculateCmd.Flags().Bool("debug", false, "Enable debug logging with detailed calculation breakdowns")
}

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/internal/output"
)

// monteCarloCmd runs probabilistic simulations using the full FERS engine.
var monteCarloCmd = &cobra.Command{
	Use:   "monte-carlo [input-file]",
	Short: "Run FERS Monte Carlo simulations using a configuration file",
	Long: `Run Monte Carlo simulations that leverage the full deterministic FERS engine.

This command reuses your YAML configuration, injects market randomness (TSP returns,
inflation, COLA, FEHB variability), and aggregates retirement outcomes across many runs.

Examples:
  fers-calc monte-carlo dr.yaml --simulations 2000 --format html
  fers-calc monte-carlo dr.yaml --format csv --output reports/monte-carlo
  fers-calc monte-carlo dr.yaml --format json --output results.json --historical=false`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		format, _ := cmd.Flags().GetString("format")
		outputPath, _ := cmd.Flags().GetString("output")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		numSimulations, _ := cmd.Flags().GetInt("simulations")
		useHistorical, _ := cmd.Flags().GetBool("historical")
		seed, _ := cmd.Flags().GetInt64("seed")
		debug, _ := cmd.Flags().GetBool("debug")
		quiet, _ := cmd.Flags().GetBool("quiet")
		stressList, _ := cmd.Flags().GetString("stress-scenarios")
		stressAll, _ := cmd.Flags().GetBool("stress-all")
		stressFile, _ := cmd.Flags().GetString("stress-file")
		stressRepeat, _ := cmd.Flags().GetBool("stress-repeat")
		stressSweep, _ := cmd.Flags().GetBool("stress-sweep")

		if !quiet {
			fmt.Fprintf(os.Stderr, "Loading configuration from %s...\n", inputFile)
		}

		parser := config.NewInputParser()
		cfg, err := parser.LoadFromFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "Configuration loaded. Loading historical data from %s...\n", dataDir)
		}

		historicalData := calculation.NewHistoricalDataManager(dataDir)
		if err := historicalData.LoadAllData(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load historical data: %v\n", err)
			os.Exit(1)
		}

		engine := calculation.NewFERSMonteCarloEngine(cfg, historicalData)
		engine.SetDebug(debug)

		mcConfig := engine.DefaultConfig()
		mcConfig.BaseConfig = cfg
		if numSimulations > 0 {
			mcConfig.NumSimulations = numSimulations
		}
		mcConfig.UseHistorical = useHistorical
		mcConfig.Seed = seed

		stressSelection, selectionOrder, err := resolveStressScenarioSelection(cfg, dataDir, stressFile, stressList, stressAll)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve stress scenarios: %v\n", err)
			os.Exit(1)
		}

		if len(selectionOrder) == 0 {
			runSingleMonteCarlo(engine, mcConfig, format, outputPath, quiet)
			return
		}

		if err := runStressBatch(engine, mcConfig, selectionOrder, stressSelection, stressRepeat, stressSweep, format, outputPath, quiet); err != nil {
			fmt.Fprintf(os.Stderr, "Batch Monte Carlo failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(monteCarloCmd)

	monteCarloCmd.Flags().Int("simulations", 1000, "Number of Monte Carlo simulations to run")
	monteCarloCmd.Flags().Bool("historical", true, "Use historical data instead of statistical distributions")
	monteCarloCmd.Flags().Int64("seed", 0, "Seed for random number generation (0 = random)")
	monteCarloCmd.Flags().String("format", "html", "Output format: html, csv, or json")
	monteCarloCmd.Flags().StringP("output", "o", "", "Output file or directory (defaults depend on format)")
	monteCarloCmd.Flags().String("data-dir", "data", "Path to historical data directory")
	monteCarloCmd.Flags().Bool("debug", false, "Enable debug logging with detailed calculation breakdowns")
	monteCarloCmd.Flags().String("stress-scenarios", "", "Comma-separated list of stress scenarios to run (by name)")
	monteCarloCmd.Flags().Bool("stress-all", false, "Run all available stress scenarios in the library/config")
	monteCarloCmd.Flags().String("stress-file", "", "Path to an external stress scenario library (defaults to data/stress/periods.yaml)")
	monteCarloCmd.Flags().Bool("stress-repeat", false, "Repeat stress scenario sequences when projection years exceed defined years")
	monteCarloCmd.Flags().Bool("stress-sweep", false, "Overlay each stress scenario by sweeping it across all possible retirement start years (HTML-only output)")
}

func generateMonteCarloOutput(format, outputPath string, result *calculation.FERSMonteCarloResult, cfg calculation.FERSMonteCarloConfig, quiet bool) error {
	switch strings.ToLower(format) {
	case "html":
		if outputPath == "" {
			outputPath = "monte_carlo_report.html"
		}
		report := output.MonteCarloHTMLReport{Result: result, Config: cfg}
		if err := report.GenerateHTMLReport(outputPath); err != nil {
			return err
		}
		if !quiet {
			fmt.Fprintf(os.Stderr, "HTML report written to %s\n", outputPath)
		}
	case "csv":
		if outputPath == "" {
			outputPath = "monte_carlo_reports"
		}
		report := output.MonteCarloCSVReport{Result: result, Config: cfg}
		if err := report.GenerateAllCSVReports(outputPath); err != nil {
			return err
		}
		if !quiet {
			absPath, _ := filepath.Abs(outputPath)
			fmt.Fprintf(os.Stderr, "CSV reports written under %s\n", absPath)
		}
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		if outputPath == "" {
			fmt.Println(string(data))
		} else {
			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return err
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "JSON output written to %s\n", outputPath)
			}
		}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func runStressScenarioSweep(engine *calculation.FERSMonteCarloEngine, baseConfig calculation.FERSMonteCarloConfig, quiet bool) ([]output.StressSweepRun, *calculation.FERSMonteCarloResult, error) {
	stress := baseConfig.StressScenario
	if stress == nil || len(stress.Years) == 0 {
		return nil, nil, fmt.Errorf("stress sweep requires a scenario with defined years")
	}

	projectionYears := baseConfig.BaseConfig.GlobalAssumptions.ProjectionYears
	if projectionYears <= 0 {
		projectionYears = len(stress.Years)
	}

	stressLen := len(stress.Years)
	maxOffset := projectionYears - stressLen
	if maxOffset < 0 {
		maxOffset = 0
	}

	runs := make([]output.StressSweepRun, 0, maxOffset+1)
	sumSuccess := decimal.Zero
	sumMedianIncome := decimal.Zero
	sumMedianTSP := decimal.Zero
	sumDepletion := decimal.Zero
	sumVolatility := decimal.Zero

	var bestCase decimal.Decimal
	var worstCase decimal.Decimal
	var extremesSet bool
	var netIncomePercentiles calculation.PercentileRanges
	var tspLongevityPercentiles calculation.PercentileRanges
	var percentileSet bool

	for offset := 0; offset <= maxOffset; offset++ {
		runCfg := baseConfig
		runCfg.StressOffset = offset

		if !quiet {
			fmt.Fprintf(os.Stderr, "  - Offset %d/%d...\n", offset, maxOffset)
		}

		result, err := engine.RunFERSMonteCarlo(runCfg)
		if err != nil {
			return nil, nil, fmt.Errorf("offset %d failed: %w", offset, err)
		}

		years, netSeries, tspSeries := output.ExtractMedianTimeSeries(result)
		runs = append(runs, output.StressSweepRun{
			Offset:             offset,
			SuccessRate:        result.SuccessRate,
			MedianNetIncome:    result.MedianNetIncome,
			MedianFinalTSP:     result.MedianFinalTSPBalance,
			TSPDepletionRate:   result.TSPDepletionRate,
			IncomeVolatility:   result.IncomeVolatility,
			BestCaseNetIncome:  result.BestCaseScenario,
			WorstCaseNetIncome: result.WorstCaseScenario,
			Years:              years,
			NetMedianSeries:    netSeries,
			TSPMedianSeries:    tspSeries,
		})

		sumSuccess = sumSuccess.Add(result.SuccessRate)
		sumMedianIncome = sumMedianIncome.Add(result.MedianNetIncome)
		sumMedianTSP = sumMedianTSP.Add(result.MedianFinalTSPBalance)
		sumDepletion = sumDepletion.Add(result.TSPDepletionRate)
		sumVolatility = sumVolatility.Add(result.IncomeVolatility)

		if !extremesSet {
			bestCase = result.BestCaseScenario
			worstCase = result.WorstCaseScenario
			extremesSet = true
		} else {
			if result.BestCaseScenario.GreaterThan(bestCase) {
				bestCase = result.BestCaseScenario
			}
			if result.WorstCaseScenario.LessThan(worstCase) {
				worstCase = result.WorstCaseScenario
			}
		}

		if !percentileSet {
			netIncomePercentiles = result.NetIncomePercentiles
			tspLongevityPercentiles = result.TSPLongevityPercentiles
			percentileSet = true
		}
	}

	countDecimal := decimal.NewFromInt(int64(len(runs)))
	aggregate := &calculation.FERSMonteCarloResult{
		SuccessRate:           sumSuccess.Div(countDecimal),
		MedianNetIncome:       sumMedianIncome.Div(countDecimal),
		MedianFinalTSPBalance: sumMedianTSP.Div(countDecimal),
		TSPDepletionRate:      sumDepletion.Div(countDecimal),
		IncomeVolatility:      sumVolatility.Div(countDecimal),
		BestCaseScenario:      bestCase,
		WorstCaseScenario:     worstCase,
		StressScenarioName:    baseConfig.StressScenarioName + " (sweep avg)",
		NumSimulations:        baseConfig.NumSimulations,
		BaseConfig:            baseConfig.BaseConfig,
	}

	if percentileSet {
		aggregate.NetIncomePercentiles = netIncomePercentiles
		aggregate.TSPLongevityPercentiles = tspLongevityPercentiles
	} else {
		return nil, nil, fmt.Errorf("no successful sweep runs generated")
	}

	return runs, aggregate, nil
}

func printSummary(quiet bool, result *calculation.FERSMonteCarloResult) {
	if quiet {
		return
	}

	successRate := result.SuccessRate.Mul(decimal.NewFromInt(100)).InexactFloat64()
	fmt.Fprintf(os.Stderr, "Monte Carlo complete. Success Rate: %.2f%%, Median Net Income: $%s\n",
		successRate, result.MedianNetIncome.StringFixed(0))
}

func runSingleMonteCarlo(engine *calculation.FERSMonteCarloEngine, cfg calculation.FERSMonteCarloConfig, format, outputPath string, quiet bool) {
	if !quiet {
		fmt.Fprintf(os.Stderr, "Running %d Monte Carlo simulations...\n", cfg.NumSimulations)
	}

	result, err := engine.RunFERSMonteCarlo(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Monte Carlo simulation failed: %v\n", err)
		os.Exit(1)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Simulations completed. Generating %s output...\n", format)
	}

	if err := generateMonteCarloOutput(format, outputPath, result, cfg, quiet); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate Monte Carlo output: %v\n", err)
		os.Exit(1)
	}

	printSummary(quiet, result)
}

func resolveStressScenarioSelection(cfg *domain.Configuration, dataDir, stressFile, requested string, useAll bool) (map[string]*domain.StressScenario, []string, error) {
	inline := cfg.GlobalAssumptions.MonteCarloSettings.StressTests
	library := make(map[string]*domain.StressScenario)

	defaultLibraryPath := filepath.Join(dataDir, "stress", "periods.yaml")
	libraryPath := defaultLibraryPath
	if stressFile != "" {
		libraryPath = stressFile
	}

	if fileExists(libraryPath) {
		loaded, err := calculation.LoadStressScenarioLibrary(libraryPath)
		if err != nil {
			return nil, nil, err
		}
		library = loaded
	}

	combined := calculation.MergeStressScenarioSources(inline, library)
	if len(combined) == 0 {
		return combined, nil, nil
	}

	var order []string
	if useAll {
		for key := range combined {
			order = append(order, key)
		}
		sort.Strings(order)
		return combined, order, nil
	}

	if requested == "" {
		return combined, nil, nil
	}

	tokens := strings.Split(requested, ",")
	seen := make(map[string]bool, len(tokens))
	for _, raw := range tokens {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" || seen[key] {
			continue
		}
		if _, ok := combined[key]; !ok {
			return nil, nil, fmt.Errorf("unknown stress scenario: %s", raw)
		}
		seen[key] = true
		order = append(order, key)
	}

	return combined, order, nil
}

func runStressBatch(engine *calculation.FERSMonteCarloEngine, baseConfig calculation.FERSMonteCarloConfig, order []string, scenarios map[string]*domain.StressScenario, repeatOverride bool, stressSweep bool, format, outputPath string, quiet bool) error {
	if len(order) == 0 {
		return fmt.Errorf("no stress scenarios selected")
	}

	var batchResults []output.ScenarioBatchResult
	for idx, key := range order {
		stress := scenarios[key]
		if stress == nil {
			return fmt.Errorf("stress scenario %s missing definition", key)
		}

		runCfg := baseConfig
		runCfg.StressScenario = stress
		runCfg.StressScenarioName = displayNameForScenario(key, stress)
		runCfg.StressRepeat = repeatOverride || stress.Repeat

		if stressSweep {
			if !strings.EqualFold(format, "html") {
				return fmt.Errorf("stress sweep output is only supported for HTML format (scenario %s)", runCfg.StressScenarioName)
			}
			if !quiet {
				fmt.Fprintf(os.Stderr, "Sweeping stress scenario '%s' across all offsets (%d of %d)...\n", runCfg.StressScenarioName, idx+1, len(order))
			}

			sweepResults, aggregateResult, err := runStressScenarioSweep(engine, runCfg, quiet)
			if err != nil {
				return fmt.Errorf("scenario %s sweep failed: %w", runCfg.StressScenarioName, err)
			}

			scenarioOutputPath := deriveScenarioOutputPath(outputPath, key, idx == 0, format)
			report := output.StressSweepHTMLReport{
				Config:        runCfg,
				Scenario:      stress,
				Results:       sweepResults,
				DisplayName:   runCfg.StressScenarioName,
				Description:   stress.Description,
				ProjectionLen: runCfg.BaseConfig.GlobalAssumptions.ProjectionYears,
			}
			if err := report.GenerateHTMLReport(scenarioOutputPath); err != nil {
				return fmt.Errorf("failed to generate sweep output for %s: %w", runCfg.StressScenarioName, err)
			}

			if !quiet {
				fmt.Fprintf(os.Stderr, "HTML sweep overlay written to %s (offsets: %d)\n", scenarioOutputPath, len(sweepResults))
			}

			batchResults = append(batchResults, output.ScenarioBatchResult{
				Key:          key,
				DisplayName:  runCfg.StressScenarioName + " (sweep)",
				Description:  stress.Description,
				Result:       aggregateResult,
				SweepResults: sweepResults,
			})
			continue
		}

		if !quiet {
			fmt.Fprintf(os.Stderr, "Running stress scenario '%s' (%d of %d)...\n", runCfg.StressScenarioName, idx+1, len(order))
		}

		result, err := engine.RunFERSMonteCarlo(runCfg)
		if err != nil {
			return fmt.Errorf("scenario %s failed: %w", runCfg.StressScenarioName, err)
		}

		scenarioOutputPath := deriveScenarioOutputPath(outputPath, key, idx == 0, format)
		if err := generateMonteCarloOutput(format, scenarioOutputPath, result, runCfg, quiet); err != nil {
			return fmt.Errorf("failed to generate output for %s: %w", runCfg.StressScenarioName, err)
		}

		printSummary(quiet, result)

		batchResults = append(batchResults, output.ScenarioBatchResult{
			Key:         key,
			DisplayName: runCfg.StressScenarioName,
			Description: stress.Description,
			Result:      result,
		})
	}

	if err := generateBatchOutputs(format, outputPath, batchResults, quiet); err != nil {
		return err
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Stress batch complete. Generated %d scenario reports and combined summary.\n", len(batchResults))
	}
	return nil
}

func displayNameForScenario(key string, scenario *domain.StressScenario) string {
	name := strings.TrimSpace(scenario.Name)
	if name != "" {
		return name
	}
	return strings.Title(strings.ReplaceAll(key, "_", " "))
}

func deriveScenarioOutputPath(basePath, scenarioKey string, isPrimary bool, format string) string {
	if basePath == "" {
		return ""
	}

	if isPrimary {
		return basePath
	}

	if strings.EqualFold(format, "csv") {
		return fmt.Sprintf("%s_%s", basePath, slugify(scenarioKey))
	}

	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext)
	return fmt.Sprintf("%s_%s%s", name, slugify(scenarioKey), ext)
}

func deriveBatchOutputPath(basePath, format string) string {
	if basePath == "" {
		switch strings.ToLower(format) {
		case "html":
			return "monte_carlo_batch_report.html"
		case "csv":
			return "monte_carlo_batch_summary.csv"
		case "json":
			return "monte_carlo_batch.json"
		default:
			return ""
		}
	}

	if strings.EqualFold(format, "csv") {
		return fmt.Sprintf("%s_batch_summary.csv", basePath)
	}

	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext)
	if strings.EqualFold(format, "json") && ext == "" {
		return fmt.Sprintf("%s_batch.json", basePath)
	}
	return fmt.Sprintf("%s_batch%s", name, ext)
}

func generateBatchOutputs(format, baseOutputPath string, scenarios []output.ScenarioBatchResult, quiet bool) error {
	if len(scenarios) == 0 {
		return nil
	}

	batch := output.MonteCarloBatchReport{Scenarios: scenarios}
	target := deriveBatchOutputPath(baseOutputPath, format)

	switch strings.ToLower(format) {
	case "html":
		if err := batch.GenerateHTMLReport(target); err != nil {
			return err
		}
	case "csv":
		if err := batch.GenerateCSVReport(target); err != nil {
			return err
		}
	case "json":
		if err := batch.GenerateJSONReport(target); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported format for batch output: %s", format)
	}

	if !quiet && target != "" {
		fmt.Fprintf(os.Stderr, "Batch %s report written to %s\n", strings.ToUpper(format), target)
	}

	return nil
}

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.ReplaceAll(input, " ", "-")
	input = strings.ReplaceAll(input, "/", "-")
	input = strings.ReplaceAll(input, "\\", "-")
	input = strings.ReplaceAll(input, "__", "_")
	return input
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

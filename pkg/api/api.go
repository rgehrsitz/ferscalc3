package api

import (
	"context"
	"fmt"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/internal/output"
)

// Service exposes calculation capabilities for UI and server callers.
type Service struct {
	parser *config.InputParser
}

// NewService constructs a new API service.
func NewService() *Service {
	return &Service{parser: config.NewInputParser()}
}

// Validate validates configuration data and returns a typed error on failure.
func (s *Service) Validate(cfg *domain.Configuration) error {
	if cfg == nil {
		return &AppError{Kind: ErrorKindConfig, Message: "configuration is required"}
	}
	if err := s.parser.ValidateConfiguration(cfg); err != nil {
		return &AppError{Kind: ErrorKindValidation, Message: err.Error(), Err: err}
	}
	return nil
}

// LoadFromFile loads configuration from a YAML file.
func (s *Service) LoadFromFile(path string) (*domain.Configuration, error) {
	if path == "" {
		return nil, &AppError{Kind: ErrorKindConfig, Field: "path", Message: "path is required"}
	}
	cfg, err := s.parser.LoadFromFile(path)
	if err != nil {
		return nil, &AppError{Kind: ErrorKindConfig, Field: "path", Message: err.Error(), Err: err}
	}
	return cfg, nil
}

// LoadFromJSON loads configuration from JSON bytes.
func (s *Service) LoadFromJSON(data []byte) (*domain.Configuration, error) {
	cfg, err := s.parser.LoadFromJSON(data)
	if err != nil {
		return nil, &AppError{Kind: ErrorKindConfig, Field: "json", Message: err.Error(), Err: err}
	}
	return cfg, nil
}

// RunScenario runs deterministic calculations for all scenarios.
func (s *Service) RunScenario(ctx context.Context, cfg *domain.Configuration, debug bool) (*domain.ScenarioComparison, error) {
	if err := s.Validate(cfg); err != nil {
		return nil, err
	}

	// Create a context with timeout for calculations
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	engine := calculation.NewCalculationEngineWithConfigAndInflation(cfg.GlobalAssumptions.FederalRules, cfg.GlobalAssumptions.InflationRate)
	engine.Debug = debug

	results, err := engine.RunScenariosWithContext(ctx, cfg)
	if err != nil {
		return nil, &AppError{Kind: ErrorKindCalculation, Message: err.Error(), Err: err}
	}
	if results == nil {
		return nil, &AppError{Kind: ErrorKindCalculation, Message: "no results returned"}
	}
	return results, nil
}

// RunScenarioWithParser allows callers to inject a parser (advanced use).
func RunScenarioWithParser(ctx context.Context, parser *config.InputParser, cfg *domain.Configuration, debug bool) (*domain.ScenarioComparison, error) {
	if parser == nil {
		return nil, &AppError{Kind: ErrorKindConfig, Message: "parser is required"}
	}
	svc := &Service{parser: parser}
	return svc.RunScenario(ctx, cfg, debug)
}

// FormatReport formats results using a supported formatter name.
func FormatReport(results *domain.ScenarioComparison, format string) ([]byte, error) {
	if results == nil {
		return nil, &AppError{Kind: ErrorKindConfig, Message: "results are required"}
	}
	formatter := outputFormatter(format)
	if formatter == nil {
		return nil, &AppError{Kind: ErrorKindConfig, Field: "format", Message: fmt.Sprintf("unsupported format: %s", format)}
	}
	data, err := formatter.Format(results)
	if err != nil {
		return nil, &AppError{Kind: ErrorKindCalculation, Message: err.Error(), Err: err}
	}
	return data, nil
}

func outputFormatter(format string) output.Formatter {
	if format == "" {
		format = "console"
	}
	return output.GetFormatterByName(format)
}

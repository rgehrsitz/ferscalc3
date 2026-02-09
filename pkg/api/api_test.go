package api

import (
	"context"
	"testing"

	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunScenario_UsesConfiguredState(t *testing.T) {
	svc := NewService()
	parser := config.NewInputParser()

	paCfg := parser.CreateExampleConfiguration()
	paCfg.GlobalAssumptions.CurrentLocation.State = "PA"
	paResult, err := svc.RunScenario(context.Background(), paCfg, false)
	require.NoError(t, err)
	require.NotEmpty(t, paResult.Scenarios)
	require.NotEmpty(t, paResult.Scenarios[0].Projection)
	paStateTax := paResult.Scenarios[0].Projection[0].StateTax

	njCfg := parser.CreateExampleConfiguration()
	njCfg.GlobalAssumptions.CurrentLocation.State = "NJ"
	njResult, err := svc.RunScenario(context.Background(), njCfg, false)
	require.NoError(t, err)
	require.NotEmpty(t, njResult.Scenarios)
	require.NotEmpty(t, njResult.Scenarios[0].Projection)
	njStateTax := njResult.Scenarios[0].Projection[0].StateTax

	assert.False(t, paStateTax.Equal(njStateTax), "PA and NJ should produce different state tax in the working year")
}

func TestRunScenario_AppliesFEHBDefaultsWhenMissing(t *testing.T) {
	svc := NewService()
	parser := config.NewInputParser()

	cfg := parser.CreateExampleConfiguration()
	cfg.GlobalAssumptions.FederalRules.FEHBConfig.PayPeriodsPerYear = 0
	cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementCalculationMethod = ""
	cfg.GlobalAssumptions.FederalRules.FEHBConfig.RetirementPremiumMultiplier = decimal.Zero

	results, err := svc.RunScenario(context.Background(), cfg, false)
	require.NoError(t, err)
	require.NotEmpty(t, results.Scenarios)
	require.NotEmpty(t, results.Scenarios[0].Projection)

	year0FEHB := results.Scenarios[0].Projection[0].FEHBPremium
	assert.True(t, year0FEHB.GreaterThan(decimal.Zero), "FEHB premium should be non-zero when employee FEHB input is present")
}

func TestRunScenario_AppliesFederalRulesDefaultsWhenMissing(t *testing.T) {
	svc := NewService()
	parser := config.NewInputParser()

	cfg := parser.CreateExampleConfiguration()
	cfg.GlobalAssumptions.FederalRules = domain.FederalRules{}
	cfg.GlobalAssumptions.CurrentLocation.State = "PA"

	results, err := svc.RunScenario(context.Background(), cfg, false)
	require.NoError(t, err)
	require.NotEmpty(t, results.Scenarios)
	require.NotEmpty(t, results.Scenarios[0].Projection)

	year0 := results.Scenarios[0].Projection[0]
	assert.True(t, year0.FederalTax.GreaterThan(decimal.Zero), "federal tax should be non-zero with working wages")
	assert.True(t, year0.StateTax.GreaterThan(decimal.Zero), "PA state tax should be non-zero with working wages")
	assert.True(t, year0.LocalTax.GreaterThan(decimal.Zero), "PA local EIT should be non-zero with working wages")
	assert.True(t, year0.FICATax.GreaterThan(decimal.Zero), "FICA should be non-zero with working wages")
	assert.True(t, year0.FEHBPremium.GreaterThan(decimal.Zero), "FEHB should be non-zero with pay period defaults")
}

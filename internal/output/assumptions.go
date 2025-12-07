package output

import (
	"fmt"
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// DefaultAssumptions lists key modeling assumptions rendered in detailed outputs.
// Future: could be loaded from configuration or generated dynamically.
var DefaultAssumptions = []string{
	"General COLA (FERS pension & SS): 2.5% annually",
	"FEHB premium inflation: 4.0% annually",
	"TSP growth pre-retirement: 7.0% annually",
	"TSP growth post-retirement: 5.0% annually",
	"Social Security wage base indexing: ~5% annually (2025 est: $168,600)",
	"Tax brackets: 2025 levels held constant (no inflation indexing)",
}

// GenerateAssumptions creates dynamic assumptions list from actual config values
func GenerateAssumptions(assumptions *domain.GlobalAssumptions) []string {
	return []string{
		fmt.Sprintf("General COLA (FERS pension & SS): %.1f%% annually", assumptions.COLAGeneralRate.Mul(decimalHundred).InexactFloat64()),
		fmt.Sprintf("FEHB premium inflation: %.1f%% annually", assumptions.FEHBPremiumInflation.Mul(decimalHundred).InexactFloat64()),
		fmt.Sprintf("TSP growth pre-retirement: %.1f%% annually", assumptions.TSPReturnPreRetirement.Mul(decimalHundred).InexactFloat64()),
		fmt.Sprintf("TSP growth post-retirement: %.1f%% annually", assumptions.TSPReturnPostRetirement.Mul(decimalHundred).InexactFloat64()),
		"Social Security wage base indexing: ~5% annually (2025 est: $168,600)",
		"Tax brackets: 2025 levels held constant (no inflation indexing)",
	}
}

var decimalHundred = decimal.NewFromInt(100)

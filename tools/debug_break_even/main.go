package main

import (
	"fmt"
	"os"

	calc "github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/shopspring/decimal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: debug_break_even <config-file>")
		return
	}
	f := os.Args[1]
	p := config.NewInputParser()
	cfg, err := p.LoadFromFile(f)
	if err != nil {
		panic(err)
	}
	engine := calc.NewCalculationEngineWithConfigAndInflation(cfg.GlobalAssumptions.FederalRules, cfg.GlobalAssumptions.InflationRate)
	res, err := engine.RunScenarios(cfg)
	if err != nil {
		panic(err)
	}
	if len(res.Scenarios) < 1 {
		fmt.Println("no scenarios")
		return
	}

	// Find the minimum projection length across scenarios
	minLen := -1
	for _, s := range res.Scenarios {
		if minLen == -1 || len(s.Projection) < minLen {
			minLen = len(s.Projection)
		}
	}
	if minLen <= 0 {
		fmt.Println("no projection data")
		return
	}

	// Header
	header := "Index,Date,Year"
	for i := range res.Scenarios {
		header += fmt.Sprintf(",S%d_Salary,S%d_Pension,S%d_TSP,S%d_SS,S%d_Net", i+1, i+1, i+1, i+1, i+1)
	}
	fmt.Println(header)

	// Iterate years and print components
	for idx := 0; idx < minLen; idx++ {
		row := fmt.Sprintf("%d,%s,%d", idx, res.Scenarios[0].Projection[idx].Date.Format("2006-01-02"), res.Scenarios[0].Projection[idx].Date.Year())
		for sidx := range res.Scenarios {
			p := res.Scenarios[sidx].Projection[idx]
			row += fmt.Sprintf(",%s,%s,%s,%s,%s", p.SalaryPersonA.Add(p.SalaryPersonB).StringFixed(0), p.PensionPersonA.Add(p.PensionPersonB).StringFixed(0), p.TSPWithdrawalPersonA.Add(p.TSPWithdrawalPersonB).StringFixed(0), p.SSBenefitPersonA.Add(p.SSBenefitPersonB).StringFixed(0), p.NetIncome.StringFixed(0))
		}
		fmt.Println(row)
	}

	// If at least two scenarios, compute cumulative diffs for first two
	if len(res.Scenarios) >= 2 {
		a := res.Scenarios[0].Projection
		b := res.Scenarios[1].Projection
		cumA := decimal.Zero
		cumB := decimal.Zero
		for i := 0; i < len(a) && i < len(b); i++ {
			cumA = cumA.Add(a[i].NetIncome)
			cumB = cumB.Add(b[i].NetIncome)
			fmt.Printf("Cumulative Year %d: cumA=%s cumB=%s diff=%s\n", a[i].Date.Year(), cumA.StringFixed(0), cumB.StringFixed(0), cumA.Sub(cumB).StringFixed(0))
		}
		be, err := calc.CalculateCumulativeBreakEven(a, b)
		fmt.Printf("\nBreakEven: %+v, err=%v\n", be, err)
	}
}

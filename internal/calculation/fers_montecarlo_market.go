package calculation

import (
	"math"
	"math/rand"
	"strings"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

type marketGenerator struct {
	config         *FERSMonteCarloConfig
	historicalData *HistoricalDataManager
}

func newMarketGenerator(cfg *FERSMonteCarloConfig, data *HistoricalDataManager) *marketGenerator {
	return &marketGenerator{
		config:         cfg,
		historicalData: data,
	}
}

func (g *marketGenerator) generateMarketConditions() MarketCondition {
	if g.config.UseHistorical {
		return g.generateEnhancedHistoricalMarketConditions()
	}
	return g.generateStatisticalMarketConditions()
}

func (g *marketGenerator) generateMarketConditionSeries(years int) MarketConditionSeries {
	if g.config.StressScenario != nil && len(g.config.StressScenario.Years) > 0 {
		return g.generateStressConditionSeries(years)
	}

	series := MarketConditionSeries{
		Years: make([]MarketCondition, years),
	}

	for i := 0; i < years; i++ {
		condition := g.generateMarketConditions()
		condition.Year = 2025 + i // Ensure correct calendar year
		series.Years[i] = condition
	}

	return series
}

func (g *marketGenerator) generateStressConditionSeries(years int) MarketConditionSeries {
	series := MarketConditionSeries{
		Years: make([]MarketCondition, years),
	}

	stress := g.config.StressScenario
	if stress == nil || len(stress.Years) == 0 {
		return series
	}

	baseAssumptions := g.config.BaseConfig.GlobalAssumptions
	count := len(stress.Years)
	repeatEnabled := g.config.StressRepeat || stress.Repeat
	offset := g.config.StressOffset
	if offset < 0 {
		offset = 0
	}
	if offset > years {
		offset = years
	}
	stressEnd := offset + count

	for i := 0; i < years; i++ {
		switch {
		case i < offset:
			condition := g.generateMarketConditions()
			condition.Year = 2025 + i
			series.Years[i] = condition
		case i >= offset && i < stressEnd:
			stressIndex := i - offset
			if stressIndex >= 0 && stressIndex < count {
				series.Years[i] = buildStressMarket(stress.Years[stressIndex], baseAssumptions)
				break
			}
			fallthrough
		case repeatEnabled && count > 0 && i >= stressEnd:
			loopIndex := (i - offset) % count
			if loopIndex < 0 {
				loopIndex = 0
			}
			series.Years[i] = buildStressMarket(stress.Years[loopIndex], baseAssumptions)
		default:
			condition := g.generateMarketConditions()
			condition.Year = 2025 + i
			series.Years[i] = condition
		}
	}

	return series
}

func (g *marketGenerator) pickStressYear(stress *domain.StressScenario, index int) domain.StressYear {
	count := len(stress.Years)
	if count == 0 {
		return domain.StressYear{}
	}

	if index < count {
		return stress.Years[index]
	}

	if g.config.StressRepeat || stress.Repeat {
		return stress.Years[index%count]
	}

	return stress.Years[count-1]
}

func stressValueOrDefault(value *decimal.Decimal, fallback decimal.Decimal) decimal.Decimal {
	if value == nil {
		return fallback
	}
	return *value
}

func buildStressMarket(yearData domain.StressYear, baseAssumptions domain.GlobalAssumptions) MarketCondition {
	tspReturns := make(map[string]decimal.Decimal, len(yearData.TSPReturns))
	for fund, value := range yearData.TSPReturns {
		tspReturns[strings.ToUpper(fund)] = value
	}

	return MarketCondition{
		Year:          yearData.Year,
		TSPReturns:    tspReturns,
		InflationRate: stressValueOrDefault(yearData.Inflation, baseAssumptions.InflationRate),
		COLARate:      stressValueOrDefault(yearData.COLA, baseAssumptions.COLAGeneralRate),
		FEHBIncrease:  stressValueOrDefault(yearData.FEHB, baseAssumptions.FEHBPremiumInflation),
	}
}

func (g *marketGenerator) generateEnhancedHistoricalMarketConditions() MarketCondition {
	minYear, maxYear, err := g.historicalData.GetAvailableYears()
	if err != nil {
		return g.generateStatisticalMarketConditions()
	}

	marketData := MarketCondition{
		TSPReturns: make(map[string]decimal.Decimal),
	}

	tspYear := minYear + rand.Intn(maxYear-minYear+1)
	inflationYear := minYear + rand.Intn(maxYear-minYear+1)
	colaYear := minYear + rand.Intn(maxYear-minYear+1)
	fehbYear := minYear + rand.Intn(maxYear-minYear+1)

	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		if baseReturn, err := g.historicalData.GetTSPReturn(fund, tspYear); err == nil {
			variability := g.randomVariability(g.config.TSPReturnVariability)
			adjusted := baseReturn.Mul(decimal.NewFromFloat(1.0).Add(variability))
			marketData.TSPReturns[fund] = adjusted
		} else {
			marketData.TSPReturns[fund] = g.generateStatisticalTSPReturn(fund)
		}
	}

	if baseInflation, err := g.historicalData.GetInflationRate(inflationYear); err == nil {
		variability := g.randomVariability(g.config.InflationVariability)
		marketData.InflationRate = baseInflation.Mul(decimal.NewFromFloat(1.0).Add(variability))
	} else {
		marketData.InflationRate = g.generateStatisticalInflation()
	}

	if baseCOLA, err := g.historicalData.GetCOLARate(colaYear); err == nil {
		variability := g.randomVariability(g.config.COLAVariability)
		marketData.COLARate = baseCOLA.Mul(decimal.NewFromFloat(1.0).Add(variability))
	} else {
		marketData.COLARate = g.generateStatisticalCOLA()
	}

	if baseFEHB, err := g.historicalData.GetInflationRate(fehbYear); err == nil {
		variability := g.randomVariability(g.config.FEHBVariability)
		marketData.FEHBIncrease = baseFEHB.Mul(decimal.NewFromFloat(1.0).Add(variability))
	} else {
		marketData.FEHBIncrease = g.generateStatisticalInflation()
	}

	marketData.Year = tspYear
	return marketData
}

func (g *marketGenerator) generateStatisticalMarketConditions() MarketCondition {
	marketData := MarketCondition{
		Year:       rand.Intn(30) + 2025,
		TSPReturns: make(map[string]decimal.Decimal),
	}

	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		marketData.TSPReturns[fund] = g.generateStatisticalTSPReturn(fund)
	}

	marketData.InflationRate = g.generateStatisticalInflation()
	marketData.COLARate = g.generateStatisticalCOLA()
	marketData.FEHBIncrease = g.generateStatisticalFEHBIncrease()

	return marketData
}

func (g *marketGenerator) randomVariability(stdDev decimal.Decimal) decimal.Decimal {
	if stdDev.IsZero() {
		return decimal.Zero
	}

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)

	variability := decimal.NewFromFloat(z).Mul(stdDev)
	maxVariability := stdDev.Mul(decimal.NewFromFloat(3.0))
	minVariability := stdDev.Mul(decimal.NewFromFloat(-3.0))

	if variability.GreaterThan(maxVariability) {
		return maxVariability
	}
	if variability.LessThan(minVariability) {
		return minVariability
	}
	return variability
}

func (g *marketGenerator) generateStatisticalTSPReturn(fund string) decimal.Decimal {
	models := g.config.BaseConfig.GlobalAssumptions.TSPStatisticalModels

	var (
		mean, stdDev decimal.Decimal
		ok           bool
	)

	switch fund {
	case "C":
		mean, stdDev, ok = models.CFund.Mean, models.CFund.StandardDev, !models.CFund.Mean.IsZero() && !models.CFund.StandardDev.IsZero()
	case "S":
		mean, stdDev, ok = models.SFund.Mean, models.SFund.StandardDev, !models.SFund.Mean.IsZero() && !models.SFund.StandardDev.IsZero()
	case "I":
		mean, stdDev, ok = models.IFund.Mean, models.IFund.StandardDev, !models.IFund.Mean.IsZero() && !models.IFund.StandardDev.IsZero()
	case "F":
		mean, stdDev, ok = models.FFund.Mean, models.FFund.StandardDev, !models.FFund.Mean.IsZero() && !models.FFund.StandardDev.IsZero()
	case "G":
		mean, stdDev, ok = models.GFund.Mean, models.GFund.StandardDev, !models.GFund.Mean.IsZero() && !models.GFund.StandardDev.IsZero()
	}

	if !ok {
		switch fund {
		case "C":
			mean = decimal.NewFromFloat(0.1125)
			stdDev = decimal.NewFromFloat(0.1744)
		case "S":
			mean = decimal.NewFromFloat(0.1117)
			stdDev = decimal.NewFromFloat(0.1933)
		case "I":
			mean = decimal.NewFromFloat(0.0634)
			stdDev = decimal.NewFromFloat(0.1863)
		case "F":
			mean = decimal.NewFromFloat(0.0532)
			stdDev = decimal.NewFromFloat(0.0565)
		case "G":
			mean = decimal.NewFromFloat(0.0493)
			stdDev = decimal.NewFromFloat(0.0165)
		default:
			mean = decimal.NewFromFloat(0.08)
			stdDev = decimal.NewFromFloat(0.15)
		}
	}

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := g.boxMullerTransform(u1, u2)

	return mean.Add(decimal.NewFromFloat(z).Mul(stdDev))
}

func (g *marketGenerator) generateStatisticalInflation() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0259)
	stdDev := decimal.NewFromFloat(0.0137)

	return g.applyBoundedNormal(mean, stdDev, decimal.Zero, decimal.NewFromFloat(0.20))
}

func (g *marketGenerator) generateStatisticalCOLA() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0255)
	stdDev := decimal.NewFromFloat(0.0182)

	return g.applyBoundedNormal(mean, stdDev, decimal.Zero, decimal.NewFromFloat(0.15))
}

func (g *marketGenerator) generateStatisticalFEHBIncrease() decimal.Decimal {
	mean := decimal.NewFromFloat(0.045)
	stdDev := decimal.NewFromFloat(0.025)

	return mean.Add(decimal.NewFromFloat(g.boxMullerTransform(rand.Float64(), rand.Float64())).Mul(stdDev))
}

func (g *marketGenerator) applyBoundedNormal(mean, stdDev, min, max decimal.Decimal) decimal.Decimal {
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := g.boxMullerTransform(u1, u2)

	value := mean.Add(decimal.NewFromFloat(z).Mul(stdDev))
	if value.LessThan(min) {
		return min
	}
	if value.GreaterThan(max) {
		return max
	}
	return value
}

func (g *marketGenerator) boxMullerTransform(u1, u2 float64) float64 {
	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

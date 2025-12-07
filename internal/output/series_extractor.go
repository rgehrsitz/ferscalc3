package output

import (
	"sort"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/shopspring/decimal"
)

// ExtractMedianTimeSeries returns the year labels plus median (P50) net income and TSP balance series.
func ExtractMedianTimeSeries(result *calculation.FERSMonteCarloResult) (years []int, netMedian []float64, tspMedian []float64) {
	if result == nil || len(result.Simulations) == 0 {
		return nil, nil, nil
	}

	firstSim := result.Simulations[0]
	if len(firstSim.ScenarioResults) == 0 || len(firstSim.ScenarioResults[0].Projection) == 0 {
		return nil, nil, nil
	}

	projectionLength := len(firstSim.ScenarioResults[0].Projection)
	years = make([]int, projectionLength)
	netMedian = make([]float64, projectionLength)
	tspMedian = make([]float64, projectionLength)

	netBuckets := make([][]decimal.Decimal, projectionLength)
	tspBuckets := make([][]decimal.Decimal, projectionLength)

	for _, sim := range result.Simulations {
		if len(sim.ScenarioResults) == 0 {
			continue
		}
		scenario := sim.ScenarioResults[0]
		for idx, year := range scenario.Projection {
			if idx >= projectionLength {
				continue
			}
			if netBuckets[idx] == nil {
				netBuckets[idx] = make([]decimal.Decimal, 0, len(result.Simulations))
				tspBuckets[idx] = make([]decimal.Decimal, 0, len(result.Simulations))
				years[idx] = year.Date.Year()
			}
			netBuckets[idx] = append(netBuckets[idx], year.NetIncome)
			totalTSP := year.TSPBalancePersonA.Add(year.TSPBalancePersonB)
			tspBuckets[idx] = append(tspBuckets[idx], totalTSP)
		}
	}

	for i := 0; i < projectionLength; i++ {
		netMedian[i] = percentileFloatDecimals(netBuckets[i], 0.50)
		tspMedian[i] = percentileFloatDecimals(tspBuckets[i], 0.50)
	}

	return years, netMedian, tspMedian
}

func percentileFloatDecimals(values []decimal.Decimal, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	floatValues := make([]float64, len(values))
	for i, v := range values {
		floatValues[i] = decimalToFloat(v)
	}

	sort.Float64s(floatValues)

	if len(floatValues) == 1 {
		return floatValues[0]
	}

	position := percentile * float64(len(floatValues)-1)
	lowerIndex := int(position)
	upperIndex := lowerIndex + 1

	if upperIndex >= len(floatValues) {
		return floatValues[len(floatValues)-1]
	}

	if position == float64(lowerIndex) {
		return floatValues[lowerIndex]
	}

	weight := position - float64(lowerIndex)
	lower := floatValues[lowerIndex]
	upper := floatValues[upperIndex]

	return lower + (upper-lower)*weight
}

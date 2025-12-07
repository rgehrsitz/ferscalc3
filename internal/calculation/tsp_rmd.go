package calculation

import (
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// RMDCalculator calculates Required Minimum Distributions.
type RMDCalculator struct {
	BirthYear int
}

func NewRMDCalculator(birthYear int) *RMDCalculator {
	return &RMDCalculator{BirthYear: birthYear}
}

func (rmd *RMDCalculator) GetRMDAge() int {
	return dateutil.GetRMDAge(rmd.BirthYear)
}

func (rmd *RMDCalculator) CalculateRMD(traditionalBalance decimal.Decimal, age int) decimal.Decimal {
	if age < rmd.GetRMDAge() {
		return decimal.Zero
	}

	distributionPeriods := map[int]decimal.Decimal{
		72:  decimal.NewFromFloat(27.4),
		73:  decimal.NewFromFloat(26.5),
		74:  decimal.NewFromFloat(25.5),
		75:  decimal.NewFromFloat(24.6),
		76:  decimal.NewFromFloat(23.7),
		77:  decimal.NewFromFloat(22.9),
		78:  decimal.NewFromFloat(22.0),
		79:  decimal.NewFromFloat(21.1),
		80:  decimal.NewFromFloat(20.2),
		81:  decimal.NewFromFloat(19.4),
		82:  decimal.NewFromFloat(18.5),
		83:  decimal.NewFromFloat(17.7),
		84:  decimal.NewFromFloat(16.8),
		85:  decimal.NewFromFloat(16.0),
		86:  decimal.NewFromFloat(15.2),
		87:  decimal.NewFromFloat(14.4),
		88:  decimal.NewFromFloat(13.7),
		89:  decimal.NewFromFloat(12.9),
		90:  decimal.NewFromFloat(12.2),
		91:  decimal.NewFromFloat(11.5),
		92:  decimal.NewFromFloat(10.8),
		93:  decimal.NewFromFloat(10.1),
		94:  decimal.NewFromFloat(9.5),
		95:  decimal.NewFromFloat(8.9),
		96:  decimal.NewFromFloat(8.4),
		97:  decimal.NewFromFloat(7.8),
		98:  decimal.NewFromFloat(7.3),
		99:  decimal.NewFromFloat(6.8),
		100: decimal.NewFromFloat(6.4),
	}

	if period, ok := distributionPeriods[age]; ok {
		return traditionalBalance.Div(period)
	}
	if age > 100 {
		return traditionalBalance.Div(decimal.NewFromFloat(6.0))
	}
	return decimal.Zero
}

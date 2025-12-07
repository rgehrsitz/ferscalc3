package decimal

import (
	"github.com/shopspring/decimal"
)

// Money represents a monetary amount with proper financial precision
type Money struct {
	decimal.Decimal
}

// NewMoney creates a new Money instance from a float64
func NewMoney(value float64) Money {
	return Money{decimal.NewFromFloat(value)}
}

// NewMoneyFromDecimal creates a new Money instance from a decimal.Decimal
func NewMoneyFromDecimal(d decimal.Decimal) Money {
	return Money{d}
}

// NewMoneyFromString creates a new Money instance from a string
func NewMoneyFromString(value string) (Money, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Money{}, err
	}
	return Money{d}, nil
}

// Round rounds the money amount to cents using banker's rounding
func (m Money) Round() Money {
	return Money{m.Decimal.Round(2)}
}

// Annual converts a monthly amount to annual
func (m Money) Annual() Money {
	return Money{m.Decimal.Mul(decimal.NewFromInt(12))}
}

// Monthly converts an annual amount to monthly
func (m Money) Monthly() Money {
	return Money{m.Decimal.Div(decimal.NewFromInt(12))}
}

// ApplyTaxRate applies a tax rate to the money amount
func (m Money) ApplyTaxRate(rate decimal.Decimal) Money {
	tax := m.Decimal.Mul(rate)
	return Money{m.Decimal.Sub(tax)}
}

// Add adds another Money amount
func (m Money) Add(other Money) Money {
	return Money{m.Decimal.Add(other.Decimal)}
}

// Sub subtracts another Money amount
func (m Money) Sub(other Money) Money {
	return Money{m.Decimal.Sub(other.Decimal)}
}

// Mul multiplies by a decimal factor
func (m Money) Mul(factor decimal.Decimal) Money {
	return Money{m.Decimal.Mul(factor)}
}

// Div divides by a decimal factor
func (m Money) Div(factor decimal.Decimal) Money {
	return Money{m.Decimal.Div(factor)}
}

// GreaterThan checks if this amount is greater than another
func (m Money) GreaterThan(other Money) bool {
	return m.Decimal.GreaterThan(other.Decimal)
}

// GreaterThanOrEqual checks if this amount is greater than or equal to another
func (m Money) GreaterThanOrEqual(other Money) bool {
	return m.Decimal.GreaterThanOrEqual(other.Decimal)
}

// LessThan checks if this amount is less than another
func (m Money) LessThan(other Money) bool {
	return m.Decimal.LessThan(other.Decimal)
}

// LessThanOrEqual checks if this amount is less than or equal to another
func (m Money) LessThanOrEqual(other Money) bool {
	return m.Decimal.LessThanOrEqual(other.Decimal)
}

// Equal checks if this amount equals another
func (m Money) Equal(other Money) bool {
	return m.Decimal.Equal(other.Decimal)
}

// IsZero checks if the amount is zero
func (m Money) IsZero() bool {
	return m.Decimal.IsZero()
}

// IsPositive checks if the amount is positive
func (m Money) IsPositive() bool {
	return m.Decimal.IsPositive()
}

// IsNegative checks if the amount is negative
func (m Money) IsNegative() bool {
	return m.Decimal.IsNegative()
}

// Min returns the minimum of two Money amounts
func Min(a, b Money) Money {
	if a.LessThan(b) {
		return a
	}
	return b
}

// Max returns the maximum of two Money amounts
func Max(a, b Money) Money {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

// Zero returns a zero Money amount
func Zero() Money {
	return Money{decimal.Zero}
}

// String returns the string representation with proper formatting
func (m Money) String() string {
	return m.Decimal.StringFixed(2)
}

// Format formats the money amount with proper currency formatting
func (m Money) Format() string {
	return "$" + m.String()
} 
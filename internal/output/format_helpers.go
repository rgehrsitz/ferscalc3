package output

import "github.com/shopspring/decimal"

// FormatCurrency formats a decimal as USD currency with 2 decimals.
// Kept here so it can be reused by multiple formatters and unit tested in isolation.
func FormatCurrency(amount decimal.Decimal) string { return "$" + amount.StringFixed(2) }

// FormatPercentage formats a decimal as a percentage with 2 decimals.
func FormatPercentage(amount decimal.Decimal) string { return amount.StringFixed(2) + "%" }

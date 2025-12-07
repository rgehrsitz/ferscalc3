package output

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFormatCurrency(t *testing.T) {
	v := decimal.NewFromFloat(1234.567)
	got := FormatCurrency(v)
	want := "$1234.57"
	if got != want {
		t.Errorf("FormatCurrency(%v) = %q, want %q", v, got, want)
	}
}

func TestFormatPercentage(t *testing.T) {
	v := decimal.NewFromFloat(12.3456)
	got := FormatPercentage(v)
	want := "12.35%"
	if got != want {
		t.Errorf("FormatPercentage(%v) = %q, want %q", v, got, want)
	}
}

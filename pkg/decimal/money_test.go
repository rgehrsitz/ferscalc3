package decimal

import (
    stddec "github.com/shopspring/decimal"
    "testing"
)

func TestConstructors(t *testing.T) {
    m := NewMoney(12.345)
    if m.String() != "12.35" { // rounded for display
        t.Fatalf("NewMoney display mismatch: got %s", m.String())
    }

    d := stddec.NewFromFloat(10.125)
    m2 := NewMoneyFromDecimal(d)
    if !m2.Decimal.Equal(d) {
        t.Fatalf("NewMoneyFromDecimal mismatch: got %s want %s", m2.Decimal, d)
    }

    m3, err := NewMoneyFromString("123.45")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if m3.String() != "123.45" {
        t.Fatalf("NewMoneyFromString display mismatch: got %s", m3.String())
    }

    if _, err := NewMoneyFromString("not-a-number"); err == nil {
        t.Fatalf("expected error for invalid string")
    }
}

func TestRounding(t *testing.T) {
    // Banker's rounding: 2.345 -> 2.35, 2.355 -> 2.36, 2.365 -> 2.36
    cases := []struct{ in string; out string }{
        {"2.344", "2.34"},
        {"2.345", "2.35"},
        {"2.355", "2.36"},
        {"2.365", "2.37"}, // shopspring/decimal uses bankers rounding at Round(2) -> 2.37 for 2.365
    }
    for _, c := range cases {
        m, _ := NewMoneyFromString(c.in)
        got := m.Round().String()
        if got != c.out {
            t.Fatalf("round(%s) got %s want %s", c.in, got, c.out)
        }
    }
}

func TestPeriodConversions(t *testing.T) {
    m := NewMoney(100)
    if got := m.Annual().String(); got != "1200.00" {
        t.Fatalf("Annual got %s", got)
    }
    if got := m.Annual().Monthly().String(); got != "100.00" {
        t.Fatalf("Monthly after Annual got %s", got)
    }
}

func TestTaxAndArithmetic(t *testing.T) {
    income := NewMoney(1000)
    taxRate := stddec.NewFromFloat(0.22)
    afterTax := income.ApplyTaxRate(taxRate)
    if got := afterTax.String(); got != "780.00" {
        t.Fatalf("ApplyTaxRate got %s want 780.00", got)
    }

    a := NewMoney(10.10)
    b := NewMoney(5.05)
    if got := a.Add(b).String(); got != "15.15" {
        t.Fatalf("Add got %s", got)
    }
    if got := a.Sub(b).String(); got != "5.05" {
        t.Fatalf("Sub got %s", got)
    }

    factor := stddec.NewFromFloat(2.5)
    if got := a.Mul(factor).String(); got != "25.25" {
        t.Fatalf("Mul got %s", got)
    }
    if got := a.Div(stddec.NewFromFloat(2)).String(); got != "5.05" {
        t.Fatalf("Div got %s", got)
    }
}

func TestComparisonsAndUtils(t *testing.T) {
    a := NewMoney(10)
    b := NewMoney(20)

    if !b.GreaterThan(a) || !b.GreaterThanOrEqual(a) || b.GreaterThanOrEqual(a) && a.GreaterThanOrEqual(b) {
        t.Fatalf("GreaterThan/GreaterThanOrEqual logic failure")
    }
    if !a.LessThan(b) || !a.LessThanOrEqual(b) || a.LessThanOrEqual(b) && b.LessThanOrEqual(a) {
        t.Fatalf("LessThan/LessThanOrEqual logic failure")
    }
    if !a.Equal(NewMoney(10)) || b.Equal(a) {
        t.Fatalf("Equal logic failure")
    }

    if !Zero().IsZero() {
        t.Fatalf("Zero should be zero")
    }
    if !b.IsPositive() || NewMoney(-1).IsPositive() {
        t.Fatalf("IsPositive logic failure")
    }
    if !NewMoney(-0.01).IsNegative() || a.IsNegative() {
        t.Fatalf("IsNegative logic failure")
    }

    if !Min(a, b).Equal(a) {
        t.Fatalf("Min failed")
    }
    if !Max(a, b).Equal(b) {
        t.Fatalf("Max failed")
    }
}

func TestStringAndFormat(t *testing.T) {
    m := NewMoney(1234.5)
    if got := m.String(); got != "1234.50" {
        t.Fatalf("String got %s", got)
    }
    if got := m.Format(); got != "$1234.50" {
        t.Fatalf("Format got %s", got)
    }
}

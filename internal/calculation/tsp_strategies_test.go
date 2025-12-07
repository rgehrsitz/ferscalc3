package calculation

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestFourPercentRule(t *testing.T) {
	strategy := NewFourPercentRule(decimal.NewFromInt(100000), decimal.NewFromFloat(0.02))

	t.Run("first year withdrawal equals initial amount", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(100000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(4000)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})

	t.Run("second year increases with inflation", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(100000), 2, decimal.Zero, 66, false, decimal.Zero)
		want := decimal.NewFromInt(4000).Mul(decimal.NewFromFloat(1.02))
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})

	t.Run("RMD overrides smaller withdrawal", func(t *testing.T) {
		rmd := decimal.NewFromInt(6000)
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(100000), 5, decimal.Zero, 75, true, rmd)
		if !got.Equal(rmd) {
			t.Fatalf("expected RMD %s got %s", rmd, got)
		}
	})

	t.Run("withdrawal capped by balance", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(2000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(2000)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})
}

func TestNeedBasedWithdrawal(t *testing.T) {
	strategy := NewNeedBasedWithdrawal(decimal.NewFromInt(500))

	t.Run("withdraws target amount", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(100000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(6000)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})

	t.Run("negative target treated as zero", func(t *testing.T) {
		negStrategy := NewNeedBasedWithdrawal(decimal.NewFromInt(-300))
		got := negStrategy.CalculateWithdrawal(decimal.NewFromInt(100000), 1, decimal.Zero, 65, false, decimal.Zero)
		if !got.Equal(decimal.Zero) {
			t.Fatalf("expected zero withdrawal got %s", got)
		}
	})

	t.Run("RMD overrides smaller target", func(t *testing.T) {
		rmd := decimal.NewFromInt(10000)
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(100000), 1, decimal.Zero, 75, true, rmd)
		if !got.Equal(rmd) {
			t.Fatalf("expected RMD %s got %s", rmd, got)
		}
	})

	t.Run("withdrawal capped by balance", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(3000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(3000)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})
}

func TestVariablePercentageWithdrawal(t *testing.T) {
	strategy := NewVariablePercentageWithdrawal(decimal.NewFromFloat(0.05))

	t.Run("withdraws percentage of balance", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(200000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(10000)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}
	})

	t.Run("RMD overrides smaller percentage", func(t *testing.T) {
		rmd := decimal.NewFromInt(15000)
		got := strategy.CalculateWithdrawal(decimal.NewFromInt(200000), 1, decimal.Zero, 75, true, rmd)
		if !got.Equal(rmd) {
			t.Fatalf("expected RMD %s got %s", rmd, got)
		}
	})

	t.Run("withdrawal capped by balance", func(t *testing.T) {
		highRate := NewVariablePercentageWithdrawal(decimal.NewFromFloat(0.90))
		got := highRate.CalculateWithdrawal(decimal.NewFromInt(1000), 1, decimal.Zero, 65, false, decimal.Zero)
		want := decimal.NewFromInt(900)
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want, got)
		}

		got = highRate.CalculateWithdrawal(decimal.NewFromInt(1000), 1, decimal.Zero, 75, true, decimal.NewFromInt(2000))
		if !got.Equal(decimal.NewFromInt(1000)) {
			t.Fatalf("expected RMD clamped to balance got %s", got)
		}
	})
}

func TestFixedAnnuity(t *testing.T) {
	// Test with $1,000,000 premium, 5.5% annual payout, no COLA, 100% survivor benefit, 10 year guarantee
	premium := decimal.NewFromInt(1000000)
	payoutRate := decimal.NewFromFloat(0.055) // 5.5% annual payout
	colaRate := decimal.Zero                  // No COLA (fixed payment)
	survivorPercent := decimal.NewFromInt(1)  // 100% to survivor
	guaranteedYears := 10

	strategy := NewFixedAnnuity(premium, payoutRate, colaRate, survivorPercent, guaranteedYears)

	t.Run("first year payment equals premium times payout rate", func(t *testing.T) {
		got := strategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 65, false, decimal.Zero)
		want := premium.Mul(payoutRate) // $1,000,000 * 0.055 = $55,000
		if !got.Equal(want) {
			t.Fatalf("expected %s got %s", want.StringFixed(2), got.StringFixed(2))
		}
	})

	t.Run("monthly payment is annual divided by 12", func(t *testing.T) {
		monthlyPayment := strategy.GetMonthlyPayment(1)
		expectedMonthly := premium.Mul(payoutRate).Div(decimal.NewFromInt(12))
		if !monthlyPayment.Equal(expectedMonthly) {
			t.Fatalf("expected monthly %s got %s", expectedMonthly.StringFixed(2), monthlyPayment.StringFixed(2))
		}
	})

	t.Run("fixed payment stays same without COLA", func(t *testing.T) {
		year1 := strategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 65, false, decimal.Zero)
		year5 := strategy.CalculateWithdrawal(decimal.Zero, 5, decimal.Zero, 69, false, decimal.Zero)
		year10 := strategy.CalculateWithdrawal(decimal.Zero, 10, decimal.Zero, 74, false, decimal.Zero)

		if !year1.Equal(year5) || !year1.Equal(year10) {
			t.Fatalf("fixed annuity should have same payment every year: Y1=%s Y5=%s Y10=%s",
				year1.StringFixed(2), year5.StringFixed(2), year10.StringFixed(2))
		}
	})

	t.Run("annuity payments not subject to RMD", func(t *testing.T) {
		// Annuity payments satisfy RMD requirements automatically
		payment := strategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 75, true, decimal.NewFromInt(100000))
		expectedPayment := premium.Mul(payoutRate)
		if !payment.Equal(expectedPayment) {
			t.Fatalf("annuity payment should not be affected by RMD: expected %s got %s",
				expectedPayment.StringFixed(2), payment.StringFixed(2))
		}
	})

	t.Run("annuity with COLA increases payments", func(t *testing.T) {
		colaStrategy := NewFixedAnnuity(premium, payoutRate, decimal.NewFromFloat(0.02), survivorPercent, guaranteedYears)

		year1 := colaStrategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 65, false, decimal.Zero)
		year2 := colaStrategy.CalculateWithdrawal(decimal.Zero, 2, decimal.Zero, 66, false, decimal.Zero)
		year5 := colaStrategy.CalculateWithdrawal(decimal.Zero, 5, decimal.Zero, 69, false, decimal.Zero)

		// Year 2 should be 2% higher than year 1
		expectedYear2 := year1.Mul(decimal.NewFromFloat(1.02))
		if !year2.Sub(expectedYear2).Abs().LessThan(decimal.NewFromFloat(0.01)) {
			t.Fatalf("year 2 with 2%% COLA: expected %s got %s",
				expectedYear2.StringFixed(2), year2.StringFixed(2))
		}

		// Year 5 should be (1.02)^4 times year 1
		expectedYear5 := year1.Mul(decimal.NewFromFloat(1.02).Pow(decimal.NewFromInt(4)))
		if !year5.Sub(expectedYear5).Abs().LessThan(decimal.NewFromFloat(0.01)) {
			t.Fatalf("year 5 with 2%% COLA: expected %s got %s",
				expectedYear5.StringFixed(2), year5.StringFixed(2))
		}
	})

	t.Run("partial TSP to annuity scenario", func(t *testing.T) {
		// Convert only 50% of TSP to annuity
		halfPremium := decimal.NewFromInt(500000)
		partialStrategy := NewFixedAnnuity(halfPremium, payoutRate, decimal.Zero, survivorPercent, guaranteedYears)

		payment := partialStrategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 65, false, decimal.Zero)
		expectedPayment := halfPremium.Mul(payoutRate) // $500,000 * 0.055 = $27,500
		if !payment.Equal(expectedPayment) {
			t.Fatalf("partial annuity: expected %s got %s",
				expectedPayment.StringFixed(2), payment.StringFixed(2))
		}
	})

	t.Run("realistic PersonA scenario - full TSP conversion", func(t *testing.T) {
		// PersonA TSP balance: $1,966,168.86
		personATSP := decimal.NewFromFloat(1966168.86)
		// Assume 5.5% payout with 2% COLA
		realisticStrategy := NewFixedAnnuity(
			personATSP,
			decimal.NewFromFloat(0.055),
			decimal.NewFromFloat(0.02),
			decimal.NewFromInt(1),
			10,
		)

		year1Payment := realisticStrategy.CalculateWithdrawal(decimal.Zero, 1, decimal.Zero, 62, false, decimal.Zero)
		expectedYear1 := personATSP.Mul(decimal.NewFromFloat(0.055)) // $108,139.29 annually

		if !year1Payment.Sub(expectedYear1).Abs().LessThan(decimal.NewFromFloat(0.01)) {
			t.Fatalf("PersonA annuity year 1: expected %s got %s",
				expectedYear1.StringFixed(2), year1Payment.StringFixed(2))
		}

		// Check monthly payment
		monthlyPayment := realisticStrategy.GetMonthlyPayment(1)
		expectedMonthly := expectedYear1.Div(decimal.NewFromInt(12)) // $9,011.61/month
		if !monthlyPayment.Sub(expectedMonthly).Abs().LessThan(decimal.NewFromFloat(0.01)) {
			t.Fatalf("PersonA annuity monthly: expected %s got %s",
				expectedMonthly.StringFixed(2), monthlyPayment.StringFixed(2))
		}
	})
}

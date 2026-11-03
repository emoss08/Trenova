//go:build fuzz

package money

import (
	"testing"

	"github.com/shopspring/decimal"
)

func FuzzMinorUnitsRoundTripAndRoundHalfToEven(f *testing.F) {
	for _, seed := range []struct {
		minor int64
		extra int64
	}{{0, 0}, {1, 5}, {-1, 5}, {250, 50}, {999_999_999_999_999, 9}, {-12_345, 4999}} {
		f.Add(seed.minor, seed.extra)
	}
	f.Fuzz(func(t *testing.T, minor, extra int64) {
		minor %= 1_000_000_000_000_000
		if got := MinorUnits(DecimalFromMinor(minor)); got != minor {
			t.Fatalf("%d minor units came back as %d", minor, got)
		}

		extra %= 10_000
		if extra < 0 {
			extra = -extra
		}
		value := DecimalFromMinor(minor).Add(decimal.New(extra, -6))
		got := MinorUnits(value)
		if diff := value.Sub(DecimalFromMinor(got)).Abs(); diff.GreaterThan(decimal.New(5, -3)) {
			t.Fatalf("%s rounded to %d minor units, %s away", value, got, diff)
		}
		if got := FromDecimal("usd", value); got.CurrencyCode != "USD" ||
			got.Minor != MinorUnits(value) {
			t.Fatalf("%s as an amount is %+v", value, got)
		}
	})
}

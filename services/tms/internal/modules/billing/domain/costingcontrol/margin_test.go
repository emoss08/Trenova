package costingcontrol

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveTargetMarginPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target decimal.NullDecimal
		want   decimal.Decimal
	}{
		{name: "unset falls back", target: decimal.NullDecimal{}, want: DefaultTargetMarginPercent},
		{
			name:   "zero falls back",
			target: decimal.NewNullDecimal(decimal.Zero),
			want:   DefaultTargetMarginPercent,
		},
		{
			name:   "configured wins",
			target: decimal.NewNullDecimal(decimal.NewFromInt(18)),
			want:   decimal.NewFromInt(18),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.True(t, tt.want.Equal(EffectiveTargetMarginPercent(tt.target)))
		})
	}
}

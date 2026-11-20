package temperaturerange

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func degrees(value int16) *int16 {
	return &value
}

func TestLoad_IsTheRangeEveryLineSharesOrTheirSpan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ranges []Range
		want   Range
	}{
		{name: "no ranges", ranges: nil, want: Range{}},
		{
			name:   "one range",
			ranges: []Range{Of(degrees(33), degrees(38))},
			want:   Of(degrees(33), degrees(38)),
		},
		{
			name:   "overlapping ranges keep what all of them allow",
			ranges: []Range{Of(degrees(28), degrees(38)), Of(degrees(32), degrees(40))},
			want:   Of(degrees(32), degrees(38)),
		},
		{
			name:   "a one-sided bound narrows only its side",
			ranges: []Range{Of(degrees(28), degrees(38)), Of(nil, degrees(34))},
			want:   Of(degrees(28), degrees(34)),
		},
		{
			name:   "frozen and chilled share nothing, so the load spans both",
			ranges: []Range{Of(degrees(-10), degrees(0)), Of(degrees(34), degrees(38))},
			want:   Of(degrees(-10), degrees(38)),
		},
		{
			name:   "one-sided ranges that share nothing span every bound given",
			ranges: []Range{Of(nil, degrees(0)), Of(degrees(34), nil)},
			want:   Of(degrees(0), degrees(34)),
		},
		{
			name:   "unset ranges are skipped",
			ranges: []Range{{}, Of(degrees(33), nil)},
			want:   Of(degrees(33), nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Load(tt.ranges))
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  Range
		fields []string
	}{
		{name: "unset", value: Range{}},
		{name: "a range", value: Of(degrees(-10), degrees(0))},
		{name: "one bound", value: Of(nil, degrees(-5))},
		{name: "inverted", value: Of(degrees(40), degrees(34)), fields: []string{"temperatureMax"}},
		{name: "too cold", value: Of(degrees(-101), nil), fields: []string{"temperatureMin"}},
		{name: "too warm", value: Of(nil, degrees(151)), fields: []string{"temperatureMax"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tt.value.Validate(multiErr)
			require.Equal(t, len(tt.fields) > 0, multiErr.HasErrors())
			for _, field := range tt.fields {
				assert.True(t, multiErr.HasErrorAt(field), field)
			}
		})
	}
}

func TestValidateWithin_ChecksOnlyTheBoundsALineSets(t *testing.T) {
	t.Parallel()

	low, high := -20, 10
	catalog := FromDegrees(&low, &high)
	tests := []struct {
		name   string
		line   Range
		fields []string
	}{
		{name: "inside", line: Of(degrees(-10), degrees(0))},
		{name: "colder", line: Of(degrees(-30), degrees(0)), fields: []string{"temperatureMin"}},
		{name: "warmer", line: Of(degrees(-10), degrees(20)), fields: []string{"temperatureMax"}},
		{name: "one-sided inside", line: Of(nil, degrees(-5))},
		{name: "unset", line: Range{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			multiErr := errortypes.NewMultiError()
			tt.line.ValidateWithin(catalog, "Ice cream", multiErr)
			require.Equal(t, len(tt.fields) > 0, multiErr.HasErrors())
			for _, field := range tt.fields {
				assert.True(t, multiErr.HasErrorAt(field), field)
			}
		})
	}
}

func TestFromDegrees_DropsBoundsNoStopCouldHold(t *testing.T) {
	t.Parallel()

	cold, absurd := -40, 400
	got := FromDegrees(&cold, &absurd)
	assert.Equal(t, degrees(-40), got.Min)
	assert.Nil(t, got.Max)
}

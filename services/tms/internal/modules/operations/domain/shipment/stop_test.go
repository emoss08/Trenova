package shipment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStop_EffectiveScheduledCutoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		stop *Stop
		want int64
	}{
		{
			name: "nil stop returns zero",
			stop: nil,
			want: 0,
		},
		{
			name: "uses scheduled window end when present",
			stop: &Stop{
				ScheduledWindowStart: 100,
				ScheduledWindowEnd:   new(int64(200)),
			},
			want: 200,
		},
		{
			name: "falls back to scheduled window start",
			stop: &Stop{
				ScheduledWindowStart: 100,
				ScheduledWindowEnd:   nil,
			},
			want: 100,
		},
		{
			name: "missing schedule returns zero",
			stop: &Stop{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.stop.EffectiveScheduledCutoff())
			assert.Equal(t, tt.want, tt.stop.EffectiveScheduledWindowEnd())
		})
	}
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}

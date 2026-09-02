package resolver_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type timedStop struct {
	ScheduledWindowStart int64
	ScheduledWindowEnd   *int64
	ActualArrival        *int64
	ActualDeparture      *int64
}

type timedMove struct {
	Stops []timedStop
}

type timedShipment struct {
	Moves []timedMove
}

func int64Ptr(v int64) *int64 { return &v }

func TestComputeTotalHours(t *testing.T) {
	t.Parallel()

	r := resolver.NewResolver()
	resolver.RegisterDefaultComputed(r)

	compute, ok := r.GetComputed("computeTotalHours")
	require.True(t, ok)

	tests := []struct {
		name   string
		entity *timedShipment
		want   float64
	}{
		{
			name: "actual arrival to actual departure",
			entity: &timedShipment{
				Moves: []timedMove{{
					Stops: []timedStop{
						{
							ScheduledWindowStart: 900,
							ActualArrival:        int64Ptr(1000),
							ActualDeparture:      int64Ptr(4600),
						},
						{
							ScheduledWindowStart: 5000,
							ActualArrival:        int64Ptr(6000),
							ActualDeparture:      int64Ptr(8200),
						},
					},
				}},
			},
			want: 2.0,
		},
		{
			name: "scheduled windows when no actuals recorded",
			entity: &timedShipment{
				Moves: []timedMove{{
					Stops: []timedStop{
						{ScheduledWindowStart: 1000, ScheduledWindowEnd: int64Ptr(2000)},
						{ScheduledWindowStart: 7000, ScheduledWindowEnd: int64Ptr(8200)},
					},
				}},
			},
			want: 2.0,
		},
		{
			name: "mixed actuals and scheduled windows",
			entity: &timedShipment{
				Moves: []timedMove{{
					Stops: []timedStop{
						{
							ScheduledWindowStart: 500,
							ActualArrival:        int64Ptr(1000),
						},
						{ScheduledWindowStart: 4600, ScheduledWindowEnd: int64Ptr(8200)},
					},
				}},
			},
			want: 2.0,
		},
		{
			name:   "no moves",
			entity: &timedShipment{},
			want:   0.0,
		},
		{
			name: "single instantaneous stop",
			entity: &timedShipment{
				Moves: []timedMove{{
					Stops: []timedStop{
						{ScheduledWindowStart: 1000},
					},
				}},
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := compute(tt.entity)
			require.NoError(t, err)
			assert.InDelta(t, tt.want, got, 0.0001)
		})
	}
}

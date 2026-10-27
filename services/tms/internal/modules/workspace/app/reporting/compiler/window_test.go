package compiler

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/stretchr/testify/assert"
)

func TestParameterWindowIsTheRangeTheRunFilteredOn(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	ranAt := time.Date(2026, 10, 27, 15, 0, 0, 0, time.UTC)
	day := func(month time.Month, d int) int64 {
		return time.Date(2026, month, d, 0, 0, 0, 0, chicago).Unix()
	}

	tests := []struct {
		name  string
		op    dbtype.Operator
		value any
		start int64
		end   int64
		ok    bool
	}{
		{"last n days ends after today", dbtype.OpLastNDays, 30.0, day(9, 27), day(10, 28), true},
		{"next n days starts today", dbtype.OpNextNDays, int64(7), day(10, 27), day(11, 4), true},
		{
			"a date range includes its last day",
			dbtype.OpDateRange,
			[]any{"2026-10-01", "2026-10-31"},
			day(10, 1),
			day(11, 1),
			true,
		},
		{"a day count that is not whole", dbtype.OpLastNDays, 1.5, 0, 0, false},
		{"an operator without a window", dbtype.OpIn, []any{"Posted"}, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start, end, ok := ParameterWindow(tt.op, tt.value, chicago, ranAt)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.start, start)
			assert.Equal(t, tt.end, end)
		})
	}
}

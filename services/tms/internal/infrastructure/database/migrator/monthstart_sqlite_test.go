package migrator_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// The fleet safety trend buckets by UTC month through the dialect helper. The
// Postgres spelling is obvious; the SQLite one goes through strftime and back,
// so it is worth proving it lands on the first instant of the right month.
func TestSQLiteMonthStartEpochBucketsByUTCMonth(t *testing.T) {
	ctx := t.Context()
	db := newSQLiteDB(t)

	cases := []struct {
		at   time.Time
		want time.Time
	}{
		{
			at:   time.Date(2026, time.September, 4, 13, 45, 0, 0, time.UTC),
			want: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			at:   time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
		// The last second of a month must not spill into the next one.
		{
			at:   time.Date(2026, time.February, 28, 23, 59, 59, 0, time.UTC),
			want: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	expr := dbdialect.SQLite.MonthStartEpoch("?")
	for _, tc := range cases {
		var got int64
		require.NoError(t,
			db.NewRaw("SELECT "+expr, tc.at.Unix()).Scan(ctx, &got),
			"month start for %s", tc.at,
		)
		require.Equalf(t, tc.want.Unix(), got, "month start for %s", tc.at)
	}
}

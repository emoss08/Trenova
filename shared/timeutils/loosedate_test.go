package timeutils_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLooseDateReadsTheLayoutsFactorsAndCardFeedsUse(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	want := time.Date(2026, time.October, 7, 0, 0, 0, 0, time.UTC).Unix()

	for _, raw := range []string{"2026-10-07", "10/07/2026", "10/7/2026", "07-Oct-2026", "Oct 7, 2026", "20261007", "46302"} {
		got, parseErr := timeutils.ParseLooseDate(raw, time.UTC)
		require.NoError(t, parseErr, raw)
		assert.Equal(t, want, got, raw)
	}

	local, err := timeutils.ParseLooseDate("10/07/2026", chicago)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.October, 7, 0, 0, 0, 0, chicago).Unix(), local)
}

func TestParseLooseDateNamesWhyItCannotReadADate(t *testing.T) {
	t.Parallel()

	_, err := timeutils.ParseLooseDate("  ", time.UTC)
	require.ErrorIs(t, err, timeutils.ErrDateMissing)

	_, err = timeutils.ParseLooseDate("next tuesday", time.UTC)
	require.ErrorIs(t, err, timeutils.ErrDateUnrecognized)
}

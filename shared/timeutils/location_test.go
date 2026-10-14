package timeutils_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupLocationResolvesAndReusesAZone(t *testing.T) {
	t.Parallel()

	first, err := timeutils.LookupLocation("America/Chicago")
	require.NoError(t, err)
	second, err := timeutils.LookupLocation("America/Chicago")
	require.NoError(t, err)

	assert.Equal(t, "America/Chicago", first.String())
	assert.Same(t, first, second, "a zone is read from disk once")
}

func TestLookupLocationTreatsNoNameAsUTC(t *testing.T) {
	t.Parallel()

	location, err := timeutils.LookupLocation("")

	require.NoError(t, err)
	assert.Equal(t, time.UTC, location)
}

func TestLookupLocationRefusesUnknownAndOverlongNames(t *testing.T) {
	t.Parallel()

	_, err := timeutils.LookupLocation("Mars/Olympus_Mons")
	require.Error(t, err)

	_, err = timeutils.LookupLocation(strings.Repeat("A", timeutils.MaxLocationNameLength+1))
	require.ErrorIs(t, err, timeutils.ErrLocationNameTooLong)
}

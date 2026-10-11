package agenttoolstest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func LocalInstant(t *testing.T, zone string, year, month, day, hour, minute int) int64 {
	t.Helper()

	loc, err := time.LoadLocation(zone)
	require.NoError(t, err)

	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc).Unix()
}

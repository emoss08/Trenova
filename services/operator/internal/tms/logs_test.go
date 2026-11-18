package tms_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

type fixedToken string

func (f fixedToken) Token(context.Context) (string, error)   { return string(f), nil }
func (f fixedToken) Refresh(context.Context) (string, error) { return string(f), nil }

func logClient(t *testing.T) (*tmstest.Server, *tms.Client) {
	t.Helper()
	srv := tmstest.New(t, "token")
	c, err := tms.New(tms.Params{URL: srv.URL, Tokens: fixedToken("token")})
	require.NoError(t, err)
	return srv, c
}

func TestALogPullIsAllowedOnlyUnderALiveGrant(t *testing.T) {
	t.Parallel()
	srv, c := logClient(t)
	srv.Grant("sam@trenova.app", true)

	require.NoError(t, c.AuthorizeLogPull(t.Context(), "sam@trenova.app", tms.LogPullBundle))
	require.ErrorIs(t, c.AuthorizeLogPull(t.Context(), "eve@trenova.app", tms.LogPullLines), tms.ErrNoGrant)

	srv.FailLogPullsNext(1)
	err := c.AuthorizeLogPull(t.Context(), "sam@trenova.app", tms.LogPullLines)
	require.Error(t, err)
	assert.NotErrorIs(t, err, tms.ErrNoGrant, "an unreachable TMS is not a refusal")

	assert.Equal(t, []tmstest.LogPull{
		{Email: "sam@trenova.app", Kind: tms.LogPullBundle},
		{Email: "eve@trenova.app", Kind: tms.LogPullLines},
	}, srv.LogPulls())
}

func TestLogRequestsAreListedAndUpdated(t *testing.T) {
	t.Parallel()
	srv, c := logClient(t)
	since := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	srv.AddLogRequest(tms.LogRequest{
		ID: "lrq_01", Since: since, Until: since.Add(time.Hour),
		RequestedBy: tms.LogRequester{Email: "dana@acme.example", Name: "Dana"},
	})

	list, err := c.LogRequests(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "lrq_01", list[0].ID)
	assert.True(t, since.Equal(list[0].Since))

	require.NoError(t, c.UpdateLogRequest(t.Context(), "lrq_01", tms.LogRequestUpdate{
		Status: tms.LogRequestSent, BundleID: "8a0b4f0e-8d5c-4c4f-9a43-2f1c3c7e2b11",
	}))
	list, err = c.LogRequests(t.Context())
	require.NoError(t, err)
	assert.Empty(t, list, "a sent request is no longer pending")
	require.Error(t, c.UpdateLogRequest(t.Context(), "lrq_99", tms.LogRequestUpdate{Status: tms.LogRequestCollecting}))
}

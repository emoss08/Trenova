package networkpulseservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRepo struct {
	counts    *repositories.NetworkPulseCounts
	err       error
	calls     int
	since     int64
	laneLimit int
}

func (r *stubRepo) GetNetworkPulse(
	_ context.Context,
	since int64,
	laneLimit int,
) (*repositories.NetworkPulseCounts, error) {
	r.calls++
	r.since = since
	r.laneLimit = laneLimit
	if r.err != nil {
		return nil, r.err
	}
	return r.counts, nil
}

func newService(t *testing.T, enabled bool, repo *stubRepo) *Service {
	t.Helper()

	cfg := &config.Config{}
	cfg.System.NetworkPulse.Enabled = enabled

	return New(Params{
		Logger: zap.NewNop(),
		Config: cfg,
		Repo:   repo,
	})
}

func TestGetNetworkPulse_DisabledByDefault(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{}}
	svc := newService(t, false, repo)

	pulse, err := svc.GetNetworkPulse(t.Context())

	require.ErrorIs(t, err, ErrDisabled)
	assert.Nil(t, pulse)
	assert.Zero(t, repo.calls, "a disabled pulse must not touch the database")
}

func TestGetNetworkPulse_ReportsPercentToOneDecimal(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{
		LoadsInMotion: 12480,
		OnTimeCount:   3374,
		OnTimeTotal:   3421,
	}}
	svc := newService(t, true, repo)

	pulse, err := svc.GetNetworkPulse(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 12480, pulse.LoadsInMotion)
	assert.InDelta(t, 98.6, pulse.OnTimePercent, 0.0001)
	assert.Equal(t, 3421, pulse.SampleSize)
	assert.Equal(t, WindowDays, pulse.WindowDays)
}

func TestGetNetworkPulse_ScoresOverTheReportingWindow(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{}}
	svc := newService(t, true, repo)

	before := time.Now().Unix()
	_, err := svc.GetNetworkPulse(t.Context())
	require.NoError(t, err)

	expected := before - int64(WindowDays)*secondsPerDay
	assert.InDelta(t, expected, repo.since, 5)
}

// An empty window must not read as a perfect miss: the percentage is meaningless and
// SampleSize is what tells the caller so.
func TestGetNetworkPulse_EmptyWindowReportsZeroSample(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{LoadsInMotion: 4}}
	svc := newService(t, true, repo)

	pulse, err := svc.GetNetworkPulse(t.Context())

	require.NoError(t, err)
	assert.Zero(t, pulse.SampleSize)
	assert.Zero(t, pulse.OnTimePercent)
}

// The endpoint is unauthenticated, so repeated calls must not each become a database
// aggregate.
func TestGetNetworkPulse_CachesWithinTheTTL(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{LoadsInMotion: 7}}
	svc := newService(t, true, repo)

	for range 5 {
		_, err := svc.GetNetworkPulse(t.Context())
		require.NoError(t, err)
	}

	assert.Equal(t, 1, repo.calls)
}

func TestGetNetworkPulse_RecomputesOnceTheTTLExpires(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{LoadsInMotion: 7}}
	svc := newService(t, true, repo)
	svc.cfg.System.NetworkPulse.CacheTTL = time.Nanosecond

	_, err := svc.GetNetworkPulse(t.Context())
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	_, err = svc.GetNetworkPulse(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 2, repo.calls)
}

func TestGetNetworkPulse_PropagatesRepositoryFailure(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	repo := &stubRepo{err: sentinel}
	svc := newService(t, true, repo)

	_, err := svc.GetNetworkPulse(t.Context())

	require.ErrorIs(t, err, sentinel)
}

func TestNetworkPulseCacheTTL_FallsBackToTheDefault(t *testing.T) {
	t.Parallel()

	assert.Equal(t, config.DefaultNetworkPulseCacheTTL, config.NetworkPulseConfig{}.GetCacheTTL())
	assert.Equal(
		t,
		30*time.Second,
		config.NetworkPulseConfig{CacheTTL: 30 * time.Second}.GetCacheTTL(),
	)
}

func TestGetNetworkPulse_MapsLanesToStatePairs(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{
		Lanes: []repositories.NetworkPulseLane{
			{OriginState: "CA", DestinationState: "AZ", Status: "InTransit", Count: 12},
			{OriginState: "TX", DestinationState: "GA", Status: "Assigned", Count: 4},
		},
	}}
	svc := newService(t, true, repo)

	pulse, err := svc.GetNetworkPulse(t.Context())

	require.NoError(t, err)
	require.Len(t, pulse.Lanes, 2)
	assert.Equal(t, "CA", pulse.Lanes[0].From)
	assert.Equal(t, "AZ", pulse.Lanes[0].To)
	assert.Equal(t, "InTransit", pulse.Lanes[0].Status)
	assert.Equal(t, 12, pulse.Lanes[0].Count)
	assert.Equal(t, LaneLimit, repo.laneLimit)
}

// An instance with no active shipments must serialize an empty array, not null: the
// panel treats "no lanes" as "render nothing" and should not have to defend against a
// second spelling of it.
func TestGetNetworkPulse_EmptyLanesAreNotNil(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{counts: &repositories.NetworkPulseCounts{}}
	svc := newService(t, true, repo)

	pulse, err := svc.GetNetworkPulse(t.Context())

	require.NoError(t, err)
	assert.NotNil(t, pulse.Lanes)
	assert.Empty(t, pulse.Lanes)
}

package usage_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
)

var hour = time.Date(2026, 11, 5, 13, 0, 0, 0, time.UTC)

func allowance(meter, app string, unit instanceapi.SpecAllowanceUnit) instanceapi.SpecAllowance {
	a := instanceapi.SpecAllowance{Meter: meter, Unit: unit, Quantity: 100, HardCap: true}
	if app != "" {
		a.AppMeter = &app
	}
	return a
}

var allowances = []instanceapi.SpecAllowance{
	allowance("shipments", "shipments.total", instanceapi.SpecAllowanceUnitCount),
	allowance("ai_credits", "ai.spend_cents", instanceapi.SpecAllowanceUnitCents),
	allowance("storage", "documents.storage_bytes", instanceapi.SpecAllowanceUnitBytes),
	allowance("users", "", instanceapi.SpecAllowanceUnitCount),
}

func TestMapsReadingsToCatalogMeters(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		allowances []instanceapi.SpecAllowance
		in         []tms.Reading
		want       []instanceapi.UsageReading
		dropped    int
	}{
		{name: "nothing", allowances: allowances, want: []instanceapi.UsageReading{}},
		{
			name:       "mapped",
			allowances: allowances,
			in: []tms.Reading{
				{Meter: "shipments.total", Hour: hour, Quantity: 12},
				{Meter: "documents.storage_bytes", Hour: hour.In(time.FixedZone("x", 3600)), Quantity: 1 << 30},
			},
			want: []instanceapi.UsageReading{
				{Meter: "shipments", Hour: hour, Quantity: 12},
				{Meter: "storage", Hour: hour, Quantity: 1 << 30},
			},
		},
		{
			name:       "unmapped and negative dropped",
			allowances: allowances,
			in: []tms.Reading{
				{Meter: "document_intelligence.extractions", Hour: hour, Quantity: 3},
				{Meter: "ai.spend_cents", Hour: hour, Quantity: -1},
				{Meter: "ai.spend_cents", Hour: hour.Add(-time.Hour), Quantity: 0},
			},
			want:    []instanceapi.UsageReading{{Meter: "ai_credits", Hour: hour.Add(-time.Hour)}},
			dropped: 2,
		},
		{
			name: "the first allowance for an app meter wins",
			allowances: []instanceapi.SpecAllowance{
				allowance("shipments", "shipments.total", instanceapi.SpecAllowanceUnitCount),
				allowance("loads", "shipments.total", instanceapi.SpecAllowanceUnitCount),
			},
			in:   []tms.Reading{{Meter: "shipments.total", Hour: hour, Quantity: 1}},
			want: []instanceapi.UsageReading{{Meter: "shipments", Hour: hour, Quantity: 1}},
		},
		{
			name:       "no allowances",
			allowances: nil,
			in:         []tms.Reading{{Meter: "shipments.total", Hour: hour, Quantity: 1}},
			want:       []instanceapi.UsageReading{},
			dropped:    1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, dropped := usage.MapReadings(tc.allowances, tc.in)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.dropped, dropped)
		})
	}
}

func TestMapsTheAckToTMSMeters(t *testing.T) {
	t.Parallel()
	window := instanceapi.UsageWindow{Start: hour.Add(-13 * time.Hour), End: hour.Add(11 * time.Hour)}
	cases := []struct {
		name string
		ack  instanceapi.UsageAck
		want tms.State
	}{
		{
			name: "no window: everything stops",
			ack:  instanceapi.UsageAck{Meters: []instanceapi.MeterUsage{}},
			want: tms.State{Meters: []tms.Meter{}},
		},
		{
			name: "mapped back",
			ack: instanceapi.UsageAck{
				Window:  &window,
				Overage: true,
				Meters: []instanceapi.MeterUsage{
					{
						Meter: "shipments", Unit: "count",
						Quantity: 120, Allowance: 100, HardCap: true, Stopped: true,
					},
					{Meter: "storage", Unit: "bytes", Quantity: 5, Allowance: 10},
					{Meter: "users", Unit: "count", Quantity: 3, Allowance: 5},
					{Meter: "unknown", Unit: "count", Quantity: 1},
				},
			},
			want: tms.State{
				Window:  &tms.Window{Start: window.Start, End: window.End},
				Overage: true,
				Meters: []tms.Meter{
					{
						Meter: "shipments.total", Unit: "count",
						Quantity: 120, Allowance: 100, HardCap: true, Stopped: true,
					},
					{Meter: "documents.storage_bytes", Unit: "bytes", Quantity: 5, Allowance: 10},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, usage.MapState(allowances, tc.ack))
		})
	}
}

func TestChunks(t *testing.T) {
	t.Parallel()
	readings := func(n int) []instanceapi.UsageReading {
		out := make([]instanceapi.UsageReading, n)
		for i := range out {
			out[i] = instanceapi.UsageReading{Meter: "shipments", Hour: hour, Quantity: int64(i)}
		}
		return out
	}
	cases := []struct {
		n    int
		size int
		want []int
	}{
		{n: 0, size: 1000, want: []int{0}},
		{n: 1, size: 1000, want: []int{1}},
		{n: 1000, size: 1000, want: []int{1000}},
		{n: 1001, size: 1000, want: []int{1000, 1}},
		{n: 2500, size: 1000, want: []int{1000, 1000, 500}},
		{n: 3, size: 0, want: []int{1, 1, 1}},
	}
	for _, tc := range cases {
		in := readings(tc.n)
		got := usage.Chunks(in, tc.size)
		sizes := make([]int, len(got))
		var flat []instanceapi.UsageReading
		for i, c := range got {
			sizes[i] = len(c)
			flat = append(flat, c...)
		}
		require.Equal(t, tc.want, sizes, "%d by %d", tc.n, tc.size)
		if tc.n > 0 {
			require.Equal(t, in, flat, "in order, nothing lost")
		}
	}
}

type fakeTMS struct {
	readings []tms.Reading
	since    []time.Time
	states   []tms.State
	readErr  error
	putErr   error
}

func (f *fakeTMS) Readings(_ context.Context, since time.Time) ([]tms.Reading, error) {
	f.since = append(f.since, since)
	return f.readings, f.readErr
}

func (f *fakeTMS) PutState(_ context.Context, state tms.State) error {
	f.states = append(f.states, state)
	return f.putErr
}

func controlPlane(t *testing.T) (*controlplanetest.Server, *controlplane.Client) {
	t.Helper()
	spec := controlplanetest.Spec(uuid.New())
	spec.Allowances = allowances
	cp := controlplanetest.New(t, spec)
	bundle := cp.Bundle(time.Hour)
	ident, err := bundle.Identity()
	require.NoError(t, err)
	tr := &credential.Transport{}
	require.NoError(t, tr.Use(&ident))
	c, err := controlplane.New(cp.URL, &http.Client{Transport: tr, Timeout: 10 * time.Second})
	require.NoError(t, err)
	return cp, c
}

func TestRelaysUsageBothWays(t *testing.T) {
	t.Parallel()
	cp, c := controlPlane(t)
	now := time.Now().UTC().Truncate(time.Hour).Add(42 * time.Minute)
	current := now.Truncate(time.Hour)
	tm := &fakeTMS{}
	for h := current.Add(-2 * time.Hour); !h.After(current); h = h.Add(time.Hour) {
		tm.readings = append(tm.readings,
			tms.Reading{Meter: "shipments.total", Hour: h, Quantity: 40},
			tms.Reading{Meter: "document_intelligence.extractions", Hour: h, Quantity: 2},
		)
	}
	tm.readings = append(
		tm.readings,
		tms.Reading{Meter: "documents.storage_bytes", Hour: current, Quantity: 7},
	)
	relay := usage.New(usage.Params{ControlPlane: c, TMS: tm, Now: func() time.Time { return now }})

	res, err := relay.Run(t.Context(), allowances)
	require.NoError(t, err)
	require.Equal(t, usage.Result{Readings: 4, Dropped: 3, Meters: 3}, res)
	require.Equal(t, []time.Time{current.Add(-2 * time.Hour)}, tm.since)

	reports := cp.UsageReports()
	require.Len(t, reports, 1)
	require.Len(t, reports[0].Readings, 4)
	for _, rd := range reports[0].Readings {
		require.Contains(t, []string{"shipments", "storage"}, rd.Meter)
	}

	require.Len(t, tm.states, 1)
	state := tm.states[0]
	require.NotNil(t, state.Window)
	require.True(t, state.Overage)
	byMeter := map[string]tms.Meter{}
	for _, m := range state.Meters {
		byMeter[m.Meter] = m
	}
	require.Len(t, byMeter, 3, "users has no app meter; the other three map back")
	shipments := byMeter["shipments.total"]
	require.Equal(t, "count", shipments.Unit)
	require.GreaterOrEqual(t, shipments.Quantity, int64(40))
	require.Equal(t, int64(100), shipments.Allowance)
	require.Equal(t, int64(7), byMeter["documents.storage_bytes"].Quantity)
}

func TestChunksLargeReports(t *testing.T) {
	t.Parallel()
	cp, c := controlPlane(t)
	now := time.Now().UTC()
	tm := &fakeTMS{}
	for i := range 2500 {
		tm.readings = append(tm.readings, tms.Reading{
			Meter:    "shipments.total",
			Hour:     now.Truncate(time.Hour).Add(-time.Duration(i) * time.Hour),
			Quantity: 1,
		})
	}
	relay := usage.New(usage.Params{ControlPlane: c, TMS: tm, Now: func() time.Time { return now }})
	_, err := relay.Run(t.Context(), allowances)
	require.NoError(t, err)
	reports := cp.UsageReports()
	require.Len(t, reports, 3)
	require.Len(t, reports[0].Readings, 1000)
	require.Len(t, reports[2].Readings, 500)
	require.Len(t, tm.states, 1, "the state is sent once, from the last answer")
}

func TestStopsAtTheFirstFailure(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	cases := []struct {
		name      string
		tms       *fakeTMS
		limit     bool
		reports   int
		states    int
		retryable bool
	}{
		{name: "reading fails", tms: &fakeTMS{readErr: boom}},
		{name: "rate limited", tms: &fakeTMS{}, limit: true, retryable: true},
		{name: "state fails", tms: &fakeTMS{putErr: boom}, reports: 1, states: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cp, c := controlPlane(t)
			if tc.limit {
				cp.RateLimitNext(1)
			}
			_, err := usage.New(usage.Params{ControlPlane: c, TMS: tc.tms}).Run(t.Context(), allowances)
			require.Error(t, err)
			var later *controlplane.RetryLaterError
			require.Equal(t, tc.retryable, errors.As(err, &later))
			require.Len(t, cp.UsageReports(), tc.reports)
			require.Len(t, tc.tms.states, tc.states)
		})
	}
}

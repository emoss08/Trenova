package controlplane_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func telemetrySample(at time.Time) instanceapi.TelemetrySample {
	cpu := int64(250)
	return instanceapi.TelemetrySample{
		At:    at,
		Parts: []instanceapi.PartUsage{{Name: "api", Ready: true, CpuMillicores: &cpu}},
	}
}

func TestReportTelemetryIsRecordedByMinute(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	minute := time.Now().UTC().Truncate(time.Minute)

	require.NoError(t, c.ReportTelemetry(t.Context(), instanceapi.TelemetryReport{
		Samples: []instanceapi.TelemetrySample{
			telemetrySample(minute.Add(-time.Minute)),
			telemetrySample(minute),
		},
	}))

	held := cp.TelemetrySamples()
	require.Len(t, held, 2)
	require.Equal(t, int64(250), *held[minute].Parts[0].CpuMillicores)
}

func TestReportTelemetryFailures(t *testing.T) {
	t.Parallel()
	minute := time.Now().UTC().Truncate(time.Minute)
	report := instanceapi.TelemetryReport{Samples: []instanceapi.TelemetrySample{telemetrySample(minute)}}

	t.Run("refused", func(t *testing.T) {
		t.Parallel()
		cp, c := client(t)
		cp.RefuseTelemetryNext(1)
		var pe *controlplane.ProblemError
		require.True(t, errors.As(c.ReportTelemetry(t.Context(), report), &pe))
		require.Equal(t, http.StatusBadRequest, pe.Status)
		require.Empty(t, cp.TelemetrySamples())
	})
	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		cp, c := client(t)
		cp.FailTelemetryNext(1)
		var pe *controlplane.ProblemError
		require.True(t, errors.As(c.ReportTelemetry(t.Context(), report), &pe))
		require.Equal(t, http.StatusInternalServerError, pe.Status)
		require.NoError(t, c.ReportTelemetry(t.Context(), report), "the next one is accepted")
	})
	t.Run("rate limited", func(t *testing.T) {
		t.Parallel()
		cp, c := client(t)
		cp.RateLimitNext(1)
		var later *controlplane.RetryLaterError
		require.True(t, errors.As(c.ReportTelemetry(t.Context(), report), &later))
		require.Equal(t, time.Second, later.RetryAfter)
	})
	t.Run("a minute twice", func(t *testing.T) {
		t.Parallel()
		_, c := client(t)
		var pe *controlplane.ProblemError
		require.True(t, errors.As(c.ReportTelemetry(t.Context(), instanceapi.TelemetryReport{
			Samples: []instanceapi.TelemetrySample{telemetrySample(minute), telemetrySample(minute)},
		}), &pe))
		require.Equal(t, http.StatusBadRequest, pe.Status)
	})
	t.Run("revoked", func(t *testing.T) {
		t.Parallel()
		cp, c := client(t)
		cp.RevokeAll()
		require.ErrorIs(t, c.ReportTelemetry(t.Context(), report), controlplane.ErrUnauthenticated)
	})
}

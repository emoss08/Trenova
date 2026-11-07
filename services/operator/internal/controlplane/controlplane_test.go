package controlplane_test

import (
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
)

func client(t *testing.T) (*controlplanetest.Server, *controlplane.Client) {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
	bundle := cp.Bundle(time.Hour)
	ident, err := bundle.Identity()
	require.NoError(t, err)
	tr := &credential.Transport{}
	require.NoError(t, tr.Use(&ident))
	c, err := controlplane.New(cp.URL, &http.Client{Transport: tr})
	require.NoError(t, err)
	return cp, c
}

func TestSpecUsesTheETag(t *testing.T) {
	t.Parallel()
	cp, c := client(t)

	d, changed, err := c.Spec(t.Context(), "")
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, int64(1), d.Revision)
	require.Equal(t, `"1"`, d.ETag)
	require.Equal(t, "0.11.0", d.Spec.Release.Version)

	_, changed, err = c.Spec(t.Context(), d.ETag)
	require.NoError(t, err)
	require.False(t, changed)

	next := d.Spec
	next.Lifecycle = instanceapi.Suspended
	cp.SetSpec(next)
	d, changed, err = c.Spec(t.Context(), d.ETag)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, int64(2), d.Revision)
	require.Equal(t, instanceapi.Suspended, d.Spec.Lifecycle)
}

func TestRateLimitsCarryRetryAfter(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	cp.RateLimitNext(1)

	err := c.ReportStatus(t.Context(), instanceapi.StatusReport{Ready: true, Components: []instanceapi.ComponentStatus{}})
	var rl *controlplane.RetryLaterError
	require.True(t, errors.As(err, &rl))
	require.Equal(t, time.Second, rl.RetryAfter)

	require.NoError(t, c.ReportStatus(t.Context(), instanceapi.StatusReport{Ready: true, Components: []instanceapi.ComponentStatus{}}))
	require.Len(t, cp.Reports(), 1)
}

func TestRevokedCertificatesAreUnauthenticated(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	cp.RevokeAll()
	_, _, err := c.Spec(t.Context(), "")
	require.ErrorIs(t, err, controlplane.ErrUnauthenticated)
}

func TestRenewRefusesTheCurrentKey(t *testing.T) {
	t.Parallel()
	_, c := client(t)
	key, err := credential.GenerateKey(uuid.New())
	require.NoError(t, err)
	issued, err := c.Renew(t.Context(), key.CSRPEM)
	require.NoError(t, err)
	require.Contains(t, string(issued.CertPEM), "BEGIN CERTIFICATE")

	_, err = c.Renew(t.Context(), []byte("not a csr"))
	var pe *controlplane.ProblemError
	require.True(t, errors.As(err, &pe))
	require.Equal(t, http.StatusBadRequest, pe.Status)
	require.Equal(t, "validation_failed", pe.Code)
}

func TestAnUnreadySpecAsksToRetry(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	cp.UnavailableNext(1)

	_, _, err := c.Spec(t.Context(), "")
	var later *controlplane.RetryLaterError
	require.True(t, errors.As(err, &later))
	require.Equal(t, time.Second, later.RetryAfter)

	_, changed, err := c.Spec(t.Context(), "")
	require.NoError(t, err)
	require.True(t, changed)
}

func TestReportUsageReturnsTheAck(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	hour := time.Now().UTC().Truncate(time.Hour)
	window := instanceapi.UsageWindow{Start: hour.Add(-time.Hour), End: hour.Add(time.Hour)}
	cp.SetUsageAck(&instanceapi.UsageAck{
		Window:  &window,
		Overage: true,
		Meters: []instanceapi.MeterUsage{{
			Meter: "shipments", Unit: instanceapi.MeterUsageUnitCount, Quantity: 3, Allowance: 10,
		}},
	})

	ack, err := c.ReportUsage(t.Context(), instanceapi.UsageReport{
		Readings: []instanceapi.UsageReading{{Meter: "shipments", Hour: hour, Quantity: 3}},
	})
	require.NoError(t, err)
	require.True(t, ack.Overage)
	require.NotNil(t, ack.Window)
	require.True(t, ack.Window.Start.Equal(window.Start))
	require.Equal(t, int64(3), ack.Meters[0].Quantity)
	require.Len(t, cp.UsageReports(), 1)

	empty := instanceapi.UsageReport{Readings: []instanceapi.UsageReading{}}
	cp.RateLimitNext(1)
	_, err = c.ReportUsage(t.Context(), empty)
	var later *controlplane.RetryLaterError
	require.ErrorAs(t, err, &later)

	cp.FailUsageNext(1)
	_, err = c.ReportUsage(t.Context(), empty)
	var pe *controlplane.ProblemError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, http.StatusInternalServerError, pe.Status)

	cp.RevokeAll()
	_, err = c.ReportUsage(t.Context(), empty)
	require.ErrorIs(t, err, controlplane.ErrUnauthenticated)
}

func TestSecretsAreTheSameOnEveryCall(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	first, err := c.Secrets(t.Context())
	require.NoError(t, err)
	require.Equal(t, controlplanetest.Secrets(), first)
	second, err := c.Secrets(t.Context())
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, 2, cp.SecretCalls())
}

func TestSecretsFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		check  func(t *testing.T, err error)
	}{
		{
			name:   "rate limited",
			status: http.StatusTooManyRequests,
			check: func(t *testing.T, err error) {
				var later *controlplane.RetryLaterError
				require.ErrorAs(t, err, &later)
				require.Equal(t, time.Second, later.RetryAfter)
			},
		},
		{
			name:   "refused certificate",
			status: http.StatusUnauthorized,
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, controlplane.ErrUnauthenticated)
			},
		},
		{
			name:   "server error",
			status: http.StatusInternalServerError,
			check: func(t *testing.T, err error) {
				var pe *controlplane.ProblemError
				require.ErrorAs(t, err, &pe)
				require.Equal(t, http.StatusInternalServerError, pe.Status)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cp, c := client(t)
			cp.FailSecretsNext(1, tc.status)
			_, err := c.Secrets(t.Context())
			tc.check(t, err)
			for _, v := range []string{
				controlplanetest.Secrets().EncryptionKey,
				controlplanetest.Secrets().RlsScopeKey,
			} {
				require.NotContains(t, err.Error(), v)
			}
			got, err := c.Secrets(t.Context())
			require.NoError(t, err)
			require.Equal(t, controlplanetest.Secrets(), got)
		})
	}
}

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
	var rl *controlplane.RateLimitedError
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

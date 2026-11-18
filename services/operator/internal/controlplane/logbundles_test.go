package controlplane_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func registration(requestID string, period time.Duration) *instanceapi.LogBundleRegistration {
	since := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	return &instanceapi.LogBundleRegistration{
		RequestId:   requestID,
		Since:       since,
		Until:       since.Add(period),
		RequestedBy: instanceapi.LogRequester{Email: "dana@acme.example"},
	}
}

func TestRegisteringALogRequestTwiceAnswersTheSameBundle(t *testing.T) {
	t.Parallel()
	cp, c := client(t)

	first, err := c.RegisterLogBundle(t.Context(), registration("lrq_01", time.Hour))
	require.NoError(t, err)
	again, err := c.RegisterLogBundle(t.Context(), registration("lrq_01", time.Hour))
	require.NoError(t, err)

	require.Equal(t, first, again)
	require.Equal(t, 2, cp.LogBundleRegistrations())
}

func TestAPeriodLongerThanAWeekIsNotRegistered(t *testing.T) {
	t.Parallel()
	_, c := client(t)

	_, err := c.RegisterLogBundle(t.Context(), registration("lrq_02", 8*24*time.Hour))

	var pe *controlplane.ProblemError
	require.True(t, errors.As(err, &pe))
	require.Equal(t, http.StatusBadRequest, pe.Status)
}

func TestABundlesProgressIsRecordedAndAnUnknownOneIsLeft(t *testing.T) {
	t.Parallel()
	cp, c := client(t)
	id, err := c.RegisterLogBundle(t.Context(), registration("lrq_03", time.Hour))
	require.NoError(t, err)

	require.NoError(t, c.UpdateLogBundle(t.Context(), id, instanceapi.LogBundleUpdate{
		Status: instanceapi.LogBundleCollecting,
	}))
	key, size := "support/logs/"+id.String()+".zip", int64(2048)
	components := []instanceapi.LogBundleComponent{{Component: "api", Lines: 10}}
	require.NoError(t, c.UpdateLogBundle(t.Context(), id, instanceapi.LogBundleUpdate{
		Status: instanceapi.LogBundleStored, ObjectKey: &key, SizeBytes: &size, Components: &components,
	}))
	require.NoError(t, c.UpdateLogBundle(t.Context(), uuid.New(), instanceapi.LogBundleUpdate{
		Status: instanceapi.LogBundleCollecting,
	}), "a bundle the control plane no longer has is nothing to report")

	b, ok := cp.LogBundle(id)
	require.True(t, ok)
	require.Len(t, b.Updates, 2)
	require.Equal(t, instanceapi.LogBundleStored, b.Updates[1].Status)
}

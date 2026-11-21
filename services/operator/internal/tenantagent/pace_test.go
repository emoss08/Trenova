package tenantagent_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tenantagent"
)

type countingCP struct {
	calls map[string]int
	later map[string]time.Duration
}

func (c *countingCP) answer(op string) error {
	c.calls[op]++
	if wait, ok := c.later[op]; ok {
		delete(c.later, op)
		return &controlplane.RetryLaterError{Reason: "rate limited", RetryAfter: wait}
	}
	return nil
}

func (c *countingCP) SharedTenants(context.Context, string) (controlplane.SharedDesired, bool, error) {
	return controlplane.SharedDesired{}, false, c.answer("tenants")
}

func (c *countingCP) ReportSharedStatus(
	context.Context,
	*instanceapi.SharedStatusReport,
) (instanceapi.SharedReportAck, error) {
	return instanceapi.SharedReportAck{}, c.answer("status")
}

func (c *countingCP) SharedStorage(context.Context) (instanceapi.SharedStorage, error) {
	return instanceapi.SharedStorage{}, c.answer("storage")
}

func (c *countingCP) ReportSharedUsage(
	context.Context,
	instanceapi.SharedUsageReport,
) (instanceapi.SharedUsageAck, error) {
	return instanceapi.SharedUsageAck{}, c.answer("usage")
}

func TestEachSharedOperationWaitsAsLongAsTheControlPlaneAsks(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	cp := &countingCP{calls: map[string]int{}, later: map[string]time.Duration{
		"status": 90 * time.Second,
		"usage":  2 * time.Second,
	}}
	paced := tenantagent.NewPacedControlPlane(cp, 15*time.Second, func() time.Time { return now })

	_, err := paced.ReportSharedStatus(t.Context(), &instanceapi.SharedStatusReport{})
	var later *controlplane.RetryLaterError
	require.ErrorAs(t, err, &later)
	assert.Equal(t, 90*time.Second, later.RetryAfter)

	_, err = paced.ReportSharedUsage(t.Context(), instanceapi.SharedUsageReport{})
	require.ErrorAs(t, err, &later)
	assert.Equal(t, 15*time.Second, later.RetryAfter, "never shorter than the agent's own floor")

	now = now.Add(60 * time.Second)
	_, err = paced.ReportSharedStatus(t.Context(), &instanceapi.SharedStatusReport{})
	require.ErrorAs(t, err, &later, "the status report still waits")
	assert.Equal(t, 30*time.Second, later.RetryAfter)
	assert.Equal(t, 1, cp.calls["status"], "nothing was sent while it waited")

	_, err = paced.ReportSharedUsage(t.Context(), instanceapi.SharedUsageReport{})
	require.NoError(t, err, "the usage report waited its own time and goes again")
	_, _, err = paced.SharedTenants(t.Context(), "")
	require.NoError(t, err, "other operations are not held")

	now = now.Add(31 * time.Second)
	_, err = paced.ReportSharedStatus(t.Context(), &instanceapi.SharedStatusReport{})
	require.NoError(t, err)
	assert.Equal(t, 2, cp.calls["status"])
}

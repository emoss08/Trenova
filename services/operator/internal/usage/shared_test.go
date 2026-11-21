package usage_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/usage"
)

type refusingCP struct {
	refuse  map[uuid.UUID]instanceapi.SharedEntryRefusedCode
	reports []instanceapi.SharedUsageReport
}

func (c *refusingCP) ReportSharedUsage(
	_ context.Context,
	report instanceapi.SharedUsageReport,
) (instanceapi.SharedUsageAck, error) {
	c.reports = append(c.reports, report)
	ack := instanceapi.SharedUsageAck{Refused: []instanceapi.SharedEntryRefused{}}
	seen := map[uuid.UUID]bool{}
	for _, rd := range report.Readings {
		code, ok := c.refuse[rd.InstanceId]
		if ok && !seen[rd.InstanceId] {
			seen[rd.InstanceId] = true
			ack.Refused = append(
				ack.Refused,
				instanceapi.SharedEntryRefused{InstanceId: rd.InstanceId, Code: code},
			)
		}
	}
	return ack, nil
}

func (c *refusingCP) last() map[uuid.UUID][]time.Time {
	out := map[uuid.UUID][]time.Time{}
	for _, rd := range c.reports[len(c.reports)-1].Readings {
		out[rd.InstanceId] = append(out[rd.InstanceId], rd.Hour)
	}
	return out
}

type heldTMS struct {
	tenants []uuid.UUID
}

func (h *heldTMS) TenantReadings(
	_ context.Context,
	from, to time.Time,
) ([]tms.TenantReadings, error) {
	out := make([]tms.TenantReadings, 0, len(h.tenants))
	for _, id := range h.tenants {
		t := tms.TenantReadings{InstanceID: id.String()}
		for at := from; at.Before(to); at = at.Add(time.Hour) {
			t.Readings = append(
				t.Readings,
				tms.TenantReading{Meter: "shipments.total", Hour: at, Quantity: 3},
			)
		}
		out = append(out, t)
	}
	return out, nil
}

func (*heldTMS) PutTenantUsage(
	_ context.Context,
	tenants []tms.TenantUsage,
) (tms.TenantUsageAnswer, error) {
	return tms.TenantUsageAnswer{Applied: len(tenants)}, nil
}

func TestARefusedTenantHourIsNotSentAgainAndOthersKeepFlowing(t *testing.T) {
	t.Parallel()
	good, implausible, unknown := uuid.New(), uuid.New(), uuid.New()
	cp := &refusingCP{refuse: map[uuid.UUID]instanceapi.SharedEntryRefusedCode{
		implausible: "implausible",
		unknown:     "a_code_this_agent_has_never_heard_of",
	}}
	held := &heldTMS{tenants: []uuid.UUID{good, implausible, unknown}}
	var logs bytes.Buffer
	clock := hour.Add(5 * time.Minute)
	relay := usage.NewShared(usage.SharedParams{
		ControlPlane: cp,
		TMS:          held,
		Logger:       slog.New(slog.NewTextHandler(&logs, nil)),
		Now:          func() time.Time { return clock },
	})
	tenants := []instanceapi.SharedTenant{
		{InstanceId: good, Allowances: allowances},
		{InstanceId: implausible, Allowances: allowances},
		{InstanceId: unknown, Allowances: allowances},
	}

	result, err := relay.Run(t.Context(), tenants)
	require.NoError(t, err)
	assert.Equal(t, 6, result.Readings)
	assert.ElementsMatch(t, []string{implausible.String(), unknown.String()}, result.RefusedByCP)
	assert.Equal(t, 0, result.Withheld)

	for range 3 {
		clock = clock.Add(15 * time.Minute)
		result, err = relay.Run(t.Context(), tenants)
		require.NoError(t, err)
		sent := cp.last()
		assert.Len(t, sent[good], 2, "the accepted tenant is relayed every time")
		assert.Empty(t, sent[implausible], "a refused tenant-hour is done")
		assert.Empty(t, sent[unknown], "an unknown refusal code is honoured the same way")
		assert.Equal(t, 4, result.Withheld)
		assert.Empty(t, result.RefusedByCP)
	}
	assert.Equal(t, 1, strings.Count(logs.String(), implausible.String()),
		"logged once in the hour, not per poll")
	assert.Contains(t, logs.String(), "code=implausible")
	assert.Contains(t, logs.String(), "code=a_code_this_agent_has_never_heard_of")

	clock = hour.Add(time.Hour + 5*time.Minute)
	result, err = relay.Run(t.Context(), tenants)
	require.NoError(t, err)
	sent := cp.last()
	assert.Equal(t, []time.Time{hour}, sent[good][1:],
		"the window moved: the new hour is reported")
	assert.Equal(t, []time.Time{hour}, sent[implausible],
		"a later hour of a refused tenant is reported once")
	assert.Equal(t, 2, result.Withheld, "the hour already refused stays withheld")
	assert.Equal(t, 2, strings.Count(logs.String(), implausible.String()),
		"a refusal in a new hour is logged again, once")
}

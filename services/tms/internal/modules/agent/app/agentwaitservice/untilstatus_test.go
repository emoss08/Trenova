package agentwaitservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *streams) watchStatus(id pulid.ID, statuses ...string) (*agentwait.Wait, error) {
	return s.register(agentwait.KindRecordChanged, agentwait.Condition{
		RecordID:    id,
		UntilStatus: statuses,
	})
}

func TestRegister_AStatusWaitOnARecordAlreadyThereIsActedOnNow(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	s.records.found[shipment] = &agentwaitports.FoundRecord{
		Known: true, Exists: true, Label: "S26110001005205", Status: "Completed", HasStatus: true,
	}

	_, err := s.watchStatus(shipment, "completed", "ReadyToInvoice")

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "shipment S26110001005205 is Completed.")
	assert.Empty(t, s.feed.watched)
}

func TestRegister_AStatusWaitOnAKindWithoutOneIsRefused(t *testing.T) {
	t.Parallel()

	s := newStreams()
	customer := pulid.MustNew("cus_")
	s.records.found[customer] = &agentwaitports.FoundRecord{Known: true, Exists: true, Label: "Acme"}

	_, err := s.watchStatus(customer, "Active")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "A customer has no status a wait can name")
}

func TestRegister_AStatusWaitOnARecordNotThereYetIsSet(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	s.records.found[shipment] = &agentwaitports.FoundRecord{
		Known: true, Exists: true, Label: "S26110001005205", Status: "Delayed", HasStatus: true,
	}

	wait, err := s.watchStatus(shipment, "Completed")

	require.NoError(t, err)
	assert.Equal(t, []string{"Completed"}, wait.Condition.UntilStatus)
	require.Len(t, s.feed.watched, 1)
}

func TestNotifyRecordChanges_AStatusWaitSleepsThroughEditsShortOfTheStatus(t *testing.T) {
	t.Parallel()

	s := newStreams()
	shipment := pulid.MustNew("shp_")
	wait := openRecordWait(s, shipment, now-600)
	wait.Condition.UntilStatus = []string{"Completed"}
	colleague := pulid.MustNew("usr_")
	record := &agentwaitports.FoundRecord{
		Known: true, Exists: true, Label: "S26110001005205", Status: "InTransit", HasStatus: true,
	}
	s.records.found[shipment] = record
	change := func(at int64) []realtimeports.TenantRecordChange {
		return []realtimeports.TenantRecordChange{
			changeOf(s, shipment, permissionports.PrincipalTypeUser, colleague, at),
		}
	}

	s.svc.NotifyRecordChanges(context.Background(), change(now-120))
	assert.Empty(t, s.workflows.signals, "a driver change leaves the shipment short of Completed")

	record.Status = "Completed"
	s.svc.NotifyRecordChanges(context.Background(), change(now-60))

	require.Len(t, s.workflows.signals, 1)
	assert.Contains(t, s.workflows.signals[0].met.Detail,
		"It is now Completed, the status you waited for.")
}

func TestReachedStatus_MatchesAnyNamedStatusIgnoringCase(t *testing.T) {
	t.Parallel()

	c := &agentwait.Condition{UntilStatus: []string{"Completed", " ReadyToInvoice "}}

	assert.True(t, c.ReachedStatus("completed"))
	assert.True(t, c.ReachedStatus("ReadyToInvoice"))
	assert.False(t, c.ReachedStatus("Delayed"))
	assert.True(t, (&agentwait.Condition{}).ReachedStatus("Delayed"),
		"a wait that names no status ends on any change")
}

func TestValidate_UntilStatusNarrowsOnlyARecordChange(t *testing.T) {
	t.Parallel()

	s := newStreams()
	_, err := s.register(agentwait.KindReply, agentwait.Condition{
		ShipmentID:  pulid.MustNew("shp_"),
		UntilStatus: []string{"Completed"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "untilStatus narrows a RecordChanged wait only")
}

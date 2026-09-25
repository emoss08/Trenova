package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEscalateDetention_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	deadline := int64(1)
	occurrence := &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("do_"),
		Status:             detention.OccurrenceStatusAccruing,
		BillableMinutes:    75,
		NotificationStatus: detention.NotificationStatusPending,
		NoticeDeadlineAt:   &deadline,
		Version:            4,
	}
	before := *occurrence
	stub := &stubEscalator{occurrence: occurrence}
	tool := newEscalateDetentionTool(stub).(*escalateDetentionTool)
	params := deskParams(map[string]any{
		"occurrenceId": occurrence.ID.String(),
		"reason":       "the notice window closed",
	})

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	require.Len(t, preview.Changes, 2)
	update := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, update.Operation)
	assert.Equal(t, permission.ResourceDetentionPolicy, update.Resource)
	assert.Equal(t, false, fieldByPath(t, update, "requiresApproval").Before)
	assert.Equal(t, true, fieldByPath(t, update, "requiresApproval").After)
	assert.Equal(t, "Missed", fieldByPath(t, update, "notificationStatus").After)

	evidence := previewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, evidence.Operation)
	assert.Equal(t, "Escalated to a person: the notice window closed",
		fieldByPath(t, evidence, "summary").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, update, &before, occurrence,
		toolpreview.Only(detentionEscalationFields...))
}

func TestEscalateDetention_PreviewWarnsWhenItWouldBeRefused(t *testing.T) {
	t.Parallel()

	occurrence := &detention.DetentionOccurrence{
		ID:               pulid.MustNew("do_"),
		Status:           detention.OccurrenceStatusAccruing,
		RequiresApproval: true,
	}
	stub := &stubEscalator{occurrence: occurrence}
	tool := newEscalateDetentionTool(stub).(*escalateDetentionTool)

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), deskParams(map[string]any{
			"occurrenceId": occurrence.ID.String(),
			"reason":       "look at it",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestPlaceWorkerDispatchHold_PreviewReadsTheDriver(t *testing.T) {
	t.Parallel()

	driver := &worker.Worker{
		ID:            pulid.MustNew("wrk_"),
		FirstName:     "Ana",
		LastName:      "Reyes",
		CanBeAssigned: true,
		Version:       3,
	}
	before := *driver
	stub := &stubWorkerHolder{held: driver}
	tool := newPlaceWorkerDispatchHoldTool(stub).(*placeWorkerDispatchHoldTool)
	params := deskParams(map[string]any{
		"workerId": driver.ID.String(),
		"reason":   "medical card lapsed on the 3rd",
	})

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceWorker, change.Resource)
	assert.Equal(t, "Ana Reyes", change.Label)
	assert.Equal(t, true, fieldByPath(t, change, "canBeAssigned").Before)
	assert.Equal(t, false, fieldByPath(t, change, "canBeAssigned").After)
	assert.Contains(t, preview.Summary, "Ana Reyes")

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, driver, toolpreview.Only("canBeAssigned"))
}

func TestPlaceWorkerDispatchHold_PreviewWarnsForADriverAlreadyHeld(t *testing.T) {
	t.Parallel()

	driver := &worker.Worker{ID: pulid.MustNew("wrk_"), CanBeAssigned: false}
	stub := &stubWorkerHolder{held: driver}
	tool := newPlaceWorkerDispatchHoldTool(stub).(*placeWorkerDispatchHoldTool)

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), deskParams(map[string]any{
			"workerId": driver.ID.String(),
			"reason":   "medical card lapsed",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestResolveCarrierIntelEvent_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	event := openFinding()
	event.Summary = "Operating authority revoked"
	event.Version = 5
	before := *event
	stub := &stubCarrierIntel{event: event}
	tool := newResolveCarrierIntelEventTool(stub).(*resolveCarrierIntelEventTool)
	params := deskParams(map[string]any{
		"eventId":    event.ID.String(),
		"resolution": string(carrierintel.EventResolutionCarrierBlocked),
		"note":       "authority revoked; not to be tendered",
	})

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "Operating authority revoked", change.Label)
	assert.Equal(t, "Open", fieldByPath(t, change, "status").Before)
	assert.Equal(t, "Resolved", fieldByPath(t, change, "status").After)
	assert.True(t, fieldByPath(t, change, "resolvedAt").Volatile)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, event,
		toolpreview.Volatile(carrierIntelVolatileFields...))
}

func TestAcknowledgeCarrierIntelEvent_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	event := openFinding()
	before := *event
	stub := &stubCarrierIntel{event: event}
	tool := newAcknowledgeCarrierIntelEventTool(stub).(*acknowledgeCarrierIntelEventTool)
	params := deskParams(map[string]any{
		"eventId": event.ID.String(),
		"note":    "new mailing address, nothing to do",
	})

	preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, "Acknowledged", fieldByPath(t, change, "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, event,
		toolpreview.Volatile(carrierIntelVolatileFields...))
}

func TestCarrierIntelEventTools_PreviewWarnsOnAClosedFinding(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func(carrierIntelActor) serviceports.AgentTool{
		"acknowledge": newAcknowledgeCarrierIntelEventTool,
		"resolve":     newResolveCarrierIntelEventTool,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := openFinding()
			event.Status = carrierintel.EventStatusResolved
			stub := &stubCarrierIntel{event: event}
			tool := build(stub).(serviceports.ToolPreviewer)

			preview := previewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
				return tool.Preview(t.Context(), deskParams(map[string]any{
					"eventId":    event.ID.String(),
					"resolution": string(carrierintel.EventResolutionCarrierUpdated),
					"note":       "already done",
				}))
			})

			assert.Empty(t, preview.Changes)
			requireWarning(t, preview, agent.PreviewWarningWouldFail)
		})
	}
}

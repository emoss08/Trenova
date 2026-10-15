package masterdatatools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
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
	tool := NewEscalateDetentionTool(stub).(*EscalateDetentionTool)
	params := agenttoolstest.DeskParams(map[string]any{
		"occurrenceId": occurrence.ID.String(),
		"reason":       "the notice window closed",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	require.Len(t, preview.Changes, 2)
	update := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationUpdate, update.Operation)
	assert.Equal(t, permission.ResourceDetention, update.Resource)
	assert.Equal(t, false, agenttoolstest.FieldByPath(t, update, "requiresApproval").Before)
	assert.Equal(t, true, agenttoolstest.FieldByPath(t, update, "requiresApproval").After)
	assert.Equal(t, "Missed", agenttoolstest.FieldByPath(t, update, "notificationStatus").After)

	evidence := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, agent.PreviewOperationCreate, evidence.Operation)
	assert.Equal(t, "Escalated to a person: the notice window closed",
		agenttoolstest.FieldByPath(t, evidence, "summary").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, update, &before, occurrence,
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
	tool := NewEscalateDetentionTool(stub).(*EscalateDetentionTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"occurrenceId": occurrence.ID.String(),
			"reason":       "look at it",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
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
	tool := NewPlaceWorkerDispatchHoldTool(stub).(*PlaceWorkerDispatchHoldTool)
	params := agenttoolstest.DeskParams(map[string]any{
		"workerId": driver.ID.String(),
		"reason":   "medical card lapsed on the 3rd",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceWorker, change.Resource)
	assert.Equal(t, "Ana Reyes", change.Label)
	assert.Equal(t, true, agenttoolstest.FieldByPath(t, change, "canBeAssigned").Before)
	assert.Equal(t, false, agenttoolstest.FieldByPath(t, change, "canBeAssigned").After)
	assert.Contains(t, preview.Summary, "Ana Reyes")

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, driver, toolpreview.Only("canBeAssigned"))
}

func TestPlaceWorkerDispatchHold_PreviewWarnsForADriverAlreadyHeld(t *testing.T) {
	t.Parallel()

	driver := &worker.Worker{ID: pulid.MustNew("wrk_"), CanBeAssigned: false}
	stub := &stubWorkerHolder{held: driver}
	tool := NewPlaceWorkerDispatchHoldTool(stub).(*PlaceWorkerDispatchHoldTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"workerId": driver.ID.String(),
			"reason":   "medical card lapsed",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestResolveCarrierIntelEvent_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	event := openFinding()
	event.Summary = "Operating authority revoked"
	event.Version = 5
	before := *event
	stub := &stubCarrierIntel{event: event}
	tool := NewResolveCarrierIntelEventTool(stub).(*ResolveCarrierIntelEventTool)
	params := agenttoolstest.DeskParams(map[string]any{
		"eventId":    event.ID.String(),
		"resolution": string(carrierintel.EventResolutionCarrierBlocked),
		"note":       "authority revoked; not to be tendered",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Operating authority revoked", change.Label)
	assert.Equal(t, "Open", agenttoolstest.FieldByPath(t, change, "status").Before)
	assert.Equal(t, "Resolved", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.True(t, agenttoolstest.FieldByPath(t, change, "resolvedAt").Volatile)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, event,
		toolpreview.Volatile(carrierIntelVolatileFields...))
}

func TestAcknowledgeCarrierIntelEvent_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	event := openFinding()
	before := *event
	stub := &stubCarrierIntel{event: event}
	tool := NewAcknowledgeCarrierIntelEventTool(stub).(*AcknowledgeCarrierIntelEventTool)
	params := agenttoolstest.DeskParams(map[string]any{
		"eventId": event.ID.String(),
		"note":    "new mailing address, nothing to do",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Acknowledged", agenttoolstest.FieldByPath(t, change, "status").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, event,
		toolpreview.Volatile(carrierIntelVolatileFields...))
}

func TestCarrierIntelEventTools_PreviewWarnsOnAClosedFinding(t *testing.T) {
	t.Parallel()

	for name, build := range map[string]func(carrierIntelActor) agentports.AgentTool{
		"acknowledge": NewAcknowledgeCarrierIntelEventTool,
		"resolve":     NewResolveCarrierIntelEventTool,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := openFinding()
			event.Status = carrierintel.EventStatusResolved
			stub := &stubCarrierIntel{event: event}
			tool := build(stub).(agentports.ToolPreviewer)

			preview := agenttoolstest.PreviewWithoutWrites(t, &stub.guard, func() (*agent.ToolPreview, error) {
				return tool.Preview(t.Context(), agenttoolstest.DeskParams(map[string]any{
					"eventId":    event.ID.String(),
					"resolution": string(carrierintel.EventResolutionCarrierUpdated),
					"note":       "already done",
				}))
			})

			assert.Empty(t, preview.Changes)
			agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
		})
	}
}

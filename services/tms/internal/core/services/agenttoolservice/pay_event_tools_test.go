package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePayEvents struct {
	event      *driversettlement.PayEvent
	settlement *driversettlement.Settlement
	autoAttach bool
	locked     bool

	held       pulid.ID
	heldReason string
	released   pulid.ID
	attached   []pulid.ID
	detached   pulid.ID
	target     pulid.ID
}

func (f *fakePayEvents) PlanPayEventHold(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
	hold bool,
) (*driversettlementservice.PayEventPlan, error) {
	after := *f.event
	plan := &driversettlementservice.PayEventPlan{Before: f.event, After: &after}
	if hold {
		plan.Changed, plan.Refusal = driversettlementservice.PlanHoldPayEvent(&after, reason)
	} else {
		plan.Changed = driversettlementservice.PlanReleasePayEvent(&after)
		plan.AutoAttach = plan.Changed && f.autoAttach
	}

	return plan, nil
}

func (f *fakePayEvents) HoldPayEvent(
	_ context.Context,
	_ pagination.TenantInfo,
	eventID pulid.ID,
	reason string,
	_ *serviceports.RequestActor,
) (*driversettlement.PayEvent, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	f.held, f.heldReason = eventID, reason

	return f.event, nil
}

func (f *fakePayEvents) ReleasePayEvent(
	_ context.Context,
	_ pagination.TenantInfo,
	eventID pulid.ID,
	_ *serviceports.RequestActor,
) (*driversettlement.PayEvent, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	f.released = eventID

	return f.event, nil
}

func (f *fakePayEvents) PlanEventTransfer(
	_ context.Context,
	req *driversettlementservice.EventTransferRequest,
) (*driversettlementservice.ActionPlan, error) {
	after := driversettlementservice.CloneSettlement(f.settlement)
	plan := &settlementshared.ActionPlan[*driversettlement.Settlement]{
		Before: f.settlement,
		After:  after,
	}
	if after.Status != driversettlement.StatusDraft {
		plan.Refusal = errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Pay events can be moved only on a draft settlement",
		)

		return plan, nil
	}
	if req.Detach {
		after.ShipmentCount--
	} else {
		after.Lines = append(after.Lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryEarning,
			Description: "Late load",
			AmountMinor: 42000,
		})
		after.ShipmentCount += len(req.EventIDs)
		after.SyncTotals()
	}

	return plan, nil
}

func (f *fakePayEvents) AttachPayEvents(
	_ context.Context,
	_ pagination.TenantInfo,
	settlementID pulid.ID,
	eventIDs []pulid.ID,
	_ *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	f.target, f.attached = settlementID, eventIDs

	return f.settlement, nil
}

func (f *fakePayEvents) DetachPayEvent(
	_ context.Context,
	_ pagination.TenantInfo,
	settlementID, eventID pulid.ID,
	_ *serviceports.RequestActor,
) (*driversettlement.Settlement, error) {
	if f.locked {
		return nil, errPerformedDuringPreview
	}
	f.target, f.detached = settlementID, eventID

	return f.settlement, nil
}

func accruedPayEvent() *driversettlement.PayEvent {
	return &driversettlement.PayEvent{
		ID:        pulid.MustNew("dpe_"),
		ProNumber: "PRO-7781",
		Status:    driversettlement.PayEventStatusAccrued,
		Version:   2,
		Worker:    &worker.Worker{FirstName: "Ana", LastName: "Ruiz"},
	}
}

func TestHoldDriverPayEvent_TellsTheDriverWhyInTheirPortal(t *testing.T) {
	t.Parallel()

	events := &fakePayEvents{event: accruedPayEvent(), locked: true}
	tool := &payEventHoldTool{events: events, hold: true}

	policy := tool.Policy()
	assert.Equal(t, []agent.EgressClass{agent.EgressDriverVisible}, policy.Egress)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)

	missing := executeParams(map[string]any{paramPayEventID: events.event.ID.String()})
	require.Error(t, tool.Execute(t.Context(), missing))

	params := executeParams(map[string]any{
		paramPayEventID: events.event.ID.String(),
		paramHoldReason: "  Proof of delivery is missing for this load  ",
	})
	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would hold Ana Ruiz's pay for PRO-7781.")
	assert.Contains(t, preview.Summary, "driver portal")
	require.Len(t, preview.Changes, 2)
	assert.Equal(t, true, fieldByPath(t, previewChange(t, preview, 0), fieldOnHold).After)
	message := previewChange(t, preview, 1).Message
	require.NotNil(t, message)
	assert.Equal(t, agent.MessageChannelDash, message.Channel)
	assert.Equal(t, "Proof of delivery is missing for this load", message.Body)

	events.locked = false
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, events.event.ID, events.held)
	assert.Equal(t, "Proof of delivery is missing for this load", events.heldReason)
}

func TestHoldDriverPayEvent_SettledPayCannotBeHeld(t *testing.T) {
	t.Parallel()

	event := accruedPayEvent()
	event.Status = driversettlement.PayEventStatusSettled
	tool := &payEventHoldTool{events: &fakePayEvents{event: event, locked: true}, hold: true}
	params := executeParams(map[string]any{
		paramPayEventID: event.ID.String(),
		paramHoldReason: "Disputed load",
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), params))
}

func TestReleaseDriverPayEvent_SaysWhenNothingIsHeld(t *testing.T) {
	t.Parallel()

	events := &fakePayEvents{event: accruedPayEvent(), autoAttach: true, locked: true}
	tool := &payEventHoldTool{events: events}
	params := executeParams(map[string]any{paramPayEventID: events.event.ID.String()})

	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, tool.Policy().Egress)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "is already released; nothing would change")
	assert.Empty(t, preview.Changes)

	events.event.OnHold = true
	events.event.HoldReason = "Disputed load"
	preview, err = tool.Preview(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, false, fieldByPath(t, previewChange(t, preview, 0), fieldOnHold).After)
	assert.Contains(t, preview.Summary, "open draft settlement")

	events.locked = false
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, events.event.ID, events.released)
}

func TestAttachPayEvents_PreviewsTheNewNetOnTheDraft(t *testing.T) {
	t.Parallel()

	settlement := pendingDriverSettlement()
	settlement.Status = driversettlement.StatusDraft
	events := &fakePayEvents{settlement: settlement, locked: true}
	tool := &payEventTransferTool{events: events}
	first, second := pulid.MustNew("dpe_"), pulid.MustNew("dpe_")
	params := executeParams(map[string]any{
		paramSettlementID: settlement.ID.String(),
		paramPayEventIDs:  []any{first.String(), second.String()},
	})

	target, ok := tool.Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceDriverSettlement, target.Resource)
	assert.Equal(t, settlement.ID, target.ID)

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would add 2 pay events to driver settlement DS-2041")
	change := previewChange(t, preview, 0)
	require.NotNil(t, change.Money)
	assert.Equal(t, "420", change.Money.Delta.Decimal.String())

	require.Error(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramSettlementID: settlement.ID.String(),
		paramPayEventIDs:  []any{},
	})))

	events.locked = false
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, settlement.ID, events.target)
	assert.Equal(t, []pulid.ID{first, second}, events.attached)
}

func TestDetachPayEvent_RefusesAnythingButADraft(t *testing.T) {
	t.Parallel()

	settlement := pendingDriverSettlement()
	events := &fakePayEvents{settlement: settlement, locked: true}
	tool := &payEventTransferTool{events: events, detach: true}
	eventID := pulid.MustNew("dpe_")
	params := executeParams(map[string]any{
		paramSettlementID: settlement.ID.String(),
		paramPayEventID:   eventID.String(),
	})

	preview, err := tool.Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	require.Error(t, tool.Validate(t.Context(), params))

	settlement.Status = driversettlement.StatusDraft
	require.NoError(t, tool.Validate(t.Context(), params))
	events.locked = false
	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, eventID, events.detached)
}

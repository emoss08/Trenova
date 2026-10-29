package agenttrigger_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/stretchr/testify/assert"
)

func TestKnownEvents_EveryKindIsValidAndCarriesASubject(t *testing.T) {
	t.Parallel()

	events := agenttrigger.KnownEvents()
	assert.NotEmpty(t, events)

	seen := make(map[agenttrigger.EventKind]struct{}, len(events))
	for _, event := range events {
		assert.True(t, event.Kind.IsValid(), event.Kind)
		assert.True(t, event.SubjectType.IsValid(), event.Kind)
		assert.NotEmpty(t, event.Label, event.Kind)
		assert.NotEmpty(t, event.Description, event.Kind)
		_, duplicate := seen[event.Kind]
		assert.False(t, duplicate, "duplicate event kind %s", event.Kind)
		seen[event.Kind] = struct{}{}
	}

	assert.False(t, agenttrigger.EventKind("shipment.teleported").IsValid())
	assert.Equal(
		t,
		agent.SubjectBillingQueueItem,
		agenttrigger.EventBillingQueueItemException.SubjectType(),
	)
	assert.Equal(t, agent.SubjectInsight, agenttrigger.EventInsightDetected.SubjectType())
	assert.Equal(t, agent.SubjectBankReceipt, agenttrigger.EventBankReceiptException.SubjectType())
}

func TestRunTrigger_IsValid(t *testing.T) {
	t.Parallel()

	for _, trigger := range []agent.RunTrigger{
		agent.RunTriggerManual,
		agent.RunTriggerChat,
		agent.RunTriggerScheduled,
		agent.RunTriggerEvent,
		agent.RunTriggerContinuous,
	} {
		assert.True(t, trigger.IsValid(), trigger)
	}
	assert.False(t, agent.RunTrigger("Whim").IsValid())
	assert.True(t, agent.TypeGeneral.IsValid())
	assert.True(t, agent.SubjectOrganization.IsValid())
}

func TestKnownEvents_MoveUnassignedDescribesOnlyALostAssignment(t *testing.T) {
	t.Parallel()

	for _, event := range agenttrigger.KnownEvents() {
		if event.Kind != agenttrigger.EventShipmentMoveUnassigned {
			continue
		}

		assert.Equal(
			t,
			"A shipment move lost its assignment and has nobody to run it.",
			event.Description,
		)
		return
	}

	t.Fatalf("%s is not a known event", agenttrigger.EventShipmentMoveUnassigned)
}

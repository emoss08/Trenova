package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestKnownEvents_EveryKindIsValidAndCarriesASubject(t *testing.T) {
	t.Parallel()

	events := agent.KnownEvents()
	assert.NotEmpty(t, events)

	seen := make(map[agent.EventKind]struct{}, len(events))
	for _, event := range events {
		assert.True(t, event.Kind.IsValid(), event.Kind)
		assert.True(t, event.SubjectType.IsValid(), event.Kind)
		assert.NotEmpty(t, event.Label, event.Kind)
		assert.NotEmpty(t, event.Description, event.Kind)
		_, duplicate := seen[event.Kind]
		assert.False(t, duplicate, "duplicate event kind %s", event.Kind)
		seen[event.Kind] = struct{}{}
	}

	assert.False(t, agent.EventKind("shipment.teleported").IsValid())
	assert.Equal(t, agent.SubjectBillingQueueItem, agent.EventBillingQueueItemException.SubjectType())
	assert.Equal(t, agent.SubjectInsight, agent.EventInsightDetected.SubjectType())
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

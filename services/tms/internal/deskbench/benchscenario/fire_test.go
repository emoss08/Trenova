package benchscenario

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fireScenario(fire *Fire) *Scenario {
	return &Scenario{Name: "unattended", Steps: []Step{{Fire: fire}}}
}

func TestFire_Validation(t *testing.T) {
	t.Parallel()

	event := &Fire{
		Agent:   string(agenttemplate.TemplateBillingException),
		Event:   agenttrigger.EventBillingQueueItemException,
		Subject: "select id from billing_queue_items limit 1",
	}
	require.NoError(t, fireScenario(event).Validate())
	assert.True(t, fireScenario(event).Unattended())

	schedule := &Fire{Agent: string(agenttemplate.TemplateLoadMonitor), Schedule: true}
	require.NoError(t, fireScenario(schedule).Validate())

	cases := map[string]*Fire{
		"an event the template does not answer": {
			Agent: string(
				agenttemplate.TemplateBillingException,
			), Event: agenttrigger.EventShipmentCreated,
			Subject: "select 1",
		},
		"an event run with no subject": {
			Agent: string(agenttemplate.TemplateBillingException),
			Event: agenttrigger.EventBillingQueueItemException,
		},
		"a scheduled agent fired by an event": {
			Agent: string(
				agenttemplate.TemplateLoadMonitor,
			), Event: agenttrigger.EventShipmentCreated,
		},
		"a chat agent": {
			Agent:    string(agenttemplate.TemplateDispatchAssistant),
			Schedule: true,
		},
		"an unknown template": {Agent: "NoSuchDesk", Schedule: true},
		"a subject that writes": {
			Agent: string(agenttemplate.TemplateBillingException),
			Event: agenttrigger.EventBillingQueueItemException, Subject: "delete from shipments",
		},
		"a ceiling that is no tier": {
			Agent: string(
				agenttemplate.TemplateLoadMonitor,
			), Schedule: true, Ceiling: "Sometimes",
		},
		"a tier that is no tier": {
			Agent: string(agenttemplate.TemplateLoadMonitor), Schedule: true,
			ToolTiers: map[string]agent.AutonomyTier{
				agentdefinition.CoreToolRaiseException: "Loud",
			},
		},
		"a tier on a tool the agent does not hold": {
			Agent: string(agenttemplate.TemplateLoadMonitor), Schedule: true,
			ToolTiers: map[string]agent.AutonomyTier{"invoice_post": agent.TierPropose},
		},
	}
	for name, fire := range cases {
		assert.Error(t, fireScenario(fire).Validate(), name)
	}

	shaped := &Fire{
		Agent:   string(agenttemplate.TemplateBillingException),
		Event:   agenttrigger.EventBillingQueueItemException,
		Subject: "select id from billing_queue_items limit 1",
		Ceiling: agent.TierPropose,
		ToolTiers: map[string]agent.AutonomyTier{
			agentdefinition.CoreToolRaiseException: agent.TierPropose,
		},
		StopWhileWaiting: true,
	}
	require.NoError(t, fireScenario(shaped).Validate(), "a core tool's tier may be set")
	assert.Equal(
		t,
		agent.TierPropose,
		shaped.AutonomyCeiling(agenttemplate.TemplateBillingException),
	)
	assert.Equal(t, agentpreset.StarterCeiling(agenttemplate.TemplateBillingException),
		event.AutonomyCeiling(agenttemplate.TemplateBillingException))

	twoSteps := fireScenario(schedule)
	twoSteps.Steps = append(twoSteps.Steps, Step{Say: "and then?"})
	require.Error(t, twoSteps.Validate())

	graded := &Scenario{Name: "chat", Steps: []Step{{Say: "hi", Expect: Expect{Run: "Completed"}}}}
	require.Error(t, graded.Validate(), "run belongs on a fire step")
}

package deskcase_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func holder(name string, tools ...string) *deskcase.Holder {
	held := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		held[tool] = struct{}{}
	}

	return &deskcase.Holder{ID: pulid.MustNew("agd_"), Name: name, Tools: held}
}

func permitted(tools ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		out[tool] = struct{}{}
	}

	return out
}

/*
A step is only worth sending to an agent that holds a tool for it, and only
when the person may use that tool. Shipment Assistant, asked to mark a
shipment ready to invoice, could only say it could not; the checklist now
hands that step to an agent that can, or opens the page where the person
does it.
*/
func TestAbilities(t *testing.T) {
	t.Parallel()

	shipments := holder("Shipment Assistant", "get_shipment_tracking")
	billing := holder("Billing Assistant", "transfer_to_billing", "email_customer")
	steps := []deskcase.StepKey{
		deskcase.StepTrackDelivery,
		deskcase.StepMarkReady,
		deskcase.StepSendInvoice,
		deskcase.StepKey("custom:callshipper"),
	}

	got := deskcase.Abilities(&deskcase.AbilityInputs{
		Steps:     steps,
		Own:       shipments,
		Others:    []*deskcase.Holder{billing},
		Permitted: permitted("get_shipment_tracking", "transfer_to_billing", "send_invoice"),
	})

	assert.Equal(t, deskcase.ViaAgent, got[deskcase.StepTrackDelivery].Via)
	require.Equal(t, deskcase.ViaAsk, got[deskcase.StepMarkReady].Via)
	assert.Equal(t, billing.ID, got[deskcase.StepMarkReady].AgentID)
	assert.Equal(t, "Billing Assistant", got[deskcase.StepMarkReady].AgentName)
	assert.Equal(t, deskcase.ViaPerson, got[deskcase.StepSendInvoice].Via,
		"no agent the person may use holds send_invoice")
	assert.Equal(t, deskcase.ViaAgent, got[deskcase.StepKey("custom:callshipper")].Via,
		"a step the organization added names no tool; the agent may try")
}

func TestAbilities_AToolThePersonMayNotUseIsNotTheAgents(t *testing.T) {
	t.Parallel()

	own := holder("Billing Assistant", "transfer_to_billing")

	got := deskcase.Abilities(&deskcase.AbilityInputs{
		Steps:     []deskcase.StepKey{deskcase.StepMarkReady},
		Own:       own,
		Permitted: permitted(),
	})

	assert.Equal(t, deskcase.ViaPerson, got[deskcase.StepMarkReady].Via)
}

func TestNeedsOthers(t *testing.T) {
	t.Parallel()

	own := holder("Billing Assistant", "transfer_to_billing")
	allowed := permitted("transfer_to_billing")

	assert.False(t, deskcase.NeedsOthers([]deskcase.StepKey{deskcase.StepMarkReady}, own, allowed),
		"the other agents are not read when the conversation's agent can take every step")
	assert.True(t, deskcase.NeedsOthers([]deskcase.StepKey{deskcase.StepSendInvoice}, own, allowed))
}

func TestChecklistSteps_NextFirstAndEachOnce(t *testing.T) {
	t.Parallel()

	checklist := &deskcase.Checklist{
		Next: deskcase.StepRequestPOD,
		Items: []*deskcase.Item{
			{Key: deskcase.ItemDelivered, Step: deskcase.StepTrackDelivery},
			{Key: deskcase.ItemPOD, Step: deskcase.StepRequestPOD},
			{Key: deskcase.ItemSent},
		},
	}

	assert.Equal(t,
		[]deskcase.StepKey{deskcase.StepRequestPOD, deskcase.StepTrackDelivery},
		checklist.Steps())
	assert.Nil(t, (*deskcase.Checklist)(nil).Steps())
}

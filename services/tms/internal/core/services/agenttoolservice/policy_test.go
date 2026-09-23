package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/stretchr/testify/assert"
)

func callLimit(call serviceports.CallPolicy) agent.AutonomyTier {
	limit := call.Egress.Ceiling()
	if call.MaxTier.IsValid() {
		limit = limit.AtMost(call.MaxTier)
	}

	return limit
}

// Every tool that sends something outside the organization declares that a
// person approves it, whatever it earns.
func TestEgressTools_NeverRunWithoutAPerson(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentTool{
		&emailCustomerTool{},
		&sendDetentionNoticeTool{},
		&notifyDriverTool{},
		&requestMissingDocsTool{},
		&requestCredentialRenewalTool{},
		&tenderToRoutingGuideTool{},
		&tenderToCarriersTool{},
		&replyToInboundMessageTool{},
		&scheduleReportTool{},
		&cancelShipmentTool{},
		&updateShipmentTool{},
		&resolveServiceFailureTool{},
		&rejectWorkerPTOTool{},
		&cancelWorkerPTOTool{},
	} {
		policy := tool.Policy()
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
		assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(policy), tool.Name())
	}

	assert.Equal(t, agent.TierAutoExecute,
		agenttoolpolicy.Promotable((&addShipmentCommentTool{}).Policy()),
		"an internal note has no ceiling of its own")
}

// A note on the internal thread changes nothing and stays automatic; one a
// customer or a driver will read waits for a person.
func TestAddShipmentComment_HoldsANoteOutsidersReadForApproval(t *testing.T) {
	t.Parallel()

	policy := (&addShipmentCommentTool{}).Policy()
	cases := map[string]struct {
		class agent.EgressClass
		limit agent.AutonomyTier
	}{
		"":           {class: agent.EgressInternal, limit: agent.TierAutoExecute},
		"Internal":   {class: agent.EgressInternal, limit: agent.TierAutoExecute},
		"Operations": {class: agent.EgressInternal, limit: agent.TierAutoExecute},
		"Accounting": {class: agent.EgressInternal, limit: agent.TierAutoExecute},
		"Customer":   {class: agent.EgressCustomerVisible, limit: agent.TierActWithApproval},
		"Driver":     {class: agent.EgressDriverVisible, limit: agent.TierActWithApproval},
	}
	for visibility, want := range cases {
		params := serviceports.ToolExecuteParams{Params: map[string]any{"visibility": visibility}}
		call := policy.Classified(params)
		assert.Equal(t, want.class, call.Egress, visibility)
		assert.Equal(t, want.limit, callLimit(call), visibility)
	}
}

// A private table view is the person's own, as a private report is; a shared
// one, or one no person is driving, is not.
func TestSaveTableView_ClassifiesAPrivateViewAsPersonal(t *testing.T) {
	t.Parallel()

	policy := (&saveTableViewTool{}).Policy()
	assert.True(t, policy.PersonalRunsUnasked)

	person := personParams(map[string]any{"entity": "shipment"})
	assert.Equal(t, agent.EgressPersonal, policy.Classified(person).Egress)

	shared := personParams(map[string]any{"entity": "shipment", "shared": true})
	assert.Equal(t, agent.EgressInternal, policy.Classified(shared).Egress)

	agentCall := personParams(map[string]any{"entity": "shipment"})
	agentCall.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	assert.Equal(t, agent.EgressInternal, policy.Classified(agentCall).Egress)
}

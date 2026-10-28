package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/alerttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/inboundtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/tendertools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
)

// Every tool that sends something outside the organization declares that a
// person approves it, whatever it earns.
func TestEgressTools_NeverRunWithoutAPerson(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		&operationstools.EmailCustomerTool{},
		&operationstools.SendDetentionNoticeTool{},
		&operationstools.NotifyDriverTool{},
		&workforcetools.RequestMissingDocsTool{},
		&masterdatatools.RequestCredentialRenewalTool{},
		&tendertools.TenderToRoutingGuideTool{},
		&tendertools.TenderToCarriersTool{},
		&inboundtools.ReplyToInboundMessageTool{},
		&alerttools.ScheduleReportTool{},
		&shipmenttools.CancelShipmentTool{},
		&operationstools.UpdateShipmentTool{},
		&operationstools.ResolveServiceFailureTool{},
		&operationstools.RejectWorkerPTOTool{},
		&operationstools.CancelWorkerPTOTool{},
	} {
		policy := tool.Policy()
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, tool.Name())
		assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(policy), tool.Name())
	}

	assert.Equal(t, agent.TierAutoExecute,
		agenttoolpolicy.Promotable((&shipmenttools.AddShipmentCommentTool{}).Policy()),
		"an internal note has no ceiling of its own")
}

// A note on the internal thread changes nothing and stays automatic; one a
// customer or a driver will read waits for a person.
func TestAddShipmentComment_HoldsANoteOutsidersReadForApproval(t *testing.T) {
	t.Parallel()

	policy := (&shipmenttools.AddShipmentCommentTool{}).Policy()
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
		params := agenttoolports.ToolExecuteParams{Params: map[string]any{"visibility": visibility}}
		call := policy.Classified(params)
		assert.Equal(t, want.class, call.Egress, visibility)
		assert.Equal(t, want.limit, agenttoolstest.CallLimit(call), visibility)
	}
}

// A private table view is the person's own, as a private report is; a shared
// one, or one no person is driving, is not.
func TestSaveTableView_ClassifiesAPrivateViewAsPersonal(t *testing.T) {
	t.Parallel()

	policy := (&masterdatatools.SaveTableViewTool{}).Policy()
	assert.True(t, policy.PersonalRunsUnasked)

	person := agenttoolstest.PersonParams(map[string]any{"entity": "shipment"})
	assert.Equal(t, agent.EgressPersonal, policy.Classified(person).Egress)

	shared := agenttoolstest.PersonParams(map[string]any{"entity": "shipment", "shared": true})
	assert.Equal(t, agent.EgressInternal, policy.Classified(shared).Egress)

	agentCall := agenttoolstest.PersonParams(map[string]any{"entity": "shipment"})
	agentCall.Actor.PrincipalType = permissionports.PrincipalTypeAgent
	assert.Equal(t, agent.EgressInternal, policy.Classified(agentCall).Egress)
}

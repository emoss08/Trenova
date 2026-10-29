package triageworktools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

var triageWorkToolCases = []agenttoolstest.TriageToolCase{
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newDraftInboundReplyTool(f, f)
		},
		Name:     "inbox_draft_reply",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newDiscardInboundReplyDraftTool(f)
		},
		Name:     "inbox_discard_reply_draft",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newDraftInboundLoadTool(f, f)
		},
		Name:     "inbox_draft_load",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newCreateInboundLoadTool(f)
		},
		Name:     "inbox_create_load",
		Required: []string{"messageId", "fields"}, Resource: permission.ResourceShipment,
		Operation: permission.OpCreate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newDiscardInboundLoadTool(f, f)
		},
		Name:     "inbox_discard_load",
		Required: []string{"messageId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newFileInboundAttachmentTool(f, f)
		},
		Name:     "inbox_file_attachment",
		Required: []string{"messageId", "attachmentId", "shipmentId"},
		Resource: permission.ResourceDocument, Operation: permission.OpUpdate,
		Egress: agent.EgressInternal, MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newUnfileInboundAttachmentTool(f, f)
		},
		Name:     "inbox_unfile_attachment",
		Required: []string{"messageId", "attachmentId"}, Resource: permission.ResourceDocument,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newLogInboundExceptionTool(f, f)
		},
		Name:     "inbox_record_discrepancy",
		Required: []string{"messageId", "kind", "quantity", "unit"},
		Resource: permission.ResourceShipmentHold, Operation: permission.OpCreate,
		Egress: agent.EgressInternal, MaxTier: agent.TierAutoExecute, Condition: true,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newSettleDetentionDisputeTool(f, f)
		},
		Name:     "detention_resolve_dispute",
		Required: []string{"messageId", "decision"}, Resource: permission.ResourceDetention,
		Operation: permission.OpUpdate, Egress: agent.EgressMoney,
		MaxTier: agent.TierPropose,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newResolveInboundConflictTool(agenttoolstest.FakeConflictDesk{f})
		},
		Name:     "inbox_resolve_conflict",
		Required: []string{"messageId", "conflictId", "resolution"},
		Resource: permission.ResourceInboundMessage, Operation: permission.OpUpdate,
		Egress: agent.EgressInternal, MaxTier: agent.TierActWithApproval,
	},
	{
		Build: func(f *agenttoolstest.FakeTriage) agenttoolports.AgentTool {
			return newReopenInboundConflictTool(f, agenttoolstest.FakeConflictDesk{f})
		},
		Name:     "inbox_reopen_conflict",
		Required: []string{"messageId", "conflictId"}, Resource: permission.ResourceInboundMessage,
		Operation: permission.OpUpdate, Egress: agent.EgressInternal,
		MaxTier: agent.TierAutoExecute, Condition: true,
	},
}

func TestTriageWorkTools_DeclareTheirSchemaAndPolicy(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolSchemaAndPolicy(t, triageWorkToolCases)
}

func TestTriageWorkTools_RunOnlyAsFarAsTheMailboxAllows(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolMailboxCondition(t, triageWorkToolCases)
}

func TestTriageWorkTools_NameTheMessageTheyChange(t *testing.T) {
	t.Parallel()

	agenttoolstest.CheckTriageToolTargets(t, triageWorkToolCases)
}

/*
Settling a disputed detention charge gives up or keeps revenue, so a person
always decides; drafting a reply only drafts, and nothing in the set sends.
*/
func TestTriageWorkTools_KeepTheirCaps(t *testing.T) {
	t.Parallel()

	policies := agenttoolstest.TriageToolPolicies(
		agenttoolstest.NewFakeTriage(),
		triageWorkToolCases,
	)
	for name, policy := range policies {
		assert.Falsef(t, policy.HasEgress(agent.EgressExternalRecipient),
			"%s must not send anything outside the organization", name)
	}

	settle := policies["detention_resolve_dispute"]
	assert.Equal(t, agent.TierPropose, agenttoolpolicy.Promotable(&settle))
	assert.True(t, settle.HeldWhenTainted(), "a decision that moves money waits once tainted")

	draft := policies["inbox_draft_reply"]
	assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, draft.Egress,
		"a draft is only seen inside the organization")

	create := policies["inbox_create_load"]
	assert.Equal(t, agent.TierActWithApproval, agenttoolpolicy.Promotable(&create),
		"a load is never created unattended")
	assert.True(t, create.Idempotent)
}

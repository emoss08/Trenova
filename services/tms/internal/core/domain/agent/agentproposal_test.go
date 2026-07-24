package agent_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func baseProposal() *agent.AgentProposal {
	return &agent.AgentProposal{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		RunID:          pulid.MustNew("ar_"),
		ToolName:       "transition_item_to_in_review",
		Rationale:      "The item is ready for review",
		AutonomyTier:   agent.TierPropose,
		Status:         agent.ProposalStatusPending,
		Confidence:     decimal.NewFromFloat(0.9),
	}
}

func TestAgentProposal_Validate_RequiresEvidence(t *testing.T) {
	proposal := baseProposal()
	proposal.Evidence = nil

	me := errortypes.NewMultiError()
	proposal.Validate(me)

	require.True(t, me.HasErrors(), "a proposal with no evidence must fail validation")
	require.Contains(t, me.Error(), "evidence")
}

func TestAgentProposal_Validate_RejectsEvidenceWithBlankFields(t *testing.T) {
	proposal := baseProposal()
	proposal.Evidence = []agent.EvidenceRef{{Type: "", ID: ""}}

	me := errortypes.NewMultiError()
	proposal.Validate(me)

	require.True(t, me.HasErrors(), "evidence with blank type/id must fail validation")
}

func TestAgentProposal_Validate_PassesWithEvidence(t *testing.T) {
	proposal := baseProposal()
	proposal.Evidence = []agent.EvidenceRef{
		{Type: "billing_queue_item", ID: "bqi_123", Note: "blocked item"},
	}

	me := errortypes.NewMultiError()
	proposal.Validate(me)

	require.False(t, me.HasErrors(), "a proposal with evidence must pass validation")
}

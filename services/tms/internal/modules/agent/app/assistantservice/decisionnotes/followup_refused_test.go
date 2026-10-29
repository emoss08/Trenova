package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnchorChanged_ReadsOnlyTheRefusedRecordsNotAlreadyInPlay(t *testing.T) {
	t.Parallel()

	inPlay := pulid.MustNew("shp_").String()
	unread := pulid.MustNew("shp_").String()
	anchors := &assistanttest.RecordedAnchors{}
	svc := &Notes{Anchors: anchors}
	actor := assistanttest.Actor()

	got := svc.AnchorChanged(t.Context(),
		[]agentturn.RuntimeAnchor{{Kind: "shipment", ID: inPlay}},
		[]agentpagecontext.EntityRef{{ID: inPlay}, {ID: unread}, {ID: unread}},
		actor, actor.TenantInfo())

	require.Len(t, anchors.Asked, 1)
	require.Len(t, anchors.Asked[0].Records, 1, "a record already in play is not read twice")
	assert.Equal(t, unread, anchors.Asked[0].Records[0].ID)
	require.Len(t, got, 2)
	assert.Equal(t, unread, got[1].ID)

	untouched := svc.AnchorChanged(t.Context(), got, nil, actor, actor.TenantInfo())
	assert.Equal(t, got, untouched)
	assert.Len(t, anchors.Asked, 1, "nothing refused, nothing read")
}

func TestRefusedOverChange_IsOnlyAnApprovalRefusedOverAMovedRecord(t *testing.T) {
	t.Parallel()

	moved := assistanttest.RefusedProposal()
	failed := &agent.AgentProposal{
		ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusExecutionFailed,
		ExecutionError: "validation failed: bol: taken",
	}
	executed := &agent.AgentProposal{ID: pulid.MustNew("ap_"), Status: agent.ProposalStatusExecuted}

	refused := refusedOverChange([]*agent.AgentProposal{failed, moved, executed, nil})

	require.Len(t, refused, 1)
	assert.Equal(t, moved.ID, refused[0].ID)
}

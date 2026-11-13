package proposalexecutor

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

// What an approved proposal writes is the agent's work. The audit entry used
// to read as the approver's own edit, so nothing could tell a person later
// changing what an agent wrote from a person changing their own work.
func TestApprovedAuditSource_MarksTheWriteAsTheAgentsWork(t *testing.T) {
	t.Parallel()

	runID, agentID := pulid.MustNew("ar_"), pulid.MustNew("agdef_")

	inRun := auditctx.From(approvedAuditSource(t.Context(), &approvedRun{
		proposal: &agent.AgentProposal{RunID: runID},
		params:   &agenttoolports.ToolExecuteParams{DefinitionID: agentID},
	}))
	assert.Equal(t, auditctx.Source{
		Channel: auditctx.ChannelAgentRun,
		AIRunID: runID.String(),
		AgentID: agentID.String(),
	}, inRun)

	inConversation := auditctx.From(approvedAuditSource(t.Context(), &approvedRun{
		proposal: &agent.AgentProposal{RunID: runID},
		params: &agenttoolports.ToolExecuteParams{
			DefinitionID: agentID,
			ThreadID:     pulid.MustNew("athr_"),
		},
	}))
	assert.Equal(t, auditctx.ChannelAgentConversation, inConversation.Channel)

	none := auditctx.From(approvedAuditSource(t.Context(), &approvedRun{
		proposal: &agent.AgentProposal{},
		params:   &agenttoolports.ToolExecuteParams{},
	}))
	assert.Equal(t, auditctx.Source{}, none, "a proposal with no run names none")
}

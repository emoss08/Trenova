package decisionnotes

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestDecisionLine_FencesTheFailureAndTheHeadlineShowsItsWords(t *testing.T) {
	t.Parallel()

	line := decisionLine(&agent.AgentProposal{
		ToolName:       "shipment_create",
		Status:         agent.ProposalStatusExecutionFailed,
		ExecutionError: "name taken",
	})

	assert.Equal(t,
		"Approved shipment_create, but it failed when it ran: "+
			toolresult.FenceInline("name taken"),
		line,
	)
	assert.Equal(t,
		"Approved shipment_create, but it failed when it ran: name taken",
		DecisionHeadline(line+"\nDecision on proposal ap_1 (shipment_create)."),
	)
}

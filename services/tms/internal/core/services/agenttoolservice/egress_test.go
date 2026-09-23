package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
)

// Every tool that sends something outside the organization declares that a
// person approves it, whatever it earns.
func TestEgressTools_NeverRunWithoutAPerson(t *testing.T) {
	t.Parallel()

	for _, tool := range []any{
		&emailCustomerTool{},
		&sendDetentionNoticeTool{},
		&notifyDriverTool{},
		&requestMissingDocsTool{},
		&requestCredentialRenewalTool{},
		&tenderToRoutingGuideTool{},
		&tenderToCarriersTool{},
		&replyToInboundMessageTool{},
	} {
		assert.Equal(t, agent.TierActWithApproval, serviceports.CeilingOf(tool), "%T", tool)
	}

	assert.Equal(t, agent.TierAutoExecute, serviceports.CeilingOf(&addShipmentCommentTool{}),
		"an internal note has no ceiling of its own")
}

// A note on the internal thread changes nothing and stays automatic; one a
// customer or a driver will read waits for a person.
func TestAddShipmentComment_HoldsANoteOutsidersReadForApproval(t *testing.T) {
	t.Parallel()

	tool := &addShipmentCommentTool{}
	cases := map[string]agent.AutonomyTier{
		"":           agent.TierAutoExecute,
		"Internal":   agent.TierAutoExecute,
		"Operations": agent.TierAutoExecute,
		"Accounting": agent.TierAutoExecute,
		"Customer":   agent.TierActWithApproval,
		"Driver":     agent.TierActWithApproval,
	}
	for visibility, want := range cases {
		params := serviceports.ToolExecuteParams{Params: map[string]any{"visibility": visibility}}
		assert.Equal(t, want, tool.TierLimit(t.Context(), params), visibility)
	}
}

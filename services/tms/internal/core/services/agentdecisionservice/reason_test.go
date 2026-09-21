package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
)

// The chat card asks for a click, not a reason. A decision without one is
// still a decision, and is recorded under a code that says as much.
func TestReasonCodeFor_FillsInACodeWhenNoneWasGiven(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "approved_without_reason", reasonCodeFor(agent.DecisionAccepted, ""))
	assert.Equal(t, "modified_without_reason", reasonCodeFor(agent.DecisionModified, "  "))
	assert.Equal(t, "rejected_without_reason", reasonCodeFor(agent.DecisionRejected, ""))
	assert.Equal(t, "Wrong trailer", reasonCodeFor(agent.DecisionRejected, " Wrong trailer "))
}

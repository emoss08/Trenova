package assistantjobs

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/agentguard"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

// Preparing a turn is timed by how it ended, because a refusal is decided in
// milliseconds and an answered question carries the context build: one
// histogram of both would hide the slow one.
func TestPrepareOutcome(t *testing.T) {
	t.Parallel()

	allowed := &assistantservice.TurnPlan{Decision: agentguard.Decision{Allowed: true}}
	refused := &assistantservice.TurnPlan{Decision: agentguard.Decision{Allowed: false}}

	assert.Equal(t, "allowed", prepareOutcome(allowed, nil))
	assert.Equal(t, "refused", prepareOutcome(refused, nil))
	assert.Equal(t, "rejected", prepareOutcome(nil, errortypes.NewBusinessError("Agent is disabled")))
	assert.Equal(t, "failed", prepareOutcome(nil, errors.New("connection refused")))
}

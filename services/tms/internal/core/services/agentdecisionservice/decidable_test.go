package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A decision used to be recorded against whatever state the proposal was in,
// so a second click or a retry accepted a rejected proposal, or approved an
// approved one again and ran its write twice.
func TestDecidable_RefusesAProposalAlreadyDecided(t *testing.T) {
	t.Parallel()

	for _, status := range []agent.ProposalStatus{
		agent.ProposalStatusAccepted,
		agent.ProposalStatusRejected,
		agent.ProposalStatusExecuted,
		agent.ProposalStatusExpired,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			err := decidable(&agent.AgentProposal{Status: status})
			require.Error(t, err)
			assert.True(t, errortypes.IsBusinessError(err))
			assert.Contains(t, err.Error(), "already been decided")
		})
	}
}

func TestDecidable_AllowsAPendingProposal(t *testing.T) {
	t.Parallel()

	assert.NoError(t, decidable(&agent.AgentProposal{Status: agent.ProposalStatusPending}))
}

// The sweeper marks these every quarter hour; between sweeps the clock is the
// authority, so a proposal is never approved in the minutes after its window
// closed just because the row had not caught up.
func TestDecidable_RefusesAPendingProposalPastItsExpiry(t *testing.T) {
	t.Parallel()

	err := decidable(&agent.AgentProposal{
		Status:    agent.ProposalStatusPending,
		ExpiresAt: 1, // 1970: as expired as it gets
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// A proposal made before expiry existed has none, and never expires on its own.
func TestDecidable_AllowsAPendingProposalWithNoExpiry(t *testing.T) {
	t.Parallel()

	assert.NoError(t, decidable(&agent.AgentProposal{Status: agent.ProposalStatusPending, ExpiresAt: 0}))
}

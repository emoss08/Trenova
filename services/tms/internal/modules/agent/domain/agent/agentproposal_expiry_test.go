package agent

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// A proposal is a judgement about the world as it was. It gets a window on
// insert, whichever path created it, so no creation site can forget.
func TestAgentProposal_GetsAnExpiryOnInsert(t *testing.T) {
	t.Parallel()

	proposal := &AgentProposal{}
	require.NoError(t, proposal.BeforeAppendModel(t.Context(), &bun.InsertQuery{}))

	expected := timeutils.NowUnix() + int64(DefaultProposalTTL.Seconds())
	assert.InDelta(t, expected, proposal.ExpiresAt, 2)
}

func TestAgentProposal_KeepsAnExpiryAlreadySet(t *testing.T) {
	t.Parallel()

	proposal := &AgentProposal{ExpiresAt: 42}
	require.NoError(t, proposal.BeforeAppendModel(t.Context(), &bun.InsertQuery{}))

	assert.Equal(t, int64(42), proposal.ExpiresAt)
}

func TestAgentProposal_Expired(t *testing.T) {
	t.Parallel()

	now := time.Now().Unix()
	assert.True(t, (&AgentProposal{ExpiresAt: now - 1}).Expired(now))
	assert.True(t, (&AgentProposal{ExpiresAt: now}).Expired(now), "the boundary is closed")
	assert.False(t, (&AgentProposal{ExpiresAt: now + 1}).Expired(now))
	assert.False(t, (&AgentProposal{}).Expired(now), "no expiry never expires")
}

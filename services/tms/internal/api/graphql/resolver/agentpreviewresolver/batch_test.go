package agentpreviewresolver

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func batchOf(h *previewHarness, n int) []string {
	ids := make([]string, 0, n)
	for range n {
		proposal := &agent.AgentProposal{ID: pulid.MustNew("ap_")}
		h.proposals.Batch = append(h.proposals.Batch, proposal)
		ids = append(ids, proposal.ID.String())
	}

	return ids
}

// A batch preview needs the same right as one proposal's preview, and reads
// nothing for a reader without it.
func TestAgentProposalPreviews_NeedsProposalRead(t *testing.T) {
	t.Parallel()

	denied := newPreviewHarness(t)
	_, err := denied.resolver.AgentProposalPreviews(denied.ctx, batchOf(denied, 2))
	require.Error(t, err)
	assert.Empty(t, denied.previews.Batched)

	allowed := newPreviewHarness(t, proposalRead)
	results, err := allowed.resolver.AgentProposalPreviews(allowed.ctx, batchOf(allowed, 2))
	require.NoError(t, err)
	assert.Len(t, results, 2)
	require.NotNil(t, allowed.previews.Viewer)
	assert.NotNil(t, allowed.previews.Viewer.Reads, "reads are checked for this reader")
}

// Results come back in the order asked. A proposal that is not there says
// so; one whose preview failed for a reason written for the reader gives
// it, and an internal failure is never shown.
func TestAgentProposalPreviews_KeepsTheOrderAndSaysWhatFailed(t *testing.T) {
	t.Parallel()

	h := newPreviewHarness(t, proposalRead)
	ids := batchOf(h, 3)
	missing := pulid.MustNew("ap_").String()
	refused, _ := pulid.MustParse(ids[1])
	broken, _ := pulid.MustParse(ids[2])
	h.previews.Failures = map[pulid.ID]error{
		refused: errortypes.NewBusinessError("The shipment is closed"),
		broken:  errors.New("pq: connection reset by peer"),
	}
	asked := []string{ids[2], missing, ids[0], ids[1]}

	results, err := h.resolver.AgentProposalPreviews(h.ctx, asked)
	require.NoError(t, err)
	require.Len(t, results, 4)
	for i, result := range results {
		assert.Equal(t, asked[i], result.ProposalID.String())
	}

	errorOf := func(i int) *string {
		t.Helper()
		message, rErr := (&AgentProposalPreviewResultResolver{h.resolver.Deps}).Error(h.ctx, results[i])
		require.NoError(t, rErr)

		return message
	}
	require.NotNil(t, errorOf(0))
	assert.Equal(t, "This change could not be previewed. Open it on its own.", *errorOf(0),
		"an internal failure is not shown")
	assert.Nil(t, results[0].Preview)
	require.NotNil(t, errorOf(1))
	assert.Equal(t, "Proposal not found", *errorOf(1))
	assert.Nil(t, errorOf(2))
	require.NotNil(t, results[2].Preview)
	assert.Equal(t, "d-"+ids[0], results[2].Preview.Digest)
	require.NotNil(t, errorOf(3))
	assert.Equal(t, "The shipment is closed", *errorOf(3))
	assert.Equal(t, []pulid.ID{broken, pulid.ID(ids[0]), refused}, h.previews.Batched,
		"only the proposals found are previewed, in the order asked")
}

// A batch names between one and agentdecision.MaxBatch proposals, each once.
func TestAgentProposalPreviews_RefusesABatchOutOfBounds(t *testing.T) {
	t.Parallel()

	h := newPreviewHarness(t, proposalRead)
	ids := batchOf(h, agentdecision.MaxBatch+1)

	for name, asked := range map[string][]string{
		"empty":     {},
		"too many":  ids,
		"duplicate": {ids[0], ids[1], ids[0]},
	} {
		_, err := h.resolver.AgentProposalPreviews(h.ctx, asked)
		require.Error(t, err, name)
		assert.True(t, errortypes.IsMultiError(err), name)
	}
	assert.Empty(t, h.previews.Batched, "nothing is previewed for a refused batch")
}

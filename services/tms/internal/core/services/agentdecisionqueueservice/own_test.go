package agentdecisionqueueservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type owningDecider struct {
	stubDecider

	mine    map[pulid.ID]bool
	checked []pulid.ID
}

func (d *owningDecider) AssertOwnProposal(
	_ context.Context,
	proposalID pulid.ID,
	_ pagination.TenantInfo,
	_ *services.RequestActor,
) error {
	d.checked = append(d.checked, proposalID)
	if !d.mine[proposalID] {
		return errortypes.NewNotFoundError(
			"That proposal was not raised in one of your conversations",
		)
	}

	return nil
}

func TestDecideManyOwn_DecidesTheCallersOwnProposalsOfOneTool(t *testing.T) {
	t.Parallel()

	first, second := proposal("post_invoice"), proposal("post_invoice")
	proposals := &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{
		first.ID: first, second.ID: second,
	}}
	decider := &owningDecider{
		stubDecider: stubDecider{digests: map[pulid.ID]string{}},
		mine:        map[pulid.ID]bool{first.ID: true, second.ID: true},
	}
	svc := newService(proposals, &decider.stubDecider)
	svc.decisions = decider

	results, err := svc.DecideManyOwn(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs:    []pulid.ID{first.ID, second.ID},
		Decision:       agent.DecisionAccepted,
		PreviewDigests: map[pulid.ID]string{first.ID: "sha256:first"},
	}, actor())
	require.NoError(t, err)

	require.Len(t, results, 2)
	assert.True(t, results[0].Executed)
	assert.True(t, results[1].Executed)
	assert.Equal(t, []pulid.ID{first.ID, second.ID}, decider.checked)
	assert.Equal(t, "sha256:first", decider.digests[first.ID],
		"each approval carries the digest of the preview it was shown")
	assert.Empty(t, decider.digests[second.ID])
}

func TestDecideManyOwn_RefusesTheWholeBatchWhenOneIsNotTheCallers(t *testing.T) {
	t.Parallel()

	mine, theirs := proposal("post_invoice"), proposal("post_invoice")
	proposals := &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{
		mine.ID: mine, theirs.ID: theirs,
	}}
	decider := &owningDecider{mine: map[pulid.ID]bool{mine.ID: true}}
	svc := newService(proposals, &decider.stubDecider)
	svc.decisions = decider

	_, err := svc.DecideManyOwn(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{mine.ID, theirs.ID},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.Error(t, err)
	assert.True(t, errortypes.IsMultiError(err))
	assert.Contains(t, err.Error(), "proposalIds[1]")
	assert.Empty(t, decider.decided, "nothing is decided when one is not the caller's")
}

func TestDecideManyOwn_KeepsTheBatchRules(t *testing.T) {
	t.Parallel()

	first, other := proposal("post_invoice"), proposal("send_invoice")
	proposals := &stubProposals{byID: map[pulid.ID]*agent.AgentProposal{
		first.ID: first, other.ID: other,
	}}
	decider := &owningDecider{mine: map[pulid.ID]bool{first.ID: true, other.ID: true}}
	svc := newService(proposals, &decider.stubDecider)
	svc.decisions = decider

	_, err := svc.DecideManyOwn(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: []pulid.ID{first.ID, other.ID},
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.ErrorContains(t, err, "one tool at a time")
	decider.checked = nil

	many := make([]pulid.ID, 0, MaxBatch+1)
	for range MaxBatch + 1 {
		many = append(many, pulid.MustNew("ap_"))
	}
	_, err = svc.DecideManyOwn(t.Context(), &services.DecideAgentProposalsRequest{
		ProposalIDs: many,
		Decision:    agent.DecisionAccepted,
	}, actor())
	require.ErrorContains(t, err, "at most 50")
	assert.Empty(t, decider.checked, "an oversized batch is refused before any lookup")
	assert.Empty(t, decider.decided)
}

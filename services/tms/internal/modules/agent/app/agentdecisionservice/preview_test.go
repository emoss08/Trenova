package agentdecisionservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePreviews answers as the preview service would: stale when the pinned
// record has moved, and with the digest of what it shows.
type fakePreviews struct {
	agentruntimeports.ProposalPreviewService

	stale  bool
	digest string
	asked  []*agentruntimeports.ProposalPreviewRequest
}

func (f *fakePreviews) ForProposal(
	_ context.Context,
	req *agentruntimeports.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	f.asked = append(f.asked, req)
	version := int64(3)
	preview := &agent.ProposalPreview{
		Schema:        agent.PreviewSchemaVersion,
		ProposalID:    req.Proposal.ID,
		Tool:          req.Proposal.ToolName,
		Coverage:      agent.PreviewCoverageFull,
		TargetVersion: &version,
		Digest:        f.digest,
		Staleness:     &agent.PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: 3},
	}
	if f.stale {
		preview.Staleness.CurrentVersion = 5
	}

	return preview, nil
}

type statusRecorder struct {
	*traceProposals

	updated int
}

func (p *statusRecorder) UpdateStatus(
	ctx context.Context,
	req agentports.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	p.updated++

	return p.traceProposals.UpdateStatus(ctx, req)
}

func previewHarness(t *testing.T, previews *fakePreviews) (*decideHarness, *statusRecorder) {
	t.Helper()

	h := newDecideHarness(t)
	recorder := &statusRecorder{traceProposals: h.proposals}
	h.service.proposalRepo = recorder
	h.service.previews = previews

	return h, recorder
}

func decideWith(
	t *testing.T,
	h *decideHarness,
	decision agent.DecisionType,
	digest string,
) (*agentports.DecisionOutcome, error) {
	t.Helper()

	return h.service.DecideWithOutcome(t.Context(), &agentports.DecideAgentProposalRequest{
		ProposalID:    h.proposals.proposal.ID,
		Decision:      decision,
		ReasonCode:    "looks_right",
		PreviewDigest: digest,
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}, h.actor)
}

func TestDecide_AStaleApprovalIsRecordedAsRefusedAndReported(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("a", 64)
	h, recorder := previewHarness(t, &fakePreviews{stale: true, digest: digest})
	followUps := &recordedFollowUps{}
	h.service.followUps = followUps

	outcome, err := decideWith(t, h, agent.DecisionAccepted, "")
	require.NoError(t, err, "the refusal is the decision's outcome, not an error to the caller")

	require.ErrorIs(t, outcome.ExecutionError, agent.ErrTargetChanged)
	assert.Contains(t, outcome.ExecutionError.Error(),
		"was at version 3 when this was proposed and is at version 5 now, so nothing was changed")
	assert.Nil(t, outcome.ExecutedTargetVersion)

	created := h.decisions.created
	require.NotNil(t, created, "the approval is recorded")
	assert.Equal(t, agent.DecisionAccepted, created.Decision)
	require.NotNil(t, created.Preview)
	assert.True(t, created.Preview.IsStale(), "with the preview it was refused on")
	assert.Equal(t, digest, created.PreviewDigest)
	assert.False(t, created.PreviewReviewed)
	assert.Nil(t, created.CommitsAt)

	assert.Equal(t, 1, recorder.updated, "the proposal leaves Pending once, conditionally")
	require.Len(t, h.proposals.executed, 1)
	refused := h.proposals.executed[0]
	assert.Equal(t, agent.ProposalStatusExecutionFailed, refused.Status,
		"the proposal is closed as not carried out")
	assert.True(t, agent.TargetChanged(refused.ExecutionError))
	assert.True(t, refused.ExecutedByUserID.IsNil(), "no write ran as anyone")

	require.Len(t, followUps.requests, 1, "the conversation is told why nothing happened")
	assert.Equal(t, h.proposals.proposal.ID, followUps.requests[0].ProposalID)
}

func TestDecide_AStaleApprovalIsRefusedWhateverDigestItWasReviewedOn(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{stale: true, digest: strings.Repeat("a", 64)})

	outcome, err := decideWith(t, h, agent.DecisionAccepted, strings.Repeat("b", 64))
	require.NoError(t, err, "a record that moved on is a refusal, not a preview conflict")
	require.ErrorIs(t, outcome.ExecutionError, agent.ErrTargetChanged)
}

func TestDecide_AStaleRefusalLeavesTheTrustLedgerAlone(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{stale: true, digest: strings.Repeat("a", 64)})
	trust := &recordedTrust{}
	h.service.trust = trust

	_, err := decideWith(t, h, agent.DecisionAccepted, "")
	require.NoError(t, err)
	assert.Zero(t, trust.decisions)
	assert.Zero(t, trust.failures, "a record someone else moved is not the tool's failure")
}

type recordedTrust struct {
	agentports.AgentTrustService

	decisions int
	failures  int
}

func (r *recordedTrust) RecordDecision(
	context.Context,
	*agent.AgentProposal,
	*agent.AgentDecision,
) error {
	r.decisions++

	return nil
}

func (r *recordedTrust) RecordExecutionFailure(context.Context, *agent.AgentProposal) error {
	r.failures++

	return nil
}

func TestDecide_AStalePlanStepIsRefusedEvenWhereItWouldNeedReview(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{})
	h.proposals.proposal.EgressClass = agent.EgressMoney
	version := int64(5)
	step := &agent.ProposalPreview{
		ProposalID:    h.proposals.proposal.ID,
		TargetVersion: &version,
		Staleness:     &agent.PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: 5},
	}

	outcome, err := h.service.DecideWithOutcome(t.Context(), &agentports.DecideAgentProposalRequest{
		ProposalID:  h.proposals.proposal.ID,
		Decision:    agent.DecisionAccepted,
		ReasonCode:  "decided_as_plan",
		WithinPlan:  true,
		StepPreview: step,
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}, h.actor)
	require.NoError(t, err)
	require.ErrorIs(t, outcome.ExecutionError, agent.ErrTargetChanged)
	require.Len(t, h.proposals.executed, 1)
	assert.Equal(t, agent.ProposalStatusExecutionFailed, h.proposals.executed[0].Status)
}

func TestDecide_ADigestThatNoLongerMatchesIsAConflictAndWritesNothing(t *testing.T) {
	t.Parallel()

	h, recorder := previewHarness(t, &fakePreviews{digest: strings.Repeat("a", 64)})

	_, err := decideWith(t, h, agent.DecisionAccepted, strings.Repeat("b", 64))

	require.Error(t, err)
	var conflict *errortypes.ConflictError
	require.True(t, errors.As(err, &conflict), "a changed preview is a conflict: %v", err)
	assert.Nil(t, h.decisions.created)
	assert.Zero(t, recorder.updated)
	assert.Empty(t, h.proposals.executed)
}

func TestDecide_RecordsWhatTheApproverSawAndWhetherTheyReviewedIt(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("c", 64)
	previews := &fakePreviews{digest: digest}
	h, _ := previewHarness(t, previews)

	outcome, err := decideWith(t, h, agent.DecisionAccepted, digest)
	require.NoError(t, err)
	require.NoError(t, outcome.ExecutionError)

	created := h.decisions.created
	require.NotNil(t, created)
	require.NotNil(t, created.Preview)
	assert.Equal(t, digest, created.PreviewDigest)
	assert.True(t, created.PreviewReviewed)
	require.NotNil(t, created.PreviewTargetVersion)
	assert.Equal(t, int64(3), *created.PreviewTargetVersion)
	require.Len(t, previews.asked, 1)
	assert.True(t, previews.asked[0].Deciding)
	assert.Equal(t, h.actor, previews.asked[0].Viewer.Actor, "previewed as the decider")
	require.NotNil(t, outcome.ExecutedTargetVersion)
	assert.Equal(t, int64(4), *outcome.ExecutedTargetVersion)
}

func TestDecide_AnApprovalWithoutADigestIsRecordedUnreviewed(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("d", 64)
	h, _ := previewHarness(t, &fakePreviews{digest: digest})

	_, err := decideWith(t, h, agent.DecisionAccepted, "")
	require.NoError(t, err)

	created := h.decisions.created
	require.NotNil(t, created)
	assert.Equal(t, digest, created.PreviewDigest, "the preview it ran against is recorded")
	assert.False(t, created.PreviewReviewed)
}

// A rejection is never refused over its preview: a person must always be
// able to say no, whatever the change has become since they read it.
func TestDecide_ARejectionIsNeverRefusedOverItsDigest(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{digest: strings.Repeat("a", 64)})

	_, err := decideWith(t, h, agent.DecisionRejected, "not-a-digest-the-client-could-send")
	require.NoError(t, err)

	require.NotNil(t, h.decisions.created)
	assert.Empty(t, h.decisions.created.PreviewDigest, "a malformed digest is not kept")
	assert.False(t, h.decisions.created.PreviewReviewed)
}

func TestDecide_ARejectionRecordsOnlyTheDigestItWasSent(t *testing.T) {
	t.Parallel()

	previews := &fakePreviews{stale: true, digest: strings.Repeat("e", 64)}
	h, _ := previewHarness(t, previews)

	sent := strings.Repeat("f", 64)
	_, err := decideWith(t, h, agent.DecisionRejected, sent)
	require.NoError(t, err, "a stale proposal can always be rejected")

	created := h.decisions.created
	require.NotNil(t, created)
	assert.Nil(t, created.Preview)
	assert.Equal(t, sent, created.PreviewDigest)
	assert.False(t, created.PreviewReviewed)
	assert.Empty(t, previews.asked)
}

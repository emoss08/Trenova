package dispatchautoassignservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type pendingProposals struct {
	agentproposalports.AgentProposalRepository

	proposals   []*agent.AgentProposal
	failed      []*agent.AgentProposal
	asked       *agentproposalports.ListPendingByTargetsRequest
	askedFailed *agentproposalports.ListFailedByTargetsRequest
}

func (r *pendingProposals) ListPendingByTargets(
	_ context.Context,
	req *agentproposalports.ListPendingByTargetsRequest,
) ([]*agent.AgentProposal, error) {
	r.asked = req

	return r.proposals, nil
}

func (r *pendingProposals) ListFailedByTargets(
	_ context.Context,
	req *agentproposalports.ListFailedByTargetsRequest,
) ([]*agent.AgentProposal, error) {
	r.askedFailed = req

	return r.failed, nil
}

type namedWorkers struct {
	workerports.WorkerRepository

	workers []*worker.Worker
	calls   int
}

func (r *namedWorkers) GetByIDs(
	_ context.Context,
	_ workerports.GetWorkersByIDsRequest,
) ([]*worker.Worker, error) {
	r.calls++

	return r.workers, nil
}

func coverageProposal(moveID, workerID pulid.ID, version int64) *agent.AgentProposal {
	return &agent.AgentProposal{
		ID:       pulid.MustNew("ap_"),
		ToolName: assignMoveTool,
		ToolParams: map[string]any{
			"shipmentMoveId":  moveID.String(),
			"primaryWorkerId": workerID.String(),
			"tractorId":       pulid.MustNew("trc_").String(),
		},
		Confidence:     decimal.NewFromFloat(0.91),
		Rationale:      "18 empty miles to the pickup",
		TargetResource: "shipment_move",
		TargetID:       moveID,
		TargetVersion:  version,
		CreatedAt:      1_800_000_000,
	}
}

func pendingService(
	proposals *pendingProposals,
	workers *namedWorkers,
) *Service {
	return &Service{l: zap.NewNop(), proposalRepo: proposals, workerRepo: workers}
}

func pendingRequest(
	moves ...dispatchconsoleports.CoverageMove,
) *dispatchconsoleports.CoverageProposalsRequest {
	return &dispatchconsoleports.CoverageProposalsRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Moves:      moves,
	}
}

func TestCoverageProposals_NamesTheProposedDriverForEachMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	dana := &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: "Dana", LastName: "Ortiz"}
	proposals := &pendingProposals{proposals: []*agent.AgentProposal{
		coverageProposal(moveID, dana.ID, 3),
	}}

	pending, err := pendingService(proposals, &namedWorkers{workers: []*worker.Worker{dana}}).
		CoverageProposals(t.Context(), pendingRequest(dispatchconsoleports.CoverageMove{
			MoveID:      moveID,
			MoveVersion: 3,
		}))
	require.NoError(t, err)

	require.Contains(t, pending, moveID)
	coverage := pending[moveID].Pending
	assert.Equal(t, dana.ID, coverage.WorkerID)
	assert.Equal(t, "Dana Ortiz", coverage.WorkerName)
	assert.False(t, coverage.Stale)
	assert.Equal(t, assignMoveTool, proposals.asked.ToolName)
	assert.Equal(t, "shipment_move", proposals.asked.TargetResource)
	assert.Equal(t, []pulid.ID{moveID}, proposals.asked.TargetIDs)
	assert.Positive(t, proposals.asked.Now, "expired proposals are left out by the time asked")
}

func TestCoverageProposals_MarksAProposalPlannedAgainstAnOlderMoveAsStale(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	proposals := &pendingProposals{proposals: []*agent.AgentProposal{
		coverageProposal(moveID, pulid.MustNew("wrk_"), 3),
	}}

	pending, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 4}),
	)
	require.NoError(t, err)

	assert.True(t, pending[moveID].Pending.Stale)
}

func TestCoverageProposals_KeepsOnlyTheNewestProposalForAMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	newest := coverageProposal(moveID, pulid.MustNew("wrk_"), 1)
	older := coverageProposal(moveID, pulid.MustNew("wrk_"), 1)
	proposals := &pendingProposals{proposals: []*agent.AgentProposal{newest, older}}

	pending, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 1}),
	)
	require.NoError(t, err)

	assert.Equal(t, newest.ID, pending[moveID].Pending.ProposalID)
}

func TestCoverageProposals_SkipsAProposalWhoseParametersDoNotDecode(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	broken := coverageProposal(moveID, pulid.MustNew("wrk_"), 1)
	broken.ToolParams["primaryWorkerId"] = "not-an-id"
	proposals := &pendingProposals{proposals: []*agent.AgentProposal{broken}}

	pending, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 1}),
	)
	require.NoError(t, err)

	assert.Empty(t, pending)
}

func TestCoverageProposals_AsksNothingForABoardWithNoMoves(t *testing.T) {
	t.Parallel()

	proposals := &pendingProposals{}
	workers := &namedWorkers{}

	pending, err := pendingService(
		proposals,
		workers,
	).CoverageProposals(t.Context(), pendingRequest())
	require.NoError(t, err)

	assert.Empty(t, pending)
	assert.Nil(t, proposals.asked)
	assert.Nil(t, proposals.askedFailed)
	assert.Zero(t, workers.calls)
}

func failedProposal(moveID, workerID pulid.ID, version, failedAt int64) *agent.AgentProposal {
	proposal := coverageProposal(moveID, workerID, version)
	proposal.Status = agent.ProposalStatusExecutionFailed
	proposal.ExecutedAt = &failedAt
	proposal.ExecutionError = "validation failed:\n- moves[0].stops[0].scheduledWindowEnd: " +
		"Scheduled window end must be before the next stop's scheduled window start"

	return proposal
}

func TestCoverageProposals_KeepsAFailedApprovalOnItsMoveWithAReasonInWords(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	dana := &worker.Worker{ID: pulid.MustNew("wrk_"), FirstName: "Dana", LastName: "Ortiz"}
	failure := failedProposal(moveID, dana.ID, 3, 1_800_000_500)
	proposals := &pendingProposals{failed: []*agent.AgentProposal{failure}}

	got, err := pendingService(proposals, &namedWorkers{workers: []*worker.Worker{dana}}).
		CoverageProposals(t.Context(), pendingRequest(dispatchconsoleports.CoverageMove{
			MoveID:      moveID,
			MoveVersion: 3,
		}))
	require.NoError(t, err)

	require.Contains(t, got, moveID)
	assert.Nil(t, got[moveID].Pending)
	assert.Equal(t, &dispatchconsoleports.FailedCoverage{
		ProposalID: failure.ID,
		MoveID:     moveID,
		WorkerID:   dana.ID,
		WorkerName: "Dana Ortiz",
		Reason: "Move 1, stop 1: Scheduled window end must be before the next stop's " +
			"scheduled window start",
		FailedAt: 1_800_000_500,
	}, got[moveID].Failed)
	assert.Equal(t, []pulid.ID{moveID}, proposals.askedFailed.TargetIDs)
	assert.Equal(t, assignMoveTool, proposals.askedFailed.ToolName)
}

func TestCoverageProposals_DropsAFailureOnceTheMoveHasChanged(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	proposals := &pendingProposals{failed: []*agent.AgentProposal{
		failedProposal(moveID, pulid.MustNew("wrk_"), 3, 1_800_000_500),
	}}

	got, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 4}),
	)
	require.NoError(t, err)

	assert.Empty(t, got)
}

func TestCoverageProposals_ANewerProposalReplacesAnOlderFailure(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	newer := coverageProposal(moveID, pulid.MustNew("wrk_"), 3)
	newer.CreatedAt = 1_800_000_900
	proposals := &pendingProposals{
		proposals: []*agent.AgentProposal{newer},
		failed: []*agent.AgentProposal{
			failedProposal(moveID, pulid.MustNew("wrk_"), 3, 1_800_000_500),
		},
	}

	got, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 3}),
	)
	require.NoError(t, err)

	assert.Equal(t, newer.ID, got[moveID].Pending.ProposalID)
	assert.Nil(t, got[moveID].Failed)
}

func TestCoverageProposals_KeepsOnlyTheNewestFailureForAMove(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("sm_")
	newest := failedProposal(moveID, pulid.MustNew("wrk_"), 3, 1_800_000_800)
	older := failedProposal(moveID, pulid.MustNew("wrk_"), 3, 1_800_000_500)
	proposals := &pendingProposals{failed: []*agent.AgentProposal{newest, older}}

	got, err := pendingService(proposals, &namedWorkers{}).CoverageProposals(
		t.Context(),
		pendingRequest(dispatchconsoleports.CoverageMove{MoveID: moveID, MoveVersion: 3}),
	)
	require.NoError(t, err)

	assert.Equal(t, newest.ID, got[moveID].Failed.ProposalID)
}

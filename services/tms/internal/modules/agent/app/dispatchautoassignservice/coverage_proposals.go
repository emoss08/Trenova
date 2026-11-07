package dispatchautoassignservice

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) CoverageProposals(
	ctx context.Context,
	req *dispatchconsoleports.CoverageProposalsRequest,
) (map[pulid.ID]*dispatchconsoleports.MoveCoverageProposals, error) {
	out := make(map[pulid.ID]*dispatchconsoleports.MoveCoverageProposals, len(req.Moves))
	if len(req.Moves) == 0 {
		return out, nil
	}

	versions := make(map[pulid.ID]int64, len(req.Moves))
	moveIDs := make([]pulid.ID, 0, len(req.Moves))
	for _, move := range req.Moves {
		if _, seen := versions[move.MoveID]; seen {
			continue
		}
		versions[move.MoveID] = move.MoveVersion
		moveIDs = append(moveIDs, move.MoveID)
	}

	pending, err := s.proposalRepo.ListPendingByTargets(
		ctx,
		&agentproposalports.ListPendingByTargetsRequest{
			TenantInfo:     req.TenantInfo,
			ToolName:       assignMoveTool,
			TargetResource: string(permission.ResourceShipmentMove),
			TargetIDs:      moveIDs,
			Now:            timeutils.NowUnix(),
		},
	)
	if err != nil {
		return nil, err
	}
	failed, err := s.proposalRepo.ListFailedByTargets(
		ctx,
		&agentproposalports.ListFailedByTargetsRequest{
			TenantInfo:     req.TenantInfo,
			ToolName:       assignMoveTool,
			TargetResource: string(permission.ResourceShipmentMove),
			TargetIDs:      moveIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	entry := func(moveID pulid.ID) *dispatchconsoleports.MoveCoverageProposals {
		if out[moveID] == nil {
			out[moveID] = &dispatchconsoleports.MoveCoverageProposals{}
		}
		return out[moveID]
	}
	workerIDs := make([]pulid.ID, 0, len(pending)+len(failed))

	for _, proposal := range pending {
		if out[proposal.TargetID] != nil {
			continue
		}
		coverage, decodeErr := pendingCoverageOf(proposal, versions[proposal.TargetID])
		if decodeErr != nil {
			s.skipUndecodable(proposal, decodeErr)
			continue
		}
		entry(proposal.TargetID).Pending = coverage
		workerIDs = append(workerIDs, coverage.WorkerID)
	}

	for _, proposal := range failed {
		current := out[proposal.TargetID]
		if current != nil && current.Failed != nil {
			continue
		}
		if !failureStillApplies(proposal, versions[proposal.TargetID], current) {
			continue
		}
		coverage, decodeErr := failedCoverageOf(proposal)
		if decodeErr != nil {
			s.skipUndecodable(proposal, decodeErr)
			continue
		}
		entry(proposal.TargetID).Failed = coverage
		workerIDs = append(workerIDs, coverage.WorkerID)
	}

	if err = s.nameWorkers(ctx, req, workerIDs, out); err != nil {
		return nil, err
	}

	return out, nil
}

func failureStillApplies(
	proposal *agent.AgentProposal,
	moveVersion int64,
	current *dispatchconsoleports.MoveCoverageProposals,
) bool {
	if proposal.TargetVersion != moveVersion {
		return false
	}
	if current != nil && current.Pending != nil && current.Pending.CreatedAt > failedAt(proposal) {
		return false
	}

	return true
}

func failedAt(proposal *agent.AgentProposal) int64 {
	if proposal.ExecutedAt != nil {
		return *proposal.ExecutedAt
	}

	return proposal.UpdatedAt
}

func (s *Service) skipUndecodable(proposal *agent.AgentProposal, err error) {
	s.l.Warn(
		"skipping a coverage proposal whose parameters do not decode",
		zap.String("proposalId", proposal.ID.String()),
		zap.Error(err),
	)
}

func (s *Service) nameWorkers(
	ctx context.Context,
	req *dispatchconsoleports.CoverageProposalsRequest,
	workerIDs []pulid.ID,
	proposals map[pulid.ID]*dispatchconsoleports.MoveCoverageProposals,
) error {
	if len(workerIDs) == 0 {
		return nil
	}

	workers, err := s.workerRepo.GetByIDs(ctx, workerports.GetWorkersByIDsRequest{
		TenantInfo: req.TenantInfo,
		WorkerIDs:  workerIDs,
	})
	if err != nil {
		return fmt.Errorf("name proposed drivers: %w", err)
	}

	names := make(map[pulid.ID]string, len(workers))
	for _, w := range workers {
		names[w.ID] = w.FullName()
	}
	for _, move := range proposals {
		if move.Pending != nil {
			move.Pending.WorkerName = names[move.Pending.WorkerID]
		}
		if move.Failed != nil {
			move.Failed.WorkerName = names[move.Failed.WorkerID]
		}
	}

	return nil
}

func proposedWorkers(proposal *agent.AgentProposal) (assignMoveToolParams, error) {
	var params assignMoveToolParams
	raw, err := sonic.Marshal(proposal.ToolParams)
	if err != nil {
		return params, err
	}
	err = sonic.Unmarshal(raw, &params)

	return params, err
}

func failedCoverageOf(proposal *agent.AgentProposal) (*dispatchconsoleports.FailedCoverage, error) {
	params, err := proposedWorkers(proposal)
	if err != nil {
		return nil, err
	}
	workerID, err := pulid.Parse(params.PrimaryWorkerID)
	if err != nil {
		return nil, fmt.Errorf("primary worker: %w", err)
	}

	return &dispatchconsoleports.FailedCoverage{
		ProposalID: proposal.ID,
		MoveID:     proposal.TargetID,
		WorkerID:   workerID,
		Reason:     errortypes.SummaryOfText(proposal.ExecutionError),
		FailedAt:   failedAt(proposal),
	}, nil
}

func pendingCoverageOf(
	proposal *agent.AgentProposal,
	moveVersion int64,
) (*dispatchconsoleports.PendingCoverage, error) {
	params, err := proposedWorkers(proposal)
	if err != nil {
		return nil, err
	}

	workerID, err := pulid.Parse(params.PrimaryWorkerID)
	if err != nil {
		return nil, fmt.Errorf("primary worker: %w", err)
	}
	tractorID, err := pulid.Parse(params.TractorID)
	if err != nil {
		return nil, fmt.Errorf("tractor: %w", err)
	}
	trailerID := pulid.Nil
	if params.TrailerID != "" {
		if trailerID, err = pulid.Parse(params.TrailerID); err != nil {
			return nil, fmt.Errorf("trailer: %w", err)
		}
	}

	return &dispatchconsoleports.PendingCoverage{
		ProposalID: proposal.ID,
		MoveID:     proposal.TargetID,
		WorkerID:   workerID,
		TractorID:  tractorID,
		TrailerID:  trailerID,
		Confidence: proposal.Confidence,
		Rationale:  proposal.Rationale,
		CreatedAt:  proposal.CreatedAt,
		ExpiresAt:  proposal.ExpiresAt,
		Stale:      proposal.TargetVersion != moveVersion,
	}, nil
}

package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

type ScoreCandidateRequest struct {
	Driver   *repositories.BoardDriver
	Move     *repositories.BoardMove
	Snapshot *FleetSnapshot
}

func (s *Service) ScoreCandidate(req *ScoreCandidateRequest) *CandidateScore {
	if req == nil || req.Driver == nil || req.Move == nil || req.Snapshot == nil {
		return nil
	}
	if req.Snapshot.Control == nil {
		return nil
	}

	return s.scoreDriver(&scoreDriverParams{
		Driver:       req.Driver,
		Move:         req.Move,
		Requirements: requirementsFor(req.Move),
		Snapshot:     req.Snapshot,
		Weights:      req.Snapshot.Control.ResolvedScoringWeights(),
	})
}

type CommitPlannedMoveRequest struct {
	Snapshot *FleetSnapshot
	Move     *repositories.BoardMove
	Score    *CandidateScore
}

func CommitPlannedMove(req *CommitPlannedMoveRequest) {
	if req == nil || req.Snapshot == nil || req.Move == nil || req.Score == nil {
		return
	}
	if req.Score.WorkerID.IsNil() {
		return
	}

	if req.Snapshot.CommitmentsByWorker == nil {
		req.Snapshot.CommitmentsByWorker = make(
			map[pulid.ID][]*repositories.WorkerCommitment,
			1,
		)
	}

	move := req.Move
	workerID := req.Score.WorkerID

	req.Snapshot.CommitmentsByWorker[workerID] = append(
		req.Snapshot.CommitmentsByWorker[workerID],
		&repositories.WorkerCommitment{
			WorkerID:       workerID,
			MoveID:         move.MoveID,
			ShipmentID:     move.ShipmentID,
			ProNumber:      move.ProNumber,
			MoveStatus:     move.MoveStatus,
			WindowStart:    move.OriginWindowStart,
			WindowEnd:      PlannedCompletion(move, req.Score),
			DestinationID:  move.DestinationLocationID,
			DestinationCty: move.DestinationCity,
			DestinationSt:  move.DestinationState,
			DestLatitude:   move.DestinationLatitude,
			DestLongitude:  move.DestinationLongitude,
			TrailerID:      req.Score.TrailerID,
		},
	)
}

func PlannedCompletion(move *repositories.BoardMove, score *CandidateScore) int64 {
	if move == nil || score == nil {
		return 0
	}

	drivenBy := score.ProjectedAvailable + score.EstimatedDriveMs/1000

	return max(MoveWindowEnd(move), drivenBy)
}

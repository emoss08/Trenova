package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
)

type CommitPlannedMoveRequest struct {
	Snapshot *FleetSnapshot
	Move     *dispatchboardports.BoardMove
	Score    *dispatchplanports.CandidateScore
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
			map[pulid.ID][]*dispatchboardports.WorkerCommitment,
			1,
		)
	}

	workerID := req.Score.WorkerID
	req.Snapshot.CommitmentsByWorker[workerID] = append(
		req.Snapshot.CommitmentsByWorker[workerID],
		PlannedCommitment(req.Move, req.Score),
	)
	req.Snapshot.RefreshProfile(workerID)
}

func PlannedCommitment(
	move *dispatchboardports.BoardMove,
	score *dispatchplanports.CandidateScore,
) *dispatchboardports.WorkerCommitment {
	if move == nil || score == nil || score.WorkerID.IsNil() {
		return nil
	}
	completion := PlannedCompletion(move, score)

	return &dispatchboardports.WorkerCommitment{
		WorkerID:       score.WorkerID,
		MoveID:         move.MoveID,
		ShipmentID:     move.ShipmentID,
		ProNumber:      move.ProNumber,
		MoveStatus:     move.MoveStatus,
		WindowStart:    move.OriginWindowStart,
		WindowEnd:      completion,
		ExpectedEnd:    completion,
		Distance:       move.Distance,
		DestinationID:  move.DestinationLocationID,
		DestinationCty: move.DestinationCity,
		DestinationSt:  move.DestinationState,
		DestLatitude:   move.DestinationLatitude,
		DestLongitude:  move.DestinationLongitude,
		TrailerID:      score.TrailerID,
	}
}

func PlannedCompletion(
	move *dispatchboardports.BoardMove,
	score *dispatchplanports.CandidateScore,
) int64 {
	if move == nil || score == nil {
		return 0
	}
	if score.ProjectedFreeAt > 0 {
		return score.ProjectedFreeAt
	}

	return MoveWindowEnd(move)
}

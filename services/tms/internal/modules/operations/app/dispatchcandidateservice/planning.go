package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
)

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

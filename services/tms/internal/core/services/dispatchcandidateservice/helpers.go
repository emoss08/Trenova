package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func CustomerIDsOf(moves []*repositories.BoardMove) []pulid.ID {
	return uniqueMoveIDs(moves, func(move *repositories.BoardMove) pulid.ID {
		return move.CustomerID
	})
}

func TrailerIDsOf(moves []*repositories.BoardMove) []pulid.ID {
	return uniqueMoveIDs(moves, resolveTrailer)
}

func uniqueMoveIDs(
	moves []*repositories.BoardMove,
	pick func(*repositories.BoardMove) pulid.ID,
) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(moves))
	ids := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		id := pick(move)
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func ProjectedTimeAvailable(commitments []*repositories.WorkerCommitment, now int64) int64 {
	latest := now
	for _, commitment := range commitments {
		if commitment.WindowEnd > latest {
			latest = commitment.WindowEnd
		}
	}
	return latest
}

func ResolveWindow(
	start, end int64,
	control *dispatchcontrol.DispatchControl,
	now int64,
) (resolvedStart, resolvedEnd int64) {
	if start <= 0 {
		start = now
	}
	if end <= 0 {
		end = now + int64(control.PlanningHorizonHours())*3600
	}
	return start, end
}

func MoveWindowEnd(move *repositories.BoardMove) int64 {
	end := move.DestinationWindowStart
	if move.DestinationWindowEnd != nil && *move.DestinationWindowEnd > end {
		end = *move.DestinationWindowEnd
	}
	return end
}

func CompareRank(left, right *CandidateScore) int {
	leftBlocked, rightBlocked := left.Blocked(), right.Blocked()
	if leftBlocked != rightBlocked {
		if rightBlocked {
			return -1
		}
		return 1
	}
	if left.Score != right.Score {
		if left.Score > right.Score {
			return -1
		}
		return 1
	}
	return 0
}

func PTOInclusiveEnd(endDate int64) int64 {
	if endDate <= 0 {
		return endDate
	}
	if dayEnd, err := timeutils.DayEndUnix(endDate, "UTC"); err == nil {
		return dayEnd
	}
	return endDate
}

package dispatchcandidateservice

import (
	"cmp"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
)

func candidateFrom(snapshot *FleetSnapshot, eval *evaluation) *dispatchplanports.CandidateScore {
	p := eval.profile
	t := eval.trip
	hard, soft := snapshot.Model.Split(&eval.features)

	score := &dispatchplanports.CandidateScore{
		WorkerID:            p.driver.WorkerID,
		WorkerName:          p.name,
		TractorID:           p.driver.TractorID,
		TrailerID:           eval.trailerID,
		MoveID:              eval.move.MoveID,
		Cost:                snapshot.Model.Explain(&eval.features),
		HardCents:           hard,
		SoftCents:           soft,
		Verdict:             eval.verdict,
		OriginSource:        p.originSource,
		LoadedMiles:         t.loadedMiles,
		EstimatedDriveMs:    t.driveMs,
		RestMs:              t.restMs,
		ProjectedArrival:    t.pickupArrival,
		ProjectedDelivery:   t.deliveryArrival,
		ProjectedFreeAt:     t.freeAt,
		MinutesOfSlack:      int64(t.slackMinutes),
		SlackKnown:          t.appointmentKnown,
		DeliveryLateMinutes: int64(t.deliveryLateMinutes),
		ProjectedAvailable:  p.availableAt,
		Findings:            eval.findings,
	}
	if t.emptyKnown {
		miles := t.emptyMiles
		score.DeadheadMiles = &miles
	}
	if state := p.hosState; state != nil && p.hosKnown {
		score.HOSKnown = true
		score.DriveRemainingMs = state.DriveRemainingMs
		score.ShiftRemainingMs = state.ShiftRemainingMs
		score.CycleRemainingMs = state.CycleRemainingMs
	}
	if projection := eval.projection; projection != nil {
		score.HOSStrategy = string(projection.Strategy)
		score.HOSRestStartDeadline = projection.RestStartDeadline
		score.HOSProjectedDriveMs = projection.DriveAvailableMs
		score.HOSProjectedShiftMs = projection.ShiftAvailableMs
		score.HOSProjectedCycleMs = projection.CycleAvailableMs
	}

	return score
}

type RankRequest struct {
	Move           *dispatchboardports.BoardMove
	Snapshot       *FleetSnapshot
	Limit          int
	IncludeBlocked bool
}

func (s *Service) RankCandidates(req *RankRequest) []*dispatchplanports.CandidateScore {
	if req.Move == nil || req.Snapshot == nil {
		return nil
	}

	results := make([]*dispatchplanports.CandidateScore, 0, len(req.Snapshot.Drivers))
	for _, driver := range req.Snapshot.Drivers {
		profile := req.Snapshot.profile(driver)
		if profile == nil {
			continue
		}
		score := candidateFrom(req.Snapshot, evaluate(req.Snapshot, profile, req.Move, nil))
		if score.Blocked() && !req.IncludeBlocked {
			continue
		}
		results = append(results, score)
	}

	slices.SortStableFunc(results, func(a, b *dispatchplanports.CandidateScore) int {
		if c := dispatchplanports.CompareRank(a, b); c != 0 {
			return c
		}
		return cmp.Compare(a.WorkerName, b.WorkerName)
	})

	if req.Limit > 0 && len(results) > req.Limit {
		results = results[:req.Limit]
	}

	return results
}

type ScoreCandidateRequest struct {
	Driver   *dispatchboardports.BoardDriver
	Move     *dispatchboardports.BoardMove
	Snapshot *FleetSnapshot
}

func (s *Service) ScoreCandidate(req *ScoreCandidateRequest) *dispatchplanports.CandidateScore {
	if req == nil || req.Driver == nil || req.Move == nil || req.Snapshot == nil {
		return nil
	}
	profile := req.Snapshot.profile(req.Driver)
	if profile == nil {
		return nil
	}

	return candidateFrom(req.Snapshot, evaluate(req.Snapshot, profile, req.Move, nil))
}

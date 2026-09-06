package workersafetyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"golang.org/x/sync/errgroup"
)

const (
	defaultFleetWindowMonths = 12
	maxFleetWindowMonths     = 36
	defaultFleetRankLimit    = 10
	maxFleetRankLimit        = 50
)

// FleetRequest scopes the fleet safety view.
type FleetRequest struct {
	TenantInfo   pagination.TenantInfo
	WindowMonths int
	FleetCodeID  pulid.ID
	RankLimit    int
	AsOf         int64
}

func (r *FleetRequest) normalise() {
	if r.WindowMonths <= 0 {
		r.WindowMonths = defaultFleetWindowMonths
	}
	if r.WindowMonths > maxFleetWindowMonths {
		r.WindowMonths = maxFleetWindowMonths
	}
	if r.RankLimit <= 0 {
		r.RankLimit = defaultFleetRankLimit
	}
	if r.RankLimit > maxFleetRankLimit {
		r.RankLimit = maxFleetRankLimit
	}
	if r.AsOf <= 0 {
		r.AsOf = timeutils.NowUnix()
	}
}

// Fleet is the whole fleet's safety picture. Seven grouped queries run
// concurrently because they touch unrelated tables and the page shows them
// together; any of them failing fails the call, since a silently missing
// section would read as "nothing happening there".
func (s *Service) Fleet(ctx context.Context, req *FleetRequest) (*worker.FleetSafety, error) {
	req.normalise()

	base := &repositories.FleetSafetyRequest{
		TenantInfo:   req.TenantInfo,
		WindowMonths: req.WindowMonths,
		FleetCodeID:  req.FleetCodeID,
		RankLimit:    req.RankLimit,
		Now:          req.AsOf,
	}

	var (
		ratings   []repositories.FleetSafetyRatingRow
		terminals []repositories.FleetSafetyTerminalRow
		kinds     []repositories.FleetSafetyKindRow
		basics    []repositories.FleetSafetyBasicRow
		inferred  []repositories.FleetSafetyEventBasicRow
		trend     []repositories.FleetSafetyTrendRow
		worst     []repositories.FleetSafetyRankRow
		best      []repositories.FleetSafetyRankRow
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		ratings, err = s.repo.FleetRatings(groupCtx, base)
		return err
	})
	group.Go(func() error {
		var err error
		terminals, err = s.repo.FleetTerminals(groupCtx, base)
		return err
	})
	group.Go(func() error {
		var err error
		kinds, err = s.repo.FleetKinds(groupCtx, base)
		return err
	})
	group.Go(func() error {
		var err error
		basics, err = s.repo.FleetBasics(groupCtx, base)
		return err
	})
	group.Go(func() error {
		var err error
		inferred, err = s.repo.FleetEventBasics(groupCtx, base)
		return err
	})
	group.Go(func() error {
		var err error
		trend, err = s.repo.FleetTrend(groupCtx, base)
		return err
	})
	group.Go(func() error {
		worstReq := *base
		var err error
		worst, err = s.repo.FleetRanking(groupCtx, &worstReq)
		return err
	})
	group.Go(func() error {
		bestReq := *base
		bestReq.RankBest = true
		var err error
		best, err = s.repo.FleetRanking(groupCtx, &bestReq)
		return err
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return composeFleet(&fleetInput{
		req:       req,
		ratings:   ratings,
		terminals: terminals,
		kinds:     kinds,
		basics:    basics,
		inferred:  inferred,
		trend:     trend,
		worst:     worst,
		best:      best,
	}), nil
}

type fleetInput struct {
	req       *FleetRequest
	ratings   []repositories.FleetSafetyRatingRow
	terminals []repositories.FleetSafetyTerminalRow
	kinds     []repositories.FleetSafetyKindRow
	basics    []repositories.FleetSafetyBasicRow
	inferred  []repositories.FleetSafetyEventBasicRow
	trend     []repositories.FleetSafetyTrendRow
	worst     []repositories.FleetSafetyRankRow
	best      []repositories.FleetSafetyRankRow
}

func composeFleet(in *fleetInput) *worker.FleetSafety {
	out := &worker.FleetSafety{
		AsOf:         in.req.AsOf,
		WindowMonths: int32(in.req.WindowMonths), //nolint:gosec // bounded above
		Ratings:      make([]worker.FleetSafetyRatingCount, 0, len(in.ratings)),
		Kinds:        make([]worker.FleetSafetyKind, 0, len(in.kinds)),
		Terminals:    make([]worker.FleetSafetyTerminal, 0, len(in.terminals)),
		Trend:        make([]worker.FleetSafetyTrendPoint, 0, len(in.trend)),
		Worst:        make([]worker.FleetSafetyRank, 0, len(in.worst)),
		Best:         make([]worker.FleetSafetyRank, 0, len(in.best)),
	}

	var totalScore int32
	for _, row := range in.ratings {
		workers := int32(row.Workers) //nolint:gosec // a headcount
		out.Workers += workers
		totalScore += int32(row.TotalScore) //nolint:gosec // a score sum
		out.Ratings = append(out.Ratings, worker.FleetSafetyRatingCount{
			Rating:  row.Rating,
			Workers: workers,
		})
		switch row.Rating {
		case worker.SafetyRatingAtRisk:
			out.AtRisk += workers
		case worker.SafetyRatingWatch:
			out.Watch += workers
		case worker.SafetyRatingExcellent, worker.SafetyRatingGood:
		}
	}
	out.AverageScore = worker.AverageScore(totalScore, out.Workers)

	for _, row := range in.kinds {
		kind := worker.FleetSafetyKind{
			Kind:         row.Kind,
			Events:       int32(row.Events),       //nolint:gosec // a count
			Points:       int32(row.Points),       //nolint:gosec // a point sum
			Preventable:  int32(row.Preventable),  //nolint:gosec // a count
			OutOfService: int32(row.OutOfService), //nolint:gosec // a count
			Open:         int32(row.Open),         //nolint:gosec // a count
		}
		out.Kinds = append(out.Kinds, kind)
		out.TotalEvents += kind.Events
		out.TotalPoints += kind.Points
		out.OpenEvents += kind.Open
		out.OutOfServiceOrders += kind.OutOfService
	}

	for _, row := range in.terminals {
		workers := int32(row.Workers) //nolint:gosec // a headcount
		out.Terminals = append(out.Terminals, worker.FleetSafetyTerminal{
			FleetCodeID:  row.FleetCodeID,
			Code:         row.FleetCodeCode,
			Description:  row.FleetCodeDescription,
			Color:        row.FleetCodeColor,
			Workers:      workers,
			AtRisk:       int32(row.AtRisk),                                   //nolint:gosec // a count
			Watch:        int32(row.Watch),                                    //nolint:gosec // a count
			AverageScore: worker.AverageScore(int32(row.TotalScore), workers), //nolint:gosec // a score sum
		})
	}

	for _, row := range in.trend {
		out.Trend = append(out.Trend, worker.FleetSafetyTrendPoint{
			PeriodStart:  row.PeriodStart,
			Events:       int32(row.Events),       //nolint:gosec // a count
			Accidents:    int32(row.Accidents),    //nolint:gosec // a count
			Preventable:  int32(row.Preventable),  //nolint:gosec // a count
			Citations:    int32(row.Citations),    //nolint:gosec // a count
			Inspections:  int32(row.Inspections),  //nolint:gosec // a count
			OutOfService: int32(row.OutOfService), //nolint:gosec // a count
			Points:       int32(row.Points),       //nolint:gosec // a point sum
		})
	}

	violationInputs := make([]worker.FleetSafetyBasicInput, 0, len(in.basics))
	for _, row := range in.basics {
		violationInputs = append(violationInputs, worker.FleetSafetyBasicInput{
			Basic:        row.Basic,
			Bucket:       row.Bucket,
			Violations:   int32(row.Violations),   //nolint:gosec // a count
			SeveritySum:  int32(row.SeveritySum),  //nolint:gosec // a weight sum
			OutOfService: int32(row.OutOfService), //nolint:gosec // a count
		})
	}
	eventInputs := make([]worker.FleetSafetyEventBasicInput, 0, len(in.inferred))
	for _, row := range in.inferred {
		eventInputs = append(eventInputs, worker.FleetSafetyEventBasicInput{
			Kind:             row.Kind,
			InspectionResult: row.InspectionResult,
			Bucket:           row.Bucket,
			Events:           int32(row.Events),       //nolint:gosec // a count
			Points:           int32(row.Points),       //nolint:gosec // a point sum
			OutOfService:     int32(row.OutOfService), //nolint:gosec // a count
		})
	}
	out.Basics = worker.BuildFleetBasics(violationInputs, eventInputs)
	for _, basic := range out.Basics {
		if basic.Inferred {
			out.BasicsInferred = true
			break
		}
	}

	out.Worst = toRanks(in.worst)
	out.Best = toRanks(in.best)

	return out
}

func toRanks(rows []repositories.FleetSafetyRankRow) []worker.FleetSafetyRank {
	ranks := make([]worker.FleetSafetyRank, 0, len(rows))
	for _, row := range rows {
		ranks = append(ranks, worker.FleetSafetyRank{
			WorkerID:     row.WorkerID,
			Name:         row.FirstName + " " + row.LastName,
			FleetCodeID:  row.FleetCodeID,
			FleetCode:    row.FleetCodeCode,
			FleetColor:   row.FleetCodeColor,
			Rating:       row.Rating,
			Score:        int32(row.Score),        //nolint:gosec // a 0..100 score
			ActivePoints: int32(row.ActivePoints), //nolint:gosec // a point sum
			Events:       int32(row.Events),       //nolint:gosec // a count
			LastEventAt:  row.LastEventAt,
		})
	}
	return ranks
}

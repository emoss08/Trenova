package dispatchcandidateservice

import (
	"cmp"
	"context"
	"math"
	"runtime"
	"slices"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchurgency"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/quadtree"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

const (
	prefetchFactor = 3
	prefetchSlack  = 20
	routedPerMove  = 8
)

type Arc struct {
	Driver    int
	CostCents int64
	Candidate *dispatchplanports.CandidateScore
}

type Graph struct {
	Moves    []*dispatchboardports.BoardMove
	Drivers  []*dispatchboardports.BoardDriver
	Arcs     [][]Arc
	Coverage []int64
	Rejected []*dispatchplanports.CandidateScore
}

type GraphRequest struct {
	TenantInfo pagination.TenantInfo
	Moves      []*dispatchboardports.BoardMove
	Snapshot   *FleetSnapshot
}

type pairEval struct {
	driver int
	eval   *evaluation
	hard   int64
	soft   int64
}

func (s *Service) BuildGraph(ctx context.Context, req *GraphRequest) *Graph {
	snapshot := req.Snapshot
	if snapshot.profiles == nil {
		snapshot.Prepare()
	}

	graph := &Graph{
		Moves:    req.Moves,
		Drivers:  snapshot.Drivers,
		Arcs:     make([][]Arc, len(req.Moves)),
		Coverage: make([]int64, len(req.Moves)),
		Rejected: make([]*dispatchplanports.CandidateScore, len(req.Moves)),
	}
	limit := snapshot.Model.Economics().CandidatesPerMove
	index := newDriverIndex(snapshot)
	pairs := make([][]pairEval, len(req.Moves))

	work := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
	for i, move := range req.Moves {
		work.Go(func() {
			graph.Coverage[i] = snapshot.Model.CoveragePrice(moveValue(move, snapshot.Now))
			var rejected *evaluation
			pairs[i], rejected = bestPairs(snapshot, index, move, limit)
			if rejected != nil {
				graph.Rejected[i] = candidateFrom(snapshot, rejected)
			}
		})
	}
	work.Wait()

	s.routeTopPairs(ctx, req.TenantInfo, snapshot, req.Moves, pairs)

	for i, candidates := range pairs {
		graph.Arcs[i] = arcsFrom(snapshot, candidates)
	}

	return graph
}

func moveValue(move *dispatchboardports.BoardMove, now int64) dispatchcost.MoveValue {
	value := dispatchcost.MoveValue{Urgency: dispatchurgency.Of(move, now)}
	if move.Revenue != nil {
		value.RevenueCents = int64(math.Round(*move.Revenue * 100))
	}

	return value
}

func bestPairs(
	snapshot *FleetSnapshot,
	index *driverIndex,
	move *dispatchboardports.BoardMove,
	limit int,
) ([]pairEval, *evaluation) {
	candidates := index.near(move, limit*prefetchFactor+prefetchSlack)
	pairs := make([]pairEval, 0, len(candidates))
	var rejected *evaluation
	rejectedCost := int64(0)
	for _, driverIndex := range candidates {
		profile := snapshot.profile(snapshot.Drivers[driverIndex])
		if profile == nil {
			continue
		}
		eval := evaluate(snapshot, profile, move, nil)
		if !admissible(snapshot, eval) {
			if cost := snapshot.Model.Total(&eval.features); rejected == nil ||
				cost < rejectedCost {
				rejected, rejectedCost = eval, cost
			}
			continue
		}
		hard, soft := snapshot.Model.Split(&eval.features)
		pairs = append(pairs, pairEval{driver: driverIndex, eval: eval, hard: hard, soft: soft})
	}
	if len(pairs) == 0 && rejected == nil && index.unlocated >= 0 {
		rejected = evaluate(
			snapshot,
			snapshot.profile(snapshot.Drivers[index.unlocated]),
			move,
			nil,
		)
	}
	slices.SortFunc(pairs, func(a, b pairEval) int {
		return cmp.Compare(a.hard+a.soft, b.hard+b.soft)
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}

	return pairs, rejected
}

func admissible(snapshot *FleetSnapshot, eval *evaluation) bool {
	if eval.blocked() || !eval.trip.emptyKnown {
		return false
	}

	return withinDeadheadCap(snapshot.Control, eval.trip.emptyMiles)
}

func Admissible(snapshot *FleetSnapshot, score *dispatchplanports.CandidateScore) bool {
	if score == nil || score.Blocked() || !score.LocationKnown() {
		return false
	}

	return withinDeadheadCap(snapshot.Control, *score.DeadheadMiles)
}

func withinDeadheadCap(control *dispatchcontrol.DispatchControl, emptyMiles float64) bool {
	if control == nil || control.AutoAssignMaxDeadheadMiles == nil {
		return true
	}

	return emptyMiles <= float64(*control.AutoAssignMaxDeadheadMiles)
}

func arcsFrom(snapshot *FleetSnapshot, pairs []pairEval) []Arc {
	if len(pairs) == 0 {
		return nil
	}
	floor := pairs[0].soft
	for _, pair := range pairs {
		floor = min(floor, pair.soft)
	}

	arcs := make([]Arc, 0, len(pairs))
	for _, pair := range pairs {
		arcs = append(arcs, Arc{
			Driver:    pair.driver,
			CostCents: pair.hard + pair.soft - floor,
			Candidate: candidateFrom(snapshot, pair.eval),
		})
	}

	return arcs
}

func (s *Service) routeTopPairs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshot *FleetSnapshot,
	moves []*dispatchboardports.BoardMove,
	pairs [][]pairEval,
) {
	type ref struct{ move, pair int }

	legs := make([]routeestimateports.Leg, 0, len(moves)*routedPerMove)
	refs := make([]ref, 0, cap(legs))
	for i, candidates := range pairs {
		move := moves[i]
		if move.OriginLatitude == nil || move.OriginLongitude == nil {
			continue
		}
		for j := range min(len(candidates), routedPerMove) {
			origin := candidates[j].eval.profile.origin
			legs = append(legs, routeestimateports.Leg{
				From: routeestimateports.Point{Latitude: origin.lat, Longitude: origin.lon},
				To: routeestimateports.Point{
					Latitude:  *move.OriginLatitude,
					Longitude: *move.OriginLongitude,
				},
			})
			refs = append(refs, ref{move: i, pair: j})
		}
	}

	for start := 0; start < len(legs); start += routeestimateports.MaxLegsPerEstimate {
		end := min(start+routeestimateports.MaxLegsPerEstimate, len(legs))
		estimates, err := s.routes.EstimateLegs(ctx, tenantInfo, legs[start:end])
		if err != nil {
			s.l.Warn("routing candidate empty legs failed; planning on straight-line miles",
				zap.Error(err))
			return
		}
		for k, estimate := range estimates {
			if !estimate.Routed {
				continue
			}
			r := refs[start+k]
			pair := &pairs[r.move][r.pair]
			pair.eval = evaluate(snapshot, pair.eval.profile, moves[r.move], &routedLeg{
				miles:   estimate.Miles,
				seconds: estimate.DriveSeconds,
			})
			pair.hard, pair.soft = snapshot.Model.Split(&pair.eval.features)
		}
	}

	for i := range pairs {
		slices.SortFunc(pairs[i], func(a, b pairEval) int {
			return int(a.hard + a.soft - b.hard - b.soft)
		})
	}
}

type indexedDriver struct {
	point orb.Point
	index int
}

func (d indexedDriver) Point() orb.Point { return d.point }

type driverIndex struct {
	tree      *quadtree.Quadtree
	all       int
	unlocated int
}

func newDriverIndex(snapshot *FleetSnapshot) *driverIndex {
	index := &driverIndex{
		tree: quadtree.New(orb.Bound{
			Min: orb.Point{-180 * geoutils.PlanarLongitudeScale, -90},
			Max: orb.Point{180 * geoutils.PlanarLongitudeScale, 90},
		}),
		all:       len(snapshot.Drivers),
		unlocated: -1,
	}
	for i, driver := range snapshot.Drivers {
		profile := snapshot.profile(driver)
		if profile == nil {
			continue
		}
		if profile.originSource != dispatchplanports.OriginUnknown {
			err := index.tree.Add(indexedDriver{point: plane(profile.origin), index: i})
			if err == nil {
				continue
			}
			profile.originSource = dispatchplanports.OriginUnknown
		}
		if index.unlocated < 0 {
			index.unlocated = i
		}
	}

	return index
}

func plane(point geoPoint) orb.Point {
	x, y := geoutils.Planar(point.lat, point.lon)

	return orb.Point{x, y}
}

func (d *driverIndex) near(move *dispatchboardports.BoardMove, k int) []int {
	if move.OriginLatitude == nil || move.OriginLongitude == nil {
		out := make([]int, d.all)
		for i := range out {
			out[i] = i
		}
		return out
	}

	found := d.tree.KNearest(nil, plane(geoPoint{
		lat: *move.OriginLatitude,
		lon: *move.OriginLongitude,
	}), k)
	out := make([]int, 0, len(found))
	for _, pointer := range found {
		if driver, ok := pointer.(indexedDriver); ok {
			out = append(out, driver.index)
		}
	}

	return out
}

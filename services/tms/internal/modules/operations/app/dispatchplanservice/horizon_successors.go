package dispatchplanservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/quadtree"
)

type moveOrigin struct {
	point orb.Point
	index int
}

func (m moveOrigin) Point() orb.Point { return m.point }

func successorsOf(moves []*dispatchboardports.BoardMove, limit int) [][]int {
	out := make([][]int, len(moves))
	if limit <= 0 || len(moves) < 2 {
		return out
	}

	tree := quadtree.New(orb.Bound{
		Min: orb.Point{-180 * geoutils.PlanarLongitudeScale, -90},
		Max: orb.Point{180 * geoutils.PlanarLongitudeScale, 90},
	})
	for i, move := range moves {
		if move.OriginLatitude == nil || move.OriginLongitude == nil {
			continue
		}
		origin := moveOrigin{
			point: successorPlane(*move.OriginLatitude, *move.OriginLongitude),
			index: i,
		}
		if err := tree.Add(origin); err != nil {
			continue
		}
	}

	found := make([]orb.Pointer, 0, limit)
	for i, move := range moves {
		if move.DestinationLatitude == nil || move.DestinationLongitude == nil {
			continue
		}
		ready := earliestFree(move)
		found = tree.KNearestMatching(
			found[:0],
			successorPlane(*move.DestinationLatitude, *move.DestinationLongitude),
			limit,
			func(pointer orb.Pointer) bool {
				next, ok := pointer.(moveOrigin)
				return ok && next.index != i && reachable(move, moves[next.index], ready)
			},
		)
		out[i] = make([]int, 0, len(found))
		for _, pointer := range found {
			if next, ok := pointer.(moveOrigin); ok {
				out[i] = append(out[i], next.index)
			}
		}
	}

	return out
}

func successorPlane(latitude, longitude float64) orb.Point {
	x, y := geoutils.Planar(latitude, longitude)

	return orb.Point{x, y}
}

func earliestFree(move *dispatchboardports.BoardMove) int64 {
	loaded := 0.0
	switch {
	case move.Distance != nil:
		loaded = *move.Distance
	case move.OriginLatitude != nil && move.OriginLongitude != nil:
		loaded = geoutils.PlanningRoadMiles(
			*move.OriginLatitude,
			*move.OriginLongitude,
			*move.DestinationLatitude,
			*move.DestinationLongitude,
		)
	}

	return move.OriginWindowStart + geoutils.PlanningDriveSeconds(loaded)
}

func reachable(from, next *dispatchboardports.BoardMove, ready int64) bool {
	latest := next.OriginWindowStart
	if next.OriginWindowEnd != nil {
		latest = max(latest, *next.OriginWindowEnd)
	}
	if latest <= 0 {
		return true
	}
	deadhead := geoutils.PlanningRoadMiles(
		*from.DestinationLatitude,
		*from.DestinationLongitude,
		*next.OriginLatitude,
		*next.OriginLongitude,
	)

	return ready+geoutils.PlanningDriveSeconds(deadhead) <= latest
}

func opensOf(moves []*dispatchboardports.BoardMove) []int64 {
	opens := make([]int64, len(moves))
	for i, move := range moves {
		opens[i] = move.OriginWindowStart
	}

	return opens
}

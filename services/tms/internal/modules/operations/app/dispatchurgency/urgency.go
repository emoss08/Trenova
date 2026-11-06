package dispatchurgency

import "github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"

type Bucket string

const (
	Late     = Bucket("Late")
	Now      = Bucket("Now")
	Today    = Bucket("Today")
	Tomorrow = Bucket("Tomorrow")
	Planned  = Bucket("Planned")
)

func Of(move *dispatchboardports.BoardMove, now int64) Bucket {
	if arrived(move.OriginActualArrive) {
		return deliveryUrgency(move, now)
	}
	if move.OriginWindowStart <= 0 {
		return Planned
	}
	if WindowClose(move.OriginWindowStart, move.OriginWindowEnd) < now {
		return Late
	}

	return bucketFor(max(move.OriginWindowStart-now, 0))
}

func WindowClose(start int64, end *int64) int64 {
	if end != nil && *end > start {
		return *end
	}

	return start
}

func deliveryUrgency(move *dispatchboardports.BoardMove, now int64) Bucket {
	if arrived(move.DestinationActualArrive) || move.DestinationWindowStart <= 0 {
		return Planned
	}
	if WindowClose(move.DestinationWindowStart, move.DestinationWindowEnd) < now {
		return Late
	}

	return bucketFor(max(move.DestinationWindowStart-now, 0))
}

func arrived(at *int64) bool {
	return at != nil && *at > 0
}

func bucketFor(delta int64) Bucket {
	switch {
	case delta < 0:
		return Late
	case delta <= dispatchboardports.UrgencyNowSeconds:
		return Now
	case delta <= dispatchboardports.UrgencyTodaySeconds:
		return Today
	case delta <= 2*dispatchboardports.UrgencyTodaySeconds:
		return Tomorrow
	default:
		return Planned
	}
}

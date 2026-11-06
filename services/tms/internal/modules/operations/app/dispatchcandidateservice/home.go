package dispatchcandidateservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const secondsPerDay = 24 * 60 * 60

func lastTimeOffEnds(
	timeOff []*dispatchboardports.WorkerTimeOff,
	now int64,
) map[pulid.ID]int64 {
	ends := make(map[pulid.ID]int64, len(timeOff))
	for _, pto := range timeOff {
		end := min(pto.EndDate, now)
		if end > ends[pto.WorkerID] {
			ends[pto.WorkerID] = end
		}
	}

	return ends
}

func applyHomeProfile(snapshot *FleetSnapshot, profile *driverProfile) {
	latitude, longitude, ok := profile.worker.HomePoint()
	if ok {
		profile.home, profile.homeKnown = geoPoint{lat: latitude, lon: longitude}, true
	}

	lastHome := snapshot.LastTimeOffEnd[profile.driver.WorkerID]
	if recorded := profile.worker.LastHomeAt; recorded != nil {
		lastHome = max(lastHome, *recorded)
	}
	if position := snapshot.PosByTractor[profile.driver.TractorID]; ok && position != nil &&
		worker.AtHome(latitude, longitude, position.Latitude, position.Longitude) {
		lastHome = max(lastHome, position.RecordedAt)
	}
	profile.lastHomeAt = lastHome
}

func homeOverdueDays(profile *driverProfile, freeAt, targetDays int64) float64 {
	if profile.lastHomeAt <= 0 || freeAt <= profile.lastHomeAt {
		return 0
	}
	daysOut := float64(freeAt-profile.lastHomeAt) / secondsPerDay

	return max(0, daysOut-float64(targetDays))
}

func homewardShare(profile *driverProfile, move *dispatchboardports.BoardMove) float64 {
	if !profile.homeKnown || move.DestinationLatitude == nil || move.DestinationLongitude == nil ||
		profile.originSource == dispatchplanports.OriginUnknown {
		return 0
	}
	fromOrigin := geoutils.HaversineMiles(
		profile.origin.lat, profile.origin.lon, profile.home.lat, profile.home.lon,
	)
	if fromOrigin <= worker.HomeRadiusMiles {
		return 0
	}
	fromDestination := geoutils.HaversineMiles(
		*move.DestinationLatitude, *move.DestinationLongitude, profile.home.lat, profile.home.lon,
	)

	return min(1, max(0, (fromOrigin-fromDestination)/fromOrigin))
}

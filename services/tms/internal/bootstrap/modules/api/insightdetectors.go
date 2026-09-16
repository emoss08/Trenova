package api

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detectors"
)

// newDetectorRegistry names every insight rule this deployment runs.
//
// The list is explicit rather than assembled from an fx group so the order is
// visible and stable: a refresh produces findings in this sequence, which makes
// a run reproducible and a failure easy to attribute. Removing a detector from
// here stops it producing new findings; the ones it already produced stay
// stored, and are hidden from readers because nothing is left to say what
// permission they required.
func newDetectorRegistry(metrics repositories.InsightMetricsRepository) *detector.Registry {
	return detector.NewRegistry(
		detectors.NewOnTimeDecline(metrics),
		detectors.NewUnbilledAging(metrics),
		detectors.NewUnbilledDetention(metrics),
		detectors.NewEmptyMiles(metrics),
		detectors.NewCredentialExpiry(metrics),
	)
}

package routeestimateports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Leg struct {
	From Point
	To   Point
}

type LegEstimate struct {
	Miles        float64
	DriveSeconds int64
	Routed       bool
}

type RouteEstimator interface {
	EstimateLegs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		legs []Leg,
	) ([]LegEstimate, error)
}

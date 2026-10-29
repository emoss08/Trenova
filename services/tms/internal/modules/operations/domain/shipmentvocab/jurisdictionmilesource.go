package shipmentvocab

import (
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
)

const (
	JurisdictionDistanceUnitsMiles      = distanceprofile.DefaultDistanceUnits
	JurisdictionDistanceUnitsKilometers = "Kilometers"
)

type JurisdictionMileSource string

const (
	JurisdictionMileSourceRouteCalculation = JurisdictionMileSource("RouteCalculation")
	JurisdictionMileSourceManual           = JurisdictionMileSource("Manual")
)

func (s JurisdictionMileSource) String() string { return string(s) }

func (s JurisdictionMileSource) IsValid() bool {
	return s == JurisdictionMileSourceRouteCalculation || s == JurisdictionMileSourceManual
}

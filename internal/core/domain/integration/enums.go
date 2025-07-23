// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package integration

// Type represents the type of integration
type Type string

const (
	// GoogleMapsIntegrationType represents Google Maps integration
	GoogleMapsIntegrationType = Type("GoogleMaps")

	// PCMilerIntegrationType represents PCMiler integration
	PCMilerIntegrationType = Type("PCMiler")
)

type Category string

const (
	// Mapping & Routing category
	MappingRoutingCategory = Category("MappingRouting")

	// Freight & Logistics category
	FreightLogisticsCategory = Category("FreightLogistics")

	// Telematics category
	TelematicsCategory = Category("Telematics")
)

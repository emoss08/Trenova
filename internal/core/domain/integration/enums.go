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

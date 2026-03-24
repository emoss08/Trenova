package integration

type Type string

const (
	TypeGoogleMaps = Type("GoogleMaps")
	TypeSamsara    = Type("Samsara")
	TypeHERE       = Type("HERE")
	TypeOpenAI     = Type("OpenAI")
	// TypePCMiler    Type = "PCMiler"
	// TypeMotive     Type = "Motive"
)

type Category string

const (
	CategoryMappingRouting         = Category("MappingRouting")
	CategoryFreightLogistics       = Category("FreightLogistics")
	CategoryTelematics             = Category("Telematics")
	CategoryArtificialIntelligence = Category("ArtificialIntelligence")
)

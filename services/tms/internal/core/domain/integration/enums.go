package integration

type Type string

const (
	TypeGoogleMaps      = Type("GoogleMaps")
	TypeSamsara         = Type("Samsara")
	TypeHERE            = Type("HERE")
	TypeOpenAI          = Type("OpenAI")
	TypeOpenWeatherMap  = Type("OpenWeatherMap")
	TypeExchangeRateAPI = Type("ExchangeRateAPI")
	// TypePCMiler    Type = "PCMiler"
	// TypeMotive     Type = "Motive"
)

type Category string

const (
	CategoryMappingRouting         = Category("MappingRouting")
	CategoryFreightLogistics       = Category("FreightLogistics")
	CategoryTelematics             = Category("Telematics")
	CategoryArtificialIntelligence = Category("ArtificialIntelligence")
	CategoryWeather                = Category("Weather")
	CategoryFinancialData          = Category("FinancialData")
)

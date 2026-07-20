package integration

type Type string

const (
	TypeGoogleMaps         = Type("GoogleMaps")
	TypeSamsara            = Type("Samsara")
	TypeHERE               = Type("HERE")
	TypeOpenAI             = Type("OpenAI")
	TypeOpenWeatherMap     = Type("OpenWeatherMap")
	TypeOANDAExchangeRates = Type("OANDAExchangeRates")
	TypeEIAFuelPrices      = Type("EIAFuelPrices")
	TypePCMiler            = Type("PCMiler")
	TypeResend             = Type("Resend")
	TypeAmazonSES          = Type("AmazonSES")
	TypeSendGrid           = Type("SendGrid")
	TypeMailgun            = Type("Mailgun")
	TypePostmark           = Type("Postmark")
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
	CategoryEmail                  = Category("Email")
)

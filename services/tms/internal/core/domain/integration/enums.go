package integration

type Type string

const (
	TypeGoogleMaps         = Type("GoogleMaps")
	TypeSamsara            = Type("Samsara")
	TypeHERE               = Type("HERE")
	TypeOpenAI             = Type("OpenAI")
	TypeAnthropic          = Type("Anthropic")
	TypeOpenWeatherMap     = Type("OpenWeatherMap")
	TypeOANDAExchangeRates = Type("OANDAExchangeRates")
	TypeEIAFuelPrices      = Type("EIAFuelPrices")
	TypePCMiler            = Type("PCMiler")
	TypeResend             = Type("Resend")
	TypeAmazonSES          = Type("AmazonSES")
	TypeSendGrid           = Type("SendGrid")
	TypeMailgun            = Type("Mailgun")
	TypePostmark           = Type("Postmark")
	TypeWEXFuel            = Type("WEXFuel")
	TypeComdataFuel        = Type("ComdataFuel")
	TypeRampFuel           = Type("RampFuel")
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
	CategoryFuelCards              = Category("FuelCards")
)

func (v Type) IsValid() bool {
	switch v {
	case TypeGoogleMaps,
		TypeSamsara,
		TypeHERE,
		TypeOpenAI,
		TypeAnthropic,
		TypeOpenWeatherMap,
		TypeOANDAExchangeRates,
		TypeEIAFuelPrices,
		TypePCMiler,
		TypeResend,
		TypeAmazonSES,
		TypeSendGrid,
		TypeMailgun,
		TypePostmark,
		TypeWEXFuel,
		TypeComdataFuel,
		TypeRampFuel:
		return true
	default:
		return false
	}
}

func (v Category) IsValid() bool {
	switch v {
	case CategoryMappingRouting,
		CategoryFreightLogistics,
		CategoryTelematics,
		CategoryArtificialIntelligence,
		CategoryWeather,
		CategoryFinancialData,
		CategoryEmail,
		CategoryFuelCards:
		return true
	default:
		return false
	}
}

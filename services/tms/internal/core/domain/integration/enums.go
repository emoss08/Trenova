package integration

type Type string

const (
	TypeGoogleMaps         = Type("GoogleMaps")
	TypeSamsara            = Type("Samsara")
	TypeHERE               = Type("HERE")
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
	TypeCarrierOK          = Type("CarrierOK")
	TypeFMCSAQCMobile      = Type("FMCSAQCMobile")
	TypeQuickBooksOnline   = Type("QuickBooksOnline")
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
	CategoryCarrierCompliance      = Category("CarrierCompliance")
	CategoryAccounting             = Category("Accounting")
)

func (v Type) String() string { return string(v) }

func (v Category) String() string { return string(v) }

func (v Type) IsValid() bool {
	switch v {
	case TypeGoogleMaps,
		TypeSamsara,
		TypeHERE,
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
		TypeRampFuel,
		TypeCarrierOK,
		TypeFMCSAQCMobile,
		TypeQuickBooksOnline:
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
		CategoryFuelCards,
		CategoryCarrierCompliance,
		CategoryAccounting:
		return true
	default:
		return false
	}
}

func (v Type) SupportsCarrierIntelligence() bool {
	switch v {
	case TypeCarrierOK, TypeFMCSAQCMobile:
		return true
	default:
		return false
	}
}

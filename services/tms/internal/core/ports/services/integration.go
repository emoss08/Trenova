package services

import (
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	catalogEmailCategoryLabel   = "Email"
	catalogDocsLabel            = "Docs"
	catalogWebsiteLabel         = "Website"
	catalogViewIntegrationLabel = "View Integration"
	catalogPostmarkLogoURL      = "/integrations/logos/postmark_all.png"
	catalogSlateColor           = "#0f172a"
	catalogGoogleMapsLogoURL    = "/integrations/logos/googleMaps.svg"
	catalogOandaAPIURL          = "https://exchange-rates-api.oanda.com/"
)

var CatalogDefinitions = []CatalogItem{
	{
		Type:          integration.TypeResend,
		Name:          "Resend",
		Description:   "Transactional email delivery for invoices, authentication, reporting, and operational notifications.",
		Category:      integration.CategoryEmail,
		CategoryLabel: catalogEmailCategoryLabel,
		LogoURL:       "/integrations/logos/resend_logo_light.svg",
		LogoLightURL:  "/integrations/logos/resend_logo_light.svg",
		LogoDarkURL:   "/integrations/logos/resend_logo_dark.svg",
		DocsURL:       "https://resend.com/docs",
		WebsiteURL:    "https://resend.com/",
		Color:         "#111827",
		GlowFrom:      "#111827",
		GlowTo:        "#22c55e",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://resend.com/docs",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://resend.com/",
			},
		},
		Featured:           true,
		SortOrder:          5,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	plannedEmailCatalogItem(&plannedEmailCatalogItemParams{
		Type:         integration.TypeAmazonSES,
		Name:         "Amazon SES",
		Description:  "Planned support for Amazon Simple Email Service transactional delivery.",
		DocsURL:      "https://docs.aws.amazon.com/ses/",
		WebsiteURL:   "https://aws.amazon.com/ses/",
		Color:        "#ff9900",
		LogoLightURL: "/integrations/logos/aws_light.svg",
		LogoDarkURL:  "/integrations/logos/aws_dark.svg",
		SortOrder:    55,
	}),
	plannedEmailCatalogItem(&plannedEmailCatalogItemParams{
		Type:         integration.TypeSendGrid,
		Name:         "SendGrid",
		Description:  "Planned support for SendGrid transactional email delivery.",
		DocsURL:      "https://www.twilio.com/docs/sendgrid",
		WebsiteURL:   "https://sendgrid.com/",
		Color:        "#1a82e2",
		LogoLightURL: "/integrations/logos/sendgrid_light.svg",
		LogoDarkURL:  "/integrations/logos/sendgrid_dark.svg",
		SortOrder:    56,
	}),
	plannedEmailCatalogItem(&plannedEmailCatalogItemParams{
		Type:         integration.TypeMailgun,
		Name:         "Mailgun",
		Description:  "Planned support for Mailgun transactional email delivery.",
		DocsURL:      "https://documentation.mailgun.com/",
		WebsiteURL:   "https://www.mailgun.com/",
		Color:        "#c21f32",
		LogoLightURL: "/integrations/logos/mailgun_light.svg",
		LogoDarkURL:  "/integrations/logos/mailgun_dark.svg",
		SortOrder:    57,
	}),
	{
		Type:          integration.TypePostmark,
		Name:          "Postmark",
		Description:   "Transactional email delivery with server streams, attachments, and delivery event webhooks.",
		Category:      integration.CategoryEmail,
		CategoryLabel: catalogEmailCategoryLabel,
		LogoURL:       catalogPostmarkLogoURL,
		LogoLightURL:  catalogPostmarkLogoURL,
		LogoDarkURL:   catalogPostmarkLogoURL,
		DocsURL:       "https://postmarkapp.com/developer",
		WebsiteURL:    "https://postmarkapp.com/",
		Color:         "#ffde00",
		GlowFrom:      "#ffde00",
		GlowTo:        catalogSlateColor,
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://postmarkapp.com/developer",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://postmarkapp.com/",
			},
		},
		Featured:           true,
		SortOrder:          6,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeSamsara,
		Name:          "Samsara",
		Description:   "Seamlessly connect your Samsara account to Trenova for real-time telematics data, driver performance insights, and streamlined fleet management. Unlock the full potential of your fleet with our powerful integration.",
		Category:      integration.CategoryTelematics,
		CategoryLabel: "Telematics",
		LogoURL:       "/integrations/logos/samsara.webp",
		LogoLightURL:  "/integrations/logos/samsara.webp",
		LogoDarkURL:   "/integrations/logos/samsara_logo_white.webp",
		DocsURL:       "https://developers.samsara.com/docs/tms-integration",
		WebsiteURL:    "https://www.samsara.com/",
		Color:         "#002e8a",
		GlowFrom:      "#002e8a",
		GlowTo:        "#00b4d8",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://developers.samsara.com/docs/tms-integration",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://www.samsara.com/",
			},
		},
		Featured:           true,
		SortOrder:          10,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeGoogleMaps,
		Name:          "Google Maps",
		Description:   "Routing and geocoding",
		Category:      integration.CategoryMappingRouting,
		CategoryLabel: "Mapping & Routing",
		LogoURL:       catalogGoogleMapsLogoURL,
		LogoLightURL:  catalogGoogleMapsLogoURL,
		LogoDarkURL:   catalogGoogleMapsLogoURL,
		DocsURL:       "https://developers.google.com/maps/documentation",
		WebsiteURL:    "https://maps.google.com/",
		Color:         "#8a0000",
		GlowFrom:      "#4285f4",
		GlowTo:        "#34a853",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://developers.google.com/maps/documentation",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://maps.google.com/",
			},
		},
		Featured:           false,
		SortOrder:          20,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeOpenAI,
		Name:          "OpenAI",
		Description:   "AI-powered document classification and structured extraction for document intelligence workflows.",
		Category:      integration.CategoryArtificialIntelligence,
		CategoryLabel: "AI & Automation",
		LogoURL:       "/integrations/logos/openai_logo.svg",
		LogoLightURL:  "/integrations/logos/openai_logo.svg",
		LogoDarkURL:   "/integrations/logos/openai_logo_white.svg",
		DocsURL:       "https://platform.openai.com/docs",
		WebsiteURL:    "https://openai.com/",
		Color:         catalogSlateColor,
		GlowFrom:      catalogSlateColor,
		GlowTo:        "#10a37f",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://platform.openai.com/docs",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://openai.com/",
			},
		},
		Featured:           true,
		SortOrder:          30,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeOpenWeatherMap,
		Name:          "OpenWeatherMap",
		Description:   "Real-time weather map overlays including wind speed, cloud cover, temperature, and atmospheric pressure layers for fleet route planning.",
		Category:      integration.CategoryWeather,
		CategoryLabel: "Weather",
		LogoURL:       "/integrations/logos/open_weather_logo.webp",
		LogoLightURL:  "/integrations/logos/open_weather_logo.webp",
		LogoDarkURL:   "/integrations/logos/open_weather_dark.webp",
		DocsURL:       "https://openweathermap.org/api/weathermaps",
		WebsiteURL:    "https://openweathermap.org/",
		Color:         "#eb6e4b",
		GlowFrom:      "#eb6e4b",
		GlowTo:        "#f9d423",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://openweathermap.org/api/weathermaps",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://openweathermap.org/",
			},
		},
		Featured:           false,
		SortOrder:          25,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeOANDAExchangeRates,
		Name:          "OANDA Exchange Rates",
		Description:   "Settlement-grade FX data for multi-currency accounting, invoicing, and audit workflows with bid, ask, midpoint, and historical fixing support.",
		Category:      integration.CategoryFinancialData,
		CategoryLabel: "Financial Data",
		LogoURL:       "/integrations/logos/oanada-light.svg",
		LogoLightURL:  "/integrations/logos/oanada-light.svg",
		LogoDarkURL:   "/integrations/logos/oanada-dark.svg",
		DocsURL:       catalogOandaAPIURL,
		WebsiteURL:    "https://www.oanda.com/foreign-exchange-data-services/en/exchange-rates-api/",
		Color:         "#00a86b",
		GlowFrom:      "#00a86b",
		GlowTo:        "#372563",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   catalogOandaAPIURL,
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://www.oanda.com/foreign-exchange-data-services/en/exchange-rates-api/",
			},
			{
				Kind:  CatalogLinkKindAPI,
				Label: "API Reference",
				URL:   catalogOandaAPIURL,
			},
		},
		Featured:           false,
		SortOrder:          40,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypeEIAFuelPrices,
		Name:          "EIA Fuel Prices",
		Description:   "Automatic weekly DOE/EIA on-highway diesel price ingestion (U.S. average, PADD regions, California) that drives fuel surcharge programs and billing.",
		Category:      integration.CategoryFinancialData,
		CategoryLabel: "Financial Data",
		LogoURL:       "/integrations/logos/eia-light.svg",
		LogoLightURL:  "/integrations/logos/eia-light.svg",
		LogoDarkURL:   "/integrations/logos/eia-dark.svg",
		DocsURL:       "https://www.eia.gov/opendata/documentation.php",
		WebsiteURL:    "https://www.eia.gov/petroleum/gasdiesel/",
		Color:         "#1a6fb5",
		GlowFrom:      "#1a6fb5",
		GlowTo:        "#0e3a61",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://www.eia.gov/opendata/documentation.php",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: "Gasoline & Diesel Fuel Update",
				URL:   "https://www.eia.gov/petroleum/gasdiesel/",
			},
			{
				Kind:  CatalogLinkKindAPI,
				Label: "API Browser",
				URL:   "https://www.eia.gov/opendata/browser/petroleum/pri/gnd",
			},
		},
		Featured:           false,
		SortOrder:          41,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
	{
		Type:          integration.TypePCMiler,
		Name:          "PC*Miler",
		Description:   "Server-side mileage rating with Trimble PC*Miler route reports, data versions, and truck routing options.",
		Category:      integration.CategoryMappingRouting,
		CategoryLabel: "Mapping & Routing",
		LogoURL:       "/integrations/logos/pc-miler-logo-light.png",
		LogoLightURL:  "/integrations/logos/pc-miler-logo-light.png",
		LogoDarkURL:   "/integrations/logos/pc-miler-logo-dark.svg",
		DocsURL:       "https://developer.trimblemaps.com/restful-apis/routing/route-reports/post-route-reports/",
		WebsiteURL:    "https://maps.trimble.com/pcmiler/",
		Color:         "#155e75",
		GlowFrom:      "#155e75",
		GlowTo:        "#84cc16",
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   "https://developer.trimblemaps.com/restful-apis/routing/route-reports/post-route-reports/",
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   "https://maps.trimble.com/pcmiler/",
			},
		},
		Featured:           false,
		SortOrder:          21,
		PrimaryActionLabel: catalogViewIntegrationLabel,
	},
}

type plannedEmailCatalogItemParams struct {
	Type         integration.Type
	Name         string
	Description  string
	DocsURL      string
	WebsiteURL   string
	Color        string
	LogoLightURL string
	LogoDarkURL  string
	SortOrder    int
}

func plannedEmailCatalogItem(p *plannedEmailCatalogItemParams) CatalogItem {
	return CatalogItem{
		Type:          p.Type,
		Name:          p.Name,
		Description:   p.Description,
		Category:      integration.CategoryEmail,
		CategoryLabel: catalogEmailCategoryLabel,
		LogoURL:       p.LogoLightURL,
		LogoLightURL:  p.LogoLightURL,
		LogoDarkURL:   p.LogoDarkURL,
		Color:         p.Color,
		GlowFrom:      p.Color,
		GlowTo:        "#64748b",
		DocsURL:       p.DocsURL,
		WebsiteURL:    p.WebsiteURL,
		Links: []CatalogLink{
			{
				Kind:  CatalogLinkKindDocs,
				Label: catalogDocsLabel,
				URL:   p.DocsURL,
			},
			{
				Kind:  CatalogLinkKindWebsite,
				Label: catalogWebsiteLabel,
				URL:   p.WebsiteURL,
			},
		},
		Featured:           false,
		SortOrder:          p.SortOrder,
		PrimaryActionLabel: "Planned",
		ConfigSpec:         []integration.ConfigFieldSpec{},
	}
}

type CatalogLinkKind string

const (
	CatalogLinkKindDocs    CatalogLinkKind = "docs"
	CatalogLinkKindWebsite CatalogLinkKind = "website"
	CatalogLinkKindSupport CatalogLinkKind = "support"
	CatalogLinkKindAPI     CatalogLinkKind = "api"
)

type CatalogConnectionStatus string

const (
	CatalogConnectionStatusConnected    CatalogConnectionStatus = "connected"
	CatalogConnectionStatusDisconnected CatalogConnectionStatus = "disconnected"
)

type CatalogConfigurationStatus string

const (
	CatalogConfigurationStatusConfigured CatalogConfigurationStatus = "configured"
	CatalogConfigurationStatusNeedsSetup CatalogConfigurationStatus = "needs_setup"
)

type CatalogLink struct {
	Kind  CatalogLinkKind `json:"kind"`
	Label string          `json:"label"`
	URL   string          `json:"url"`
}

type CatalogStatus struct {
	Connection         CatalogConnectionStatus    `json:"connection"`
	ConnectionLabel    string                     `json:"connectionLabel"`
	Configuration      CatalogConfigurationStatus `json:"configuration"`
	ConfigurationLabel string                     `json:"configurationLabel"`
}

type CatalogItem struct {
	Type                integration.Type              `json:"type"`
	Name                string                        `json:"name"`
	Description         string                        `json:"description"`
	Category            integration.Category          `json:"category"`
	CategoryLabel       string                        `json:"categoryLabel"`
	LogoURL             string                        `json:"logoUrl"`
	LogoLightURL        string                        `json:"logoLightUrl,omitempty"`
	LogoDarkURL         string                        `json:"logoDarkUrl,omitempty"`
	Color               string                        `json:"color,omitempty"`
	GlowFrom            string                        `json:"glowFrom,omitempty"`
	GlowTo              string                        `json:"glowTo,omitempty"`
	DocsURL             string                        `json:"docsUrl,omitempty"`
	WebsiteURL          string                        `json:"websiteUrl,omitempty"`
	Links               []CatalogLink                 `json:"links"`
	Featured            bool                          `json:"featured"`
	SortOrder           int                           `json:"sortOrder"`
	PrimaryActionLabel  string                        `json:"primaryActionLabel"`
	Enabled             bool                          `json:"enabled"`
	Configured          bool                          `json:"configured"`
	Status              CatalogStatus                 `json:"status"`
	ConfigSpec          []integration.ConfigFieldSpec `json:"configSpec"`
	SupportsTestConnect bool                          `json:"supportsTestConnect"`
}

type CatalogResponse struct {
	Items []CatalogItem `json:"items"`
}

type TestConnectionResponse struct {
	Provider  integration.Type `json:"provider"`
	Success   bool             `json:"success"`
	CheckedAt int64            `json:"checkedAt"`
}

type UpdateConfigRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	Enabled       bool                  `json:"enabled"`
	Configuration map[string]string     `json:"configuration"`
}

type ConfigFieldValue struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	HasValue bool   `json:"hasValue"`
}

type ConfigResponse struct {
	Type      integration.Type              `json:"type"`
	Enabled   bool                          `json:"enabled"`
	Fields    []ConfigFieldValue            `json:"fields"`
	Spec      []integration.ConfigFieldSpec `json:"spec"`
	UpdatedAt int64                         `json:"updatedAt"`
}

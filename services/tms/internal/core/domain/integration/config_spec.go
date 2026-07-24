package integration

import "strings"

type ConfigFieldType string

const (
	ConfigFieldTypeString   ConfigFieldType = "string"
	ConfigFieldTypeURL      ConfigFieldType = "url"
	ConfigFieldTypePassword ConfigFieldType = "password"
	ConfigFieldTypeSelect   ConfigFieldType = "select"
	ConfigFieldTypeBoolean  ConfigFieldType = "boolean"
	ConfigFieldTypeNumber   ConfigFieldType = "number"
	ConfigFieldTypeMulti    ConfigFieldType = "multi-select"
)

type ConfigFieldSpec struct {
	Key         string          `json:"key"`
	Label       string          `json:"label"`
	Type        ConfigFieldType `json:"type"`
	Required    bool            `json:"required"`
	Sensitive   bool            `json:"sensitive"`
	Placeholder string          `json:"placeholder,omitempty"`
	HelpText    string          `json:"helpText,omitempty"`
	Default     string          `json:"default,omitempty"`
	Options     []string        `json:"options,omitempty"`
}

type IntegrationSpec struct {
	Fields              []ConfigFieldSpec `json:"fields"`
	SupportsTestConnect bool              `json:"supportsTestConnect"`
}

var ConfigSpecs = map[Type]IntegrationSpec{
	TypeGoogleMaps: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
		},
	},
	TypeSamsara: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "token",
				Label:     "API Token",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
			{
				Key:     "baseUrl",
				Label:   "Base URL",
				Type:    ConfigFieldTypeURL,
				Default: "https://api.samsara.com",
			},
			{
				Key:       "webhookSecret",
				Label:     "Webhook Secret",
				Type:      ConfigFieldTypePassword,
				Sensitive: true,
				HelpText:  "Base64 signing secret from the Samsara webhook configuration, used to verify X-Samsara-Signature on inbound events.",
			},
			{
				Key:       "webhookToken",
				Label:     "Webhook Token",
				Type:      ConfigFieldTypeString,
				Sensitive: false,
			},
		},
		SupportsTestConnect: true,
	},
	TypeHERE: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
		},
	},
	TypeOpenAI: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
		},
	},
	TypeAnthropic: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
		},
	},
	TypeOpenWeatherMap: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Free tier includes 1,000 API calls/day for weather map tiles.",
			},
		},
	},
	TypeOANDAExchangeRates: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "OANDA FX Data Services API key. The key is sent with Bearer authorization and never exposed to browsers.",
			},
			{
				Key:         "baseUrl",
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://exchange-rates-api.oanda.com",
				Placeholder: "https://exchange-rates-api.oanda.com",
			},
			{
				Key:      "defaultRateType",
				Label:    "Default Rate Type",
				Type:     ConfigFieldTypeSelect,
				Default:  "mid",
				Options:  []string{"mid", "bid", "ask"},
				HelpText: "Midpoint is the default settlement policy for FX quotes.",
			},
		},
		SupportsTestConnect: true,
	},
	TypeEIAFuelPrices: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Free EIA Open Data API key from eia.gov/opendata. The key is sent as a query parameter to the EIA API only and never exposed to browsers.",
			},
			{
				Key:         "baseUrl",
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://api.eia.gov/v2",
				Placeholder: "https://api.eia.gov/v2",
			},
		},
		SupportsTestConnect: true,
	},
	TypePCMiler: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Trimble Maps API key. The key is used only by the server and is never exposed to browsers.",
			},
			{
				Key:         "baseUrl",
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://pcmiler.alk.com/apis/rest/v1.0/Service.svc",
				Placeholder: "https://pcmiler.alk.com/apis/rest/v1.0/Service.svc",
			},
		},
		SupportsTestConnect: true,
	},
	TypeResend: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "apiKey",
				Label:     "API Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Resend API key used by Trenova for transactional email sends.",
			},
			{
				Key:         "baseUrl",
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://api.resend.com",
				Placeholder: "https://api.resend.com",
			},
			{
				Key:       "webhookSigningSecret",
				Label:     "Webhook Signing Secret",
				Type:      ConfigFieldTypePassword,
				Sensitive: true,
				HelpText:  "Svix signing secret from the Resend webhook endpoint.",
			},
			{
				Key:       "webhookToken",
				Label:     "Webhook Token",
				Type:      ConfigFieldTypeString,
				Sensitive: false,
			},
		},
		SupportsTestConnect: true,
	},
	TypePostmark: {
		Fields: []ConfigFieldSpec{
			{
				Key:       "serverToken",
				Label:     "Server Token",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Postmark server token used by Trenova for transactional email sends.",
			},
			{
				Key:         "baseUrl",
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://api.postmarkapp.com",
				Placeholder: "https://api.postmarkapp.com",
			},
			{
				Key:         "messageStream",
				Label:       "Message Stream",
				Type:        ConfigFieldTypeString,
				Default:     "outbound",
				Placeholder: "outbound",
			},
			{
				Key:       "webhookToken",
				Label:     "Webhook Token",
				Type:      ConfigFieldTypeString,
				Sensitive: false,
			},
		},
		SupportsTestConnect: true,
	},
}

func HasRequiredConfiguration(configuration map[string]any, spec IntegrationSpec) bool {
	for _, field := range spec.Fields {
		if !field.Required {
			continue
		}
		if ReadConfigString(configuration, field.Key) == "" {
			return false
		}
	}
	return true
}

func ReadConfigString(configuration map[string]any, key string) string {
	if len(configuration) == 0 {
		return ""
	}

	value, ok := configuration[key]
	if !ok || value == nil {
		return ""
	}

	stringValue, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(stringValue)
}

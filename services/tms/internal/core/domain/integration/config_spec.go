package integration

import "strings"

type ConfigFieldType string

const (
	ConfigFieldTypeString   ConfigFieldType = "string"
	ConfigFieldTypeURL      ConfigFieldType = "url"
	ConfigFieldTypePassword ConfigFieldType = "password"
	ConfigFieldTypeSelect   ConfigFieldType = "select"
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

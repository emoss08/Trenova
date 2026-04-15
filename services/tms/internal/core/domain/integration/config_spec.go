package integration

type ConfigFieldType string

const (
	ConfigFieldTypeString   ConfigFieldType = "string"
	ConfigFieldTypeURL      ConfigFieldType = "url"
	ConfigFieldTypePassword ConfigFieldType = "password"
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
}

package integration

const (
	ConfigKeyCarrierIntelAPIKey  = "apiKey"
	ConfigKeyCarrierIntelWebKey  = "webKey"
	ConfigKeyCarrierIntelBaseURL = "baseUrl"
	ConfigKeyCarrierIntelRole    = "role"

	CarrierIntelRolePrimary  = "primary"
	CarrierIntelRoleFallback = "fallback"

	DefaultCarrierOKBaseURL = "https://api.carrierok.com"
	DefaultFMCSABaseURL     = "https://mobile.fmcsa.dot.gov/qc/services"
)

func carrierOKSpec() IntegrationSpec {
	return IntegrationSpec{
		Fields: []ConfigFieldSpec{
			{
				Key:         ConfigKeyCarrierIntelAPIKey,
				Label:       "API Key",
				Type:        ConfigFieldTypePassword,
				Required:    true,
				Sensitive:   true,
				Placeholder: "sk_live_...",
				HelpText:    "Your CarrierOk API key from developers.carrierok.com. Live keys start with sk_live_ and are billed per carrier; sandbox keys start with sk_test_ and only return the ten fixture carriers.",
			},
			{
				Key:         ConfigKeyCarrierIntelBaseURL,
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     DefaultCarrierOKBaseURL,
				Placeholder: DefaultCarrierOKBaseURL,
			},
		},
		SupportsTestConnect: true,
		BindSecretsToTenant: true,
	}
}

func fmcsaQCMobileSpec() IntegrationSpec {
	return IntegrationSpec{
		Fields: []ConfigFieldSpec{
			{
				Key:       ConfigKeyCarrierIntelWebKey,
				Label:     "Web Key",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
				HelpText:  "Free FMCSA QCMobile web key from mobile.fmcsa.dot.gov/QCDevsite. Used only by the server.",
			},
			{
				Key:         ConfigKeyCarrierIntelBaseURL,
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     DefaultFMCSABaseURL,
				Placeholder: DefaultFMCSABaseURL,
			},
			{
				Key:      ConfigKeyCarrierIntelRole,
				Label:    "Role",
				Type:     ConfigFieldTypeSelect,
				Default:  CarrierIntelRolePrimary,
				Options:  []string{CarrierIntelRolePrimary, CarrierIntelRoleFallback},
				HelpText: "Primary makes FMCSA the organization's carrier intelligence source. Fallback keeps another provider primary and uses FMCSA only for lookups while that provider is unavailable.",
			},
		},
		SupportsTestConnect: true,
		BindSecretsToTenant: true,
	}
}

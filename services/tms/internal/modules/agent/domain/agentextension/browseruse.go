package agentextension

import (
	"github.com/emoss08/trenova/internal/modules/kernel/domain/configspec"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	BrowserUseModelDefault = "browser-use-2.0"
	BrowserUseModelLuna    = "gpt-5.6-luna"
	BrowserUseModelSonnet  = "claude-sonnet-5"
	BrowserUseModelGemini  = "gemini-3-flash-preview"

	ProxyCountryUS   = "us"
	ProxyCountryCA   = "ca"
	ProxyCountryMX   = "mx"
	ProxyCountryNone = "none"
)

var browserUseModels = []string{
	BrowserUseModelDefault,
	BrowserUseModelLuna,
	BrowserUseModelSonnet,
	BrowserUseModelGemini,
}

var proxyCountries = []string{ProxyCountryUS, ProxyCountryCA, ProxyCountryMX, ProxyCountryNone}

func browserUseSpec() Spec {
	return Spec{
		Capability: CapabilityPortalAutomation,
		Fields: portalAutomationFields(&portalFieldsParams{
			apiKey: APIKeyField(
				"Your Browser Use API key",
				"Create a key at cloud.browser-use.com. Trenova stores it encrypted and only sends it to api.browser-use.com.",
			),
			vendor: []configspec.Field{
				{
					Key:      ConfigKeyBrowserModel,
					Label:    "Browser model",
					Type:     configspec.FieldTypeSelect,
					Default:  BrowserUseModelDefault,
					Options:  browserUseModels,
					HelpText: "The model that drives the browser. Browser Use's own model is built for it and costs least per step; the others reason more carefully on complicated portals and cost more.",
				},
				{
					Key:      ConfigKeyProxyCountry,
					Label:    "Browse from",
					Type:     configspec.FieldTypeSelect,
					Default:  ProxyCountryUS,
					Options:  proxyCountries,
					HelpText: "The country the browser appears to browse from. Some portals refuse visitors from abroad; none uses the vendor's own network.",
				},
			},
		}),
		SupportsTestConnect: true,
		DefaultDailyLimit:   DefaultPortalDailyTasks,
		Validate: func(values map[string]string, multiErr *errortypes.MultiError) {
			parseBrowserUseSettings(values, multiErr)
		},
	}
}

type BrowserUseSettings struct {
	PortalAutomationSettings
	Model        string
	ProxyCountry string
}

func ParseBrowserUseSettings(values map[string]string) (BrowserUseSettings, error) {
	multiErr := errortypes.NewMultiError()
	settings := parseBrowserUseSettings(values, multiErr)
	if multiErr.HasErrors() {
		return BrowserUseSettings{}, multiErr
	}

	return settings, nil
}

func parseBrowserUseSettings(
	values map[string]string,
	multiErr *errortypes.MultiError,
) BrowserUseSettings {
	return BrowserUseSettings{
		PortalAutomationSettings: parsePortalAutomationSettings(values, multiErr),
		Model: parseChoice(&choiceField{
			values:   values,
			key:      ConfigKeyBrowserModel,
			fallback: BrowserUseModelDefault,
			options:  browserUseModels,
			message:  "Browser model must be one of the models offered",
		}, multiErr),
		ProxyCountry: parseChoice(&choiceField{
			values:   values,
			key:      ConfigKeyProxyCountry,
			fallback: ProxyCountryUS,
			options:  proxyCountries,
			message:  "Browse from must be us, ca, mx or none",
		}, multiErr),
	}
}

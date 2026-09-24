package quickbooks

import "strings"

type Environment string

const (
	EnvironmentSandbox    = Environment("sandbox")
	EnvironmentProduction = Environment("production")
)

const (
	productionAPIBaseURL = "https://quickbooks.api.intuit.com"
	sandboxAPIBaseURL    = "https://sandbox-quickbooks.api.intuit.com"
	authorizeURL         = "https://appcenter.intuit.com/connect/oauth2"
	oauthBaseURL         = "https://oauth.platform.intuit.com"
	oauthExchangePath    = "/oauth2/v1/tokens/bearer"
	revokeBaseURL        = "https://developer.api.intuit.com"
	revokePath           = "/v2/oauth2/tokens/revoke"
	productionAppURL     = "https://app.qbo.intuit.com"
	sandboxAppURL        = "https://app.sandbox.qbo.intuit.com"
	accountingScope      = "com.intuit.quickbooks.accounting"
	minorVersion         = "75"
)

func ParseEnvironment(value string) (Environment, bool) {
	switch Environment(strings.ToLower(strings.TrimSpace(value))) {
	case EnvironmentSandbox:
		return EnvironmentSandbox, true
	case EnvironmentProduction:
		return EnvironmentProduction, true
	default:
		return "", false
	}
}

func (e Environment) APIBaseURL() string {
	if e == EnvironmentSandbox {
		return sandboxAPIBaseURL
	}
	return productionAPIBaseURL
}

func (e Environment) AppBaseURL() string {
	if e == EnvironmentSandbox {
		return sandboxAppURL
	}
	return productionAppURL
}

func AllowedHosts() []string {
	return []string{
		"quickbooks.api.intuit.com",
		"sandbox-quickbooks.api.intuit.com",
		"appcenter.intuit.com",
		"oauth.platform.intuit.com",
		"developer.api.intuit.com",
	}
}

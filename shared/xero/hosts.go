package xero

const (
	apiBaseURL      = "https://api.xero.com"
	identityBaseURL = "https://identity.xero.com"
	loginBaseURL    = "https://login.xero.com"
	authorizePath   = "/identity/connect/authorize"
	tokenPath       = "/connect/token"
	revocationPath  = "/connect/revocation"
	connectionsPath = "/connections"
	accountingRoot  = "api.xro/2.0"
)

func AllowedHosts() []string {
	return []string{
		"api.xero.com",
		"identity.xero.com",
		"login.xero.com",
	}
}

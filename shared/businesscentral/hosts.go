package businesscentral

const (
	apiBaseURL       = "https://api.businesscentral.dynamics.com"
	loginBaseURL     = "https://login.microsoftonline.com"
	webBaseURL       = "https://businesscentral.dynamics.com"
	authorizePath    = "/organizations/oauth2/v2.0/authorize"
	tokenPath        = "/organizations/oauth2/v2.0/token"
	environmentsPath = "/environments/v1.2"
	apiVersionRoot   = "v2.0"
	apiSegment       = "api/v2.0"
)

func AllowedHosts() []string {
	return []string{
		"api.businesscentral.dynamics.com",
		"login.microsoftonline.com",
		"businesscentral.dynamics.com",
	}
}

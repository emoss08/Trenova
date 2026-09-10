package config

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

var ValidEnvs = []string{EnvDevelopment, EnvStaging, EnvProduction, EnvTest}

type RoutingProvider string

const (
	RoutingProviderHERE    = RoutingProvider("here")
	RoutingProviderPCMiler = RoutingProvider("pcmiler")
)

const (
	TrustedPlatformCloudflare      = "cloudflare"
	TrustedPlatformGoogleAppEngine = "google-app-engine"
	TrustedPlatformFlyIO           = "flyio"
)

var DefaultTrustedProxies = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"fe80::/10",
	"fc00::/7",
}

type PlatformMode string

const (
	PlatformModeCommunity   = PlatformMode("community")
	PlatformModeSelfHosted  = PlatformMode("self_hosted")
	PlatformModeDevelopment = PlatformMode("development")
	PlatformModeCloud       = PlatformMode("cloud")
	PlatformModeEnterprise  = PlatformMode("enterprise")
)

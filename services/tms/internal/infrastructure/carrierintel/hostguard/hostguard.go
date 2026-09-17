package hostguard

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/httpsafe"
)

const (
	schemeHTTPS    = "https"
	schemeHTTP     = "http"
	localhostName  = "localhost"
	loopbackIPv4   = "127.0.0.1"
	DefaultTimeout = 30 * time.Second
)

type Error struct {
	Message string
}

func (e *Error) Error() string { return e.Message }

func newError(message string) error {
	return &Error{Message: message}
}

type Target struct {
	BaseURL  string
	Loopback bool
}

func ValidateBaseURL(raw, fallback string, allowed []string) (string, error) {
	target, err := validate(raw, fallback, allowed)
	if err != nil {
		return "", err
	}
	return target.BaseURL, nil
}

func Resolve(raw, fallback string, cfg *config.Config) (*Target, error) {
	target, err := validate(raw, fallback, cfg.CarrierIntelligence.GetAllowedHosts())
	if err != nil {
		return nil, err
	}
	if target.Loopback && !cfg.CarrierIntelligence.IsSandboxAllowed(&cfg.App) {
		return nil, newError("Local base URLs are not allowed in this environment")
	}
	return target, nil
}

func (t *Target) HTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return httpsafe.NewClientWithPolicy(timeout, httpsafe.Policy{AllowPrivateNetworks: t.Loopback})
}

func IsLoopbackHost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	return normalized == localhostName || normalized == loopbackIPv4
}

func validate(raw, fallback string, allowed []string) (*Target, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" {
		return nil, newError("Base URL is required")
	}

	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host == "" || parsed.Opaque != "" {
		return nil, newError("Base URL is not a valid URL")
	}
	if parsed.User != nil {
		return nil, newError("Base URL must not include credentials")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, newError("Base URL must not include a query string or fragment")
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, newError("Base URL is not a valid URL")
	}
	loopback := IsLoopbackHost(host)

	scheme := strings.ToLower(parsed.Scheme)
	switch {
	case scheme == schemeHTTPS:
	case scheme == schemeHTTP && loopback:
	default:
		return nil, newError("Base URL must use HTTPS")
	}

	if !loopback && !hostAllowed(host, allowed) {
		return nil, newError("Base URL host is not an approved carrier intelligence host")
	}

	hostPort := host
	if port := parsed.Port(); port != "" {
		hostPort = net.JoinHostPort(host, port)
	}

	path := strings.TrimRight(parsed.EscapedPath(), "/")
	return &Target{
		BaseURL:  scheme + "://" + hostPort + path,
		Loopback: loopback,
	}, nil
}

func hostAllowed(host string, allowed []string) bool {
	for _, entry := range allowed {
		if strings.EqualFold(strings.TrimSpace(entry), host) {
			return true
		}
	}
	return false
}

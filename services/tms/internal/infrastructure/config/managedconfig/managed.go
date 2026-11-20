package managedconfig

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	defaultHost      = "127.0.0.1"
	defaultPort      = 8090
	minTokenLength   = 32
	maxModelLength   = 200
	maxSupportURLLen = 2048
)

var (
	ErrTokenRequired = fmt.Errorf(
		"managed.token must be at least %d characters when managed.enabled is true",
		minTokenLength,
	)
	ErrInstanceIDInvalid = errors.New(
		"managed.instanceId must be the instance's ID, a lowercase UUID, when managed.enabled is true",
	)
	ErrPortInvalid         = errors.New("managed.port must be between 1 and 65535")
	ErrEmbeddingURLInvalid = errors.New(
		"managed.embedding.url must be an http or https URL with a host and nothing after it",
	)
	ErrEmbeddingModelMissing = fmt.Errorf(
		"managed.embedding.model must name the model the service serves, at most %d characters, when managed.embedding.url is set",
		maxModelLength,
	)
	ErrEmbeddingModelWithoutURL = errors.New(
		"managed.embedding.model is set without managed.embedding.url",
	)
	ErrTenancyInvalid = errors.New(
		"managed.tenancy must be dedicated or shared",
	)
	ErrSharedTenancyNotManaged = errors.New(
		"managed.tenancy shared requires managed.enabled",
	)
	ErrSupportURLInvalid = fmt.Errorf(
		"managed.supportUrl and managed.newSupportRequestUrl must each be empty or an http or https URL with a host, no user information, fragment or space, at most %d characters",
		maxSupportURLLen,
	)
)

type Tenancy string

const (
	TenancyDedicated = Tenancy("dedicated")
	TenancyShared    = Tenancy("shared")
)

type Config struct {
	Enabled              bool      `mapstructure:"enabled"`
	Tenancy              Tenancy   `mapstructure:"tenancy"`
	InstanceID           string    `mapstructure:"instanceId"`
	Host                 string    `mapstructure:"host"                 validate:"omitempty,hostname|ip"`
	Port                 int       `mapstructure:"port"`
	Token                string    `mapstructure:"token"`
	Embedding            Embedding `mapstructure:"embedding"`
	SupportURL           string    `mapstructure:"supportUrl"`
	NewSupportRequestURL string    `mapstructure:"newSupportRequestUrl"`
}

type Support struct {
	RequestsURL   string
	NewRequestURL string
}

type Embedding struct {
	URL   string `mapstructure:"url"`
	Model string `mapstructure:"model"`
}

func Defaults() map[string]any {
	return map[string]any{
		"enabled":              false,
		"tenancy":              string(TenancyDedicated),
		"instanceId":           "",
		"host":                 defaultHost,
		"port":                 defaultPort,
		"token":                "",
		"embedding.url":        "",
		"embedding.model":      "",
		"supportUrl":           "",
		"newSupportRequestUrl": "",
	}
}

func (c *Config) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return defaultHost
	}
	return c.Host
}

func (c *Config) GetPort() int {
	if c.Port == 0 {
		return defaultPort
	}
	return c.Port
}

func (c *Config) SharedTenancy() bool {
	return c != nil && c.Enabled && c.Tenancy == TenancyShared
}

func (c *Config) BuiltInEmbedding() (Embedding, bool) {
	if c == nil || !c.Enabled || !c.Embedding.Configured() {
		return Embedding{}, false
	}
	return Embedding{
		URL:   strings.TrimRight(strings.TrimSpace(c.Embedding.URL), "/"),
		Model: strings.TrimSpace(c.Embedding.Model),
	}, true
}

func (c *Config) Support() Support {
	if c == nil || !c.Enabled {
		return Support{}
	}
	return Support{
		RequestsURL:   strings.TrimSpace(c.SupportURL),
		NewRequestURL: strings.TrimSpace(c.NewSupportRequestURL),
	}
}

func (e Embedding) Configured() bool {
	return strings.TrimSpace(e.URL) != "" && strings.TrimSpace(e.Model) != ""
}

func (c *Config) Validate() error {
	switch c.Tenancy {
	case "", TenancyDedicated:
	case TenancyShared:
		if !c.Enabled {
			return ErrSharedTenancyNotManaged
		}
	default:
		return ErrTenancyInvalid
	}
	if !c.Enabled {
		return nil
	}
	if parsed, err := uuid.Parse(c.InstanceID); err != nil || parsed.String() != c.InstanceID {
		return ErrInstanceIDInvalid
	}
	if len(strings.TrimSpace(c.Token)) < minTokenLength {
		return ErrTokenRequired
	}
	if p := c.GetPort(); p < 1 || p > 65535 {
		return ErrPortInvalid
	}
	if !validSupportURL(c.SupportURL) || !validSupportURL(c.NewSupportRequestURL) {
		return ErrSupportURLInvalid
	}
	return c.Embedding.validate()
}

func validSupportURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	if len(raw) > maxSupportURLLen || strings.ContainsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == ""
}

func (e Embedding) validate() error {
	rawURL := strings.TrimSpace(e.URL)
	model := strings.TrimSpace(e.Model)
	if rawURL == "" {
		if model != "" {
			return ErrEmbeddingModelWithoutURL
		}
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" || parsed.User != nil ||
		strings.Trim(parsed.Path, "/") != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ErrEmbeddingURLInvalid
	}
	if model == "" || len(model) > maxModelLength || strings.ContainsAny(model, " \t\r\n") {
		return ErrEmbeddingModelMissing
	}
	return nil
}

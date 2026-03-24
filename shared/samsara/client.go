package samsara

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/samsara/addresses"
	"github.com/emoss08/trenova/shared/samsara/assets"
	"github.com/emoss08/trenova/shared/samsara/compliance"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/emoss08/trenova/shared/samsara/forms"
	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/liveshares"
	"github.com/emoss08/trenova/shared/samsara/messages"
	"github.com/emoss08/trenova/shared/samsara/routes"
	"github.com/emoss08/trenova/shared/samsara/vehicles"
	"github.com/emoss08/trenova/shared/samsara/webhooks"
)

const (
	defaultBaseURL        = "https://api.samsara.com"
	defaultTimeout        = 30 * time.Second
	defaultMaxAttempts    = 4
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
)

type Client struct {
	Addresses  addresses.Service
	Assets     assets.Service
	Compliance compliance.Service
	Drivers    drivers.Service
	Forms      forms.Service
	LiveShares liveshares.Service
	Messages   messages.Service
	Routes     routes.Service
	Vehicles   vehicles.Service
	Webhooks   webhooks.Service
}

func New(apiKey string, opts ...Option) (*Client, error) {
	mergedCfg, err := applyConfigDefaults(apiKey, opts...)
	if err != nil {
		return nil, err
	}

	transport, err := httpx.New(httpx.Config{
		Token:      mergedCfg.Token,
		BaseURL:    mergedCfg.BaseURL,
		Timeout:    mergedCfg.Timeout,
		UserAgent:  mergedCfg.UserAgent,
		HTTPClient: mergedCfg.httpClient,
		Retry: httpx.RetryConfig{
			Enabled:        mergedCfg.Retry.Enabled,
			MaxAttempts:    mergedCfg.Retry.MaxAttempts,
			InitialBackoff: mergedCfg.Retry.InitialBackoff,
			MaxBackoff:     mergedCfg.Retry.MaxBackoff,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Addresses:  addresses.NewService(transport),
		Assets:     assets.NewService(transport),
		Compliance: compliance.NewService(transport),
		Drivers:    drivers.NewService(transport),
		Forms:      forms.NewService(transport),
		LiveShares: liveshares.NewService(transport),
		Messages:   messages.NewService(transport),
		Routes:     routes.NewService(transport),
		Vehicles:   vehicles.NewService(transport),
		Webhooks:   webhooks.NewService(transport),
	}, nil
}

type mergedConfig struct {
	Token      string
	BaseURL    string
	Timeout    time.Duration
	Retry      RetryConfig
	UserAgent  string
	httpClient *http.Client
}

//nolint:cyclop,funlen,gocognit,nestif,gocritic // constructor validation intentionally centralizes all defaults and guardrails.
func applyConfigDefaults(apiKey string, opts ...Option) (*mergedConfig, error) {
	cfg := &mergedConfig{
		Token:   strings.TrimSpace(apiKey),
		BaseURL: defaultBaseURL,
		Timeout: defaultTimeout,
		Retry: RetryConfig{
			Enabled:        true,
			MaxAttempts:    defaultMaxAttempts,
			InitialBackoff: defaultInitialBackoff,
			MaxBackoff:     defaultMaxBackoff,
		},
	}

	if cfg.Token == "" {
		return nil, ErrTokenRequired
	}

	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid samsara base URL: %w", err)
	}

	optSet := &options{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(optSet)
	}

	if optSet.baseURL != nil {
		baseURL := strings.TrimSpace(*optSet.baseURL)
		cfg.BaseURL = baseURL
		if cfg.BaseURL == "" {
			return nil, ErrBaseURLOverrideEmpty
		}
		if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid samsara base URL override: %w", err)
		}
	}

	if optSet.userAgent != nil {
		cfg.UserAgent = strings.TrimSpace(*optSet.userAgent)
	}

	if optSet.timeout != nil {
		if *optSet.timeout <= 0 {
			return nil, ErrTimeoutInvalid
		}
		cfg.Timeout = *optSet.timeout
	}

	if optSet.retry != nil {
		cfg.Retry = *optSet.retry
		if cfg.Retry.MaxAttempts <= 0 {
			return nil, ErrRetryMaxAttemptsInvalid
		}
		if cfg.Retry.MaxAttempts > 10 {
			return nil, ErrRetryMaxAttemptsTooHigh
		}
		if cfg.Retry.InitialBackoff <= 0 {
			return nil, ErrRetryInitialBackoffInvalid
		}
		if cfg.Retry.MaxBackoff <= 0 {
			return nil, ErrRetryMaxBackoffInvalid
		}
		if cfg.Retry.InitialBackoff > cfg.Retry.MaxBackoff {
			return nil, ErrRetryBackoffOrderInvalid
		}
	}

	if optSet.httpClient != nil && optSet.httpClient.Transport == nil {
		return nil, ErrCustomHTTPClientTransportEmpty
	}

	cfg.httpClient = optSet.httpClient
	return cfg, nil
}

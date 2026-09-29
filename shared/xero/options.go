package xero

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

const (
	defaultTimeout        = 30 * time.Second
	defaultUserAgent      = "trenova-xero/1"
	defaultMaxAttempts    = 4
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 15 * time.Second
	defaultLimiterPrefix  = "xero"
	requestsPerMinute     = 50
	concurrentRequests    = 5
)

type Option func(*options)

type options struct {
	baseURL          string
	identityURL      string
	loginURL         string
	httpClient       *http.Client
	timeout          time.Duration
	userAgent        string
	retry            restx.RetryConfig
	limiter          restx.Limiter
	limiterKeyPrefix string
	observer         restx.Observer
}

func defaultOptions() options {
	return options{
		baseURL:     apiBaseURL,
		identityURL: identityBaseURL,
		loginURL:    loginBaseURL,
		timeout:     defaultTimeout,
		userAgent:   defaultUserAgent,
		retry: restx.RetryConfig{
			Enabled:        true,
			MaxAttempts:    defaultMaxAttempts,
			InitialBackoff: defaultInitialBackoff,
			MaxBackoff:     defaultMaxBackoff,
		},
		limiterKeyPrefix: defaultLimiterPrefix,
	}
}

func resolveOptions(opts []Option) options {
	settings := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&settings)
		}
	}
	return settings
}

func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		if baseURL != "" {
			opts.baseURL = baseURL
		}
	}
}

func WithIdentityURL(identityURL string) Option {
	return func(opts *options) {
		if identityURL != "" {
			opts.identityURL = identityURL
		}
	}
}

func WithLoginURL(loginURL string) Option {
	return func(opts *options) {
		if loginURL != "" {
			opts.loginURL = loginURL
		}
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) {
		opts.httpClient = client
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *options) {
		opts.timeout = timeout
	}
}

func WithUserAgent(userAgent string) Option {
	return func(opts *options) {
		opts.userAgent = userAgent
	}
}

func WithRetry(retry restx.RetryConfig) Option {
	return func(opts *options) {
		opts.retry = retry
	}
}

func WithLimiter(limiter restx.Limiter, keyPrefix string) Option {
	return func(opts *options) {
		opts.limiter = limiter
		if keyPrefix != "" {
			opts.limiterKeyPrefix = keyPrefix
		}
	}
}

func WithObserver(observer restx.Observer) Option {
	return func(opts *options) {
		opts.observer = observer
	}
}

func TenantBucket(keyPrefix, tenantID string) restx.Bucket {
	return restx.Bucket{
		Key:    keyPrefix + ":tenant:" + tenantID,
		Limit:  requestsPerMinute,
		Period: time.Minute,
		Burst:  concurrentRequests,
		Cost:   1,
	}
}

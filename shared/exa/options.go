package exa

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

type Option func(*options)

type options struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	userAgent  string
	retry      restx.RetryConfig
	observer   restx.Observer
}

func defaultOptions() options {
	return options{
		baseURL:   DefaultBaseURL,
		timeout:   defaultTimeout,
		userAgent: defaultUserAgent,
		retry: restx.RetryConfig{
			Enabled:        true,
			MaxAttempts:    defaultMaxAttempts,
			InitialBackoff: defaultInitialBackoff,
			MaxBackoff:     defaultMaxBackoff,
		},
	}
}

func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		opts.baseURL = baseURL
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

func WithRetry(retry restx.RetryConfig) Option {
	return func(opts *options) {
		opts.retry = retry
	}
}

func WithObserver(observer restx.Observer) Option {
	return func(opts *options) {
		opts.observer = observer
	}
}

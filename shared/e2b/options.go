package e2b

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
	sandboxURL string
}

func defaultOptions() options {
	return options{
		baseURL:   DefaultAPIURL,
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

// WithSandboxURL sends every request to a sandbox to one address, naming the
// sandbox and port in headers, as E2B's own stable sandbox host does. It is
// for a self-hosted deployment or a test server.
func WithSandboxURL(sandboxURL string) Option {
	return func(opts *options) {
		opts.sandboxURL = sandboxURL
	}
}

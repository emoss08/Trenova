package netsuite

import (
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/restx"
)

const (
	defaultTimeout        = 60 * time.Second
	defaultUserAgent      = "trenova-netsuite/1"
	defaultMaxAttempts    = 4
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 15 * time.Second
	defaultLimiterPrefix  = "netsuite"
	requestsPerMinute     = 300
	concurrentRequests    = 2
)

type Option func(*options)

type options struct {
	baseURL          string
	loginURL         string
	httpClient       *http.Client
	timeout          time.Duration
	userAgent        string
	retry            restx.RetryConfig
	limiter          restx.Limiter
	limiterKeyPrefix string
	observer         restx.Observer
	now              func() time.Time
}

func defaultOptions() options {
	return options{
		loginURL:  loginBaseURL,
		timeout:   defaultTimeout,
		userAgent: defaultUserAgent,
		retry: restx.RetryConfig{
			Enabled:        true,
			MaxAttempts:    defaultMaxAttempts,
			InitialBackoff: defaultInitialBackoff,
			MaxBackoff:     defaultMaxBackoff,
		},
		limiterKeyPrefix: defaultLimiterPrefix,
		now:              time.Now,
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

func (o *options) apiBaseURL(accountID string) (string, error) {
	host, err := APIHost(accountID)
	if err != nil {
		return "", err
	}
	if o.baseURL != "" {
		return o.baseURL, nil
	}
	return "https://" + host, nil
}

func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
			opts.baseURL = trimmed
		}
	}
}

func WithLoginURL(loginURL string) Option {
	return func(opts *options) {
		if trimmed := strings.TrimRight(strings.TrimSpace(loginURL), "/"); trimmed != "" {
			opts.loginURL = trimmed
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

func WithClock(now func() time.Time) Option {
	return func(opts *options) {
		if now != nil {
			opts.now = now
		}
	}
}

func AccountBucket(keyPrefix, accountID string) restx.Bucket {
	prefix := keyPrefix
	if prefix == "" {
		prefix = defaultLimiterPrefix
	}
	account, err := NormalizeAccountID(accountID)
	if err != nil {
		account = strings.TrimSpace(accountID)
	}
	return restx.Bucket{
		Key:    prefix + ":" + strings.ToLower(account),
		Limit:  requestsPerMinute,
		Period: time.Minute,
		Burst:  concurrentRequests,
		Cost:   1,
	}
}

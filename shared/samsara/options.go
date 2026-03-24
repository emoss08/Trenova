package samsara

import (
	"net/http"
	"time"
)

type Option func(*options)

type options struct {
	httpClient *http.Client
	baseURL    *string
	userAgent  *string
	retry      *RetryConfig
	timeout    *time.Duration
}

func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) {
		opts.httpClient = client
	}
}

func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		opts.baseURL = &baseURL
	}
}

func WithUserAgent(userAgent string) Option {
	return func(opts *options) {
		opts.userAgent = &userAgent
	}
}

func WithRetry(retry RetryConfig) Option {
	return func(opts *options) {
		opts.retry = &retry
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opts *options) {
		opts.timeout = &timeout
	}
}

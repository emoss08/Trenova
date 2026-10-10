package httpx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/samsara/internal/ratelimit"
	samsaratypes "github.com/emoss08/trenova/shared/samsara/types"
	"github.com/go-resty/resty/v2"
)

const defaultRateLimitPenalty = time.Second

type RetryConfig struct {
	Enabled        bool
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

type Config struct {
	Token      string
	BaseURL    string
	Timeout    time.Duration
	UserAgent  string
	HTTPClient *http.Client
	Retry      RetryConfig
	Limiter    ratelimit.Limiter
}

type Request struct {
	Method         string
	Path           string
	Query          url.Values
	Body           any
	Out            any
	ExpectedStatus []int
}

type Requester interface {
	Do(ctx context.Context, req Request) error
}

type Client struct {
	resty *resty.Client
}

type endpointContextKey struct{}

type rateLimitWaitError struct {
	err error
}

func (e *rateLimitWaitError) Error() string {
	return "wait for samsara rate limit: " + e.err.Error()
}

func (e *rateLimitWaitError) Unwrap() error {
	return e.err
}

type endpoint struct {
	method string
	path   string
}

//nolint:gocritic // constructor config is passed by value as immutable input.
func New(
	cfg Config,
) (*Client, error) {
	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	var rc *resty.Client
	if cfg.HTTPClient != nil {
		rc = resty.NewWithClient(cfg.HTTPClient)
	} else {
		rc = resty.New()
	}
	rc.SetBaseURL(baseURL.String())
	rc.SetHeader("Authorization", "Bearer "+cfg.Token)
	rc.SetHeader("Accept", "application/json")
	rc.SetTimeout(cfg.Timeout)

	if cfg.UserAgent != "" {
		rc.SetHeader("User-Agent", cfg.UserAgent)
	}

	configureRetries(rc, cfg.Retry)
	configureRateLimit(rc, cfg.Limiter)

	return &Client{resty: rc}, nil
}

//nolint:gocritic // request is passed by value to keep per-call data isolated.
func (c *Client) Do(
	ctx context.Context,
	req Request,
) error {
	request := c.resty.R().SetContext(
		context.WithValue(ctx, endpointContextKey{}, endpoint{method: req.Method, path: req.Path}),
	)
	if req.Query != nil {
		request.SetQueryParamsFromValues(req.Query)
	}

	if req.Body != nil {
		encoded, err := sonic.Marshal(req.Body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		request.SetHeader("Content-Type", "application/json")
		request.SetBody(encoded)
	}

	resp, err := request.Execute(req.Method, req.Path)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("execute request: %w", err)
	}

	expected := req.ExpectedStatus
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}

	if !containsStatus(expected, resp.StatusCode()) {
		return parseAPIError(resp.StatusCode(), resp.Body())
	}

	if req.Out == nil || len(resp.Body()) == 0 {
		return nil
	}

	if err = sonic.Unmarshal(resp.Body(), req.Out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func configureRetries(client *resty.Client, cfg RetryConfig) {
	if !cfg.Enabled {
		client.SetRetryCount(0)
		return
	}

	retries := cfg.MaxAttempts - 1
	if retries < 0 {
		retries = 0
	}
	client.SetRetryCount(retries)

	if cfg.InitialBackoff > 0 {
		client.SetRetryWaitTime(cfg.InitialBackoff)
	}
	if cfg.MaxBackoff > 0 {
		client.SetRetryMaxWaitTime(cfg.MaxBackoff)
	}

	client.AddRetryCondition(func(resp *resty.Response, err error) bool {
		if err != nil {
			var waitErr *rateLimitWaitError
			return !errors.As(err, &waitErr)
		}
		if resp == nil {
			return false
		}
		statusCode := resp.StatusCode()
		return statusCode == http.StatusTooManyRequests ||
			statusCode >= http.StatusInternalServerError
	})

	client.SetRetryAfter(func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
		if resp == nil {
			return 0, nil
		}

		d, ok := restx.ParseRetryAfter(resp.Header().Get("Retry-After"))
		if !ok {
			return 0, nil
		}
		return d, nil
	})
}

func configureRateLimit(client *resty.Client, limiter ratelimit.Limiter) {
	if limiter == nil {
		return
	}

	client.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		ctx := req.Context()
		target, ok := ctx.Value(endpointContextKey{}).(endpoint)
		if !ok {
			return nil
		}
		if err := limiter.Wait(ctx, target.method, target.path); err != nil {
			return &rateLimitWaitError{err: err}
		}
		return nil
	})

	client.OnAfterResponse(func(_ *resty.Client, resp *resty.Response) error {
		if resp == nil || resp.Request == nil ||
			resp.StatusCode() != http.StatusTooManyRequests {
			return nil
		}
		target, ok := resp.Request.Context().Value(endpointContextKey{}).(endpoint)
		if !ok {
			return nil
		}
		penalty, parsed := restx.ParseRetryAfter(resp.Header().Get("Retry-After"))
		if !parsed {
			penalty = defaultRateLimitPenalty
		}
		limiter.Penalize(target.method, target.path, penalty)
		return nil
	})
}

func containsStatus(expected []int, status int) bool {
	for _, code := range expected {
		if code == status {
			return true
		}
	}
	return false
}

func parseAPIError(statusCode int, body []byte) *samsaratypes.APIError {
	type errorBody struct {
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	}

	parsed := errorBody{}
	if len(body) > 0 {
		_ = sonic.Unmarshal(body, &parsed)
	}

	message := parsed.Message
	if message == "" {
		message = http.StatusText(statusCode)
	}

	return &samsaratypes.APIError{
		StatusCode: statusCode,
		Message:    message,
		RequestID:  parsed.RequestID,
		RawBody:    body,
	}
}

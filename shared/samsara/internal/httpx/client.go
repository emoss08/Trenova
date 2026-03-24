package httpx

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	samsaratypes "github.com/emoss08/trenova/shared/samsara/types"
	"github.com/go-resty/resty/v2"
)

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

	return &Client{resty: rc}, nil
}

//nolint:gocritic // request is passed by value to keep per-call data isolated.
func (c *Client) Do(
	ctx context.Context,
	req Request,
) error {
	request := c.resty.R().SetContext(ctx)
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
			return true
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

		d, ok := parseRetryAfter(resp.Header().Get("Retry-After"))
		if !ok {
			return 0, nil
		}
		return d, nil
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

func parseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	seconds, err := strconv.Atoi(value)
	if err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}

	t, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	until := time.Until(t)
	if until < 0 {
		return 0, true
	}
	return until, true
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

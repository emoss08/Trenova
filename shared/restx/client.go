package restx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	defaultTimeout          = 30 * time.Second
	defaultMaxResponseBytes = 32 << 20
	contentTypeJSON         = "application/json"
)

var (
	ErrNilRequest        = errors.New("restx: request is required")
	ErrInvalidBaseURL    = errors.New("restx: base URL must be an absolute http or https URL")
	ErrInvalidPath       = errors.New("restx: request path must not contain a query or fragment")
	ErrLimiterNoBucket   = errors.New("restx: a limiter requires a BucketFor function")
	ErrResponseTooLarge  = errors.New("restx: response body exceeds the configured limit")
	ErrInvalidHeaderName = errors.New("restx: header names must not be blank")
)

type Bucket struct {
	Key    string
	Limit  int
	Period time.Duration
	Burst  int
	Cost   int
}

type Limiter interface {
	Acquire(ctx context.Context, bucket Bucket) error
}

type CallInfo struct {
	Endpoint      string
	Method        string
	Path          string
	StatusCode    int
	Attempt       int
	Latency       time.Duration
	Err           error
	RetryAfter    time.Duration
	ResponseBytes int
}

type Observer func(CallInfo)

type ErrorDecoder func(status int, body []byte, header http.Header) error

type Config struct {
	BaseURL          string
	Timeout          time.Duration
	UserAgent        string
	HTTPClient       *http.Client
	Headers          map[string]string
	QueryParams      map[string]string
	Retry            RetryConfig
	Limiter          Limiter
	BucketFor        func(endpoint string) (Bucket, bool)
	Observer         Observer
	ErrorDecoder     ErrorDecoder
	RedactQueryKeys  []string
	MaxResponseBytes int64
}

type Request struct {
	Endpoint       string
	Method         string
	Path           string
	Query          url.Values
	Body           any
	Out            any
	ExpectedStatus []int
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Attempts   int
}

type Client struct {
	baseURL          *url.URL
	httpClient       *http.Client
	headers          http.Header
	queryParams      map[string]string
	retry            RetryConfig
	limiter          Limiter
	bucketFor        func(endpoint string) (Bucket, bool)
	observer         Observer
	errorDecoder     ErrorDecoder
	redactQueryKeys  []string
	redactor         redactor
	maxResponseBytes int64
}

func New(cfg Config) (*Client, error) {
	baseURL, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Limiter != nil && cfg.BucketFor == nil {
		return nil, ErrLimiterNoBucket
	}

	headers := make(http.Header, len(cfg.Headers)+2)
	headers.Set("Accept", contentTypeJSON)
	if userAgent := strings.TrimSpace(cfg.UserAgent); userAgent != "" {
		headers.Set("User-Agent", userAgent)
	}

	secrets := make([]string, 0, len(cfg.Headers)+len(cfg.QueryParams))
	for name, value := range cfg.Headers {
		if strings.TrimSpace(name) == "" {
			return nil, ErrInvalidHeaderName
		}
		headers.Set(name, value)
		if !isPublicHeader(name) {
			secrets = append(secrets, value)
		}
	}

	queryParams := make(map[string]string, len(cfg.QueryParams))
	redactKeys := make([]string, 0, len(cfg.RedactQueryKeys)+len(cfg.QueryParams))
	redactKeys = append(redactKeys, cfg.RedactQueryKeys...)
	for name, value := range cfg.QueryParams {
		queryParams[name] = value
		secrets = append(secrets, value)
		redactKeys = append(redactKeys, name)
	}

	maxResponseBytes := cfg.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseBytes
	}

	return &Client{
		baseURL:          baseURL,
		httpClient:       buildHTTPClient(cfg.HTTPClient, cfg.Timeout),
		headers:          headers,
		queryParams:      queryParams,
		retry:            cfg.Retry.normalized(),
		limiter:          cfg.Limiter,
		bucketFor:        cfg.BucketFor,
		observer:         cfg.Observer,
		errorDecoder:     cfg.ErrorDecoder,
		redactQueryKeys:  redactKeys,
		redactor:         newRedactor(secrets),
		maxResponseBytes: maxResponseBytes,
	}, nil
}

func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	target, err := c.resolve(req.Path, req.Query)
	if err != nil {
		return nil, err
	}

	var payload []byte
	if req.Body != nil {
		payload, err = sonic.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("encode %s %s request body: %w", method, req.Path, err)
		}
	}

	bucket, limited := c.resolveBucket(req.Endpoint)
	expected := req.ExpectedStatus
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}

	for attempt := 1; ; attempt++ {
		info := CallInfo{
			Endpoint: req.Endpoint,
			Method:   method,
			Path:     req.Path,
			Attempt:  attempt,
		}

		if limited {
			if err = c.limiter.Acquire(ctx, bucket); err != nil {
				info.Err = err
				info.RetryAfter = RetryAfterOf(err)
				c.observe(info)
				return nil, err
			}
		}

		started := time.Now()
		resp, sendErr := c.send(ctx, method, target, payload)
		info.Latency = time.Since(started)

		if sendErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				info.Err = ctxErr
				c.observe(info)
				return nil, ctxErr
			}
			transportErr := &TransportError{
				Method: method,
				Path:   req.Path,
				Err:    sanitizeTransportError(sendErr),
				redact: c.redactor.apply,
			}
			info.Err = transportErr
			c.observe(info)
			if errors.Is(sendErr, ErrResponseTooLarge) || attempt >= c.retry.MaxAttempts {
				return nil, transportErr
			}
			if err = sleep(ctx, c.retry.backoff(attempt)); err != nil {
				return nil, err
			}
			continue
		}

		resp.Attempts = attempt
		info.StatusCode = resp.StatusCode
		info.ResponseBytes = len(resp.Body)

		if slices.Contains(expected, resp.StatusCode) {
			if req.Out != nil && len(bytes.TrimSpace(resp.Body)) > 0 {
				if err = sonic.Unmarshal(resp.Body, req.Out); err != nil {
					decodeErr := fmt.Errorf("decode %s %s response: %w", method, req.Path, err)
					info.Err = decodeErr
					c.observe(info)
					return resp, decodeErr
				}
			}
			c.observe(info)
			return resp, nil
		}

		retryAfter, hasRetryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
		apiErr := c.decodeError(resp)
		info.Err = apiErr
		info.RetryAfter = retryAfter
		c.observe(info)

		if !retryableStatus(resp.StatusCode) || attempt >= c.retry.MaxAttempts {
			return resp, apiErr
		}
		if err = sleep(ctx, c.retry.delay(attempt, retryAfter, hasRetryAfter)); err != nil {
			return nil, err
		}
	}
}

func (c *Client) RedactURL(u *url.URL) string {
	return RedactURL(u, c.redactQueryKeys)
}

func (c *Client) Redact(value string) string {
	return c.redactor.apply(value)
}

func (c *Client) resolveBucket(endpoint string) (Bucket, bool) {
	if c.limiter == nil || c.bucketFor == nil {
		return Bucket{}, false
	}
	bucket, ok := c.bucketFor(endpoint)
	if !ok {
		return Bucket{}, false
	}
	if bucket.Cost <= 0 {
		bucket.Cost = 1
	}
	return bucket, true
}

func (c *Client) resolve(path string, query url.Values) (*url.URL, error) {
	if strings.ContainsAny(path, "?#") {
		return nil, ErrInvalidPath
	}

	escaped := strings.TrimRight(c.baseURL.EscapedPath(), "/")
	if trimmed := strings.TrimLeft(path, "/"); trimmed != "" {
		escaped += "/" + trimmed
	}
	if escaped == "" {
		escaped = "/"
	}

	decoded, err := url.PathUnescape(escaped)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid escape sequence", ErrInvalidPath)
	}

	target := *c.baseURL
	target.Path = decoded
	target.RawPath = escaped

	values := make(url.Values, len(query)+len(c.queryParams))
	for name, items := range query {
		values[name] = slices.Clone(items)
	}
	for name, value := range c.queryParams {
		values.Set(name, value)
	}
	target.RawQuery = values.Encode()

	return &target, nil
}

func (c *Client) send(
	ctx context.Context,
	method string,
	target *url.URL,
	payload []byte,
) (*Response, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header = c.headers.Clone()
	if payload != nil {
		httpReq.Header.Set("Content-Type", contentTypeJSON)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(httpResp.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if int64(len(data)) > c.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Header:     httpResp.Header,
		Body:       data,
	}, nil
}

func (c *Client) decodeError(resp *Response) error {
	var err error
	if c.errorDecoder != nil {
		err = c.errorDecoder(resp.StatusCode, resp.Body, resp.Header)
	}
	if err == nil {
		err = DecodeAPIError(resp.StatusCode, resp.Body, resp.Header)
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr != nil {
		if apiErr.StatusCode == 0 {
			apiErr.StatusCode = resp.StatusCode
		}
		if apiErr.RetryAfter == 0 {
			if retryAfter, ok := ParseRetryAfter(resp.Header.Get("Retry-After")); ok {
				apiErr.RetryAfter = retryAfter
			}
		}
		apiErr.Code = c.redactor.apply(apiErr.Code)
		apiErr.Message = c.redactor.apply(apiErr.Message)
		apiErr.Hint = c.redactor.apply(apiErr.Hint)
		apiErr.RequestID = c.redactor.apply(apiErr.RequestID)
		apiErr.Body = c.redactor.bytes(apiErr.Body)
	}
	return err
}

func (c *Client) observe(info CallInfo) {
	if c.observer != nil {
		c.observer(info)
	}
}

func parseBaseURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrInvalidBaseURL
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBaseURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, ErrInvalidBaseURL
	}
	if parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return nil, ErrInvalidBaseURL
	}
	return parsed, nil
}

func buildHTTPClient(base *http.Client, timeout time.Duration) *http.Client {
	if base == nil {
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		return &http.Client{Timeout: timeout}
	}
	clone := *base
	switch {
	case timeout > 0:
		clone.Timeout = timeout
	case clone.Timeout <= 0:
		clone.Timeout = defaultTimeout
	}
	return &clone
}

func isPublicHeader(name string) bool {
	switch http.CanonicalHeaderKey(strings.TrimSpace(name)) {
	case "Accept", "Content-Type", "User-Agent", "Accept-Language", "Accept-Encoding":
		return true
	default:
		return false
	}
}

func sanitizeTransportError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}

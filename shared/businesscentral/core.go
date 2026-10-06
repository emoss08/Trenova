package businesscentral

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/restx"
)

const (
	maxPages            = 200
	acceptLanguage      = "en-US"
	authorizationHeader = "Authorization"
	languageHeader      = "Accept-Language"
	wildcardETag        = "*"
	maxETagLength       = 512
	suffixCreate        = "-create"
	suffixGet           = "-get"
	suffixSearch        = "-search"
	suffixDelete        = "-delete"
	suffixUpdate        = "-update"
)

type core struct {
	reads    *restx.Client
	writes   *restx.Client
	base     *url.URL
	pageSize int
}

type call struct {
	endpoint string
	method   string
	path     string
	query    url.Values
	ifMatch  string
	body     any
	out      any
	expected []int
}

type collection[T any] struct {
	Value    []T    `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

type listCall struct {
	endpoint string
	path     string
	filter   string
	unpaged  bool
}

type pageCursor struct {
	path   string
	query  url.Values
	skip   int
	linked bool
}

func newCore(settings *options, token string, bucket *restx.Bucket) (*core, error) {
	cfg := restx.Config{
		BaseURL:    settings.baseURL,
		Timeout:    settings.timeout,
		UserAgent:  settings.userAgent,
		HTTPClient: headerClient(settings.httpClient),
		Headers: map[string]string{
			authorizationHeader: "Bearer " + token,
			languageHeader:      acceptLanguage,
		},
		Retry:        settings.retry,
		Observer:     settings.observer,
		ErrorDecoder: decodeAPIError,
	}
	if settings.limiter != nil && bucket != nil {
		limited := *bucket
		cfg.Limiter = settings.limiter
		cfg.BucketFor = func(string) (restx.Bucket, bool) { return limited, true }
	}

	reads, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("businesscentral: configure transport: %w", err)
	}
	cfg.Retry = restx.RetryConfig{}
	writes, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("businesscentral: configure write transport: %w", err)
	}
	base, err := url.Parse(strings.TrimSpace(settings.baseURL))
	if err != nil {
		return nil, fmt.Errorf("businesscentral: parse base url: %w", err)
	}
	return &core{reads: reads, writes: writes, base: base, pageSize: settings.pageSize}, nil
}

func (c *core) do(ctx context.Context, req *call) (*restx.Response, error) {
	transport := c.reads
	if req.method != http.MethodGet {
		transport = c.writes
	}
	expected := req.expected
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}
	resp, err := transport.Do(withRequestHeaders(ctx, requestHeaders{ifMatch: req.ifMatch}),
		&restx.Request{
			Endpoint:       req.endpoint,
			Method:         req.method,
			Path:           req.path,
			Query:          req.query,
			Body:           req.body,
			Out:            req.out,
			ExpectedStatus: expected,
		})
	if err != nil {
		redactAPIError(err, transport.Redact)
	}
	return resp, err
}

func fetchOne[W, T any](ctx context.Context, c *core, req *call, convert func(*W) T) (*T, error) {
	var wire W
	req.out = &wire
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(bytes.TrimSpace(resp.Body)) == 0 {
		return nil, ErrUnexpectedPayload
	}
	out := convert(&wire)
	return &out, nil
}

func fetchFirst[W, T any](
	ctx context.Context,
	c *core,
	req *listCall,
	convert func(*W) T,
) (*T, error) {
	var wire collection[W]
	if _, err := c.do(ctx, &call{
		endpoint: req.endpoint,
		method:   http.MethodGet,
		path:     req.path,
		out:      &wire,
	}); err != nil {
		return nil, err
	}
	if len(wire.Value) == 0 {
		return nil, ErrUnexpectedPayload
	}
	out := convert(&wire.Value[0])
	return &out, nil
}

func collect[W, T any](
	ctx context.Context,
	c *core,
	req *listCall,
	convert func(*W) T,
) ([]T, error) {
	query := url.Values{}
	if !req.unpaged {
		query.Set(topParam, strconv.Itoa(c.pageSize))
	}
	if req.filter != "" {
		query.Set(filterParam, req.filter)
	}
	cursor := pageCursor{path: req.path, query: query, linked: req.unpaged}
	out := make([]T, 0)
	for range maxPages {
		var wire collection[W]
		_, err := c.do(ctx, &call{
			endpoint: req.endpoint,
			method:   http.MethodGet,
			path:     cursor.path,
			query:    cursor.query,
			out:      &wire,
		})
		if err != nil {
			return nil, err
		}
		for idx := range wire.Value {
			out = append(out, convert(&wire.Value[idx]))
		}
		more, err := c.advance(&cursor, wire.NextLink, len(wire.Value))
		if err != nil {
			return nil, err
		}
		if !more {
			return out, nil
		}
	}
	return nil, ErrTooManyResults
}

func (c *core) advance(cursor *pageCursor, nextLink string, received int) (bool, error) {
	if nextLink != "" {
		path, query, err := c.resolveNextLink(nextLink)
		if err != nil {
			return false, err
		}
		cursor.path, cursor.query, cursor.linked = path, query, true
		return true, nil
	}
	if cursor.linked || received < c.pageSize {
		return false, nil
	}
	cursor.skip += received
	cursor.query.Set(skipParam, strconv.Itoa(cursor.skip))
	return true, nil
}

func (c *core) resolveNextLink(raw string) (string, url.Values, error) {
	link, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || link.User != nil || link.Fragment != "" ||
		!strings.EqualFold(link.Scheme, c.base.Scheme) ||
		!strings.EqualFold(link.Host, c.base.Host) {
		return "", nil, ErrForeignNextLink
	}
	prefix := strings.TrimRight(c.base.EscapedPath(), "/")
	escaped := link.EscapedPath()
	if !strings.HasPrefix(escaped, prefix+"/") || strings.Contains(escaped+"/", "/../") ||
		strings.Contains(escaped+"/", "/./") {
		return "", nil, ErrForeignNextLink
	}
	query, err := url.ParseQuery(link.RawQuery)
	if err != nil {
		return "", nil, ErrForeignNextLink
	}
	return strings.TrimPrefix(escaped, prefix), query, nil
}

func ifMatch(etag string) (string, error) {
	value := strings.TrimSpace(etag)
	if value == "" {
		return wildcardETag, nil
	}
	if len(value) > maxETagLength {
		return "", ErrInvalidETag
	}
	for idx := range len(value) {
		if value[idx] < ' ' || value[idx] > '~' {
			return "", ErrInvalidETag
		}
	}
	return value, nil
}

func environmentPath(tenantID, environment string) string {
	return "/" + apiVersionRoot + "/" + tenantID + "/" + url.PathEscape(environment) +
		"/" + apiSegment
}

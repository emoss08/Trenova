package businesscentral

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	maxPages            = 200
	acceptLanguage      = "en-US"
	authorizationHeader = "Authorization"
	languageHeader      = "Accept-Language"
	wildcardETag        = "*"
	maxETagLength       = 512
	SuffixCreate        = "-create"
	SuffixGet           = "-get"
	SuffixSearch        = "-search"
	SuffixDelete        = "-delete"
	SuffixUpdate        = "-update"
)

type Core struct {
	reads    *restx.Client
	writes   *restx.Client
	base     *url.URL
	pageSize int
}

type Call struct {
	Endpoint string
	Method   string
	Path     string
	Query    url.Values
	IfMatch  string
	Body     any
	Out      any
	Expected []int
}

type collection[T any] struct {
	Value    []T    `json:"value"`
	NextLink string `json:"@odata.nextLink"`
}

type ListCall struct {
	Endpoint string
	Path     string
	Filter   string
	Unpaged  bool
}

type pageCursor struct {
	path   string
	query  url.Values
	skip   int
	linked bool
}

func newCore(settings *options, token string, bucket *restx.Bucket) (*Core, error) {
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
		ErrorDecoder: businesscentralerrors.DecodeAPIError,
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
	return &Core{reads: reads, writes: writes, base: base, pageSize: settings.pageSize}, nil
}

func (c *Core) do(ctx context.Context, req *Call) (*restx.Response, error) {
	transport := c.reads
	if req.Method != http.MethodGet {
		transport = c.writes
	}
	expected := req.Expected
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}
	resp, err := transport.Do(withRequestHeaders(ctx, requestHeaders{ifMatch: req.IfMatch}),
		&restx.Request{
			Endpoint:       req.Endpoint,
			Method:         req.Method,
			Path:           req.Path,
			Query:          req.Query,
			Body:           req.Body,
			Out:            req.Out,
			ExpectedStatus: expected,
		})
	if err != nil {
		businesscentralerrors.RedactAPIError(err, transport.Redact)
	}
	return resp, err
}

func FetchOne[W, T any](ctx context.Context, c *Core, req *Call, convert func(*W) T) (*T, error) {
	var wire W
	req.Out = &wire
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(bytes.TrimSpace(resp.Body)) == 0 {
		return nil, businesscentralerrors.ErrUnexpectedPayload
	}
	out := convert(&wire)
	return &out, nil
}

func FetchFirst[W, T any](
	ctx context.Context,
	c *Core,
	req *ListCall,
	convert func(*W) T,
) (*T, error) {
	var wire collection[W]
	if _, err := c.do(ctx, &Call{
		Endpoint: req.Endpoint,
		Method:   http.MethodGet,
		Path:     req.Path,
		Out:      &wire,
	}); err != nil {
		return nil, err
	}
	if len(wire.Value) == 0 {
		return nil, businesscentralerrors.ErrUnexpectedPayload
	}
	out := convert(&wire.Value[0])
	return &out, nil
}

func Collect[W, T any](
	ctx context.Context,
	c *Core,
	req *ListCall,
	convert func(*W) T,
) ([]T, error) {
	query := url.Values{}
	if !req.Unpaged {
		query.Set(topParam, strconv.Itoa(c.pageSize))
	}
	if req.Filter != "" {
		query.Set(filterParam, req.Filter)
	}
	cursor := pageCursor{path: req.Path, query: query, linked: req.Unpaged}
	out := make([]T, 0)
	for range maxPages {
		var wire collection[W]
		_, err := c.do(ctx, &Call{
			Endpoint: req.Endpoint,
			Method:   http.MethodGet,
			Path:     cursor.path,
			Query:    cursor.query,
			Out:      &wire,
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
	return nil, businesscentralerrors.ErrTooManyResults
}

func (c *Core) advance(cursor *pageCursor, nextLink string, received int) (bool, error) {
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

func (c *Core) resolveNextLink(raw string) (string, url.Values, error) {
	link, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || link.User != nil || link.Fragment != "" ||
		!strings.EqualFold(link.Scheme, c.base.Scheme) ||
		!strings.EqualFold(link.Host, c.base.Host) {
		return "", nil, businesscentralerrors.ErrForeignNextLink
	}
	prefix := strings.TrimRight(c.base.EscapedPath(), "/")
	escaped := link.EscapedPath()
	if !strings.HasPrefix(escaped, prefix+"/") || strings.Contains(escaped+"/", "/../") ||
		strings.Contains(escaped+"/", "/./") {
		return "", nil, businesscentralerrors.ErrForeignNextLink
	}
	query, err := url.ParseQuery(link.RawQuery)
	if err != nil {
		return "", nil, businesscentralerrors.ErrForeignNextLink
	}
	return strings.TrimPrefix(escaped, prefix), query, nil
}

func IfMatch(etag string) (string, error) {
	value := strings.TrimSpace(etag)
	if value == "" {
		return wildcardETag, nil
	}
	if len(value) > maxETagLength {
		return "", businesscentralerrors.ErrInvalidETag
	}
	for idx := range len(value) {
		if value[idx] < ' ' || value[idx] > '~' {
			return "", businesscentralerrors.ErrInvalidETag
		}
	}
	return value, nil
}

func environmentPath(tenantID, environment string) string {
	return "/" + apiVersionRoot + "/" + tenantID + "/" + url.PathEscape(environment) +
		"/" + APISegment
}

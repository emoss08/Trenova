package netsuite

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	authorizationHeader = "Authorization"
	locationHeader      = "Location"
	preferTransient     = "transient"
)

type core struct {
	reads  *restx.Client
	writes *restx.Client
	base   *url.URL
}

type call struct {
	endpoint   string
	method     string
	path       string
	query      url.Values
	body       any
	out        any
	expected   []int
	prefer     string
	idempotent bool
}

type coreParams struct {
	settings *options
	baseURL  string
	token    string
	bucket   restx.Bucket
}

func newCore(params *coreParams) (*core, error) {
	settings := params.settings
	cfg := restx.Config{
		BaseURL:    params.baseURL,
		Timeout:    settings.timeout,
		UserAgent:  settings.userAgent,
		HTTPClient: headerClient(settings.httpClient),
		Headers: map[string]string{
			authorizationHeader: "Bearer " + params.token,
		},
		Retry:        settings.retry,
		Observer:     settings.observer,
		ErrorDecoder: netsuiteerrors.DecodeAPIError,
	}
	if settings.limiter != nil {
		bucket := params.bucket
		cfg.Limiter = settings.limiter
		cfg.BucketFor = func(string) (restx.Bucket, bool) { return bucket, true }
	}

	reads, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("netsuite: configure transport: %w", err)
	}
	cfg.Retry = restx.RetryConfig{}
	writes, err := restx.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("netsuite: configure write transport: %w", err)
	}
	base, err := url.Parse(params.baseURL)
	if err != nil {
		return nil, fmt.Errorf("netsuite: parse base url: %w", err)
	}
	return &core{reads: reads, writes: writes, base: base}, nil
}

func (c *core) do(ctx context.Context, req *call) (*restx.Response, error) {
	transport := c.writes
	if req.method == http.MethodGet || req.idempotent {
		transport = c.reads
	}
	expected := req.expected
	if len(expected) == 0 {
		expected = []int{http.StatusOK}
	}
	resp, err := transport.Do(withRequestHeaders(ctx, requestHeaders{prefer: req.prefer}),
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
		netsuiteerrors.RedactAPIError(err, transport.Redact)
	}
	return resp, err
}

func (c *core) locationID(resp *restx.Response, recordType string) (string, error) {
	if resp == nil {
		return "", netsuiteerrors.ErrUnexpectedPayload
	}
	raw := strings.TrimSpace(resp.Header.Get(locationHeader))
	if raw == "" {
		return "", netsuiteerrors.ErrUnexpectedPayload
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return "", netsuiteerrors.ErrUnexpectedPayload
	}
	link := c.base.ResolveReference(ref)
	if link.User != nil || !strings.EqualFold(link.Scheme, c.base.Scheme) ||
		!strings.EqualFold(link.Host, c.base.Host) {
		return "", netsuiteerrors.ErrForeignLocation
	}
	prefix := strings.TrimRight(c.base.EscapedPath(), "/") + recordPath + "/"
	rest, ok := strings.CutPrefix(link.EscapedPath(), prefix)
	if !ok {
		return "", netsuiteerrors.ErrForeignLocation
	}
	segment, id, found := strings.Cut(rest, "/")
	if !found || !strings.EqualFold(segment, recordType) || !isInternalID(id) {
		return "", netsuiteerrors.ErrUnexpectedPayload
	}
	return id, nil
}

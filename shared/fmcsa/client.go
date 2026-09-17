package fmcsa

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	defaultBaseURL        = "https://mobile.fmcsa.dot.gov/qc/services"
	defaultTimeout        = 30 * time.Second
	defaultUserAgent      = "trenova-fmcsa/1"
	defaultMaxAttempts    = 3
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
	defaultLimiterPrefix  = "fmcsa"
	webKeyParam           = "webKey"
	bucketSuffix          = "qcmobile"
	defaultLimitPerMinute = 120
	defaultBurst          = 4
	compositeConcurrency  = 3
)

type Endpoint string

const (
	EndpointCarrier                 Endpoint = "carrier"
	EndpointCarriersByDocket        Endpoint = "carriers-by-docket"
	EndpointCarriersByName          Endpoint = "carriers-by-name"
	EndpointBasics                  Endpoint = "basics"
	EndpointCargoCarried            Endpoint = "cargo-carried"
	EndpointOperationClassification Endpoint = "operation-classification"
	EndpointOOS                     Endpoint = "oos"
	EndpointDocketNumbers           Endpoint = "docket-numbers"
	EndpointAuthority               Endpoint = "authority"
)

type Client struct {
	transport *restx.Client
}

func DefaultPolicy() restx.Bucket {
	return restx.Bucket{
		Limit:  defaultLimitPerMinute,
		Period: time.Minute,
		Burst:  defaultBurst,
		Cost:   1,
	}
}

func New(webKey string, opts ...Option) (*Client, error) {
	key := strings.TrimSpace(webKey)
	if key == "" {
		return nil, ErrWebKeyRequired
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	restCfg := restx.Config{
		BaseURL:         cfg.baseURL,
		Timeout:         cfg.timeout,
		UserAgent:       cfg.userAgent,
		HTTPClient:      cfg.httpClient,
		QueryParams:     map[string]string{webKeyParam: key},
		RedactQueryKeys: []string{webKeyParam},
		Retry:           cfg.retry,
		Observer:        cfg.observer,
		ErrorDecoder:    decodeError,
	}
	if cfg.limiter != nil {
		prefix := strings.TrimSpace(cfg.limiterKeyPrefix)
		if prefix == "" {
			prefix = defaultLimiterPrefix
		}
		bucket := DefaultPolicy()
		bucket.Key = prefix + ":" + bucketSuffix
		restCfg.Limiter = cfg.limiter
		restCfg.BucketFor = func(string) (restx.Bucket, bool) {
			return bucket, true
		}
	}

	transport, err := restx.New(restCfg)
	if err != nil {
		return nil, fmt.Errorf("fmcsa: configure transport: %w", err)
	}
	return &Client{transport: transport}, nil
}

func (c *Client) CarrierByDOT(ctx context.Context, dot string) (*Carrier, error) {
	normalized, err := normalizeDOT(dot)
	if err != nil {
		return nil, err
	}
	content, err := c.fetchContent(ctx, EndpointCarrier, carrierPath(normalized), nil)
	if err != nil {
		return nil, err
	}
	return decodeCarrierContent(content)
}

func (c *Client) CarriersByDocket(ctx context.Context, docket string) ([]Carrier, error) {
	digits := stringutils.DigitsOnly(docket)
	if digits == "" {
		return nil, fmt.Errorf("%w: docket number must contain digits", ErrInvalidArgument)
	}
	content, err := c.fetchContent(
		ctx,
		EndpointCarriersByDocket,
		"/carriers/docket-number/"+url.PathEscape(digits),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return decodeCarrierList(EndpointCarriersByDocket, content)
}

func (c *Client) CarriersByName(
	ctx context.Context,
	name string,
	start, size int,
) ([]Carrier, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return nil, fmt.Errorf("%w: carrier name is required", ErrInvalidArgument)
	}
	if start < 0 || size < 0 {
		return nil, fmt.Errorf("%w: start and size must not be negative", ErrInvalidArgument)
	}

	query := make(url.Values, 2)
	if start > 0 {
		query.Set("start", strconv.Itoa(start))
	}
	if size > 0 {
		query.Set("size", strconv.Itoa(size))
	}

	content, err := c.fetchContent(
		ctx,
		EndpointCarriersByName,
		"/carriers/name/"+url.PathEscape(trimmed),
		query,
	)
	if err != nil {
		return nil, err
	}
	return decodeCarrierList(EndpointCarriersByName, content)
}

func (c *Client) Basics(ctx context.Context, dot string) ([]Basic, error) {
	content, err := c.fetchSubresource(ctx, EndpointBasics, dot)
	if err != nil {
		return nil, err
	}
	return decodeList(EndpointBasics, content, "basic", decodeBasic)
}

func (c *Client) CargoCarried(ctx context.Context, dot string) ([]string, error) {
	content, err := c.fetchSubresource(ctx, EndpointCargoCarried, dot)
	if err != nil {
		return nil, err
	}
	return decodeDescriptions(EndpointCargoCarried, content, "cargoClassDesc")
}

func (c *Client) OperationClassification(ctx context.Context, dot string) ([]string, error) {
	content, err := c.fetchSubresource(ctx, EndpointOperationClassification, dot)
	if err != nil {
		return nil, err
	}
	return decodeDescriptions(EndpointOperationClassification, content, "operationClassDesc")
}

func (c *Client) OOS(ctx context.Context, dot string) ([]OOSEntry, error) {
	content, err := c.fetchSubresource(ctx, EndpointOOS, dot)
	if err != nil {
		return nil, err
	}
	return decodeList(EndpointOOS, content, "oos", decodeOOSEntry)
}

func (c *Client) DocketNumbers(ctx context.Context, dot string) ([]Docket, error) {
	content, err := c.fetchSubresource(ctx, EndpointDocketNumbers, dot)
	if err != nil {
		return nil, err
	}
	return decodeList(EndpointDocketNumbers, content, "docketNumber", decodeDocket)
}

func (c *Client) Authority(ctx context.Context, dot string) ([]Authority, error) {
	content, err := c.fetchSubresource(ctx, EndpointAuthority, dot)
	if err != nil {
		return nil, err
	}
	return decodeList(EndpointAuthority, content, "carrierAuthority", decodeAuthority)
}

type compositeDocument struct {
	Carrier                 sonic.NoCopyRawMessage `json:"carrier"`
	Basics                  sonic.NoCopyRawMessage `json:"basics"`
	CargoCarried            sonic.NoCopyRawMessage `json:"cargoCarried"`
	OperationClassification sonic.NoCopyRawMessage `json:"operationClassification"`
	OOS                     sonic.NoCopyRawMessage `json:"oos"`
	DocketNumbers           sonic.NoCopyRawMessage `json:"docketNumbers"`
	Authority               sonic.NoCopyRawMessage `json:"authority"`
}

func (c *Client) Composite(ctx context.Context, dot string) (*CompositeCarrier, error) {
	normalized, err := normalizeDOT(dot)
	if err != nil {
		return nil, err
	}

	carrierContent, err := c.fetchContent(ctx, EndpointCarrier, carrierPath(normalized), nil)
	if err != nil {
		return nil, err
	}
	carrier, err := decodeCarrierContent(carrierContent)
	if err != nil {
		return nil, err
	}

	endpoints := [...]Endpoint{
		EndpointBasics,
		EndpointCargoCarried,
		EndpointOperationClassification,
		EndpointOOS,
		EndpointDocketNumbers,
		EndpointAuthority,
	}
	contents, err := c.fetchConcurrently(ctx, normalized, endpoints[:])
	if err != nil {
		return nil, err
	}

	composite := &CompositeCarrier{Carrier: *carrier}
	if composite.Basics, err = decodeOptionalList(
		EndpointBasics, contents[0], "basic", decodeBasic,
	); err != nil {
		return nil, err
	}
	if composite.Cargo, err = decodeOptionalDescriptions(
		EndpointCargoCarried, contents[1], "cargoClassDesc",
	); err != nil {
		return nil, err
	}
	if composite.Operations, err = decodeOptionalDescriptions(
		EndpointOperationClassification, contents[2], "operationClassDesc",
	); err != nil {
		return nil, err
	}
	if composite.OOS, err = decodeOptionalList(
		EndpointOOS, contents[3], "oos", decodeOOSEntry,
	); err != nil {
		return nil, err
	}
	if composite.Dockets, err = decodeOptionalList(
		EndpointDocketNumbers, contents[4], "docketNumber", decodeDocket,
	); err != nil {
		return nil, err
	}
	if composite.Authorities, err = decodeOptionalList(
		EndpointAuthority, contents[5], "carrierAuthority", decodeAuthority,
	); err != nil {
		return nil, err
	}

	composite.Raw, err = sonic.Marshal(compositeDocument{
		Carrier:                 carrier.Raw,
		Basics:                  contents[0],
		CargoCarried:            contents[1],
		OperationClassification: contents[2],
		OOS:                     contents[3],
		DocketNumbers:           contents[4],
		Authority:               contents[5],
	})
	if err != nil {
		return nil, fmt.Errorf("fmcsa: encode composite document: %w", err)
	}
	return composite, nil
}

func (c *Client) fetchConcurrently(
	ctx context.Context,
	dot string,
	endpoints []Endpoint,
) ([][]byte, error) {
	groupCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	contents := make([][]byte, len(endpoints))
	semaphore := make(chan struct{}, compositeConcurrency)

	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	for idx, endpoint := range endpoints {
		wg.Go(func() {
			select {
			case semaphore <- struct{}{}:
			case <-groupCtx.Done():
				return
			}
			defer func() { <-semaphore }()

			content, err := c.fetchContent(
				groupCtx,
				endpoint,
				carrierPath(dot)+"/"+string(endpoint),
				nil,
			)
			if err != nil {
				if IsNotFound(err) {
					return
				}
				once.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}
			contents[idx] = content
		})
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return contents, nil
}

func (c *Client) fetchSubresource(
	ctx context.Context,
	endpoint Endpoint,
	dot string,
) ([]byte, error) {
	normalized, err := normalizeDOT(dot)
	if err != nil {
		return nil, err
	}
	return c.fetchContent(ctx, endpoint, carrierPath(normalized)+"/"+string(endpoint), nil)
}

func (c *Client) fetchContent(
	ctx context.Context,
	endpoint Endpoint,
	path string,
	query url.Values,
) ([]byte, error) {
	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: string(endpoint),
		Method:   http.MethodGet,
		Path:     path,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}

	content := resp.Body
	if jsonflex.KindOf(resp.Body) == jsonflex.KindObject {
		obj, decodeErr := jsonflex.DecodeObject(resp.Body)
		if decodeErr != nil {
			return nil, fmt.Errorf("fmcsa %s: %w: %w", endpoint, ErrUnexpectedPayload, decodeErr)
		}
		if raw, ok := obj["content"]; ok {
			content = raw
		}
	}

	switch jsonflex.KindOf(content) {
	case jsonflex.KindNull, jsonflex.KindInvalid:
		return nil, fmt.Errorf("fmcsa %s: %w", endpoint, ErrNotFound)
	case jsonflex.KindArray:
		items, decodeErr := jsonflex.DecodeArray(content)
		if decodeErr != nil {
			return nil, fmt.Errorf("fmcsa %s: %w: %w", endpoint, ErrUnexpectedPayload, decodeErr)
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("fmcsa %s: %w", endpoint, ErrNotFound)
		}
		return slices.Clone(content), nil
	case jsonflex.KindObject:
		return slices.Clone(content), nil
	case jsonflex.KindString:
		message := ""
		if value, ok := jsonflex.ParseString(content); ok {
			message = value.Value()
		}
		if isCredentialMessage(message) {
			return nil, &restx.APIError{
				StatusCode: http.StatusUnauthorized,
				Message:    c.transport.Redact(message),
			}
		}
		return nil, fmt.Errorf("fmcsa %s: %w", endpoint, ErrNotFound)
	default:
		return nil, fmt.Errorf("fmcsa %s: %w", endpoint, ErrUnexpectedPayload)
	}
}

func normalizeDOT(dot string) (string, error) {
	trimmed := strings.TrimSpace(dot)
	if trimmed == "" || stringutils.DigitsOnly(trimmed) != trimmed {
		return "", fmt.Errorf("%w: DOT number must contain only digits", ErrInvalidArgument)
	}
	return trimmed, nil
}

func carrierPath(dot string) string {
	return "/carriers/" + dot
}

func decodeCarrierContent(content []byte) (*Carrier, error) {
	if jsonflex.KindOf(content) == jsonflex.KindArray {
		carriers, err := decodeCarrierList(EndpointCarrier, content)
		if err != nil {
			return nil, err
		}
		if len(carriers) == 0 {
			return nil, fmt.Errorf("fmcsa %s: %w", EndpointCarrier, ErrNotFound)
		}
		return &carriers[0], nil
	}
	carrier, err := DecodeCarrier(content)
	if err != nil {
		return nil, fmt.Errorf("fmcsa %s: %w", EndpointCarrier, err)
	}
	return carrier, nil
}

func decodeCarrierList(endpoint Endpoint, content []byte) ([]Carrier, error) {
	items, err := contentItems(endpoint, content)
	if err != nil {
		return nil, err
	}
	carriers := make([]Carrier, 0, len(items))
	for _, item := range items {
		if jsonflex.KindOf(item) != jsonflex.KindObject {
			continue
		}
		var carrier Carrier
		if err = carrier.UnmarshalJSON(item); err != nil {
			return nil, fmt.Errorf("fmcsa %s: %w", endpoint, err)
		}
		carriers = append(carriers, carrier)
	}
	return carriers, nil
}

func decodeList[T any](
	endpoint Endpoint,
	content []byte,
	wrapperKey string,
	decode func(item jsonflex.Object, raw []byte) T,
) ([]T, error) {
	items, err := contentItems(endpoint, content)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		unwrapped := jsonflex.Unwrap(item, wrapperKey)
		if jsonflex.KindOf(unwrapped) != jsonflex.KindObject {
			continue
		}
		obj, decodeErr := jsonflex.DecodeObject(unwrapped)
		if decodeErr != nil {
			return nil, fmt.Errorf(
				"fmcsa %s: %w: %w",
				endpoint,
				ErrUnexpectedPayload,
				decodeErr,
			)
		}
		out = append(out, decode(obj, slices.Clone(unwrapped)))
	}
	return out, nil
}

func decodeOptionalList[T any](
	endpoint Endpoint,
	content []byte,
	wrapperKey string,
	decode func(item jsonflex.Object, raw []byte) T,
) ([]T, error) {
	if content == nil {
		return []T{}, nil
	}
	return decodeList(endpoint, content, wrapperKey, decode)
}

func decodeDescriptions(endpoint Endpoint, content []byte, key string) ([]string, error) {
	items, err := contentItems(endpoint, content)
	if err != nil {
		return nil, err
	}
	descriptions := make([]string, 0, len(items))
	for _, item := range items {
		if jsonflex.KindOf(item) != jsonflex.KindObject {
			continue
		}
		obj, decodeErr := jsonflex.DecodeObject(item)
		if decodeErr != nil {
			return nil, fmt.Errorf(
				"fmcsa %s: %w: %w",
				endpoint,
				ErrUnexpectedPayload,
				decodeErr,
			)
		}
		if description := obj.Text(key); description != "" {
			descriptions = append(descriptions, description)
		}
	}
	return descriptions, nil
}

func decodeOptionalDescriptions(
	endpoint Endpoint,
	content []byte,
	key string,
) ([]string, error) {
	if content == nil {
		return []string{}, nil
	}
	return decodeDescriptions(endpoint, content, key)
}

func contentItems(endpoint Endpoint, content []byte) ([]sonic.NoCopyRawMessage, error) {
	switch jsonflex.KindOf(content) {
	case jsonflex.KindArray:
		items, err := jsonflex.DecodeArray(content)
		if err != nil {
			return nil, fmt.Errorf("fmcsa %s: %w: %w", endpoint, ErrUnexpectedPayload, err)
		}
		return items, nil
	case jsonflex.KindObject:
		return []sonic.NoCopyRawMessage{sonic.NoCopyRawMessage(content)}, nil
	default:
		return nil, fmt.Errorf("fmcsa %s: %w", endpoint, ErrUnexpectedPayload)
	}
}

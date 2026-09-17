package carrierok

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	defaultBaseURL        = "https://api.carrierok.com"
	defaultTimeout        = 30 * time.Second
	defaultUserAgent      = "trenova-carrierok/2"
	defaultMaxAttempts    = 3
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
	defaultLimiterPrefix  = "carrierok"
	sandboxKeyPrefix      = "sk_test_"
)

type Client struct {
	transport *restx.Client
	sandbox   bool
}

type Suggestion struct {
	DOTNumber string
	LegalName string
	DBAName   string
	City      string
	State     string
}

type SearchResult struct {
	Items      []Profile
	TotalCount int64
	Raw        []byte
}

type monitoringBody struct {
	ProfileIDs []string `json:"profile_ids"`
}

func IsSandboxKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), sandboxKeyPrefix)
}

func New(apiKey string, opts ...Option) (*Client, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, ErrAPIKeyRequired
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	sandbox := IsSandboxKey(key)
	restCfg := restx.Config{
		BaseURL:      cfg.baseURL,
		Timeout:      cfg.timeout,
		UserAgent:    cfg.userAgent,
		HTTPClient:   cfg.httpClient,
		Headers:      map[string]string{"Authorization": "Bearer " + key},
		Retry:        cfg.retry,
		Observer:     cfg.observer,
		ErrorDecoder: decodeError,
	}
	if cfg.limiter != nil && !sandbox {
		restCfg.Limiter = cfg.limiter
		restCfg.BucketFor = bucketResolver(cfg.limiterKeyPrefix)
	}

	transport, err := restx.New(restCfg)
	if err != nil {
		return nil, fmt.Errorf("carrierok: configure transport: %w", err)
	}
	return &Client{transport: transport, sandbox: sandbox}, nil
}

func (c *Client) Sandbox() bool {
	return c.sandbox
}

func bucketResolver(keyPrefix string) func(endpoint string) (restx.Bucket, bool) {
	prefix := strings.TrimSpace(keyPrefix)
	if prefix == "" {
		prefix = defaultLimiterPrefix
	}
	policies := DefaultPolicies()
	buckets := make(map[string]restx.Bucket, len(policies))
	for endpoint, bucket := range policies {
		bucket.Key = prefix + ":" + string(endpoint)
		buckets[string(endpoint)] = bucket
	}
	return func(endpoint string) (restx.Bucket, bool) {
		bucket, ok := buckets[endpoint]
		return bucket, ok
	}
}

func (c *Client) Profile(ctx context.Context, query ProfileQuery) (*Profile, error) {
	return c.firstProfile(ctx, EndpointProfile, pathProfile, &query)
}

func (c *Client) Profiles(ctx context.Context, query ProfileQuery) ([]Profile, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	return c.profiles(ctx, EndpointProfile, pathProfile, query.values())
}

func (c *Client) ProfileLite(ctx context.Context, query ProfileQuery) (*Profile, error) {
	return c.firstProfile(ctx, EndpointProfileLite, pathProfileLite, &query)
}

func (c *Client) ProfileFMCSA(ctx context.Context, query FMCSAQuery) (*Profile, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	profiles, err := c.profiles(ctx, EndpointProfileFMCSA, pathProfileFMCSA, query.values())
	if err != nil {
		return nil, err
	}
	return firstOrNotFound(EndpointProfileFMCSA, profiles)
}

func (c *Client) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: string(EndpointSearch),
		Method:   http.MethodGet,
		Path:     pathSearch,
		Query:    params.values(),
	})
	if err != nil {
		return nil, err
	}

	items, total, err := parseEnvelope(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", EndpointSearch, err)
	}
	profiles, err := decodeProfiles(items)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", EndpointSearch, err)
	}
	if total == nil {
		count := int64(len(profiles))
		total = &count
	}
	return &SearchResult{Items: profiles, TotalCount: *total, Raw: resp.Body}, nil
}

func (c *Client) Autocomplete(ctx context.Context, q string, limit int) ([]Suggestion, error) {
	if err := validateAutocomplete(q, limit); err != nil {
		return nil, err
	}

	query := url.Values{"q": []string{strings.TrimSpace(q)}}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: string(EndpointAutocomplete),
		Method:   http.MethodGet,
		Path:     pathAutocomplete,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}

	items, _, err := parseEnvelope(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", EndpointAutocomplete, err)
	}

	suggestions := make([]Suggestion, 0, len(items))
	for _, item := range items {
		obj, decodeErr := jsonflex.DecodeObject(item)
		if decodeErr != nil {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			DOTNumber: obj.Text("dot_number"),
			LegalName: obj.Text("legal_name"),
			DBAName:   obj.Text("dba_name"),
			City:      obj.Text("city"),
			State:     obj.Text("state"),
		})
	}
	return suggestions, nil
}

func (c *Client) AddToMonitoring(
	ctx context.Context,
	profileIDs []string,
) (*MonitoringWriteResult, error) {
	return c.writeMonitoring(ctx, EndpointMonitoringAdd, pathMonitoringAdd, profileIDs)
}

func (c *Client) RemoveFromMonitoring(
	ctx context.Context,
	profileIDs []string,
) (*MonitoringWriteResult, error) {
	return c.writeMonitoring(ctx, EndpointMonitoringRemove, pathMonitoringRemove, profileIDs)
}

func (c *Client) ListMonitoring(
	ctx context.Context,
	params MonitoringListParams,
) (*MonitoringListResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: string(EndpointMonitoringList),
		Method:   http.MethodGet,
		Path:     pathMonitoringList,
		Query:    params.values(),
	})
	if err != nil {
		return nil, err
	}

	result, err := decodeMonitoringList(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", EndpointMonitoringList, err)
	}
	return result, nil
}

func (c *Client) writeMonitoring(
	ctx context.Context,
	endpoint Endpoint,
	path string,
	profileIDs []string,
) (*MonitoringWriteResult, error) {
	ids := sliceutils.DedupeStrings(profileIDs)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: at least one profile id is required", ErrInvalidQuery)
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint:       string(endpoint),
		Method:         http.MethodPost,
		Path:           path,
		Body:           monitoringBody{ProfileIDs: ids},
		ExpectedStatus: []int{http.StatusOK, http.StatusCreated, http.StatusMultiStatus},
	})
	if err != nil {
		return nil, err
	}

	result, err := decodeMonitoringWrite(resp.StatusCode, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", endpoint, err)
	}
	return result, nil
}

func (c *Client) firstProfile(
	ctx context.Context,
	endpoint Endpoint,
	path string,
	query *ProfileQuery,
) (*Profile, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	profiles, err := c.profiles(ctx, endpoint, path, query.values())
	if err != nil {
		return nil, err
	}
	return firstOrNotFound(endpoint, profiles)
}

func (c *Client) profiles(
	ctx context.Context,
	endpoint Endpoint,
	path string,
	query url.Values,
) ([]Profile, error) {
	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: string(endpoint),
		Method:   http.MethodGet,
		Path:     path,
		Query:    query,
	})
	if err != nil {
		return nil, err
	}

	items, _, err := parseEnvelope(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", endpoint, err)
	}
	profiles, err := decodeProfiles(items)
	if err != nil {
		return nil, fmt.Errorf("carrierok %s: %w", endpoint, err)
	}
	return profiles, nil
}

func firstOrNotFound(endpoint Endpoint, profiles []Profile) (*Profile, error) {
	if len(profiles) == 0 {
		return nil, fmt.Errorf("carrierok %s: %w", endpoint, ErrNotFound)
	}
	return &profiles[0], nil
}

func decodeProfiles(items []sonic.NoCopyRawMessage) ([]Profile, error) {
	profiles := make([]Profile, 0, len(items))
	for idx, item := range items {
		if jsonflex.KindOf(item) != jsonflex.KindObject {
			continue
		}
		var profile Profile
		if err := profile.UnmarshalJSON(item); err != nil {
			return nil, fmt.Errorf("decode item %d: %w", idx, err)
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func parseEnvelope(body []byte) ([]sonic.NoCopyRawMessage, *int64, error) {
	switch jsonflex.KindOf(body) {
	case jsonflex.KindNull:
		return nil, nil, nil
	case jsonflex.KindInvalid:
		if len(bytes.TrimSpace(body)) == 0 {
			return nil, nil, nil
		}
		return nil, nil, ErrUnexpectedPayload
	case jsonflex.KindArray:
		items, err := jsonflex.DecodeArray(body)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", ErrUnexpectedPayload, err)
		}
		return items, nil, nil
	case jsonflex.KindObject:
		obj, err := jsonflex.DecodeObject(body)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %w", ErrUnexpectedPayload, err)
		}
		total := obj.Int("total_count", "totalCount", "total").Ptr()
		if !obj.Has("items") {
			if obj.Has("dot_number") || obj.Has("legal_name") || obj.Has("docket_number") {
				return []sonic.NoCopyRawMessage{sonic.NoCopyRawMessage(body)}, total, nil
			}
			return nil, nil, fmt.Errorf("%w: missing items", ErrUnexpectedPayload)
		}
		switch jsonflex.KindOf(obj["items"]) {
		case jsonflex.KindArray:
			items, _ := obj.Array("items")
			return items, total, nil
		case jsonflex.KindObject:
			return []sonic.NoCopyRawMessage{obj["items"]}, total, nil
		case jsonflex.KindNull:
			return nil, total, nil
		default:
			return nil, nil, fmt.Errorf("%w: items is not a list", ErrUnexpectedPayload)
		}
	default:
		return nil, nil, ErrUnexpectedPayload
	}
}

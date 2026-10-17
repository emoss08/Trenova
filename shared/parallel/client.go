package parallel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	DefaultBaseURL = "https://api.parallel.ai"

	defaultTimeout        = 45 * time.Second
	defaultUserAgent      = "trenova-parallel/1"
	defaultMaxAttempts    = 2
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	maxResponseBytes      = 8 << 20

	endpointSearch  = "search"
	endpointExtract = "extract"
	pathSearch      = "/v1/search"
	pathExtract     = "/v1/extract"

	ModeTurbo    = "turbo"
	ModeFast     = "fast"
	ModeBasic    = "basic"
	ModeAdvanced = "advanced"

	MaxSearchQueries    = 5
	MaxQueryLength      = 400
	MaxResults          = 40
	MaxSourceFilters    = 200
	MaxExtractURLs      = 20
	MaxCharsPerResult   = 1_000_000
	includedResults     = 10
	minFetchAgeSeconds  = 600
	afterDateLayout     = time.DateOnly
)

var (
	searchPriceFast     = decimal.RequireFromString("0.001")
	searchPriceAdvanced = decimal.RequireFromString("0.005")
	extraResultPrice    = decimal.RequireFromString("0.001")
	extractPricePerURL  = decimal.RequireFromString("0.001")
)

var (
	ErrAPIKeyRequired    = errors.New("parallel: API key is required")
	ErrInvalidRequest    = errors.New("parallel: invalid request")
	ErrUnexpectedPayload = errors.New("parallel: unexpected response payload")
	ErrContentNotFound   = errors.New("parallel: the page could not be retrieved")
)

type Client struct {
	transport *restx.Client
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

	transport, err := restx.New(restx.Config{
		BaseURL:          cfg.baseURL,
		Timeout:          cfg.timeout,
		UserAgent:        cfg.userAgent,
		HTTPClient:       cfg.httpClient,
		Headers:          map[string]string{"x-api-key": key},
		Retry:            cfg.retry,
		Observer:         cfg.observer,
		MaxResponseBytes: maxResponseBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("parallel: configure transport: %w", err)
	}

	return &Client{transport: transport}, nil
}

type SourcePolicy struct {
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	AfterDate      string   `json:"after_date,omitempty"`
}

type FetchPolicy struct {
	MaxAgeSeconds        *int     `json:"max_age_seconds,omitempty"`
	TimeoutSeconds       *float64 `json:"timeout_seconds,omitempty"`
	DisableCacheFallback bool     `json:"disable_cache_fallback,omitempty"`
}

type ExcerptSettings struct {
	MaxCharsPerResult int `json:"max_chars_per_result,omitempty"`
}

type FullContentSettings struct {
	MaxCharsPerResult int `json:"max_chars_per_result,omitempty"`
}

type SearchSettings struct {
	SourcePolicy    *SourcePolicy    `json:"source_policy,omitempty"`
	FetchPolicy     *FetchPolicy     `json:"fetch_policy,omitempty"`
	ExcerptSettings *ExcerptSettings `json:"excerpt_settings,omitempty"`
	Location        string           `json:"location,omitempty"`
	MaxResults      int              `json:"max_results,omitempty"`
}

type SearchRequest struct {
	Objective        string          `json:"objective,omitempty"`
	SearchQueries    []string        `json:"search_queries"`
	Mode             string          `json:"mode,omitempty"`
	MaxCharsTotal    int             `json:"max_chars_total,omitempty"`
	AdvancedSettings *SearchSettings `json:"advanced_settings,omitempty"`
}

type ExtractSettings struct {
	ExcerptSettings *ExcerptSettings     `json:"excerpt_settings,omitempty"`
	FullContent     *FullContentSettings `json:"full_content,omitempty"`
	FetchPolicy     *FetchPolicy         `json:"fetch_policy,omitempty"`
}

type ExtractRequest struct {
	URLs             []string         `json:"urls"`
	Objective        string           `json:"objective,omitempty"`
	AdvancedSettings *ExtractSettings `json:"advanced_settings,omitempty"`
}

type Result struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	PublishDate string   `json:"publish_date"`
	Excerpts    []string `json:"excerpts"`
	FullContent string   `json:"full_content"`
}

type Usage struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type SearchResponse struct {
	SearchID string   `json:"search_id"`
	Results  []Result `json:"results"`
	Usage    []Usage  `json:"usage"`

	mode string
}

type ExtractError struct {
	URL            string `json:"url"`
	ErrorType      string `json:"error_type"`
	HTTPStatusCode int    `json:"http_status_code"`
}

type ExtractResponse struct {
	ExtractID string         `json:"extract_id"`
	Results   []Result       `json:"results"`
	Errors    []ExtractError `json:"errors"`
	Usage     []Usage        `json:"usage"`
}

func (r SearchRequest) Validate() error {
	if len(r.SearchQueries) == 0 {
		return fmt.Errorf("%w: at least one search query is required", ErrInvalidRequest)
	}
	if len(r.SearchQueries) > MaxSearchQueries {
		return fmt.Errorf("%w: too many search queries", ErrInvalidRequest)
	}
	for _, query := range r.SearchQueries {
		if strings.TrimSpace(query) == "" || len(query) > MaxQueryLength {
			return fmt.Errorf("%w: each search query must be 1 to %d characters",
				ErrInvalidRequest, MaxQueryLength)
		}
	}
	switch r.Mode {
	case "", ModeTurbo, ModeFast, ModeBasic, ModeAdvanced:
	default:
		return fmt.Errorf("%w: mode %q is not supported", ErrInvalidRequest, r.Mode)
	}
	settings := r.AdvancedSettings
	if settings == nil {
		return nil
	}
	if settings.MaxResults < 0 || settings.MaxResults > MaxResults {
		return fmt.Errorf("%w: max_results is out of range", ErrInvalidRequest)
	}
	if err := validateExcerpts(settings.ExcerptSettings); err != nil {
		return err
	}
	if err := validateFetch(settings.FetchPolicy); err != nil {
		return err
	}
	if policy := settings.SourcePolicy; policy != nil {
		if len(policy.IncludeDomains)+len(policy.ExcludeDomains) > MaxSourceFilters {
			return fmt.Errorf("%w: too many domain filters", ErrInvalidRequest)
		}
		if policy.AfterDate != "" {
			if _, err := time.Parse(afterDateLayout, policy.AfterDate); err != nil {
				return fmt.Errorf("%w: after_date must be YYYY-MM-DD", ErrInvalidRequest)
			}
		}
	}

	return nil
}

func (r ExtractRequest) Validate() error {
	if len(r.URLs) == 0 || len(r.URLs) > MaxExtractURLs {
		return fmt.Errorf("%w: between 1 and %d urls are required", ErrInvalidRequest, MaxExtractURLs)
	}
	for _, target := range r.URLs {
		if strings.TrimSpace(target) == "" {
			return fmt.Errorf("%w: urls cannot be blank", ErrInvalidRequest)
		}
	}
	settings := r.AdvancedSettings
	if settings == nil {
		return nil
	}
	if err := validateExcerpts(settings.ExcerptSettings); err != nil {
		return err
	}
	if full := settings.FullContent; full != nil &&
		(full.MaxCharsPerResult < 0 || full.MaxCharsPerResult > MaxCharsPerResult) {
		return fmt.Errorf("%w: full_content max_chars_per_result is out of range", ErrInvalidRequest)
	}

	return validateFetch(settings.FetchPolicy)
}

func validateExcerpts(settings *ExcerptSettings) error {
	if settings != nil &&
		(settings.MaxCharsPerResult < 0 || settings.MaxCharsPerResult > MaxCharsPerResult) {
		return fmt.Errorf("%w: excerpt max_chars_per_result is out of range", ErrInvalidRequest)
	}

	return nil
}

func validateFetch(policy *FetchPolicy) error {
	if policy != nil && policy.MaxAgeSeconds != nil && *policy.MaxAgeSeconds < minFetchAgeSeconds {
		return fmt.Errorf("%w: max_age_seconds must be at least %d",
			ErrInvalidRequest, minFetchAgeSeconds)
	}

	return nil
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	queries := make([]string, 0, len(req.SearchQueries))
	for _, query := range req.SearchQueries {
		queries = append(queries, strings.TrimSpace(query))
	}
	req.SearchQueries = queries
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointSearch,
		Method:   http.MethodPost,
		Path:     pathSearch,
		Body:     req,
	})
	if err != nil {
		return nil, err
	}

	out := &SearchResponse{mode: req.Mode}
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointSearch, err)
	}

	return out, nil
}

func (c *Client) Extract(ctx context.Context, req ExtractRequest) (*ExtractResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointExtract,
		Method:   http.MethodPost,
		Path:     pathExtract,
		Body:     req,
	})
	if err != nil {
		return nil, err
	}

	out := new(ExtractResponse)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointExtract, err)
	}

	return out, nil
}

func (r *SearchResponse) TotalCost() decimal.Decimal {
	if r == nil {
		return decimal.Zero
	}

	price := searchPriceAdvanced
	if r.mode == ModeTurbo || r.mode == ModeFast {
		price = searchPriceFast
	}
	if extra := len(r.Results) - includedResults; extra > 0 {
		price = price.Add(extraResultPrice.Mul(decimal.NewFromInt(int64(extra))))
	}

	return price
}

func (r *ExtractResponse) TotalCost() decimal.Decimal {
	if r == nil {
		return decimal.Zero
	}

	fetched := len(r.Results) + len(r.Errors)

	return extractPricePerURL.Mul(decimal.NewFromInt(int64(fetched)))
}

func (r *ExtractResponse) Page(target string) (*Result, bool) {
	if r == nil {
		return nil, false
	}
	for idx := range r.Results {
		if r.Results[idx].URL == target {
			return &r.Results[idx], true
		}
	}
	if len(r.Errors) == 0 && len(r.Results) == 1 {
		return &r.Results[0], true
	}

	return nil, false
}

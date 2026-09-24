package exa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	DefaultBaseURL = "https://api.exa.ai"

	defaultTimeout        = 45 * time.Second
	defaultUserAgent      = "trenova-exa/1"
	defaultMaxAttempts    = 2
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	maxResponseBytes      = 8 << 20

	endpointSearch   = "search"
	endpointContents = "contents"
	pathSearch       = "/search"
	pathContents     = "/contents"

	MaxQueryLength       = 2000
	MaxNumResults        = 100
	MaxDomainFilters     = 1200
	MaxContentCharacters = 1_000_000
)

var (
	ErrAPIKeyRequired    = errors.New("exa: API key is required")
	ErrInvalidRequest    = errors.New("exa: invalid request")
	ErrUnexpectedPayload = errors.New("exa: unexpected response payload")
	ErrContentNotFound   = errors.New("exa: the page could not be retrieved")
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
		ErrorDecoder:     decodeError,
		MaxResponseBytes: maxResponseBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("exa: configure transport: %w", err)
	}

	return &Client{transport: transport}, nil
}

type SearchRequest struct {
	Query              string          `json:"query"`
	Type               string          `json:"type,omitempty"`
	NumResults         int             `json:"numResults,omitempty"`
	IncludeDomains     []string        `json:"includeDomains,omitempty"`
	ExcludeDomains     []string        `json:"excludeDomains,omitempty"`
	StartPublishedDate string          `json:"startPublishedDate,omitempty"`
	EndPublishedDate   string          `json:"endPublishedDate,omitempty"`
	Moderation         bool            `json:"moderation,omitempty"`
	Contents           *ContentOptions `json:"contents,omitempty"`
}

type ContentOptions struct {
	Text        *TextOptions      `json:"text,omitempty"`
	Highlights  *HighlightOptions `json:"highlights,omitempty"`
	MaxAgeHours *int              `json:"maxAgeHours,omitempty"`
}

type TextOptions struct {
	MaxCharacters int `json:"maxCharacters,omitempty"`
}

type HighlightOptions struct {
	Query         string `json:"query,omitempty"`
	MaxCharacters int    `json:"maxCharacters,omitempty"`
}

type ContentsRequest struct {
	URLs        []string          `json:"urls"`
	Text        *TextOptions      `json:"text,omitempty"`
	Highlights  *HighlightOptions `json:"highlights,omitempty"`
	MaxAgeHours *int              `json:"maxAgeHours,omitempty"`
}

type Result struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	PublishedDate string   `json:"publishedDate"`
	Author        string   `json:"author"`
	Text          string   `json:"text"`
	Highlights    []string `json:"highlights"`
}

type Cost struct {
	Total decimal.Decimal `json:"total"`
}

type ContentStatus struct {
	ID     string              `json:"id"`
	Status string              `json:"status"`
	Source string              `json:"source"`
	Error  *ContentStatusError `json:"error"`
}

type ContentStatusError struct {
	Tag            string `json:"tag"`
	HTTPStatusCode int    `json:"httpStatusCode"`
}

type SearchResponse struct {
	RequestID   string   `json:"requestId"`
	Results     []Result `json:"results"`
	CostDollars *Cost    `json:"costDollars"`
}

type ContentsResponse struct {
	RequestID   string          `json:"requestId"`
	Results     []Result        `json:"results"`
	Statuses    []ContentStatus `json:"statuses"`
	CostDollars *Cost           `json:"costDollars"`
}

func (r *SearchResponse) TotalCost() decimal.Decimal {
	if r == nil || r.CostDollars == nil {
		return decimal.Zero
	}

	return r.CostDollars.Total
}

func (r *ContentsResponse) TotalCost() decimal.Decimal {
	if r == nil || r.CostDollars == nil {
		return decimal.Zero
	}

	return r.CostDollars.Total
}

func (r SearchRequest) Validate() error {
	query := strings.TrimSpace(r.Query)
	switch {
	case query == "":
		return fmt.Errorf("%w: query is required", ErrInvalidRequest)
	case len(query) > MaxQueryLength:
		return fmt.Errorf("%w: query is longer than %d characters", ErrInvalidRequest, MaxQueryLength)
	case r.NumResults < 0 || r.NumResults > MaxNumResults:
		return fmt.Errorf("%w: numResults must be between 1 and %d", ErrInvalidRequest, MaxNumResults)
	case len(r.IncludeDomains) > MaxDomainFilters || len(r.ExcludeDomains) > MaxDomainFilters:
		return fmt.Errorf("%w: at most %d domain filters are allowed", ErrInvalidRequest, MaxDomainFilters)
	}

	return r.Contents.validate()
}

func (o *ContentOptions) validate() error {
	if o == nil {
		return nil
	}

	return validateContent(o.Text, o.Highlights)
}

func (r ContentsRequest) Validate() error {
	switch {
	case len(r.URLs) == 0:
		return fmt.Errorf("%w: at least one URL is required", ErrInvalidRequest)
	case len(r.URLs) > MaxNumResults:
		return fmt.Errorf("%w: at most %d URLs are allowed", ErrInvalidRequest, MaxNumResults)
	}
	for _, target := range r.URLs {
		if strings.TrimSpace(target) == "" {
			return fmt.Errorf("%w: URLs must not be blank", ErrInvalidRequest)
		}
	}

	return validateContent(r.Text, r.Highlights)
}

func validateContent(text *TextOptions, highlights *HighlightOptions) error {
	if text != nil && (text.MaxCharacters < 0 || text.MaxCharacters > MaxContentCharacters) {
		return fmt.Errorf("%w: text maxCharacters is out of range", ErrInvalidRequest)
	}
	if highlights != nil &&
		(highlights.MaxCharacters < 0 || highlights.MaxCharacters > MaxContentCharacters) {
		return fmt.Errorf("%w: highlights maxCharacters is out of range", ErrInvalidRequest)
	}

	return nil
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	req.Query = strings.TrimSpace(req.Query)
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

	out := new(SearchResponse)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointSearch, err)
	}

	return out, nil
}

func (c *Client) Contents(ctx context.Context, req ContentsRequest) (*ContentsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointContents,
		Method:   http.MethodPost,
		Path:     pathContents,
		Body:     req,
	})
	if err != nil {
		return nil, err
	}

	out := new(ContentsResponse)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointContents, err)
	}

	return out, nil
}

func (r *ContentsResponse) Failure(target string) (ContentStatus, bool) {
	if r == nil {
		return ContentStatus{}, false
	}
	for _, status := range r.Statuses {
		if status.Status != "" && status.Status != "success" &&
			(status.ID == "" || status.ID == target) {
			return status, true
		}
	}

	return ContentStatus{}, false
}

func IsUnauthorized(err error) bool {
	return restx.IsStatus(err, http.StatusUnauthorized) ||
		restx.IsStatus(err, http.StatusForbidden)
}

func IsOutOfCredits(err error) bool {
	return restx.IsStatus(err, http.StatusPaymentRequired)
}

func IsRateLimited(err error) bool {
	return restx.IsRateLimited(err)
}

func IsInvalidRequest(err error) bool {
	return errors.Is(err, ErrInvalidRequest) || restx.IsStatus(err, http.StatusBadRequest)
}

func decodeError(status int, body []byte, header http.Header) error {
	apiErr := restx.DecodeAPIError(status, body, header)
	if obj, err := jsonflex.DecodeObject(body); err == nil {
		if tag := obj.Text("tag"); tag != "" {
			apiErr.Code = tag
		}
	}

	return apiErr
}

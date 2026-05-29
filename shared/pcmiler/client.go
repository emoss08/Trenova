package pcmiler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	defaultBaseURL = "https://pcmiler.alk.com/apis/rest/v1.0/Service.svc"
	defaultTimeout = 15 * time.Second
	maxRoutesBatch = 20
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) (*Client, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" {
		return nil, errors.New("PC*Miler API key is required")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("invalid PC*Miler base URL: %w", err)
	}

	timeout := defaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) Versions(ctx context.Context, region string) ([]Version, error) {
	endpoint := c.baseURL + "/pcmversion"
	if strings.TrimSpace(region) != "" {
		endpoint += "?region=" + url.QueryEscape(region)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create PC*Miler versions request: %w", err)
	}

	body, err := c.do(req)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		PCMVersions []string `json:"pcmversions"`
	}
	if err = sonic.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("parse PC*Miler versions response: %w", err)
	}

	versions := make([]Version, 0, len(wrapped.PCMVersions))
	for _, version := range wrapped.PCMVersions {
		versions = append(versions, Version{Name: version})
	}

	return versions, nil
}

func (c *Client) Mileage(ctx context.Context, routes []RouteRequest) ([]RouteMileage, error) {
	results := make([]RouteMileage, 0, len(routes))
	for start := 0; start < len(routes); start += maxRoutesBatch {
		end := start + maxRoutesBatch
		if end > len(routes) {
			end = len(routes)
		}

		batch, err := c.mileageBatch(ctx, routes[start:end])
		if err != nil {
			return nil, err
		}
		results = append(results, batch...)
	}

	return results, nil
}

func (c *Client) mileageBatch(ctx context.Context, routes []RouteRequest) ([]RouteMileage, error) {
	payload := buildRouteReportsPayload(routes)
	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode PC*Miler mileage request: %w", err)
	}

	endpoint, err := url.Parse(c.baseURL + "/route/routeReports")
	if err != nil {
		return nil, fmt.Errorf("create PC*Miler mileage URL: %w", err)
	}
	dataVersion := strings.TrimSpace(routes[0].Options.DataVersion)
	if dataVersion == "" {
		dataVersion = "Current"
	}
	query := endpoint.Query()
	query.Set("dataVersion", dataVersion)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create PC*Miler mileage request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	respBody, err := c.do(req)
	if err != nil {
		return nil, err
	}

	var response []routeReport
	if err = sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("parse PC*Miler mileage response: %w", err)
	}

	return parseMileageResponse(response), nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PC*Miler request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read PC*Miler response: %w", readErr)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    trimErrorBody(body),
		}
	}

	return body, nil
}

func trimErrorBody(body []byte) string {
	message := strings.TrimSpace(string(body))
	if len(message) > 256 {
		message = message[:256]
	}
	if message == "" {
		return "empty response"
	}
	return message
}

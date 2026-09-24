package exaconnector

import (
	"context"
	"net/http"
	"strings"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/httpsafe"
)

const (
	requestTimeout  = 45 * time.Second
	pageMaxAgeHours = 24
)

var _ serviceports.WebSearchProvider = (*Connector)(nil)

type Connector struct {
	httpClient *http.Client
	baseURL    string
}

func New() serviceports.WebSearchProvider {
	return &Connector{
		httpClient: httpsafe.NewClient(requestTimeout),
		baseURL:    exa.DefaultBaseURL,
	}
}

func (c *Connector) client(apiKey string) (*exa.Client, error) {
	return exa.New(apiKey, exa.WithHTTPClient(c.httpClient), exa.WithBaseURL(c.baseURL))
}

func (c *Connector) Search(
	ctx context.Context,
	req serviceports.WebSearchProviderRequest,
) (*serviceports.WebSearchProviderResponse, error) {
	client, err := c.client(req.APIKey)
	if err != nil {
		return nil, err
	}

	resp, err := client.Search(ctx, exa.SearchRequest{
		Query:              req.Query,
		Type:               req.SearchType,
		NumResults:         req.NumResults,
		ExcludeDomains:     req.ExcludeDomains,
		StartPublishedDate: req.StartPublishedDate,
		Contents: &exa.ContentOptions{
			Highlights: &exa.HighlightOptions{
				Query:         req.Query,
				MaxCharacters: req.ExcerptCharacters,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	results := make([]serviceports.WebSearchProviderResult, 0, len(resp.Results))
	for idx := range resp.Results {
		result := &resp.Results[idx]
		target := strings.TrimSpace(result.URL)
		if target == "" {
			continue
		}
		results = append(results, serviceports.WebSearchProviderResult{
			Title:         strings.TrimSpace(result.Title),
			URL:           target,
			PublishedDate: strings.TrimSpace(result.PublishedDate),
			Author:        strings.TrimSpace(result.Author),
			Excerpts:      result.Highlights,
		})
	}

	return &serviceports.WebSearchProviderResponse{
		Results: results,
		CostUSD: resp.TotalCost(),
	}, nil
}

func (c *Connector) Read(
	ctx context.Context,
	req serviceports.WebPageProviderRequest,
) (*serviceports.WebPageProviderResponse, error) {
	client, err := c.client(req.APIKey)
	if err != nil {
		return nil, err
	}

	maxAge := pageMaxAgeHours
	resp, err := client.Contents(ctx, exa.ContentsRequest{
		URLs:        []string{req.URL},
		Text:        &exa.TextOptions{MaxCharacters: req.MaxCharacters},
		MaxAgeHours: &maxAge,
	})
	if err != nil {
		return nil, err
	}

	cost := resp.TotalCost()
	if _, failed := resp.Failure(req.URL); failed || len(resp.Results) == 0 {
		return &serviceports.WebPageProviderResponse{URL: req.URL, CostUSD: cost}, exa.ErrContentNotFound
	}

	page := &resp.Results[0]

	return &serviceports.WebPageProviderResponse{
		Title:         strings.TrimSpace(page.Title),
		URL:           strings.TrimSpace(page.URL),
		PublishedDate: strings.TrimSpace(page.PublishedDate),
		Author:        strings.TrimSpace(page.Author),
		Text:          page.Text,
		CostUSD:       cost,
	}, nil
}

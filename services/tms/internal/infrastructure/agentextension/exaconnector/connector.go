package exaconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentextension/connectorkit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/httpsafe"
)

const (
	requestTimeout  = 45 * time.Second
	pageMaxAgeHours = 24
	testQuery       = "Federal Motor Carrier Safety Administration hours of service"
	emptyTestNote   = "The test search returned no pages, which can happen; the key works."
)

var (
	_ agentextensionports.ExtensionConnector = (*Connector)(nil)
	_ agentextensionports.WebSearchProvider  = (*Connector)(nil)
)

type Connector struct {
	httpClient *http.Client
	baseURL    string
}

func New() *Connector {
	return &Connector{
		httpClient: httpsafe.NewClient(requestTimeout),
		baseURL:    exa.DefaultBaseURL,
	}
}

func (c *Connector) Type() agentextension.Type { return agentextension.TypeExa }

func (c *Connector) Classify(err error) agentextensionports.ProviderFailure {
	switch {
	case errors.Is(err, exa.ErrContentNotFound):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureNotFound,
		}
	case errors.Is(err, exa.ErrInvalidRequest):
		return connectorkit.InvalidRequest(err)
	default:
		return connectorkit.ClassifyHTTP(err)
	}
}

func (c *Connector) Test(
	ctx context.Context,
	settings map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	resp, err := c.Search(ctx, &agentextensionports.WebSearchProviderRequest{
		Settings:   withSearchType(settings, agentextension.ExaSearchTypeFast),
		Query:      testQuery,
		NumResults: 1,
	})
	if err != nil {
		if resp != nil {
			return &agentextensionports.ConnectorTestResult{CostUSD: resp.CostUSD}, err
		}
		return nil, err
	}

	result := &agentextensionports.ConnectorTestResult{CostUSD: resp.CostUSD}
	if len(resp.Results) == 0 {
		result.Note = emptyTestNote
	}

	return result, nil
}

func withSearchType(settings map[string]string, searchType string) map[string]string {
	next := make(map[string]string, len(settings)+1)
	for key, value := range settings {
		next[key] = value
	}
	next[agentextension.ConfigKeySearchType] = searchType

	return next
}

func (c *Connector) client(
	settings map[string]string,
) (*exa.Client, agentextension.ExaSettings, error) {
	parsed, err := agentextension.ParseExaSettings(settings)
	if err != nil {
		return nil, agentextension.ExaSettings{}, err
	}

	client, err := exa.New(
		parsed.APIKey,
		exa.WithHTTPClient(c.httpClient),
		exa.WithBaseURL(c.baseURL),
	)

	return client, parsed, err
}

func (c *Connector) Search(
	ctx context.Context,
	req *agentextensionports.WebSearchProviderRequest,
) (*agentextensionports.WebSearchProviderResponse, error) {
	client, settings, err := c.client(req.Settings)
	if err != nil {
		return nil, err
	}

	resp, err := client.Search(ctx, exa.SearchRequest{
		Query:              req.Query,
		Type:               settings.SearchType,
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

	results := make([]agentextensionports.WebSearchProviderResult, 0, len(resp.Results))
	for idx := range resp.Results {
		result := &resp.Results[idx]
		target := strings.TrimSpace(result.URL)
		if target == "" {
			continue
		}
		results = append(results, agentextensionports.WebSearchProviderResult{
			Title:         strings.TrimSpace(result.Title),
			URL:           target,
			PublishedDate: strings.TrimSpace(result.PublishedDate),
			Author:        strings.TrimSpace(result.Author),
			Excerpts:      result.Highlights,
		})
	}

	return &agentextensionports.WebSearchProviderResponse{
		Results: results,
		CostUSD: resp.TotalCost(),
	}, nil
}

func (c *Connector) Read(
	ctx context.Context,
	req agentextensionports.WebPageProviderRequest,
) (*agentextensionports.WebPageProviderResponse, error) {
	client, _, err := c.client(req.Settings)
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
		return &agentextensionports.WebPageProviderResponse{
			URL:     req.URL,
			CostUSD: cost,
		}, exa.ErrContentNotFound
	}

	page := &resp.Results[0]

	return &agentextensionports.WebPageProviderResponse{
		Title:         strings.TrimSpace(page.Title),
		URL:           strings.TrimSpace(page.URL),
		PublishedDate: strings.TrimSpace(page.PublishedDate),
		Author:        strings.TrimSpace(page.Author),
		Text:          page.Text,
		CostUSD:       cost,
	}, nil
}

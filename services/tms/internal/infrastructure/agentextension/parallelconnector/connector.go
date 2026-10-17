package parallelconnector

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentextension/connectorkit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/parallel"
)

const (
	requestTimeout    = 45 * time.Second
	pageMaxAgeSeconds = 24 * 60 * 60
	testQuery         = "FMCSA hours of service rules"
	emptyTestNote     = "The test search returned no pages, which can happen; the key works."
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
	return newConnector(httpsafe.NewClient(requestTimeout), parallel.DefaultBaseURL)
}

func newConnector(httpClient *http.Client, baseURL string) *Connector {
	return &Connector{httpClient: httpClient, baseURL: baseURL}
}

func (c *Connector) Type() agentextension.Type { return agentextension.TypeParallel }

func (c *Connector) Classify(err error) agentextensionports.ProviderFailure {
	switch {
	case errors.Is(err, parallel.ErrContentNotFound):
		return agentextensionports.ProviderFailure{
			Kind: agentextensionports.ProviderFailureNotFound,
		}
	case errors.Is(err, parallel.ErrInvalidRequest):
		return connectorkit.InvalidRequest(err)
	default:
		return connectorkit.ClassifyHTTP(err)
	}
}

func (c *Connector) Test(
	ctx context.Context,
	settings map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	client, _, err := c.client(settings)
	if err != nil {
		return nil, err
	}

	resp, err := client.Search(ctx, parallel.SearchRequest{
		SearchQueries:    []string{testQuery},
		Mode:             parallel.ModeFast,
		AdvancedSettings: &parallel.SearchSettings{MaxResults: 1},
	})
	if err != nil {
		return nil, err
	}

	result := &agentextensionports.ConnectorTestResult{CostUSD: resp.TotalCost()}
	if len(resp.Results) == 0 {
		result.Note = emptyTestNote
	}

	return result, nil
}

func (c *Connector) client(
	settings map[string]string,
) (*parallel.Client, agentextension.ParallelSettings, error) {
	parsed, err := agentextension.ParseParallelSettings(settings)
	if err != nil {
		return nil, agentextension.ParallelSettings{}, err
	}

	client, err := parallel.New(
		parsed.APIKey,
		parallel.WithHTTPClient(c.httpClient),
		parallel.WithBaseURL(c.baseURL),
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

	advanced := &parallel.SearchSettings{
		MaxResults:      req.NumResults,
		ExcerptSettings: &parallel.ExcerptSettings{MaxCharsPerResult: req.ExcerptCharacters},
	}
	if policy := sourcePolicy(req); policy != nil {
		advanced.SourcePolicy = policy
	}

	resp, err := client.Search(ctx, parallel.SearchRequest{
		Objective:        req.Query,
		SearchQueries:    []string{req.Query},
		Mode:             settings.Mode,
		AdvancedSettings: advanced,
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
			PublishedDate: strings.TrimSpace(result.PublishDate),
			Excerpts:      result.Excerpts,
		})
	}

	return &agentextensionports.WebSearchProviderResponse{
		Results: results,
		CostUSD: resp.TotalCost(),
	}, nil
}

func sourcePolicy(req *agentextensionports.WebSearchProviderRequest) *parallel.SourcePolicy {
	afterDate := ""
	if req.StartPublishedDate != "" {
		if started, err := time.Parse(time.RFC3339, req.StartPublishedDate); err == nil {
			afterDate = started.UTC().Format(time.DateOnly)
		}
	}
	excluded := make([]string, 0, len(req.ExcludeDomains))
	for _, domain := range req.ExcludeDomains {
		excluded = append(excluded, strings.TrimPrefix(domain, "*."))
	}
	if len(excluded) == 0 && afterDate == "" {
		return nil
	}

	return &parallel.SourcePolicy{ExcludeDomains: excluded, AfterDate: afterDate}
}

func (c *Connector) Read(
	ctx context.Context,
	req agentextensionports.WebPageProviderRequest,
) (*agentextensionports.WebPageProviderResponse, error) {
	client, _, err := c.client(req.Settings)
	if err != nil {
		return nil, err
	}

	maxAge := pageMaxAgeSeconds
	resp, err := client.Extract(ctx, parallel.ExtractRequest{
		URLs: []string{req.URL},
		AdvancedSettings: &parallel.ExtractSettings{
			FullContent: &parallel.FullContentSettings{MaxCharsPerResult: req.MaxCharacters},
			FetchPolicy: &parallel.FetchPolicy{MaxAgeSeconds: &maxAge},
		},
	})
	if err != nil {
		return nil, err
	}

	cost := resp.TotalCost()
	page, found := resp.Page(req.URL)
	if !found || strings.TrimSpace(page.FullContent) == "" {
		return &agentextensionports.WebPageProviderResponse{
			URL:     req.URL,
			CostUSD: cost,
		}, parallel.ErrContentNotFound
	}

	return &agentextensionports.WebPageProviderResponse{
		Title:         strings.TrimSpace(page.Title),
		URL:           strings.TrimSpace(page.URL),
		PublishedDate: strings.TrimSpace(page.PublishDate),
		Text:          page.FullContent,
		CostUSD:       cost,
	}, nil
}

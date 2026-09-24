package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/configspec"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type AgentExtensionTool struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type AgentExtensionCategoryOption struct {
	Value agentextension.Category `json:"value"`
	Label string                  `json:"label"`
}

type AgentExtensionCatalogItem struct {
	Type                agentextension.Type         `json:"type"`
	Name                string                      `json:"name"`
	Vendor              string                      `json:"vendor"`
	Summary             string                      `json:"summary"`
	Description         string                      `json:"description"`
	Category            agentextension.Category     `json:"category"`
	CategoryLabel       string                      `json:"categoryLabel"`
	BrandDomain         string                      `json:"brandDomain"`
	DocsURL             string                      `json:"docsUrl"`
	WebsiteURL          string                      `json:"websiteUrl"`
	PricingURL          string                      `json:"pricingUrl"`
	Capabilities        []string                    `json:"capabilities"`
	Tools               []AgentExtensionTool        `json:"tools"`
	DataNotice          string                      `json:"dataNotice"`
	Featured            bool                        `json:"featured"`
	SortOrder           int                         `json:"sortOrder"`
	ReleasedAt          int64                       `json:"releasedAt"`
	Enabled             bool                        `json:"enabled"`
	Configured          bool                        `json:"configured"`
	Availability        agentextension.Availability `json:"availability"`
	ConfigSpec          []configspec.Field          `json:"configSpec"`
	SupportsTestConnect bool                        `json:"supportsTestConnect"`
	DailyRequestLimit   int                         `json:"dailyRequestLimit"`
	Usage               agentextension.UsageSummary `json:"usage"`
	EnabledAt           *int64                      `json:"enabledAt"`
	UpdatedAt           int64                       `json:"updatedAt"`
	Version             int64                       `json:"version"`
}

type AgentExtensionCatalogResponse struct {
	Items      []AgentExtensionCatalogItem    `json:"items"`
	Categories []AgentExtensionCategoryOption `json:"categories"`
}

type AgentExtensionConfigResponse struct {
	Type         agentextension.Type         `json:"type"`
	Enabled      bool                        `json:"enabled"`
	Availability agentextension.Availability `json:"availability"`
	Fields       []configspec.FieldValue     `json:"fields"`
	Spec         []configspec.Field          `json:"spec"`
	Version      int64                       `json:"version"`
	UpdatedAt    int64                       `json:"updatedAt"`
}

type UpdateAgentExtensionRequest struct {
	TenantInfo    pagination.TenantInfo       `json:"-"`
	Enabled       bool                        `json:"enabled"`
	Availability  agentextension.Availability `json:"availability"`
	Configuration map[string]string           `json:"configuration"`
	Version       int64                       `json:"version"`
}

type AgentExtensionTestResponse struct {
	Type      agentextension.Type `json:"type"`
	Success   bool                `json:"success"`
	CheckedAt int64               `json:"checkedAt"`
	LatencyMs int64               `json:"latencyMs"`
	Message   string              `json:"message"`
}

type AgentExtensionGate interface {
	ActiveExtensions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (map[agentextension.Type]agentextension.Availability, error)
}

type WebSearchQuery struct {
	Query               string
	PublishedWithinDays int
	ResultLimit         int
}

type WebSearchHit struct {
	Ref           string
	Title         string
	URL           string
	Site          string
	Official      bool
	PublishedDate string
	Author        string
	Excerpts      []string
}

type WebSearchOutcome struct {
	Hits        []WebSearchHit
	RetrievedAt int64
}

type WebPageQuery struct {
	URL  string
	Ref  string
	Part int
}

type WebPage struct {
	Title         string
	URL           string
	Site          string
	Official      bool
	PublishedDate string
	Author        string
	Text          string
	Part          int
	HasMore       bool
	RetrievedAt   int64
}

type WebResearcher interface {
	SearchWeb(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		query WebSearchQuery,
	) (*WebSearchOutcome, error)
	ReadWebPage(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		query WebPageQuery,
	) (*WebPage, error)
}

type WebSearchProviderRequest struct {
	APIKey             string
	Query              string
	SearchType         string
	NumResults         int
	ExcludeDomains     []string
	StartPublishedDate string
	ExcerptCharacters  int
}

type WebSearchProviderResult struct {
	Title         string
	URL           string
	PublishedDate string
	Author        string
	Excerpts      []string
}

type WebSearchProviderResponse struct {
	Results []WebSearchProviderResult
	CostUSD decimal.Decimal
}

type WebPageProviderRequest struct {
	APIKey        string
	URL           string
	MaxCharacters int
}

type WebPageProviderResponse struct {
	Title         string
	URL           string
	PublishedDate string
	Author        string
	Text          string
	CostUSD       decimal.Decimal
}

type WebSearchProvider interface {
	Search(
		ctx context.Context,
		req WebSearchProviderRequest,
	) (*WebSearchProviderResponse, error)
	Read(
		ctx context.Context,
		req WebPageProviderRequest,
	) (*WebPageProviderResponse, error)
}

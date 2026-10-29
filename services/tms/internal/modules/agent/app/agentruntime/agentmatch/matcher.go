package agentmatch

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrecommendationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MatcherParams struct {
	fx.In

	Logger         *zap.Logger
	QueryTools     agenttoolports.AgentQueryToolRegistry
	ActionTools    agenttoolports.AgentToolRegistry
	Permissions    permissionports.PermissionEngine
	Catalog        *agenttoolcatalog.Catalog
	Extensions     agentextensionports.AgentExtensionGate         `optional:"true"`
	Vectorizer     retrievalports.QueryVectorizer                 `optional:"true"`
	CatalogVectors retrievalports.CatalogVectorIndex              `optional:"true"`
	Definitions    agentdefinitionports.AgentDefinitionRepository `optional:"true"`
	RateLimits     kernelports.RateLimitStore                     `optional:"true"`
	AIConfig       *aiconfig.Config                               `optional:"true"`
	Disclosures    agenttoolports.ToolDisclosureObserver          `optional:"true"`
	Scripts        llmports.ScriptProviders                       `optional:"true"`
}

// Matcher recommends the agent whose tools best fit a request.
type Matcher struct {
	logger      *zap.Logger
	access      *toolaccess.Access
	permissions permissionports.PermissionEngine
	definitions agentdefinitionports.AgentDefinitionRepository
	catalog     *agenttoolcatalog.Catalog
	vectorizer  retrievalports.QueryVectorizer
	rateLimits  kernelports.RateLimitStore
	pools       matchPools
	matchable   func() []string
}

var _ agentrecommendationports.AgentRecommender = (*Matcher)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewMatcher(p MatcherParams) *Matcher {
	return &Matcher{
		logger: p.Logger.Named("service.agentruntime"),
		access: toolaccess.NewAccess(&toolaccess.AccessParams{
			Logger:         p.Logger,
			QueryTools:     p.QueryTools,
			ActionTools:    p.ActionTools,
			Permissions:    p.Permissions,
			Catalog:        p.Catalog,
			Extensions:     p.Extensions,
			Vectorizer:     p.Vectorizer,
			CatalogVectors: p.CatalogVectors,
			Definitions:    p.Definitions,
			Threshold:      toolaccess.ConfiguredThreshold(p.AIConfig),
			Disclosures:    p.Disclosures,
			Scripts:        p.Scripts,
		}),
		permissions: p.Permissions,
		definitions: p.Definitions,
		catalog:     p.Catalog,
		vectorizer:  p.Vectorizer,
		rateLimits:  p.RateLimits,
		pools:       newMatchPools(),
		matchable:   matchableTools(p.Catalog),
	}
}

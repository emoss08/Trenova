package agentmatch

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolcatalog"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentrecommendationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MatcherParams struct {
	fx.In

	Logger      *zap.Logger
	Access      *toolaccess.Access
	Permissions permissionports.PermissionEngine
	Catalog     *agenttoolcatalog.Catalog
	Vectorizer  retrievalports.QueryVectorizer
	Definitions agentdefinitionports.AgentDefinitionRepository
	RateLimits  kernelports.RateLimitStore
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
		logger:      p.Logger.Named("service.agentruntime"),
		access:      p.Access,
		permissions: p.Permissions,
		definitions: p.Definitions,
		catalog:     p.Catalog,
		vectorizer:  p.Vectorizer,
		rateLimits:  p.RateLimits,
		pools:       newMatchPools(),
		matchable:   matchableTools(p.Catalog),
	}
}

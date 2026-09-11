package graphql

import (
	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/api/graphql/resolver"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
)

const (
	queryCacheSize   = 1024
	parserTokenLimit = 15000
)

type ServerParams struct {
	fx.In

	Config        *config.Config
	Resolver      *resolver.Resolver
	Observability *ObservabilityExtension
	CostBudget    *CostBudgetExtension
	FeatureAccess *FeatureAccessExtension
}

func NewServer(p ServerParams) *gqlhandler.Server {
	registerQueryLimitErrorCodes()

	srv := gqlhandler.New(newCostLimitedSchema(generated.NewExecutableSchema(generated.Config{
		Resolvers: p.Resolver,
	})))
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](queryCacheSize))
	srv.SetParserTokenLimit(parserTokenLimit)
	srv.Use(operationDepthLimit{max: querycost.MaxOperationDepth})
	srv.Use(extension.FixedComplexityLimit(querycost.MaxOperationCost))
	srv.Use(p.CostBudget)
	srv.Use(p.FeatureAccess)
	srv.Use(p.Observability)

	if devToolingEnabled(p.Config) {
		srv.Use(extension.Introspection{})
	} else {
		srv.SetDisableSuggestion(true)
	}

	srv.SetErrorPresenter(newErrorPresenter(p.Config))
	srv.SetRecoverFunc(recoverFunc)

	return srv
}

func devToolingEnabled(cfg *config.Config) bool {
	return cfg.App.Debug || cfg.App.IsDevelopment() || cfg.App.IsTest()
}

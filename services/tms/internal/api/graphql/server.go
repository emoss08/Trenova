package graphql

import (
	"github.com/99designs/gqlgen/graphql"
	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/graphql/resolver"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
)

const (
	queryCacheSize   = 1024
	parserTokenLimit = 15000
)

const ExtensionsGroup = gqlext.ExtensionsGroup

type ServerParams struct {
	fx.In
	gqlext.Params

	AppConfig *appconfig.Config
	Resolver  *resolver.Resolver
}

func AsExtension(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(graphql.HandlerExtension)),
		fx.ResultTags(`group:"`+ExtensionsGroup+`"`),
	)
}

func NewServer(p ServerParams) *gqlhandler.Server {
	srv := gqlhandler.New(
		gqlext.NewCostLimitedSchema(generated.NewExecutableSchema(generated.Config{
			Resolvers: p.Resolver,
		})),
	)
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](queryCacheSize))
	srv.SetParserTokenLimit(parserTokenLimit)
	for _, ext := range gqlext.Extensions(p.Params) {
		srv.Use(ext)
	}

	if devToolingEnabled(p.AppConfig) {
		srv.Use(extension.Introspection{})
	} else {
		srv.SetDisableSuggestion(true)
	}

	srv.SetErrorPresenter(gqlext.ErrorPresenter(p.AppConfig))
	srv.SetRecoverFunc(gqlext.Recover)

	return srv
}

func devToolingEnabled(cfg *appconfig.Config) bool {
	return cfg.Debug || cfg.IsDevelopment() || cfg.IsTest()
}

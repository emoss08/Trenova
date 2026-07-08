package graphql

import (
	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/resolver"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
)

const complexityLimit = 1000

type ServerParams struct {
	fx.In

	Config   *config.Config
	Resolver *resolver.Resolver
}

func NewServer(p ServerParams) *gqlhandler.Server {
	srv := gqlhandler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers:  p.Resolver,
		Complexity: complexityRoot(),
	}))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.FixedComplexityLimit(complexityLimit))
	if p.Config.App.Debug || p.Config.App.IsDevelopment() || p.Config.App.IsTest() {
		srv.Use(extension.Introspection{})
	}
	srv.SetErrorPresenter(newErrorPresenter(p.Config))
	srv.SetRecoverFunc(recoverFunc)

	return srv
}

package tariff

import (
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"go.uber.org/fx"
)

var AuthoringModule = fx.Module("tariff-authoring",
	fx.Provide(
		NewTemplateResolver,
		NewChecker,
		func(r *TemplateResolver) tariffports.Resolver { return r },
		func(r *TemplateResolver) tariffports.LibraryResolver { return r },
		func(c *Checker) tariffports.Checker { return c },
	),
)

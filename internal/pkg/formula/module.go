package formula

import (
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/emoss08/trenova/internal/pkg/formula/variables/builtin"
	"go.uber.org/fx"
)

// * newVariableRegistry creates and initializes a new variable registry with builtin variables
func newVariableRegistry() *variables.Registry {
	registry := variables.NewRegistry()
	builtin.RegisterAll(registry)
	return registry
}

var Module = fx.Module("formula",
	fx.Provide(
		newVariableRegistry,
		schema.NewSchemaRegistry,
		schema.NewDefaultDataResolver,
	),
)

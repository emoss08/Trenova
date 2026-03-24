package formula

import (
	"embed"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/internal/core/services/formula/schema"
	"go.uber.org/fx"
)

//go:embed schema/definitions/*.schema.json
var schemaFiles embed.FS

func newSchemaRegistry() (*schema.Registry, error) {
	registry := schema.NewRegistry()

	shipmentSchema, err := schemaFiles.ReadFile("schema/definitions/shipment.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load shipment schema: %w", err)
	}

	if err = registry.Register("shipment", shipmentSchema); err != nil {
		return nil, fmt.Errorf("failed to register shipment schema: %w", err)
	}

	return registry, nil
}

func asFormulaCalculator(s *Service) services.FormulaCalculator {
	return s
}

var Module = fx.Module(
	"formula",
	fx.Provide(
		newSchemaRegistry,
		resolver.NewResolver,
		engine.NewEngine,
		engine.NewEnvironmentBuilder,
		NewService,
		asFormulaCalculator,
	),
)

package formula

import (
	"embed"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/schema"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"go.uber.org/fx"
)

//go:embed schema/definitions/*.schema.json
var schemaFiles embed.FS

func ShipmentSchemaJSON() ([]byte, error) {
	shipmentSchema, err := schemaFiles.ReadFile("schema/definitions/shipment.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load shipment schema: %w", err)
	}

	return shipmentSchema, nil
}

func NewSchemaRegistry() (*schema.Registry, error) {
	registry := schema.NewRegistry()

	shipmentSchema, err := ShipmentSchemaJSON()
	if err != nil {
		return nil, err
	}

	if err = registry.Register("shipment", shipmentSchema); err != nil {
		return nil, fmt.Errorf("failed to register shipment schema: %w", err)
	}

	return registry, nil
}

func asFormulaCalculator(s *Service) formulatemplateports.FormulaCalculator {
	return s
}

func asFormulaPredicateEvaluator(s *Service) formulatemplateports.FormulaPredicateEvaluator {
	return s
}

var Module = fx.Module(
	"formula",
	fx.Provide(
		NewSchemaRegistry,
		resolver.NewResolver,
		engine.NewEngine,
		engine.NewEnvironmentBuilder,
		NewService,
		asFormulaCalculator,
		asFormulaPredicateEvaluator,
	),
)

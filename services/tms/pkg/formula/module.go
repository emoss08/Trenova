package formula

import (
	"embed"
	"fmt"

	"github.com/emoss08/trenova/pkg/formula/infrastructure"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/emoss08/trenova/pkg/formula/services"
	"github.com/emoss08/trenova/pkg/formula/variables"
	"github.com/emoss08/trenova/pkg/formula/variables/builtin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed schema/definitions/*.json
var schemaDefinitions embed.FS

type VariableRegistryParams struct {
	fx.In
	Logger *zap.Logger
}

func newVariableRegistry(p VariableRegistryParams) *variables.Registry {
	log := p.Logger.With(
		zap.String("module", "formula"),
		zap.String("component", "variable_registry"),
	)

	registry := variables.NewRegistry()
	builtin.RegisterAll(registry)

	varNames := registry.ListNames()
	log.Info(
		"formula variable registry initialized",
		zap.Int("total_variables", len(varNames)),
		zap.Strings("variables", varNames),
	)

	return registry
}

type DataResolverParams struct {
	fx.In
	Logger *zap.Logger
}

func newDataResolver(p DataResolverParams) *schema.DefaultDataResolver {
	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	return resolver
}

type SchemaRegistryParams struct {
	fx.In
	Logger *zap.Logger
}

func newSchemaRegistryWithSchemas(p SchemaRegistryParams) (*schema.Registry, error) {
	log := p.Logger.With(
		zap.String("module", "formula"),
		zap.String("component", "schema_registry"),
	)

	registry := schema.NewRegistry()

	// * Load shipment schema
	log.Info("loading formula schemas...")

	schemaJSON, err := schemaDefinitions.ReadFile("schema/definitions/shipment.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load shipment schema: %w", err)
	}

	if err := registry.RegisterSchema("shipment", schemaJSON); err != nil {
		return nil, fmt.Errorf("failed to register shipment schema: %w", err)
	}

	log.Info(
		"successfully registered schema",
		zap.String("schema", "shipment"),
	)

	schemas := registry.ListSchemas()
	log.Info(
		"successfully registered schema",
		zap.Int("total_schemas", len(schemas)),
		zap.Any("schemas", schemas),
	)

	return registry, nil
}

func initializeFormulaSystem(
	varRegistry *variables.Registry,
	schemaRegistry *schema.Registry,
	resolver *schema.DefaultDataResolver,
	logger *zap.Logger,
) error {
	log := logger.With(
		zap.String("module", "formula"),
		zap.String("component", "system_initializer"),
	)

	bridge := NewSchemaVariableBridge(schemaRegistry, varRegistry)

	if err := bridge.RegisterSchemaVariables("shipment"); err != nil {
		return fmt.Errorf("failed to register shipment variables: %w", err)
	}

	log.Info(
		"successfully registered schema variables",
		zap.String("schema", "shipment"),
	)

	return nil
}

var Module = fx.Module("formula",
	fx.Provide(
		newVariableRegistry,
		newSchemaRegistryWithSchemas,
		newDataResolver,
		NewSchemaVariableBridge,
		infrastructure.NewPostgresDataLoader,
		services.NewFormulaEvaluationService,
	),
	fx.Invoke(initializeFormulaSystem),
)

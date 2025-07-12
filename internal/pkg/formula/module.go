package formula

import (
	"embed"
	"fmt"

	"github.com/emoss08/trenova/internal/pkg/formula/infrastructure"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/emoss08/trenova/internal/pkg/formula/services"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/emoss08/trenova/internal/pkg/formula/variables/builtin"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

//go:embed schema/definitions/*.json
var schemaDefinitions embed.FS

// * VariableRegistryParams contains dependencies for variable registry creation
type VariableRegistryParams struct {
	fx.In
	Logger *logger.Logger
}

// * newVariableRegistry creates and initializes a new variable registry with builtin variables
func newVariableRegistry(p VariableRegistryParams) *variables.Registry {
	log := p.Logger.With().
		Str("module", "formula").
		Str("component", "variable_registry").
		Logger()

	registry := variables.NewRegistry()
	builtin.RegisterAll(registry)

	// * Log registered variables
	varNames := registry.ListNames()
	log.Info().
		Int("total_variables", len(varNames)).
		Strs("variables", varNames).
		Msg("formula variable registry initialized")

	return registry
}

// * DataResolverParams contains dependencies for data resolver creation
type DataResolverParams struct {
	fx.In
	Logger *logger.Logger
}

// * newDataResolver creates a data resolver with registered computers
func newDataResolver(p DataResolverParams) *schema.DefaultDataResolver {
	log := p.Logger.With().
		Str("module", "formula").
		Str("component", "data_resolver").
		Logger()

	resolver := schema.NewDefaultDataResolver()
	schema.RegisterShipmentComputers(resolver)

	log.Info().
		Strs("computers", []string{
			"computeTemperatureDifferential",
			"computeHasHazmat",
			"computeRequiresTemperatureControl",
			"computeTotalStops",
		}).
		Msg("formula data resolver initialized with shipment computers")

	return resolver
}

// * SchemaRegistryParams contains dependencies for schema registry creation
type SchemaRegistryParams struct {
	fx.In
	Logger *logger.Logger
}

// * newSchemaRegistryWithSchemas creates a schema registry with preloaded schemas
func newSchemaRegistryWithSchemas(p SchemaRegistryParams) (*schema.SchemaRegistry, error) {
	log := p.Logger.With().
		Str("module", "formula").
		Str("component", "schema_registry").
		Logger()

	registry := schema.NewSchemaRegistry()

	// * Load shipment schema
	log.Info().Msg("loading formula schemas...")

	schemaJSON, err := schemaDefinitions.ReadFile("schema/definitions/shipment.json")
	if err != nil {
		return nil, fmt.Errorf("failed to load shipment schema: %w", err)
	}

	if err := registry.RegisterSchema("shipment", schemaJSON); err != nil {
		return nil, fmt.Errorf("failed to register shipment schema: %w", err)
	}

	log.Info().
		Str("schema", "shipment").
		Msg("successfully registered schema")

	// * Log summary
	schemas := registry.ListSchemas()
	log.Info().
		Int("total_schemas", len(schemas)).
		Msg("formula schema registry initialized")

	return registry, nil
}

// * initializeFormulaSystem connects all components and registers variables
func initializeFormulaSystem(
	varRegistry *variables.Registry,
	schemaRegistry *schema.SchemaRegistry,
	resolver *schema.DefaultDataResolver,
	logger *logger.Logger,
) error {
	log := logger.With().
		Str("module", "formula").
		Str("component", "system_initializer").
		Logger()

	// * Create bridge and register schema variables
	bridge := NewSchemaVariableBridge(schemaRegistry, varRegistry)
	
	if err := bridge.RegisterSchemaVariables("shipment"); err != nil {
		return fmt.Errorf("failed to register shipment variables: %w", err)
	}

	log.Info().
		Str("schema", "shipment").
		Msg("successfully registered schema variables")

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

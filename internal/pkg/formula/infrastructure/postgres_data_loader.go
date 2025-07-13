package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/formula/ports"
	"github.com/emoss08/trenova/internal/pkg/formula/schema"
	"github.com/uptrace/bun"
)

// PostgresDataLoader implements DataLoader using PostgreSQL database
type PostgresDataLoader struct {
	conn           db.Connection
	schemaRegistry *schema.SchemaRegistry
}

// NewPostgresDataLoader creates a new PostgreSQL data loader
func NewPostgresDataLoader(
	conn db.Connection,
	schemaRegistry *schema.SchemaRegistry,
) *PostgresDataLoader {
	return &PostgresDataLoader{
		conn:           conn,
		schemaRegistry: schemaRegistry,
	}
}

// LoadEntity loads an entity with all schema-defined preloads
func (l *PostgresDataLoader) LoadEntity(
	ctx context.Context,
	schemaID string,
	entityID string,
) (any, error) {
	// Get schema definition
	schemaDef, err := l.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", schemaID)
	}

	// Get database connection
	dba, err := l.conn.ReadDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Build and execute query based on schema
	entity, err := l.buildAndExecuteQuery(ctx, dba, schemaDef, entityID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load entity: %w", err)
	}

	return entity, nil
}

// LoadEntityWithRequirements loads only required fields
func (l *PostgresDataLoader) LoadEntityWithRequirements(
	ctx context.Context,
	schemaID string,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	// Get schema definition
	schemaDef, err := l.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", schemaID)
	}

	// Get database connection
	dba, err := l.conn.ReadDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Build and execute optimized query
	entity, err := l.buildAndExecuteQuery(ctx, dba, schemaDef, entityID, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to load entity: %w", err)
	}

	return entity, nil
}

// buildAndExecuteQuery builds a query based on schema and requirements
func (l *PostgresDataLoader) buildAndExecuteQuery(
	ctx context.Context,
	db *bun.DB,
	schemaDef *schema.SchemaDefinition,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	// Determine table name from schema
	tableName := schemaDef.DataSource.Table
	if tableName == "" {
		return nil, fmt.Errorf("no table specified in schema")
	}

	// Create a map to hold the result
	result := make(map[string]any)

	// Build base query
	query := db.NewSelect().
		Table(tableName).
		Where("id = ?", entityID)

	// If requirements specified, select only needed fields
	if requirements != nil && len(requirements.Fields) > 0 {
		// Map field names to database columns
		columns := l.mapFieldsToColumns(schemaDef, requirements.Fields)
		if len(columns) > 0 {
			query = query.Column(columns...)
		}
	}

	// Add preloads
	preloads := l.determinePreloads(schemaDef, requirements)
	for _, preload := range preloads {
		query = query.Relation(preload)
	}

	// Execute query
	err := query.Scan(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Transform database column names to schema field names
	transformedResult := l.transformColumnNamesToFieldNames(schemaDef, result)

	return transformedResult, nil
}

// mapFieldsToColumns maps schema field names to database columns
func (l *PostgresDataLoader) mapFieldsToColumns(
	schemaDef *schema.SchemaDefinition,
	fields []string,
) []string {
	columns := make([]string, 0, len(fields))

	for _, field := range fields {
		if fieldSource, ok := schemaDef.FieldSources[field]; ok {
			// Use the database field name from schema
			if fieldSource.Field != "" {
				columns = append(columns, fieldSource.Field)
			}
		}
	}

	// Always include ID
	columns = append(columns, "id")

	return columns
}

// determinePreloads determines what relations need to be preloaded
func (l *PostgresDataLoader) determinePreloads(
	schemaDef *schema.SchemaDefinition,
	requirements *ports.DataRequirements,
) []string {
	preloadMap := make(map[string]bool)

	// Add schema-defined preloads if no specific requirements
	if requirements == nil {
		for _, preload := range schemaDef.DataSource.Preload {
			preloadMap[preload] = true
		}
	} else {
		// Add requirement-specific preloads
		for _, preload := range requirements.Preloads {
			preloadMap[preload] = true
		}

		// Check if any fields require preloads
		for _, field := range requirements.Fields {
			if strings.Contains(field, ".") {
				// Extract relation name
				parts := strings.Split(field, ".")
				if len(parts) > 0 {
					preloadMap[parts[0]] = true
				}
			}
		}

		// Check computed fields for required preloads
		for _, computedField := range requirements.ComputedFields {
			if fieldSource, ok := schemaDef.FieldSources[computedField]; ok {
				if fieldSource.Computed && len(fieldSource.Requires) > 0 {
					// Add any preloads mentioned in requires
					for _, req := range fieldSource.Requires {
						if strings.Contains(req, ".") {
							parts := strings.Split(req, ".")
							if len(parts) > 0 {
								preloadMap[parts[0]] = true
							}
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	preloads := make([]string, 0, len(preloadMap))
	for preload := range preloadMap {
		preloads = append(preloads, preload)
	}

	return preloads
}

// transformColumnNamesToFieldNames transforms database column names to schema field names
func (l *PostgresDataLoader) transformColumnNamesToFieldNames(
	schemaDef *schema.SchemaDefinition,
	result map[string]any,
) map[string]any {
	transformed := make(map[string]any)

	// Create reverse mapping from database column to schema field name
	columnToField := make(map[string]string)
	for fieldName, fieldSource := range schemaDef.FieldSources {
		if fieldSource.Field != "" {
			columnToField[fieldSource.Field] = fieldName
		}
	}

	// Transform the result map
	for column, value := range result {
		if fieldName, exists := columnToField[column]; exists {
			transformed[fieldName] = value
		} else {
			// Keep the original column name if no mapping found
			transformed[column] = value
		}
	}

	return transformed
}

// LoadShipmentEntity is a specific implementation for loading shipments
// This demonstrates how to handle a specific entity type with proper struct
func (l *PostgresDataLoader) LoadShipmentEntity(
	ctx context.Context,
	shipmentID string,
) (any, error) {
	// Get database connection
	dba, err := l.conn.ReadDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Create a map to hold the shipment data
	// In a real implementation, you would use the actual Shipment struct
	shipment := make(map[string]any)

	// Build query with common preloads for formulas
	err = dba.NewSelect().
		Table("shipments").
		Where("id = ?", shipmentID).
		Relation("Customer").
		Relation("TractorType").
		Relation("TrailerType").
		Relation("Commodities.Commodity.HazardousMaterial").
		Scan(ctx, &shipment)

	if err != nil {
		return nil, fmt.Errorf("failed to load shipment: %w", err)
	}

	// Get shipment schema and transform column names
	schemaDef, err := l.schemaRegistry.GetSchema("shipment")
	if err == nil {
		shipment = l.transformColumnNamesToFieldNames(schemaDef, shipment)
	}

	return shipment, nil
}

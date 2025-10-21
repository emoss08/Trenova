package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/formula/ports"
	"github.com/emoss08/trenova/pkg/formula/schema"
	"github.com/uptrace/bun"
)

type PostgresDataLoader struct {
	conn           *postgres.Connection
	schemaRegistry *schema.Registry
}

func NewPostgresDataLoader(
	conn *postgres.Connection,
	schemaRegistry *schema.Registry,
) *PostgresDataLoader {
	return &PostgresDataLoader{
		conn:           conn,
		schemaRegistry: schemaRegistry,
	}
}

func (l *PostgresDataLoader) LoadEntity(
	ctx context.Context,
	schemaID string,
	entityID string,
) (any, error) {
	schemaDef, err := l.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", schemaID)
	}

	db, err := l.conn.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	entity, err := l.buildAndExecuteQuery(ctx, db, schemaDef, entityID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load entity: %w", err)
	}

	return entity, nil
}

func (l *PostgresDataLoader) LoadEntityWithRequirements(
	ctx context.Context,
	schemaID string,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	schemaDef, err := l.schemaRegistry.GetSchema(schemaID)
	if err != nil {
		return nil, fmt.Errorf("schema not found: %s", schemaID)
	}

	db, err := l.conn.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	entity, err := l.buildAndExecuteQuery(ctx, db, schemaDef, entityID, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to load entity: %w", err)
	}

	return entity, nil
}

func (l *PostgresDataLoader) buildAndExecuteQuery(
	ctx context.Context,
	db *bun.DB,
	schemaDef *schema.Definition,
	entityID string,
	requirements *ports.DataRequirements,
) (any, error) {
	tableName := schemaDef.DataSource.Table
	if tableName == "" {
		return nil, errors.New("no table specified in schema")
	}

	result := make(map[string]any)

	query := db.NewSelect().
		Table(tableName).
		Where("id = ?", entityID)

	if requirements != nil && len(requirements.Fields) > 0 {
		columns := l.mapFieldsToColumns(schemaDef, requirements.Fields)
		if len(columns) > 0 {
			query = query.Column(columns...)
		}
	}

	preloads := l.determinePreloads(schemaDef, requirements)
	for _, preload := range preloads {
		query = query.Relation(preload)
	}

	err := query.Scan(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	transformedResult := l.transformColumnNamesToFieldNames(schemaDef, result)

	return transformedResult, nil
}

func (l *PostgresDataLoader) mapFieldsToColumns(
	schemaDef *schema.Definition,
	fields []string,
) []string {
	columns := make([]string, 0, len(fields))

	for _, field := range fields {
		if fieldSource, ok := schemaDef.FieldSources[field]; ok {
			if fieldSource.Field != "" {
				columns = append(columns, fieldSource.Field)
			}
		}
	}

	columns = append(columns, "id")

	return columns
}

func (l *PostgresDataLoader) determinePreloads( //nolint:gocognit // this is a helper function
	schemaDef *schema.Definition,
	requirements *ports.DataRequirements,
) []string {
	preloadMap := make(map[string]bool)

	if requirements == nil { //nolint:nestif // this is a helper function
		for _, preload := range schemaDef.DataSource.Preload {
			preloadMap[preload] = true
		}
	} else {
		for _, preload := range requirements.Preloads {
			preloadMap[preload] = true
		}

		for _, field := range requirements.Fields {
			if strings.Contains(field, ".") {
				parts := strings.Split(field, ".")
				if len(parts) > 0 {
					preloadMap[parts[0]] = true
				}
			}
		}

		for _, computedField := range requirements.ComputedFields {
			if fieldSource, ok := schemaDef.FieldSources[computedField]; ok {
				if fieldSource.Computed && len(fieldSource.Requires) > 0 {
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

	preloads := make([]string, 0, len(preloadMap))
	for preload := range preloadMap {
		preloads = append(preloads, preload)
	}

	return preloads
}

func (l *PostgresDataLoader) transformColumnNamesToFieldNames(
	schemaDef *schema.Definition,
	result map[string]any,
) map[string]any {
	transformed := make(map[string]any)

	columnToField := make(map[string]string)
	for fieldName, fieldSource := range schemaDef.FieldSources {
		if fieldSource.Field != "" {
			columnToField[fieldSource.Field] = fieldName
		}
	}

	for column, value := range result {
		if fieldName, exists := columnToField[column]; exists {
			transformed[fieldName] = value
		} else {
			transformed[column] = value
		}
	}

	return transformed
}

func (l *PostgresDataLoader) LoadShipmentEntity(
	ctx context.Context,
	shipmentID string,
) (any, error) {
	db, err := l.conn.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	shipment := make(map[string]any)

	err = db.NewSelect().
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

	schemaDef, err := l.schemaRegistry.GetSchema("shipment")
	if err == nil {
		shipment = l.transformColumnNamesToFieldNames(schemaDef, shipment)
	}

	return shipment, nil
}

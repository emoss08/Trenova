/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package ports

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/formula/schema"
)

// DataLoader is a port for loading data required by formulas
// This follows hexagonal architecture - the port defines the interface,
// implementations (adapters) will handle actual database access
type DataLoader interface {
	// LoadEntity loads an entity by ID with schema-specified preloads
	LoadEntity(ctx context.Context, schemaID string, entityID string) (any, error)

	// LoadEntityWithRequirements loads only the fields/relations needed
	LoadEntityWithRequirements(
		ctx context.Context,
		schemaID string,
		entityID string,
		requirements *DataRequirements,
	) (any, error)
}

// DataRequirements specifies what data needs to be loaded
type DataRequirements struct {
	// Fields that need to be loaded
	Fields []string

	// Relations that need to be preloaded
	Preloads []string

	// Computed fields that are needed
	ComputedFields []string
}

// FormulaDataContext provides data loading capabilities to the formula system
type FormulaDataContext interface {
	// GetSchema returns the schema definition for an entity type
	GetSchema(schemaID string) (*schema.SchemaDefinition, error)

	// LoadData loads the required data for formula evaluation
	LoadData(ctx context.Context, schemaID string, entityID string) (any, error)

	// AnalyzeRequirements analyzes an expression to determine data needs
	AnalyzeRequirements(expression string) (*DataRequirements, error)
}

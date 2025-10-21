package ports

import (
	"context"

	"github.com/emoss08/trenova/pkg/formula/schema"
)

type DataLoader interface {
	LoadEntity(ctx context.Context, schemaID string, entityID string) (any, error)
	LoadEntityWithRequirements(
		ctx context.Context,
		schemaID string,
		entityID string,
		requirements *DataRequirements,
	) (any, error)
}

type DataRequirements struct {
	Fields         []string
	Preloads       []string
	ComputedFields []string
}

type FormulaDataContext interface {
	GetSchema(schemaID string) (*schema.Definition, error)
	LoadData(ctx context.Context, schemaID string, entityID string) (any, error)
	AnalyzeRequirements(expression string) (*DataRequirements, error)
}

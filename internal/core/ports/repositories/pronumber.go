package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ProNumberRepository interface {
	// GetNextProNumber generates the next PRO number for an organization
	GetNextProNumber(ctx context.Context, orgID pulid.ID) (string, error)

	// GetNextProNumberWithBusinessUnit generates the next PRO number for a specific business unit
	GetNextProNumberWithBusinessUnit(ctx context.Context, orgID, businessUnitID pulid.ID) (string, error)

	// GetNextProNumberBatch generates a batch of sequential PRO numbers
	GetNextProNumberBatch(ctx context.Context, orgID pulid.ID, count int) ([]string, error)

	// GetNextProNumberBatchWithBusinessUnit generates a batch of sequential PRO numbers for a specific business unit
	GetNextProNumberBatchWithBusinessUnit(ctx context.Context, orgID, businessUnitID pulid.ID, count int) ([]string, error)
}

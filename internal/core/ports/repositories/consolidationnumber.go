package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ConsolidationNumberRepository interface {
	// GetNextConsolidationNumber generates the next consolidation number for an organization
	GetNextConsolidationNumber(ctx context.Context, orgID pulid.ID) (string, error)

	// GetNextConsolidationNumberWithBusinessUnit generates the next consolidation number for a specific business unit
	GetNextConsolidationNumberWithBusinessUnit(
		ctx context.Context,
		orgID, businessUnitID pulid.ID,
	) (string, error)

	// GetNextConsolidationNumberBatch generates a batch of sequential consolidation numbers
	GetNextConsolidationNumberBatch(
		ctx context.Context,
		orgID pulid.ID,
		count int,
	) ([]string, error)

	// GetNextConsolidationNumberBatchWithBusinessUnit generates a batch of sequential consolidation numbers for a specific business unit
	GetNextConsolidationNumberBatchWithBusinessUnit(
		ctx context.Context,
		orgID, businessUnitID pulid.ID,
		count int,
	) ([]string, error)
}

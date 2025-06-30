package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

// ConsolidationRepository defines operations for consolidation management
type ConsolidationRepository interface {
	// GetNextConsolidationNumber generates the next consolidation number
	GetNextConsolidationNumber(ctx context.Context, orgID, buID pulid.ID) (string, error)

	// GetNextConsolidationNumberBatch generates a batch of consolidation numbers
	GetNextConsolidationNumberBatch(
		ctx context.Context,
		orgID, buID pulid.ID,
		count int,
	) ([]string, error)
}

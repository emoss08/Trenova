package detentionpreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
)

type DetentionDisputer interface {
	Dispute(
		ctx context.Context,
		params detentionservice.DisputeParams,
	) (*detention.DetentionOccurrence, error)
	PreviewDispute(
		ctx context.Context,
		params *detentionservice.DisputeParams,
	) (*detentionservice.OccurrenceChange, error)
}

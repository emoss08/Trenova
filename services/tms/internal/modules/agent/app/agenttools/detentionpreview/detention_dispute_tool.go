package detentionpreview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
)

type DetentionDisputer interface {
	Dispute(
		ctx context.Context,
		params detentionreview.DisputeParams,
	) (*detention.DetentionOccurrence, error)
	PreviewDispute(
		ctx context.Context,
		params *detentionreview.DisputeParams,
	) (*detentionreview.OccurrenceChange, error)
}

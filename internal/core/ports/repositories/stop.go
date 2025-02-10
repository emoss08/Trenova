package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type StopRepository interface {
	BulkInsert(ctx context.Context, stops []*shipment.Stop) ([]*shipment.Stop, error)
}

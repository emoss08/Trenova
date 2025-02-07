package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type AssignmentRepository interface {
	SingleAssign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error)
}

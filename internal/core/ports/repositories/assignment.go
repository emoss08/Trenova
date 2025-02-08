package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type GetAssignmentByIDOptions struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
}

type AssignmentRepository interface {
	SingleAssign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error)
	Reassign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error)
	GetByID(ctx context.Context, opts GetAssignmentByIDOptions) (*shipment.Assignment, error)
}

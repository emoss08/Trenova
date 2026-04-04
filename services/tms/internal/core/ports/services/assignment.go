package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type AssignmentService interface {
	List(
		ctx context.Context,
		req *repositories.ListAssignmentsRequest,
	) (*pagination.ListResult[*shipment.Assignment], error)
	Get(
		ctx context.Context,
		req *repositories.GetAssignmentByIDRequest,
	) (*shipment.Assignment, error)
	AssignToMove(
		ctx context.Context,
		req *repositories.AssignShipmentMoveRequest,
	) (*shipment.Assignment, error)
	Reassign(
		ctx context.Context,
		req *repositories.ReassignShipmentMoveRequest,
	) (*shipment.Assignment, error)
	Unassign(
		ctx context.Context,
		req *repositories.UnassignShipmentMoveRequest,
	) error
	CheckWorkerCompliance(
		ctx context.Context,
		req *repositories.CheckWorkerComplianceRequest,
	) error
}

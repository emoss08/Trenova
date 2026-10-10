package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AssignmentPlan struct {
	Assignment     *shipment.Assignment
	ShipmentBefore *shipment.Shipment
	ShipmentAfter  *shipment.Shipment
	// EquipmentInUse is the tractor or trailer the assignment names that is on
	// another move still in progress. Assigning it ahead is allowed; starting
	// this move with it is refused until that one is done.
	EquipmentInUse []EquipmentInUse
}

type EquipmentInUse struct {
	Kind           string
	EquipmentID    pulid.ID
	ShipmentMoveID pulid.ID
	ProNumber      string
}

type AssignmentService interface {
	List(
		ctx context.Context,
		req *ListAssignmentsRequest,
	) (*pagination.ListResult[*shipment.Assignment], error)
	Get(
		ctx context.Context,
		req *GetAssignmentByIDRequest,
	) (*shipment.Assignment, error)
	AssignToMove(
		ctx context.Context,
		req *AssignShipmentMoveRequest,
	) (*shipment.Assignment, error)
	PreviewAssignToMove(
		ctx context.Context,
		req *AssignShipmentMoveRequest,
	) (*AssignmentPlan, error)
	Reassign(
		ctx context.Context,
		req *ReassignShipmentMoveRequest,
	) (*shipment.Assignment, error)
	Unassign(
		ctx context.Context,
		req *UnassignShipmentMoveRequest,
	) error
	PreviewUnassign(
		ctx context.Context,
		req *UnassignShipmentMoveRequest,
	) (*AssignmentPlan, error)
	CheckWorkerCompliance(
		ctx context.Context,
		req *CheckWorkerComplianceRequest,
	) error
}

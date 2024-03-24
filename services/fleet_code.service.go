package services

import (
	"context"

	"github.com/emoss08/trenova/ent/fleetcode"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type FleetCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewFleetCodeOps creates a new fleet code service.
func NewFleetCodeOps(ctx context.Context) *FleetCodeOps {
	return &FleetCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetFleetCodes gets the fleet code for an organization.
func (r *FleetCodeOps) GetFleetCodes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.FleetCode, int, error) {
	fleetCodeCount, countErr := r.client.FleetCode.Query().Where(
		fleetcode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	fleetCodes, err := r.client.FleetCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			fleetcode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return fleetCodes, fleetCodeCount, nil
}

// CreateFleetCode creates a new accessorial charge.
func (r *FleetCodeOps) CreateFleetCode(newFleetCode ent.FleetCode) (*ent.FleetCode, error) {
	fleetCode, err := r.client.FleetCode.Create().
		SetOrganizationID(newFleetCode.OrganizationID).
		SetBusinessUnitID(newFleetCode.BusinessUnitID).
		SetStatus(newFleetCode.Status).
		SetCode(newFleetCode.Code).
		SetDescription(newFleetCode.Description).
		SetRevenueGoal(newFleetCode.RevenueGoal).
		SetDeadheadGoal(newFleetCode.DeadheadGoal).
		SetMileageGoal(newFleetCode.MileageGoal).
		SetNillableManagerID(newFleetCode.ManagerID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return fleetCode, nil
}

// UpdateFleetCode updates a fleet code.
func (r *FleetCodeOps) UpdateFleetCode(fleetCode ent.FleetCode) (*ent.FleetCode, error) {
	// Start building the update operation
	updateOp := r.client.FleetCode.UpdateOneID(fleetCode.ID).
		SetStatus(fleetCode.Status).
		SetCode(fleetCode.Code).
		SetDescription(fleetCode.Description).
		SetRevenueGoal(fleetCode.RevenueGoal).
		SetDeadheadGoal(fleetCode.DeadheadGoal).
		SetMileageGoal(fleetCode.MileageGoal).
		SetNillableManagerID(fleetCode.ManagerID)

	// If the manager ID is nil, clear the association.
	if fleetCode.ManagerID == nil {
		updateOp = updateOp.ClearManager()
	}

	// Execute the update operation
	updatedFleetCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedFleetCode, nil
}

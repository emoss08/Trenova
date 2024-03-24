package services

import (
	"context"

	"github.com/emoss08/trenova/ent/accessorialcharge"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type AccessorialChargeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewAccessorialChargeOps creates a new accessorial charge service.
func NewAccessorialChargeOps(ctx context.Context) *AccessorialChargeOps {
	return &AccessorialChargeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetAccessorialCharges gets the accessorial charges for an organization.
func (r *AccessorialChargeOps) GetAccessorialCharges(limit, offset int, orgID, buID uuid.UUID) ([]*ent.AccessorialCharge, int, error) {
	accessorialChargeCount, countErr := r.client.AccessorialCharge.Query().Where(
		accessorialcharge.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	accessorialCharges, err := r.client.AccessorialCharge.Query().
		Limit(limit).
		Offset(offset).
		Where(
			accessorialcharge.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return accessorialCharges, accessorialChargeCount, nil
}

// CreateAccessorialCharge creates a new accessorial charge.
func (r *AccessorialChargeOps) CreateAccessorialCharge(newAccessorialCharge ent.AccessorialCharge) (*ent.AccessorialCharge, error) {
	accessorialCharge, err := r.client.AccessorialCharge.Create().
		SetOrganizationID(newAccessorialCharge.OrganizationID).
		SetBusinessUnitID(newAccessorialCharge.BusinessUnitID).
		SetStatus(newAccessorialCharge.Status).
		SetCode(newAccessorialCharge.Code).
		SetDescription(newAccessorialCharge.Description).
		SetIsDetention(newAccessorialCharge.IsDetention).
		SetMethod(newAccessorialCharge.Method).
		SetAmount(newAccessorialCharge.Amount).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return accessorialCharge, nil
}

// UpdateAccessorialCharge updates a accessorial charge.
func (r *AccessorialChargeOps) UpdateAccessorialCharge(accessorialCharge ent.AccessorialCharge) (*ent.AccessorialCharge, error) {
	// Start building the update operation
	updateOp := r.client.AccessorialCharge.UpdateOneID(accessorialCharge.ID).
		SetStatus(accessorialCharge.Status).
		SetCode(accessorialCharge.Code).
		SetDescription(accessorialCharge.Description).
		SetIsDetention(accessorialCharge.IsDetention).
		SetMethod(accessorialCharge.Method).
		SetAmount(accessorialCharge.Amount)

	// Execute the update operation
	updatedAccessorialCharge, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedAccessorialCharge, nil
}

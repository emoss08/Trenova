package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/chargetype"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type ChargeTypeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewChargeTypeOps creates a new commodity service.
func NewChargeTypeOps(ctx context.Context) *ChargeTypeOps {
	return &ChargeTypeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetChargeTypes gets the charge types for an organization.
func (r *ChargeTypeOps) GetChargeTypes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.ChargeType, int, error) {
	chargeTypeCount, countErr := r.client.ChargeType.Query().Where(
		chargetype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	chargeTypes, err := r.client.ChargeType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			chargetype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return chargeTypes, chargeTypeCount, nil
}

// CreateChargeType creates a new charge type.
func (r *ChargeTypeOps) CreateChargeType(newChargeType ent.ChargeType) (*ent.ChargeType, error) {
	chargeType, err := r.client.ChargeType.Create().
		SetOrganizationID(newChargeType.OrganizationID).
		SetBusinessUnitID(newChargeType.BusinessUnitID).
		SetStatus(newChargeType.Status).
		SetName(newChargeType.Name).
		SetDescription(newChargeType.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return chargeType, nil
}

// UpdateChargeType updates a charge type.
func (r *ChargeTypeOps) UpdateChargeType(chargeType ent.ChargeType) (*ent.ChargeType, error) {
	// Start building the update operation
	updateOp := r.client.ChargeType.UpdateOneID(chargeType.ID).
		SetStatus(chargeType.Status).
		SetName(chargeType.Name).
		SetDescription(chargeType.Description)

	// Execute the update operation
	updatedChargeType, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedChargeType, nil
}

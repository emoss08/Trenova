package services

import (
	"context"

	"github.com/emoss08/trenova/ent/delaycode"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type DelayCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewDelayCodeOps creates a new delay code service.
func NewDelayCodeOps(ctx context.Context) *DelayCodeOps {
	return &DelayCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetDelayCodes gets the delay code for an organization.
func (r *DelayCodeOps) GetDelayCodes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.DelayCode, int, error) {
	delayCodeCount, countErr := r.client.DelayCode.Query().Where(
		delaycode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	delayCodes, err := r.client.DelayCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			delaycode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return delayCodes, delayCodeCount, nil
}

// CreateDelayCode creates a new delay code.
func (r *DelayCodeOps) CreateDelayCode(newDelayCode ent.DelayCode) (*ent.DelayCode, error) {
	delayCode, err := r.client.DelayCode.Create().
		SetOrganizationID(newDelayCode.OrganizationID).
		SetBusinessUnitID(newDelayCode.BusinessUnitID).
		SetStatus(newDelayCode.Status).
		SetCode(newDelayCode.Code).
		SetDescription(newDelayCode.Description).
		SetFCarrierOrDriver(newDelayCode.FCarrierOrDriver).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return delayCode, nil
}

// UpdateDelayCode updates a delay code.
func (r *DelayCodeOps) UpdateDelayCode(delayCode ent.DelayCode) (*ent.DelayCode, error) {
	// Start building the update operation
	updateOp := r.client.DelayCode.UpdateOneID(delayCode.ID).
		SetStatus(delayCode.Status).
		SetCode(delayCode.Code).
		SetDescription(delayCode.Description).
		SetFCarrierOrDriver(delayCode.FCarrierOrDriver)

	// Execute the update operation
	updatedDelayCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedDelayCode, nil
}

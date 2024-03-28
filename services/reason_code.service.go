package services

import (
	"context"

	"github.com/emoss08/trenova/ent/reasoncode"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type ReasonCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewReasonCodeOps creates a new reason code service.
func NewReasonCodeOps(ctx context.Context) *ReasonCodeOps {
	return &ReasonCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetReasonCode gets the reason code for an organization.
func (r *ReasonCodeOps) GetReasonCode(limit, offset int, orgID, buID uuid.UUID) ([]*ent.ReasonCode, int, error) {
	reasonCodeCount, countErr := r.client.ReasonCode.Query().Where(
		reasoncode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	reasonCodes, err := r.client.ReasonCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			reasoncode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return reasonCodes, reasonCodeCount, nil
}

// CreateReasonCode creates a new reason code.
func (r *ReasonCodeOps) CreateReasonCode(newReasonCode ent.ReasonCode) (*ent.ReasonCode, error) {
	reasonCode, err := r.client.ReasonCode.Create().
		SetOrganizationID(newReasonCode.OrganizationID).
		SetBusinessUnitID(newReasonCode.BusinessUnitID).
		SetStatus(newReasonCode.Status).
		SetCode(newReasonCode.Code).
		SetCodeType(newReasonCode.CodeType).
		SetDescription(newReasonCode.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return reasonCode, nil
}

// UpdateReasonCode updates a reason code.
func (r *ReasonCodeOps) UpdateReasonCode(reasonCode ent.ReasonCode) (*ent.ReasonCode, error) {
	// Start building the update operation
	updateOp := r.client.ReasonCode.UpdateOneID(reasonCode.ID).
		SetStatus(reasonCode.Status).
		SetCode(reasonCode.Code).
		SetCodeType(reasonCode.CodeType).
		SetDescription(reasonCode.Description)

	// Execute the update operation
	updateReasonCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateReasonCode, nil
}

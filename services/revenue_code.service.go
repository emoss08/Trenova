package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/revenuecode"
	"github.com/google/uuid"
)

// RevenueCodeOps is the service for revenue code.
type RevenueCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewRevenueCodeOps creates a new revenue code service.
func NewRevenueCodeOps(ctx context.Context) *RevenueCodeOps {
	return &RevenueCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetRevenueCodes gets the revenue codes for an organization.
func (r *RevenueCodeOps) GetRevenueCodes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.RevenueCode, int, error) {
	revenueCodeCount, countErr := r.client.RevenueCode.Query().Where(
		revenuecode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	revenueCodes, err := r.client.RevenueCode.Query().
		Limit(limit).
		Offset(offset).
		WithExpenseAccount().
		WithRevenueAccount().
		Where(
			revenuecode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return revenueCodes, revenueCodeCount, nil
}

// CreateRevenueCode creates a new revenue code.
func (r *RevenueCodeOps) CreateRevenueCode(newRevenueCode ent.RevenueCode) (*ent.RevenueCode, error) {
	revenueCode, err := r.client.RevenueCode.Create().
		SetOrganizationID(newRevenueCode.OrganizationID).
		SetBusinessUnitID(newRevenueCode.BusinessUnitID).
		SetStatus(newRevenueCode.Status).
		SetCode(newRevenueCode.Code).
		SetDescription(newRevenueCode.Description).
		SetNillableExpenseAccountID(newRevenueCode.ExpenseAccountID).
		SetNillableRevenueAccountID(newRevenueCode.RevenueAccountID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return revenueCode, nil
}

// UpdateRevenueCode updates a revenue code.
func (r *RevenueCodeOps) UpdateRevenueCode(revenueCode ent.RevenueCode) (*ent.RevenueCode, error) {
	// Start building the update operation
	updateOp := r.client.RevenueCode.UpdateOneID(revenueCode.ID).
		SetStatus(revenueCode.Status).
		SetCode(revenueCode.Code).
		SetDescription(revenueCode.Description).
		SetNillableExpenseAccountID(revenueCode.ExpenseAccountID).
		SetNillableRevenueAccountID(revenueCode.RevenueAccountID)

	// If the expense account ID is nil, clear the association
	if revenueCode.ExpenseAccountID == nil {
		updateOp = updateOp.ClearExpenseAccount()
	}

	// If the revenue account ID is nil, clear the association
	if revenueCode.RevenueAccountID == nil {
		updateOp = updateOp.ClearRevenueAccount()
	}

	// Execute the update operation
	updatedRevenueCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedRevenueCode, nil
}

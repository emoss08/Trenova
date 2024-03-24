package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/divisioncode"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type DivisionCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewDivisionCodeOps creates a new division code service.
func NewDivisionCodeOps(ctx context.Context) *DivisionCodeOps {
	return &DivisionCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetDivisionCodes gets the division codes for an organization.
func (r *DivisionCodeOps) GetDivisionCodes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.DivisionCode, int, error) {
	divisionCodeCount, countErr := r.client.DivisionCode.Query().Where(
		divisioncode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	divisionCodes, err := r.client.DivisionCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			divisioncode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return divisionCodes, divisionCodeCount, nil
}

// CreateDivisionCode creates a new division code.
func (r *DivisionCodeOps) CreateDivisionCode(newDivisionCode ent.DivisionCode) (*ent.DivisionCode, error) {
	divisionCode, err := r.client.DivisionCode.Create().
		SetOrganizationID(newDivisionCode.OrganizationID).
		SetBusinessUnitID(newDivisionCode.BusinessUnitID).
		SetStatus(newDivisionCode.Status).
		SetCode(newDivisionCode.Code).
		SetDescription(newDivisionCode.Description).
		SetNillableApAccountID(newDivisionCode.ApAccountID).
		SetNillableCashAccountID(newDivisionCode.CashAccountID).
		SetNillableExpenseAccountID(newDivisionCode.ExpenseAccountID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return divisionCode, nil
}

// UpdateDivisionCode updates a divison code.
func (r *DivisionCodeOps) UpdateDivisionCode(divisionCode ent.DivisionCode) (*ent.DivisionCode, error) {
	// Start building the update operation
	updateOp := r.client.DivisionCode.UpdateOneID(divisionCode.ID).
		SetStatus(divisionCode.Status).
		SetCode(divisionCode.Code).
		SetDescription(divisionCode.Description).
		SetNillableApAccountID(divisionCode.ApAccountID).
		SetNillableCashAccountID(divisionCode.CashAccountID).
		SetNillableExpenseAccountID(divisionCode.ExpenseAccountID)

	// If the ap account ID is nil, clear the association
	if divisionCode.ApAccountID == nil {
		updateOp = updateOp.ClearApAccount()
	}

	// If the cash account ID is nil, clear the association
	if divisionCode.CashAccountID == nil {
		updateOp = updateOp.ClearCashAccount()
	}

	// If the expense account ID is nil, clear the association
	if divisionCode.ExpenseAccountID == nil {
		updateOp = updateOp.ClearExpenseAccount()
	}

	// Execute the update operation
	updatedDivisionCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedDivisionCode, nil
}

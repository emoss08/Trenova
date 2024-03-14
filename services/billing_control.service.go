package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/billingcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// BillingControlOps is the service for billing control settings
type BillingControlOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewBillingControlOps creates a new billing control service
func NewBillingControlOps(ctx context.Context) *BillingControlOps {
	return &BillingControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// CreateBillingControl creates a new billing control settings for an organization
func (r *BillingControlOps) GetBillingControlByOrgID(orgID uuid.UUID) (*ent.BillingControl, error) {
	billingControl, err := r.client.BillingControl.Query().Where(
		billingcontrol.HasOrganizationWith(
			organization.ID(orgID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return billingControl, nil
}

// UpdateBillingControl updates the billing control settings for an organization
func (r *BillingControlOps) UpdateBillingControl(bc ent.BillingControl) (*ent.BillingControl, error) {
	updatedBC, err := r.client.BillingControl.
		UpdateOneID(bc.ID).
		SetRemoveBillingHistory(bc.RemoveBillingHistory).
		SetAutoBillShipment(bc.AutoBillShipment).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetValidateCustomerRates(bc.ValidateCustomerRates).
		SetAutoBillCriteria(bc.AutoBillCriteria).
		SetShipmentTransferCriteria(bc.ShipmentTransferCriteria).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedBC, nil
}

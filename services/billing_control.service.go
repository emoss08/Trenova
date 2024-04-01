package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/billingcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

// BillingControlOps is the service for billing control settings.
type BillingControlOps struct {
	client *ent.Client
}

// NewBillingControlOps creates a new billing control service.
func NewBillingControlOps() *BillingControlOps {
	return &BillingControlOps{
		client: database.GetClient(),
	}
}

// GetBillingControl gets the billing control settings for an organization.
func (r *BillingControlOps) GetBillingControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.BillingControl, error) {
	billingControl, err := r.client.BillingControl.Query().Where(
		billingcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return billingControl, nil
}

// UpdateBillingControl updates the billing control settings for an organization.
func (r *BillingControlOps) UpdateBillingControl(ctx context.Context, bc ent.BillingControl) (*ent.BillingControl, error) {
	updatedBC, err := r.client.BillingControl.
		UpdateOneID(bc.ID).
		SetRemoveBillingHistory(bc.RemoveBillingHistory).
		SetAutoBillShipment(bc.AutoBillShipment).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetValidateCustomerRates(bc.ValidateCustomerRates).
		SetAutoBillCriteria(bc.AutoBillCriteria).
		SetShipmentTransferCriteria(bc.ShipmentTransferCriteria).
		SetEnforceCustomerBilling(bc.EnforceCustomerBilling).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedBC, nil
}

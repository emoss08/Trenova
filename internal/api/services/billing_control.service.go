package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/billingcontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

// BillingControlService is the service for accounting control settings.
type BillingControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewBillingControlService creates a new accessorial charge service.
func NewBillingControlService(s *api.Server) *BillingControlService {
	return &BillingControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetBillingControl gets the billing control settings for an organization.
func (r *BillingControlService) GetBillingControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.BillingControl, error) {
	billingControl, err := r.Client.BillingControl.Query().Where(
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
// TODO: Wrap this in the withTX function.
func (r *BillingControlService) UpdateBillingControl(ctx context.Context, bc *ent.BillingControl) (*ent.BillingControl, error) {
	updatedEntity := new(ent.BillingControl)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateBillingControlEntity(ctx, tx, bc)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *BillingControlService) updateBillingControlEntity(
	ctx context.Context, tx *ent.Tx, bc *ent.BillingControl,
) (*ent.BillingControl, error) {
	updateOp := tx.BillingControl.UpdateOneID(bc.ID).
		SetRemoveBillingHistory(bc.RemoveBillingHistory).
		SetAutoBillShipment(bc.AutoBillShipment).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetAutoMarkReadyToBill(bc.AutoMarkReadyToBill).
		SetValidateCustomerRates(bc.ValidateCustomerRates).
		SetAutoBillCriteria(bc.AutoBillCriteria).
		SetShipmentTransferCriteria(bc.ShipmentTransferCriteria).
		SetEnforceCustomerBilling(bc.EnforceCustomerBilling)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}

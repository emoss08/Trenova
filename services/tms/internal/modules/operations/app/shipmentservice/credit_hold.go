package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// refuseCreditHold stops new business for a customer whose credit is held:
// a new or duplicated shipment, or an existing one moved onto that payer.
// A shipment already billed to a held payer can still be worked, since the
// freight is booked and has to move and be billed regardless.
func (s *service) refuseCreditHold(
	ctx context.Context,
	entity *shipment.Shipment,
	original *shipment.Shipment,
) error {
	if s.customerRepo == nil || entity == nil {
		return nil
	}
	payerID := entity.PayerID()
	if payerID.IsNil() || (original != nil && original.PayerID() == payerID) {
		return nil
	}

	payer, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID: payerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeBillingProfile: true},
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if payer == nil || payer.BillingProfile == nil || !payer.BillingProfile.IsOnCreditHold() {
		return nil
	}

	message := payer.Name + " is on credit hold"
	if reason := payer.BillingProfile.CreditHoldReason; reason != "" {
		message += ": " + reason
	}
	message += ". Accounts receivable has to lift the hold before new shipments are booked for them."

	field := "customerId"
	if entity.HasExplicitBillTo() {
		field = "billToCustomerId"
	}
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

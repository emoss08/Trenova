package invoiceadjustmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicevoid"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func collectBatchItemIDs(items []*invoiceadjustment.InvoiceAdjustmentBatchItem) []pulid.ID {
	ids := make([]pulid.ID, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

// voidReversedInvoice retires the original once a full reversal has credited
// it. The invoice keeps its number and lines for the audit trail, its open
// balance drops to zero, and its freight is released according to the
// disposition recorded when the void was requested (DoNotRebill unless the
// requester asked otherwise).
func (s *Service) voidReversedInvoice(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	original *invoice.Invoice,
	actor *servicesports.RequestActor,
	now int64,
) error {
	if original.Status == invoice.StatusVoided {
		return nil
	}
	if !invoice.IsAllowedTransition(original.Status, invoice.StatusVoided) {
		return errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Invoice cannot be voided from its current status",
		)
	}

	disposition := original.VoidDisposition
	if !disposition.IsValid() {
		disposition = invoice.VoidDispositionDoNotRebill
	}
	reason := strings.TrimSpace(original.VoidReason)
	if reason == "" {
		reason = strings.TrimSpace(adjustment.Reason)
	}
	if reason == "" {
		reason = "Reversed in full by adjustment " + adjustment.ID.String()
	}

	original.Status = invoice.StatusVoided
	original.VoidedAt = &now
	original.VoidedByID = actor.UserID
	original.VoidReason = reason
	original.VoidDisposition = disposition
	original.VoidedByAdjustmentID = adjustment.ID
	updated, err := s.invoiceRepo.Update(ctx, original)
	if err != nil {
		return err
	}

	_, err = invoicevoid.Release(ctx, invoicevoid.Deps{
		BillingQueueRepo:     s.billingQueueRepo,
		OrderRepo:            s.orderRepo,
		ChargeAllocationRepo: s.chargeAllocRepo,
		ShipmentRepo:         s.shipmentRepo,
		InvoiceRepo:          s.invoiceRepo,
		OrderDerivation:      s.orderDerivation,
		DetentionBilling:     s.detentionBilling,
		Renumber: func(ctx context.Context, billType billingqueue.BillType) (string, error) {
			return s.renumberReleasedItem(ctx, adjustment, billType)
		},
	}, invoicevoid.Params{
		Invoice:     updated,
		Disposition: disposition,
		ActorUserID: actor.UserID,
		Reason:      reason,
		WasPosted:   true,
		Now:         now,
	})
	if err != nil {
		return err
	}
	*original = *updated

	return nil
}

func (s *Service) renumberReleasedItem(
	ctx context.Context,
	adjustment *invoiceadjustment.InvoiceAdjustment,
	billType billingqueue.BillType,
) (string, error) {
	switch billType {
	case billingqueue.BillTypeCreditMemo:
		return s.sequenceGenerator.GenerateCreditMemoNumber(
			ctx, adjustment.OrganizationID, adjustment.BusinessUnitID, "", "",
		)
	case billingqueue.BillTypeDebitMemo:
		return s.sequenceGenerator.GenerateDebitMemoNumber(
			ctx, adjustment.OrganizationID, adjustment.BusinessUnitID, "", "",
		)
	default:
		return s.sequenceGenerator.GenerateInvoiceNumber(
			ctx, adjustment.OrganizationID, adjustment.BusinessUnitID, "", "",
		)
	}
}

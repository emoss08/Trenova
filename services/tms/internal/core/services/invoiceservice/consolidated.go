package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// ConsolidatedInvoiceParams is a set of legs that should become one invoice
// regardless of which orders they came from.
//
// QueueItems are the approved billing-queue items the legs already sit on, one
// per leg. The invoice bills those items rather than minting new ones: the
// active-item unique index allows a single live pipeline per shipment, so a
// second item for freight that is already queued can never be inserted.
//
// RunID, when set, is the invoice run that proposed this group; the invoice
// carries it so a committed statement can be traced back to the proposal an
// operator actually approved.
//
// PayerID is the customer the invoice bills. Every queue item must belong to
// that payer; each leg contributes only that payer's share of its charges. When
// unset it is taken from the first queue item.
type ConsolidatedInvoiceParams struct {
	TenantInfo  pagination.TenantInfo
	Legs        []*shipment.Shipment
	QueueItems  []*billingqueue.BillingQueueItem
	PayerID     pulid.ID
	RunID       pulid.ID
	Number      string
	InvoiceDate int64
	PeriodStart int64
	PeriodEnd   int64
}

// CreateConsolidated bills a set of shipments spanning any number of orders on a
// single invoice.
//
// The first queue item is the anchor, exactly as the order path does — that is
// what keeps the invoice's single-valued queue-item FK and its idempotency lookup
// working without a join table.
func (s *Service) CreateConsolidated(
	ctx context.Context,
	params *ConsolidatedInvoiceParams,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	if params == nil || len(params.Legs) == 0 {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrRequired,
			"A consolidated invoice must bill at least one shipment",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}

	if err := validateConsolidatedQueueItems(params.Legs, params.QueueItems); err != nil {
		return nil, err
	}
	customerID := params.PayerID
	if customerID.IsNil() {
		customerID = params.QueueItems[0].BillToCustomerID
	}
	for _, item := range params.QueueItems {
		if item.BillToCustomerID != customerID {
			return nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"All shipments on one invoice must bill the same payer",
			)
		}
	}

	number := params.Number
	if number == "" {
		// Loaded before the transaction so the number honours the customer's own
		// prefix: one consolidated invoice stands for a month of freight, and a
		// customer who asked for their prefix expects to see it on it.
		cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         customerID,
			TenantInfo: params.TenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeBillingProfile: true,
			},
		})
		if err != nil {
			return nil, err
		}

		generated, err := s.generateInvoiceNumber(
			ctx,
			params.TenantInfo,
			billingProfileOf(cus),
		)
		if err != nil {
			return nil, err
		}
		number = generated
	}

	var created *invoice.Invoice
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		created, txErr = s.createConsolidatedTx(txCtx, params, number, customerID, actor)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) createConsolidatedTx(
	txCtx context.Context,
	params *ConsolidatedInvoiceParams,
	number string,
	customerID pulid.ID,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	anchor := params.QueueItems[0]

	if err := s.guardDetentionHolds(txCtx, params.TenantInfo, legIDs(params.Legs)); err != nil {
		return nil, err
	}

	if err := invoicelines.HydrateAccessorials(
		txCtx,
		s.accessorialRepo,
		params.TenantInfo,
		params.Legs...,
	); err != nil {
		return nil, err
	}

	cus, txErr := s.customerRepo.GetByID(txCtx, repositories.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: params.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if txErr != nil {
		return nil, txErr
	}

	control, txErr := s.billingRepo.GetByOrgID(txCtx, params.TenantInfo.OrgID)
	if txErr != nil && !errortypes.IsNotFoundError(txErr) {
		return nil, txErr
	}

	shares, isSplit, txErr := payerSharesForLegs(params.Legs, customerID)
	if txErr != nil {
		return nil, txErr
	}

	periodStart := params.PeriodStart
	periodEnd := params.PeriodEnd
	entity := s.buildInvoiceEntity(&buildInvoiceParams{
		Anchor:      anchor,
		Scope:       invoice.ScopeConsolidated,
		Customer:    cus,
		Control:     control,
		Legs:        params.Legs,
		RunID:       params.RunID,
		PeriodStart: &periodStart,
		PeriodEnd:   &periodEnd,
		InvoiceDate: params.InvoiceDate,
		Number:      number,
		Shares:      shares,
		IsSplitBill: isSplit,
	})
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalidOperation,
			"Nothing on this selection could be billed",
		)
	}

	if multiErr := s.validator.ValidateCreate(txCtx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, txErr := s.repo.Create(txCtx, entity)
	if txErr != nil {
		return nil, txErr
	}

	// Link every queue item this invoice bills, so the posting sweep and the
	// double-bill guard both work from one exact predicate rather than from the
	// order id, which a consolidated invoice does not have. The attach only takes
	// items that are still approved and unbilled, so an item another biller
	// invoiced after the caller read it rolls this invoice back instead of
	// billing the freight twice.
	itemIDs := make([]pulid.ID, 0, len(params.QueueItems))
	for _, item := range params.QueueItems {
		itemIDs = append(itemIDs, item.ID)
	}
	if txErr = s.attachQueueItems(txCtx, params.TenantInfo, created.ID, itemIDs); txErr != nil {
		return nil, txErr
	}

	if txErr = s.syncDetentionBilling(txCtx, created, actor); txErr != nil {
		return nil, txErr
	}

	auditActor := actor.AuditActor()
	s.logAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Consolidated invoice created",
	)
	s.publishInvalidation(txCtx, created, auditActor, "created", created)

	return created, nil
}

// payerSharesForLegs resolves what each leg owes the payer. A leg the payer has
// no share of cannot sit on their invoice, so it is refused here rather than
// silently billed at zero.
func payerSharesForLegs(
	legs []*shipment.Shipment,
	payerID pulid.ID,
) (map[pulid.ID]*shipment.PayerShare, bool, error) {
	shares := make(map[pulid.ID]*shipment.PayerShare, len(legs))
	isSplit := false
	for _, leg := range legs {
		if leg == nil {
			continue
		}
		resolution, err := shipment.ResolveShares(leg, leg.ChargeAllocations)
		if err != nil {
			return nil, false, err
		}
		share := resolution.ShareFor(payerID)
		if share == nil {
			return nil, false, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalidOperation,
				"Shipment {0} owes nothing to this payer",
				leg.ProNumber,
			)
		}
		shares[leg.ID] = share
		isSplit = isSplit || resolution.IsSplit
	}

	return shares, isSplit, nil
}

// validateConsolidatedQueueItems checks that the queue items are exactly the
// legs' live invoice pipelines: one per leg, approved, and not yet on an invoice.
func validateConsolidatedQueueItems(
	legs []*shipment.Shipment,
	items []*billingqueue.BillingQueueItem,
) error {
	if len(items) != len(legs) {
		return errortypes.NewValidationError(
			"billingQueueItemIds",
			errortypes.ErrInvalid,
			"Every shipment on a consolidated invoice must have exactly one billing queue item",
		)
	}

	legIDs := make(map[pulid.ID]struct{}, len(legs))
	for _, leg := range legs {
		legIDs[leg.ID] = struct{}{}
	}

	for _, item := range items {
		if item == nil {
			return errortypes.NewValidationError(
				"billingQueueItemIds",
				errortypes.ErrRequired,
				"Billing queue item is required",
			)
		}
		if _, ok := legIDs[item.ShipmentID]; !ok {
			return errortypes.NewValidationError(
				"billingQueueItemIds",
				errortypes.ErrInvalid,
				"Every billing queue item must belong to a shipment on this invoice",
			)
		}
		delete(legIDs, item.ShipmentID)

		if item.BillType != billingqueue.BillTypeInvoice || item.IsAdjustmentOrigin {
			return errortypes.NewValidationError(
				"billingQueueItemIds",
				errortypes.ErrInvalid,
				"Only original invoice billing queue items can be consolidated",
			)
		}
		if item.Status != billingqueue.StatusApproved || !item.InvoiceID.IsNil() {
			return errortypes.NewValidationError(
				"billingQueueItemIds",
				errortypes.ErrInvalidOperation,
				"Some shipments are no longer approved and unbilled",
			)
		}
	}

	return nil
}

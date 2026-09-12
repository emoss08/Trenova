package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

// ConsolidatedInvoiceParams is a set of legs that should become one invoice
// regardless of which orders they came from.
//
// RunID, when set, is the invoice run that proposed this group; the invoice
// carries it so a committed statement can be traced back to the proposal an
// operator actually approved.
type ConsolidatedInvoiceParams struct {
	TenantInfo  pagination.TenantInfo
	Legs        []*shipment.Shipment
	RunID       pulid.ID
	Number      string
	InvoiceDate int64
	PeriodStart int64
	PeriodEnd   int64
}

// CreateConsolidated bills a set of shipments spanning any number of orders on a
// single invoice.
//
// Every leg gets its own approved billing-queue item and the first is the anchor,
// exactly as the order path does — that is what keeps the invoice's single-valued
// queue-item FK and its idempotency lookup working without a join table.
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

	customerID := params.Legs[0].CustomerID
	for _, leg := range params.Legs {
		if leg.CustomerID != customerID {
			return nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"All shipments on one invoice must share the billing customer",
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
	queueItems, txErr := s.createLegQueueItems(txCtx, params.TenantInfo, params.Legs, number)
	if txErr != nil {
		return nil, txErr
	}
	anchor := queueItems.Anchor

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
	// order id, which a consolidated invoice does not have.
	if _, txErr = s.billingQueueRepo.AttachInvoice(txCtx, &repositories.AttachInvoiceRequest{
		TenantInfo: params.TenantInfo,
		InvoiceID:  created.ID,
		ItemIDs:    queueItems.ItemIDs,
	}); txErr != nil {
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

// createConsolidatedInvoiceFromLegs bills an ad-hoc selection, deriving the period
// it covers from the legs themselves.
func (s *Service) createConsolidatedInvoiceFromLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	legs []*shipment.Shipment,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	start, end := legServiceWindow(legs)

	return s.CreateConsolidated(ctx, &ConsolidatedInvoiceParams{
		TenantInfo:  tenantInfo,
		Legs:        legs,
		PeriodStart: start,
		PeriodEnd:   end,
	}, actor)
}

// legServiceWindow is the span the selection actually covers. A consolidated
// invoice must state a period, and for an ad-hoc selection the only honest one is
// the earliest to latest service date of what is on it.
func legServiceWindow(legs []*shipment.Shipment) (int64, int64) {
	var start, end int64
	for _, leg := range legs {
		date := serviceDateFromShipment(leg)
		if date == nil {
			continue
		}
		if start == 0 || *date < start {
			start = *date
		}
		if *date > end {
			end = *date
		}
	}

	if start == 0 {
		start = legs[0].CreatedAt
	}
	if end <= start {
		end = start + 1
	}

	return start, end
}

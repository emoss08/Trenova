package invoiceservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// CreateFromShipments bills the shipments and returns the primary payer's
// invoice. A split-billed shipment produces one invoice per payer; callers that
// need every invoice use CreateInvoicesFromShipments.
func (s *Service) CreateFromShipments(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	result, err := s.CreateInvoicesFromShipments(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return result.Primary, nil
}

// payerPlan is one payer's part of a shipment, resolved before the transaction
// so every number is minted and every guard checked before anything is written.
type payerPlan struct {
	Share    *shipment.PayerShare
	Customer *customer.Customer
	Number   string
}

func (s *Service) CreateInvoicesFromShipments(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoicesResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if len(req.ShipmentIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrRequired,
			"One shipment is required",
		)
	}
	if len(req.ShipmentIDs) > 1 {
		return s.groupedInvoicesFromShipments(ctx, req, actor)
	}

	shp, err := s.shipmentRepo.GetByID(
		ctx,
		expandedShipmentByIDRequest(req.ShipmentIDs[0], req.TenantInfo),
	)
	if err != nil {
		return nil, err
	}
	if shp.Status != shipment.StatusReadyToInvoice && shp.Status != shipment.StatusCompleted {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"Shipment must be completed or ready to invoice",
		)
	}
	if err = s.guardDetentionHolds(ctx, req.TenantInfo, []pulid.ID{shp.ID}); err != nil {
		return nil, err
	}

	resolution, err := shipment.ResolveShares(shp, shp.ChargeAllocations)
	if err != nil {
		return nil, err
	}

	plans := make([]*payerPlan, 0, len(resolution.Shares))
	for _, share := range resolution.Shares {
		plan, planErr := s.planPayer(ctx, req.TenantInfo, shp, share, req.OffCycleReason)
		if planErr != nil {
			return nil, planErr
		}
		plans = append(plans, plan)
	}

	return s.createSingleShipmentInvoices(
		ctx,
		req.TenantInfo,
		shp,
		plans,
		req.OffCycleReason,
		actor,
	)
}

// planPayer checks one payer's guards and mints their invoice number before the
// transaction opens, so a split shipment never half-bills.
func (s *Service) planPayer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shp *shipment.Shipment,
	share *shipment.PayerShare,
	offCycleReason string,
) (*payerPlan, error) {
	cus, err := s.checkPayer(ctx, tenantInfo, shp, share, offCycleReason)
	if err != nil {
		return nil, err
	}

	number, err := s.generateInvoiceNumber(ctx, tenantInfo, billingProfileOf(cus))
	if err != nil {
		return nil, err
	}

	return &payerPlan{Share: share, Customer: cus, Number: number}, nil
}

// createSingleShipmentInvoices bills every payer of one shipment in a single
// transaction: one approved queue item and one invoice per payer, all or none.
func (s *Service) createSingleShipmentInvoices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shp *shipment.Shipment,
	plans []*payerPlan,
	offCycleReason string,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoicesResult, error) {
	result := &servicesports.CreateInvoicesResult{
		Invoices: make([]*invoice.Invoice, 0, len(plans)),
	}
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		for _, plan := range plans {
			queueItem, txErr := s.billingQueueRepo.Create(txCtx, &billingqueue.BillingQueueItem{
				OrganizationID:       tenantInfo.OrgID,
				BusinessUnitID:       tenantInfo.BuID,
				ShipmentID:           shp.ID,
				OrderID:              shp.OrderID,
				BillToCustomerID:     plan.Share.PayerID,
				AllocatedTotalAmount: plan.Share.TotalAmount,
				Status:               billingqueue.StatusApproved,
				BillType:             billingqueue.BillTypeInvoice,
				Number:               plan.Number,
			})
			if txErr != nil {
				return txErr
			}

			created, txErr := s.CreateFromApprovedBillingQueueItem(
				txCtx,
				&servicesports.CreateInvoiceFromBillingQueueRequest{
					BillingQueueItemID: queueItem.ID,
					TenantInfo:         tenantInfo,
					OffCycleReason:     offCycleReasonFor(plan.Customer, offCycleReason),
				},
				actor,
			)
			if txErr != nil {
				return txErr
			}
			result.Invoices = append(result.Invoices, created.Invoice)
			if plan.Share.PayerID == shp.PayerID() {
				result.Primary = created.Invoice
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result.Primary == nil && len(result.Invoices) > 0 {
		result.Primary = result.Invoices[0]
	}

	return result, nil
}

// collectBillableLegs gathers the shipments of an order that are ready to invoice,
// loading each leg's full detail (charges) and guarding against double-billing and
// mixed customers. A non-empty `only` set restricts the invoice to those legs; every
// requested id must be a leg of the order.
func (s *Service) collectBillableLegs(
	ctx context.Context,
	ord *order.Order,
	tenantInfo pagination.TenantInfo,
	only []pulid.ID,
) ([]*shipment.Shipment, error) {
	requested := make(map[pulid.ID]struct{}, len(only))
	for _, id := range only {
		requested[id] = struct{}{}
	}
	matched := 0

	legs := make([]*shipment.Shipment, 0, len(ord.Shipments))
	for _, leg := range ord.Shipments {
		if leg == nil {
			continue
		}
		if len(requested) > 0 {
			if _, ok := requested[leg.ID]; !ok {
				continue
			}
			matched++
			if leg.Status != shipment.StatusReadyToInvoice &&
				leg.Status != shipment.StatusCompleted {
				return nil, errortypes.NewValidationError(
					"shipmentIds",
					errortypes.ErrInvalid,
					"Every selected shipment must be completed or ready to invoice",
				)
			}
		}
		if leg.Status != shipment.StatusReadyToInvoice && leg.Status != shipment.StatusCompleted {
			continue
		}
		if leg.CustomerID != ord.CustomerID {
			return nil, errortypes.NewValidationError(
				"orderId",
				errortypes.ErrInvalid,
				"All legs of a grouped invoice must share the order's customer",
			)
		}

		full, err := s.shipmentRepo.GetByID(
			ctx,
			expandedShipmentByIDRequest(leg.ID, tenantInfo),
		)
		if err != nil {
			return nil, err
		}
		legs = append(legs, full)
	}

	if len(requested) > 0 && matched != len(requested) {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"Every selected shipment must be a leg of the order",
		)
	}

	return legs, nil
}

// groupedInvoicesFromShipments resolves the single order shared by the given legs
// and delegates to CreateInvoicesFromOrder. It rejects legs that are not all
// under one order.
func (s *Service) groupedInvoicesFromShipments(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoicesResult, error) {
	orderID, err := s.sharedOrderOf(ctx, req)
	if err != nil {
		return nil, err
	}

	return s.CreateInvoicesFromOrder(ctx, &servicesports.CreateInvoiceFromOrderRequest{
		OrderID:        orderID,
		ShipmentIDs:    req.ShipmentIDs,
		TenantInfo:     req.TenantInfo,
		OffCycleReason: req.OffCycleReason,
	}, actor)
}

// CreateFromOrder issues the grouped invoices covering every billable leg of an
// order and returns the primary payer's. Each billable leg gets one approved
// billing-queue item per payer (all carrying the order id); the first of each
// payer's items is the anchor whose id backs that invoice's single-valued FK and
// idempotency lookup.
func (s *Service) CreateFromOrder(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
	actor *servicesports.RequestActor,
) (*invoice.Invoice, error) {
	result, err := s.CreateInvoicesFromOrder(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return result.Primary, nil
}

func (s *Service) CreateInvoicesFromOrder(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoicesResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	if req.OrderID.IsNil() {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrRequired,
			"Order is required",
		)
	}

	ord, err := s.orderRepo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         req.OrderID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	orderCustomer, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         ord.CustomerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}

	number, err := s.generateInvoiceNumber(
		ctx,
		req.TenantInfo,
		billingProfileOf(orderCustomer),
	)
	if err != nil {
		return nil, err
	}

	result := &servicesports.CreateInvoicesResult{}
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		var txErr error
		result, txErr = s.createOrderInvoicesTx(txCtx, req, orderCustomer, number, actor)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// orderPayerBucket is everything one payer is billed for on an order: the legs
// they have a share in, and that share per leg.
type orderPayerBucket struct {
	PayerID pulid.ID
	Legs    []*shipment.Shipment
	Shares  map[pulid.ID]*shipment.PayerShare
	IsSplit bool
}

// bucketLegsByPayer resolves every leg's shares and groups them by payer. The
// primary payer leads: the order's customer when they have a share, otherwise
// the first payer in a stable order.
func bucketLegsByPayer(
	ord *order.Order,
	legs []*shipment.Shipment,
) ([]*orderPayerBucket, error) {
	byPayer := make(map[pulid.ID]*orderPayerBucket, 2)
	for _, leg := range legs {
		resolution, err := shipment.ResolveShares(leg, leg.ChargeAllocations)
		if err != nil {
			return nil, err
		}
		for _, share := range resolution.Shares {
			bucket, ok := byPayer[share.PayerID]
			if !ok {
				bucket = &orderPayerBucket{
					PayerID: share.PayerID,
					Shares:  make(map[pulid.ID]*shipment.PayerShare, len(legs)),
				}
				byPayer[share.PayerID] = bucket
			}
			bucket.Legs = append(bucket.Legs, leg)
			bucket.Shares[leg.ID] = share
			bucket.IsSplit = bucket.IsSplit || resolution.IsSplit
		}
	}

	ordered := make([]*orderPayerBucket, 0, len(byPayer))
	if primary, ok := byPayer[ord.CustomerID]; ok {
		ordered = append(ordered, primary)
	}
	rest := make([]pulid.ID, 0, len(byPayer))
	for payerID := range byPayer {
		if payerID != ord.CustomerID {
			rest = append(rest, payerID)
		}
	}
	slices.Sort(rest)
	for _, payerID := range rest {
		ordered = append(ordered, byPayer[payerID])
	}

	return ordered, nil
}

func (s *Service) createOrderInvoicesTx(
	txCtx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
	orderCustomer *customer.Customer,
	number string,
	actor *servicesports.RequestActor,
) (*servicesports.CreateInvoicesResult, error) {
	plan, txErr := s.planOrderInvoices(txCtx, req)
	if txErr != nil {
		return nil, txErr
	}
	ord := plan.order

	result := &servicesports.CreateInvoicesResult{
		Invoices: make([]*invoice.Invoice, 0, len(plan.buckets)),
	}
	for i, bucket := range plan.buckets {
		invoiceNumber := number
		if i > 0 {
			var err error
			invoiceNumber, err = s.generateInvoiceNumber(
				txCtx,
				req.TenantInfo,
				billingProfileOf(plan.customers[i]),
			)
			if err != nil {
				return nil, err
			}
		}

		queueItems, err := s.createLegQueueItems(
			txCtx,
			req.TenantInfo,
			bucket,
			invoiceNumber,
		)
		if err != nil {
			return nil, err
		}

		entity, chargeShare, err := s.buildBucketInvoice(
			txCtx,
			req,
			plan,
			i,
			queueItems.Anchor,
			orderCustomer,
		)
		if err != nil {
			return nil, err
		}

		created, err := s.repo.Create(txCtx, entity)
		if err != nil {
			return nil, err
		}

		if err = s.attachQueueItems(
			txCtx,
			req.TenantInfo,
			created.ID,
			queueItems.ItemIDs,
		); err != nil {
			return nil, err
		}

		if err = s.markOrderChargeSharesInvoiced(
			txCtx,
			req.TenantInfo,
			ord.ID,
			chargeShare,
			created,
		); err != nil {
			return nil, err
		}

		if err = s.syncDetentionBilling(txCtx, created, actor); err != nil {
			return nil, err
		}

		auditActor := actor.AuditActor()
		s.logAction(
			created,
			auditActor,
			permission.OpCreate,
			nil,
			created,
			"Grouped invoice created from order",
		)
		s.publishInvalidation(txCtx, created, auditActor, "created", created)

		result.Invoices = append(result.Invoices, created)
		if i == 0 {
			result.Primary = created
		}
	}

	return result, nil
}

// orderChargeShareFor resolves the payer's slice of the order charges still owed
// to them. Charges with no allocation go whole to the default payer, so they
// appear only on that payer's invoice.
func orderChargeShareFor(
	charges []*order.OrderCharge,
	defaultPayerID pulid.ID,
	payerID pulid.ID,
) (*shipment.PayerShare, bool, error) {
	if len(charges) == 0 {
		return nil, false, nil
	}

	refs := make([]shipment.OrderChargeRef, 0, len(charges))
	allocations := make([]*shipment.ChargeAllocation, 0, len(charges))
	for _, charge := range charges {
		if charge == nil {
			continue
		}
		refs = append(refs, shipment.OrderChargeRef{
			ID:          charge.ID,
			Description: charge.Description,
			Amount:      charge.Amount,
		})
		allocations = append(allocations, charge.Allocations...)
	}

	resolution, err := shipment.ResolveOrderChargeShares(refs, allocations, defaultPayerID)
	if err != nil {
		return nil, false, err
	}

	share := resolution.ShareFor(payerID)
	if share == nil || len(share.Charges) == 0 {
		return nil, resolution.IsSplit, nil
	}

	// A share for the default payer also carries the implicit rows of charges
	// allocated wholly to somebody else with a zero amount; drop them.
	kept := make([]shipment.AllocatedCharge, 0, len(share.Charges))
	for _, charge := range share.Charges {
		if charge.Amount.IsZero() && charge.AllocationID.IsNil() && charge.Partial {
			continue
		}
		kept = append(kept, charge)
	}
	share.Charges = kept
	if len(kept) == 0 {
		return nil, resolution.IsSplit, nil
	}

	return share, resolution.IsSplit, nil
}

// legQueueItems is the queue items an invoice will bill: the anchor whose id backs
// the invoice's single-valued FK and idempotency lookup, plus every id so the
// invoice back-link can be written in one statement.
type legQueueItems struct {
	Anchor  *billingqueue.BillingQueueItem
	ItemIDs []pulid.ID
}

func (s *Service) attachQueueItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
	itemIDs []pulid.ID,
) error {
	attached, err := s.billingQueueRepo.AttachInvoice(ctx, &repositories.AttachInvoiceRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  invoiceID,
		ItemIDs:    itemIDs,
	})
	if err != nil {
		return err
	}
	if attached != int64(len(itemIDs)) {
		return errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalidOperation,
			"Some shipments were invoiced elsewhere while this invoice was being created",
		)
	}

	return nil
}

// createLegQueueItems creates one approved billing-queue item per billable leg
// for one payer.
//
// Only the anchor carries the invoice number; siblings leave it null, because the
// billing-queue number is tenant-unique. Each item takes its order from its own
// leg rather than from one order-wide value, which is what lets a selection span
// orders — or carry legs with no order at all.
func (s *Service) createLegQueueItems(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	bucket *orderPayerBucket,
	number string,
) (*legQueueItems, error) {
	result := &legQueueItems{ItemIDs: make([]pulid.ID, 0, len(bucket.Legs))}

	for _, leg := range bucket.Legs {
		itemNumber := ""
		if result.Anchor == nil {
			itemNumber = number
		}
		allocated := decimal.Zero
		if share := bucket.Shares[leg.ID]; share != nil {
			allocated = share.TotalAmount
		}

		item, err := s.billingQueueRepo.Create(ctx, &billingqueue.BillingQueueItem{
			OrganizationID:       tenantInfo.OrgID,
			BusinessUnitID:       tenantInfo.BuID,
			ShipmentID:           leg.ID,
			OrderID:              leg.OrderID,
			BillToCustomerID:     bucket.PayerID,
			AllocatedTotalAmount: allocated,
			Status:               billingqueue.StatusApproved,
			BillType:             billingqueue.BillTypeInvoice,
			Number:               itemNumber,
		})
		if err != nil {
			return nil, err
		}
		if result.Anchor == nil {
			result.Anchor = item
		}
		result.ItemIDs = append(result.ItemIDs, item.ID)
	}

	return result, nil
}

// markOrderChargeSharesInvoiced stamps what this invoice billed: unallocated
// charges on the charge itself, allocated ones on their share, and the charge as
// a whole once its last share is gone.
func (s *Service) markOrderChargeSharesInvoiced(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	share *shipment.PayerShare,
	created *invoice.Invoice,
) error {
	if share == nil || len(share.Charges) == 0 {
		return nil
	}

	chargeIDs := make([]pulid.ID, 0, len(share.Charges))
	allocationIDs := make([]pulid.ID, 0, len(share.Charges))
	for _, charge := range share.Charges {
		switch {
		case charge.AllocationID.IsNotNil():
			allocationIDs = append(allocationIDs, charge.AllocationID)
		case charge.OrderChargeID.IsNotNil():
			chargeIDs = append(chargeIDs, charge.OrderChargeID)
		}
	}

	if len(chargeIDs) > 0 {
		if _, err := s.orderRepo.MarkChargesInvoiced(ctx, &repositories.MarkOrderChargesInvoicedRequest{
			TenantInfo: tenantInfo,
			OrderID:    orderID,
			ChargeIDs:  chargeIDs,
			InvoiceID:  created.ID,
			InvoicedAt: created.InvoiceDate,
		}); err != nil {
			return err
		}
	}
	if len(allocationIDs) == 0 {
		return nil
	}

	if _, err := s.chargeAllocationRepo.MarkInvoiced(ctx, &repositories.MarkChargeAllocationsInvoicedRequest{
		TenantInfo:    tenantInfo,
		AllocationIDs: allocationIDs,
		InvoiceID:     created.ID,
		InvoicedAt:    created.InvoiceDate,
	}); err != nil {
		return err
	}
	_, err := s.orderRepo.MarkChargesFullyInvoicedWhereComplete(
		ctx,
		&repositories.MarkChargesFullyInvoicedRequest{
			TenantInfo: tenantInfo,
			OrderID:    orderID,
			InvoiceID:  created.ID,
			InvoicedAt: created.InvoiceDate,
		},
	)
	return err
}

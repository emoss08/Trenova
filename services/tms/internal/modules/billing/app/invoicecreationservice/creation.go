package invoicecreationservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceassembly"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicelines"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type existingInvoiceLookupResult struct {
	Invoice *invoice.Invoice
	Found   bool
}

type invoiceDependencies struct {
	Shipment       *shipment.Shipment
	Order          *order.Order
	Customer       *customer.Customer
	BillingControl *billingcontrol.BillingControl
	// Share is the payer's slice of the shipment's charges; Shipper is the
	// ordering customer when it is not the payer.
	Share       *chargeallocation.PayerShare
	Shipper     *customer.Customer
	IsSplitBill bool
}

func (s *Service) CreateFromApprovedBillingQueueItem(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
	actor *permissionports.RequestActor,
) (*invoiceports.CreateInvoiceFromBillingQueueResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	existingLookup, err := s.getExistingInvoiceByBillingQueueItem(ctx, req)
	if err != nil {
		return nil, err
	}
	if existingLookup.Found {
		return &invoiceports.CreateInvoiceFromBillingQueueResult{
			Invoice:  existingLookup.Invoice,
			AutoPost: s.resolveAutoPost(existingLookup.Invoice.Customer),
		}, nil
	}

	item, err := s.getApprovedBillingQueueItem(ctx, req)
	if err != nil {
		return nil, err
	}

	draft, err := s.planDraftFromItem(ctx, req, item)
	if err != nil {
		return nil, err
	}
	if draft.deferred {
		return &invoiceports.CreateInvoiceFromBillingQueueResult{
			DeferredToStatement: true,
		}, nil
	}
	dependencies := draft.dependencies

	created, err := s.repo.Create(ctx, draft.entity)
	if err != nil {
		return nil, err
	}

	if err = s.syncDetentionBilling(ctx, created, actor); err != nil {
		return nil, err
	}

	if item.IsAdjustmentOrigin {
		s.syncAdjustmentLineage(ctx, created, item)
	}

	autoPost := s.shouldAutoPost(ctx, created.OrganizationID, dependencies.Customer)
	auditActor := actor.AuditActor()
	s.rec.LogAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Invoice created from approved billing queue item",
	)
	s.rec.PublishInvalidation(ctx, created, auditActor, "created", created)

	return &invoiceports.CreateInvoiceFromBillingQueueResult{
		Invoice:  created,
		AutoPost: autoPost,
	}, nil
}

// PreviewApprovalInvoice is the invoice approving the item would make: the
// draft CreateFromApprovedBillingQueueItem would save, planned by the same
// code from the item as it stands, and whether it would then post on its
// own. It saves nothing. An item that already has its invoice returns that
// one, and a statement customer's item returns DeferredToStatement.
func (s *Service) PreviewApprovalInvoice(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
) (*invoiceports.CreateInvoiceFromBillingQueueResult, error) {
	existing, err := s.getExistingInvoiceByBillingQueueItem(ctx, req)
	if err != nil {
		return nil, err
	}
	if existing.Found {
		return &invoiceports.CreateInvoiceFromBillingQueueResult{
			Invoice:  existing.Invoice,
			AutoPost: s.resolveAutoPost(existing.Invoice.Customer),
		}, nil
	}

	item, err := s.billingQueueRepo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                req.BillingQueueItemID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}
	if billingqueue.IsTerminalStatus(item.Status) {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"A billing queue item in {0} status cannot be approved",
			string(item.Status),
		)
	}

	draft, err := s.planDraftFromItem(ctx, req, item)
	if err != nil {
		return nil, err
	}
	if draft.deferred {
		return &invoiceports.CreateInvoiceFromBillingQueueResult{
			DeferredToStatement: true,
		}, nil
	}

	return &invoiceports.CreateInvoiceFromBillingQueueResult{
		Invoice:  draft.entity,
		AutoPost: s.shouldAutoPost(ctx, req.TenantInfo.OrgID, draft.dependencies.Customer),
	}, nil
}

// invoiceDraft is the invoice an approved item bills, before it is saved.
type invoiceDraft struct {
	entity       *invoice.Invoice
	dependencies *invoiceDependencies
	deferred     bool
}

// planDraftFromItem builds, and validates, the invoice an approved item
// bills. It reads and saves nothing but what the build reads.
func (s *Service) planDraftFromItem(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
	item *billingqueue.BillingQueueItem,
) (*invoiceDraft, error) {
	dependencies, err := s.getInvoiceDependencies(ctx, req, item)
	if err != nil {
		return nil, err
	}

	// Approving a statement customer's freight puts it on their statement; it does
	// not bill it. Checked before the cadence guard because this is not a
	// deviation to be justified — it is the schedule working.
	if req.DeferToStatement && isStatementBilled(dependencies.Customer) {
		return &invoiceDraft{dependencies: dependencies, deferred: true}, nil
	}

	// The last gate before an invoice exists, so it cannot be bypassed by any of
	// the paths that reach this one.
	if err = guardStatementCadence(dependencies.Customer, req.OffCycleReason); err != nil {
		return nil, err
	}
	if err = s.guardItemDetentionHolds(ctx, req.TenantInfo, item); err != nil {
		return nil, err
	}
	if err = guardItemRatingReview(item, dependencies.Shipment); err != nil {
		return nil, err
	}

	entity, err := s.buildInvoiceEntity(ctx, &buildInvoiceParams{
		Anchor:         item,
		Scope:          invoice.ScopeShipment,
		Customer:       dependencies.Customer,
		Control:        dependencies.BillingControl,
		Legs:           legsFromShipment(dependencies.Shipment),
		Order:          dependencies.Order,
		OffCycleReason: offCycleReasonFor(dependencies.Customer, req.OffCycleReason),
		Shipper:        dependencies.Shipper,
		Shares:         sharesForLeg(dependencies.Shipment, dependencies.Share),
		IsSplitBill:    dependencies.IsSplitBill,
	})
	if errors.Is(err, errNothingToBill) {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Billing queue item has no shipment or adjustment context to bill from",
		)
	}
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return &invoiceDraft{entity: entity, dependencies: dependencies}, nil
}

func (s *Service) syncAdjustmentLineage(
	ctx context.Context,
	created *invoice.Invoice,
	item *billingqueue.BillingQueueItem,
) {
	if created == nil || item == nil {
		return
	}
	if item.SourceInvoiceID != nil && item.SourceInvoiceID.IsNotNil() {
		sourceInvoice, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
			ID: *item.SourceInvoiceID,
			TenantInfo: pagination.TenantInfo{
				OrgID: item.OrganizationID,
				BuID:  item.BusinessUnitID,
			},
		})
		if err == nil && sourceInvoice != nil {
			sourceInvoice.SupersededByInvoiceID = created.ID
			sourceInvoice.CorrectionGroupID = created.CorrectionGroupID
			if _, err = s.repo.Update(ctx, sourceInvoice); err != nil {
				s.l.Warn("failed to update source invoice supersession linkage", zap.Error(err))
			}
		}
	}

	if s.adjustmentRepo != nil && item.CorrectionGroupID != nil &&
		item.CorrectionGroupID.IsNotNil() {
		group, err := s.adjustmentRepo.GetCorrectionGroup(
			ctx,
			invoiceadjustmentports.GetCorrectionGroupRequest{
				ID: *item.CorrectionGroupID,
				TenantInfo: pagination.TenantInfo{
					OrgID: item.OrganizationID,
					BuID:  item.BusinessUnitID,
				},
			},
		)
		if err == nil && group != nil {
			group.CurrentInvoiceID = created.ID
			if _, err = s.adjustmentRepo.UpdateCorrectionGroup(ctx, group); err != nil {
				s.l.Warn("failed to update correction group current invoice", zap.Error(err))
			}
		}
	}
}

func (s *Service) getExistingInvoiceByBillingQueueItem(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
) (*existingInvoiceLookupResult, error) {
	existing, err := s.repo.GetByBillingQueueItemID(
		ctx,
		invoiceports.GetInvoiceByBillingQueueItemIDRequest{
			BillingQueueItemID: req.BillingQueueItemID,
			TenantInfo:         req.TenantInfo,
		},
	)
	if err == nil {
		return &existingInvoiceLookupResult{
			Invoice: existing,
			Found:   true,
		}, nil
	}
	if errortypes.IsNotFoundError(err) {
		return &existingInvoiceLookupResult{}, nil
	}

	return nil, err
}

func (s *Service) getApprovedBillingQueueItem(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
) (*billingqueue.BillingQueueItem, error) {
	item, err := s.billingQueueRepo.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                req.BillingQueueItemID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, err
	}

	if item.Status != billingqueue.StatusApproved {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Only approved billing queue items can create invoices",
		)
	}

	return item, nil
}

// resolveDependencyShipment resolves the leg (when the queue item carries one) and the
// billing customer. Order-scoped adjustment items (credit memo / replacement for a
// grouped invoice) have no single shipment; their customer comes from the source
// invoice.
func (s *Service) resolveDependencyShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *billingqueue.BillingQueueItem,
) (*shipment.Shipment, pulid.ID, error) {
	if !item.ShipmentID.IsNil() {
		shp := item.Shipment
		if shp == nil || shp.ID != item.ShipmentID || shp.CustomerID.IsNil() {
			var err error
			shp, err = s.shipmentRepo.GetByID(
				ctx,
				invoiceassembly.ExpandedShipmentByIDRequest(item.ShipmentID, tenantInfo),
			)
			if err != nil {
				return nil, pulid.Nil, err
			}
		}
		if item.BillToCustomerID.IsNotNil() {
			return shp, item.BillToCustomerID, nil
		}
		payerID := shp.PayerID()
		return shp, payerID, nil
	}

	if item.BillToCustomerID.IsNotNil() &&
		(item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil()) {
		return nil, item.BillToCustomerID, nil
	}

	if item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"Billing queue item has no shipment or source invoice to bill from",
		)
	}
	src, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         *item.SourceInvoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, pulid.Nil, err
	}
	return nil, src.CustomerID, nil
}

func (s *Service) getInvoiceDependencies(
	ctx context.Context,
	req *invoiceports.CreateInvoiceFromBillingQueueRequest,
	item *billingqueue.BillingQueueItem,
) (*invoiceDependencies, error) {
	shp, customerID, err := s.resolveDependencyShipment(ctx, req.TenantInfo, item)
	if err != nil {
		return nil, err
	}
	if err = invoicelines.HydrateAccessorials(
		ctx,
		s.accessorialRepo,
		req.TenantInfo,
		shp,
	); err != nil {
		return nil, err
	}

	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeState:          true,
		},
	})
	if err != nil {
		return nil, err
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	orderID := item.OrderID
	if orderID.IsNil() && shp != nil {
		orderID = shp.OrderID
	}
	var ord *order.Order
	if !orderID.IsNil() {
		ord, err = s.orderRepo.GetByID(ctx, orderports.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil && !errortypes.IsNotFoundError(err) {
			return nil, err
		}
	}

	deps := &invoiceDependencies{
		Shipment:       shp,
		Order:          ord,
		Customer:       cus,
		BillingControl: control,
	}
	if shp == nil {
		return deps, nil
	}

	resolution, err := chargeallocation.ResolveShares(ctx, shp, shp.ChargeAllocations)
	if err != nil {
		return nil, err
	}
	deps.Share = resolution.ShareFor(cus.ID)
	if deps.Share == nil {
		return nil, errortypes.NewValidationError(
			"billingQueueItemId",
			errortypes.ErrInvalidOperation,
			"{0} no longer has a share of shipment {1}",
			cus.Name,
			shp.ProNumber,
		)
	}
	deps.IsSplitBill = resolution.IsSplit
	if shp.CustomerID != cus.ID {
		deps.Shipper, err = s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
			ID:         shp.CustomerID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}
	}

	return deps, nil
}

// buildInvoiceParams is what an invoice is built from, whatever its scope.
//
// Grouped by a struct rather than passed positionally because the shapes differ
// only in which identity the header carries, and a six-argument signature that
// means different things per scope is how the two builders drifted apart in the
// first place.
type buildInvoiceParams struct {
	Anchor       *billingqueue.BillingQueueItem
	Scope        invoice.Scope
	Customer     *customer.Customer
	Control      *billingcontrol.BillingControl
	Legs         []*shipment.Shipment
	Order        *order.Order
	OrderCharges []*order.OrderCharge
	RunID        pulid.ID
	PeriodStart  *int64
	PeriodEnd    *int64
	InvoiceDate  int64
	Number       string
	// OffCycleReason is set only when this invoice takes freight off a customer's
	// periodic statement.
	OffCycleReason string
	// Shipper is the ordering customer when the payer is somebody else. Shares
	// are each leg's slice for the payer; a leg with no entry is billed in full.
	// OrderChargeShare is the payer's slice of the order's own charges.
	Shipper          *customer.Customer
	Shares           map[pulid.ID]*chargeallocation.PayerShare
	OrderChargeShare *chargeallocation.PayerShare
	IsSplitBill      bool
}

// buildInvoiceEntity builds every shape of invoice from its legs.
//
// One builder, because a single-shipment invoice is the one-leg case of a
// grouped one and a consolidated invoice is the many-order case. Only the header
// differs: which identity it carries, and whether it names a period.
func (s *Service) buildInvoiceEntity(
	ctx context.Context,
	p *buildInvoiceParams,
) (*invoice.Invoice, error) {
	if p.Anchor.IsAdjustmentOrigin {
		entity, err := s.adjustmentOriginEntity(ctx, p)
		if !errors.Is(err, errNothingToBill) {
			return entity, err
		}
	}

	if len(p.Legs) == 0 {
		return nil, errNothingToBill
	}

	invoiceDate := p.InvoiceDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}
	paymentTerm := invoiceassembly.ResolvePaymentTerm(p.Customer, p.Control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}

	number := p.Number
	if number == "" {
		number = p.Anchor.Number
	}

	lines, err := invoiceLines(ctx, p)
	if err != nil {
		return nil, err
	}

	entity := &invoice.Invoice{
		OrganizationID:     p.Anchor.OrganizationID,
		BusinessUnitID:     p.Anchor.BusinessUnitID,
		BillingQueueItemID: p.Anchor.ID,
		Scope:              p.Scope,
		CustomerID:         p.Customer.ID,
		Number:             number,
		BillType:           p.Anchor.BillType,
		Status:             invoice.StatusDraft,
		PaymentTerm:        paymentTerm,
		CurrencyCode:       invoiceassembly.BillingCurrencyFromCustomer(p.Customer),
		InvoiceDate:        invoiceDate,
		DueDate:            invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		ShipmentCount:      len(p.Legs),
		BillToName:         p.Customer.Name,
		BillToCode:         p.Customer.Code,
		BillToAddressLine1: p.Customer.AddressLine1,
		BillToAddressLine2: p.Customer.AddressLine2,
		BillToCity:         p.Customer.City,
		BillToPostalCode:   p.Customer.PostalCode,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		OffCycleReason:     p.OffCycleReason,
		IsSplitBill:        p.IsSplitBill,
		Lines:              lines,
		RatingNotes:        invoicelines.RatingNotes(p.Legs...),
	}

	invoiceassembly.ApplyInvoiceDetail(entity, p.Customer)
	applyInvoiceScopeHeader(entity, p)
	applyShipper(entity, p)

	if p.Customer.State != nil {
		entity.BillToState = p.Customer.State.Abbreviation
		entity.BillToCountry = p.Customer.State.CountryName
	}

	invoiceassembly.SyncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()

	return entity, nil
}

// invoiceLines numbers every leg's lines, then the order-level charges.
func invoiceLines(ctx context.Context, p *buildInvoiceParams) ([]*invoice.InvoiceLine, error) {
	lines := make([]*invoice.InvoiceLine, 0, len(p.Legs))
	nextLineNumber := 1
	for _, leg := range p.Legs {
		share, err := legShare(ctx, leg, p.Shares)
		if err != nil {
			return nil, err
		}
		legLines := invoicelines.ForShipmentShare(
			p.Anchor.BillType,
			leg,
			share,
			nextLineNumber,
		)
		lines = append(lines, legLines...)
		nextLineNumber += len(legLines)
	}

	// Order-level charges (customs brokerage, an order-wide fuel surcharge) carry
	// no leg attribution, which is what puts them in the trailing section when the
	// invoice is rendered.
	switch {
	case p.OrderChargeShare != nil:
		for i := range p.OrderChargeShare.Charges {
			charge := &p.OrderChargeShare.Charges[i]
			if charge.Kind != shipmentvocab.ChargeAllocationKindOrderCharge {
				continue
			}
			lines = append(lines, invoicelines.ForOrderChargeShare(
				p.Anchor.BillType,
				charge,
				nextLineNumber,
			))
			nextLineNumber++
		}
	default:
		for _, charge := range p.OrderCharges {
			if charge == nil {
				continue
			}
			amount := invoicelines.SignedAmount(p.Anchor.BillType, charge.Amount)
			lines = append(lines, &invoice.InvoiceLine{
				LineNumber:  nextLineNumber,
				Type:        invoice.InvoiceLineTypeAccessorial,
				Description: charge.Description,
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   amount,
				Amount:      amount,
			})
			nextLineNumber++
		}
	}

	return lines, nil
}

// applyInvoiceScopeHeader stamps the identity fields that belong to this scope,
// and only those. A grouped invoice deliberately carries no header shipment, and
// a consolidated one carries neither shipment nor order — the lines hold the
// attribution and Scope says so outright.
func applyInvoiceScopeHeader(entity *invoice.Invoice, p *buildInvoiceParams) {
	switch p.Scope {
	case invoice.ScopeShipment:
		leg := firstLeg(p.Legs)
		entity.ShipmentID = leg.ID
		entity.ShipmentProNumber = leg.ProNumber
		entity.ShipmentBOL = leg.BOL
		entity.ServiceDate = serviceDateFromShipment(leg)
		entity.OrderID = orderIDFromOrder(p.Order)
		entity.OrderNumber = orderNumberFromOrder(p.Order)
	case invoice.ScopeOrder:
		entity.OrderID = orderIDFromOrder(p.Order)
		entity.OrderNumber = orderNumberFromOrder(p.Order)
	case invoice.ScopeConsolidated:
		entity.InvoiceRunID = p.RunID
		entity.PeriodStart = p.PeriodStart
		entity.PeriodEnd = p.PeriodEnd
	case invoice.ScopeAdjustment, invoice.ScopeMemo:
	}
}

// applyShipper records on whose behalf a single-shipment or order invoice bills
// when the payer is not the ordering customer. A consolidated invoice spans
// shippers, so its lines carry that attribution instead of the header.
func applyShipper(entity *invoice.Invoice, p *buildInvoiceParams) {
	if p.Shipper == nil || p.Shipper.ID == entity.CustomerID {
		return
	}
	switch p.Scope {
	case invoice.ScopeShipment, invoice.ScopeOrder:
		entity.ShipperCustomerID = p.Shipper.ID
	case invoice.ScopeConsolidated, invoice.ScopeAdjustment, invoice.ScopeMemo:
	}
}

func sharesForLeg(
	shp *shipment.Shipment,
	share *chargeallocation.PayerShare,
) map[pulid.ID]*chargeallocation.PayerShare {
	if shp == nil || share == nil {
		return nil
	}

	return map[pulid.ID]*chargeallocation.PayerShare{shp.ID: share}
}

func legsFromShipment(shp *shipment.Shipment) []*shipment.Shipment {
	if shp == nil {
		return nil
	}
	return []*shipment.Shipment{shp}
}

func firstLeg(legs []*shipment.Shipment) *shipment.Shipment {
	if len(legs) == 0 {
		return nil
	}
	return legs[0]
}

func orderIDFromOrder(ord *order.Order) pulid.ID {
	if ord == nil {
		return pulid.Nil
	}
	return ord.ID
}

func orderNumberFromOrder(ord *order.Order) string {
	if ord == nil {
		return ""
	}
	return ord.OrderNumber
}

type adjustmentInvoiceContext struct {
	ReplacementLines   []*invoice.InvoiceLine `json:"replacementLines"`
	SubtotalAmount     decimal.Decimal        `json:"subtotalAmount"`
	OtherAmount        decimal.Decimal        `json:"otherAmount"`
	TotalAmount        decimal.Decimal        `json:"totalAmount"`
	AccountingDate     int64                  `json:"accountingDate"`
	SourceInvoiceID    pulid.ID               `json:"sourceInvoiceId"`
	CorrectionGroupID  pulid.ID               `json:"correctionGroupId"`
	SourceAdjustmentID pulid.ID               `json:"sourceAdjustmentId"`
}

func (s *Service) adjustmentOriginEntity(
	ctx context.Context,
	p *buildInvoiceParams,
) (*invoice.Invoice, error) {
	entity, err := s.buildAdjustmentOriginInvoiceEntity(
		ctx,
		p.Anchor,
		firstLeg(p.Legs),
		p.Customer,
		p.Control,
	)
	if err != nil {
		return nil, err
	}
	if p.Order != nil {
		entity.OrderID = p.Order.ID
		entity.OrderNumber = p.Order.OrderNumber
	}

	return entity, nil
}

var errNothingToBill = errors.New("there is no leg and no adjustment context to bill from")

func (s *Service) buildAdjustmentOriginInvoiceEntity(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
	shp *shipment.Shipment,
	cus *customer.Customer,
	control *billingcontrol.BillingControl,
) (*invoice.Invoice, error) {
	if len(item.AdjustmentContext) == 0 {
		return nil, errNothingToBill
	}

	raw, err := sonic.Marshal(item.AdjustmentContext)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errNothingToBill, err)
	}

	var adjustment adjustmentInvoiceContext
	if err = sonic.Unmarshal(raw, &adjustment); err != nil {
		return nil, fmt.Errorf("%w: %w", errNothingToBill, err)
	}

	invoiceDate := adjustment.AccountingDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}
	paymentTerm := invoiceassembly.ResolvePaymentTerm(cus, control)
	if paymentTerm == "" {
		paymentTerm = invoice.PaymentTermNet30
	}
	lines := adjustment.ReplacementLines
	if len(lines) == 0 {
		if shp == nil {
			return nil, errNothingToBill
		}
		lines, err = invoicelines.ForShipment(ctx, item.BillType, shp, 1)
		if err != nil {
			return nil, err
		}
	}

	entity := &invoice.Invoice{
		OrganizationID:            item.OrganizationID,
		BusinessUnitID:            item.BusinessUnitID,
		BillingQueueItemID:        item.ID,
		Scope:                     invoice.ScopeAdjustment,
		CustomerID:                cus.ID,
		Number:                    item.Number,
		BillType:                  item.BillType,
		Status:                    invoice.StatusDraft,
		PaymentTerm:               paymentTerm,
		CurrencyCode:              invoiceassembly.BillingCurrencyFromCustomer(cus),
		InvoiceDate:               invoiceDate,
		DueDate:                   invoice.DueDateFromPaymentTerm(invoiceDate, paymentTerm),
		BillToName:                cus.Name,
		BillToCode:                cus.Code,
		BillToAddressLine1:        cus.AddressLine1,
		BillToAddressLine2:        cus.AddressLine2,
		BillToCity:                cus.City,
		BillToPostalCode:          cus.PostalCode,
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		CorrectionGroupID:         adjustment.CorrectionGroupID,
		SupersedesInvoiceID:       adjustment.SourceInvoiceID,
		SourceInvoiceAdjustmentID: adjustment.SourceAdjustmentID,
		Lines:                     lines,
		RatingNotes:               invoicelines.RatingNotes(shp),
	}
	if shp != nil {
		entity.ShipmentID = shp.ID
		entity.ShipmentProNumber = shp.ProNumber
		entity.ShipmentBOL = shp.BOL
		entity.ServiceDate = serviceDateFromShipment(shp)
		if shp.CustomerID != cus.ID {
			entity.ShipperCustomerID = shp.CustomerID
		}
	}
	for _, line := range lines {
		if line != nil && line.IsPartialShare() {
			entity.IsSplitBill = true
			break
		}
	}
	if cus.State != nil {
		entity.BillToState = cus.State.Abbreviation
		entity.BillToCountry = cus.State.CountryName
	}
	invoiceassembly.SyncInvoiceTotalsFromLines(entity)
	entity.SyncMinorAmounts()
	return entity, nil
}

func legShare(
	ctx context.Context,
	leg *shipment.Shipment,
	shares map[pulid.ID]*chargeallocation.PayerShare,
) (*chargeallocation.PayerShare, error) {
	if share := shares[leg.ID]; share != nil {
		return share, nil
	}

	resolution, err := chargeallocation.ResolveShares(ctx, leg, nil)
	if err != nil {
		return nil, err
	}

	return resolution.Primary(), nil
}

func serviceDateFromShipment(shp *shipment.Shipment) *int64 {
	switch {
	case shp.ActualDeliveryDate != nil:
		return shp.ActualDeliveryDate
	case shp.ActualShipDate != nil:
		return shp.ActualShipDate
	default:
		return nil
	}
}

func (s *Service) shouldAutoPost(
	ctx context.Context,
	orgID pulid.ID,
	cus *customer.Customer,
) bool {
	control, err := s.billingRepo.GetByOrgID(ctx, orgID)
	if err == nil && control != nil {
		return s.billingPolicyService().CanAutoPostInvoice(control, cus)
	}

	return s.resolveAutoPost(cus)
}

func (s *Service) resolveAutoPost(cus *customer.Customer) bool {
	return cus != nil && cus.BillingProfile != nil && cus.BillingProfile.AutoBill
}

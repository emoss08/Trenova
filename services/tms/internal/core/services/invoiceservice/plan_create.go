package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/invoicelines"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const unnumberedInvoice = "Unnumbered"

func (s *Service) checkPayer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shp *shipment.Shipment,
	share *shipment.PayerShare,
	offCycleReason string,
) (*customer.Customer, error) {
	exists, err := s.billingQueueRepo.ExistsByShipmentPayerAndType(
		ctx,
		tenantInfo,
		shp.ID,
		share.PayerID,
		billingqueue.BillTypeInvoice,
	)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errortypes.NewConflictError(
			"A billing queue item already exists for this shipment and payer",
		)
	}

	cus, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         share.PayerID,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, err
	}
	if err = guardStatementCadence(cus, offCycleReason); err != nil {
		return nil, err
	}

	return cus, nil
}

type orderInvoicePlan struct {
	order        *order.Order
	buckets      []*orderPayerBucket
	customers    []*customer.Customer
	control      *tenant.BillingControl
	defaultPayer pulid.ID
}

func (s *Service) planOrderInvoices(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
) (*orderInvoicePlan, error) {
	ord, err := s.orderRepo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              req.OrderID,
		TenantInfo:      req.TenantInfo,
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}

	legs, err := s.collectBillableLegs(ctx, ord, req.TenantInfo, req.ShipmentIDs)
	if err != nil {
		return nil, err
	}
	if err = invoicelines.HydrateAccessorials(ctx, s.accessorialRepo, req.TenantInfo, legs...); err != nil {
		return nil, err
	}
	if len(legs) == 0 {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalid,
			"Order has no legs that are completed or ready to invoice",
		)
	}
	if err = s.guardDetentionHolds(ctx, req.TenantInfo, legIDs(legs)); err != nil {
		return nil, err
	}

	buckets, err := bucketLegsByPayer(ord, legs)
	if err != nil {
		return nil, err
	}
	for _, bucket := range buckets {
		for _, leg := range bucket.Legs {
			exists, existsErr := s.billingQueueRepo.ExistsByShipmentPayerAndType(
				ctx,
				req.TenantInfo,
				leg.ID,
				bucket.PayerID,
				billingqueue.BillTypeInvoice,
			)
			if existsErr != nil {
				return nil, existsErr
			}
			if exists {
				return nil, errortypes.NewConflictError(
					"A billing queue item already exists for a leg of this order",
				)
			}
		}
	}

	control, err := s.billingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	plan := &orderInvoicePlan{
		order:        ord,
		buckets:      buckets,
		customers:    make([]*customer.Customer, 0, len(buckets)),
		control:      control,
		defaultPayer: buckets[0].PayerID,
	}
	for _, bucket := range buckets {
		cus, cusErr := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
			ID:         bucket.PayerID,
			TenantInfo: req.TenantInfo,
			CustomerFilterOptions: repositories.CustomerFilterOptions{
				IncludeBillingProfile: true,
				IncludeState:          true,
			},
		})
		if cusErr != nil {
			return nil, cusErr
		}
		if cusErr = guardStatementCadence(cus, req.OffCycleReason); cusErr != nil {
			return nil, cusErr
		}
		plan.customers = append(plan.customers, cus)
	}

	return plan, nil
}

func (s *Service) buildBucketInvoice(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
	plan *orderInvoicePlan,
	idx int,
	anchor *billingqueue.BillingQueueItem,
	orderCustomer *customer.Customer,
) (*invoice.Invoice, *shipment.PayerShare, error) {
	bucket := plan.buckets[idx]
	cus := plan.customers[idx]

	charges, err := s.orderRepo.ListUninvoicedChargeSharesForPayer(
		ctx,
		&repositories.ListUninvoicedChargeSharesRequest{
			TenantInfo: req.TenantInfo,
			OrderID:    plan.order.ID,
			PayerID:    bucket.PayerID,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	chargeShare, chargeSplit, err := orderChargeShareFor(charges, plan.defaultPayer, bucket.PayerID)
	if err != nil {
		return nil, nil, err
	}

	var shipper *customer.Customer
	if bucket.PayerID != plan.order.CustomerID {
		shipper = orderCustomer
	}

	entity := s.buildInvoiceEntity(&buildInvoiceParams{
		Anchor:           anchor,
		Scope:            invoice.ScopeOrder,
		Customer:         cus,
		Control:          plan.control,
		Legs:             bucket.Legs,
		Order:            plan.order,
		OrderChargeShare: chargeShare,
		OffCycleReason:   offCycleReasonFor(cus, req.OffCycleReason),
		Shipper:          shipper,
		Shares:           bucket.Shares,
		IsSplitBill:      bucket.IsSplit || chargeSplit,
	})
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, nil, multiErr
	}

	return entity, chargeShare, nil
}

func (s *Service) sharedOrderOf(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromShipmentsRequest,
) (pulid.ID, error) {
	var orderID pulid.ID
	for _, shipmentID := range req.ShipmentIDs {
		shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:         shipmentID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return pulid.Nil, err
		}
		if shp.OrderID.IsNil() {
			return pulid.Nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"Grouped invoicing requires every shipment to belong to an order",
			)
		}
		if orderID.IsNil() {
			orderID = shp.OrderID
		} else if orderID != shp.OrderID {
			return pulid.Nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"All shipments in a grouped invoice must belong to the same order",
			)
		}
	}

	return orderID, nil
}

func unnumberedAnchor(
	tenantInfo pagination.TenantInfo,
	leg *shipment.Shipment,
	payerID pulid.ID,
	share *shipment.PayerShare,
) *billingqueue.BillingQueueItem {
	item := &billingqueue.BillingQueueItem{
		ID:               pulid.MustNew("bqi_"),
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		ShipmentID:       leg.ID,
		OrderID:          leg.OrderID,
		BillToCustomerID: payerID,
		Status:           billingqueue.StatusApproved,
		BillType:         billingqueue.BillTypeInvoice,
		Number:           unnumberedInvoice,
	}
	if share != nil {
		item.AllocatedTotalAmount = share.TotalAmount
	}

	return item
}

func (s *Service) PlanCreateInvoices(
	ctx context.Context,
	req *servicesports.PlanCreateInvoicesRequest,
) (*servicesports.CreateInvoicesPlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	plan, err := s.planCreateInvoices(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, draft := range plan.Drafts {
		draft.Number = ""
	}

	return plan, nil
}

func (s *Service) planCreateInvoices(
	ctx context.Context,
	req *servicesports.PlanCreateInvoicesRequest,
) (*servicesports.CreateInvoicesPlan, error) {
	switch {
	case req.OrderID.IsNotNil():
		return s.planOrderDrafts(ctx, &servicesports.CreateInvoiceFromOrderRequest{
			OrderID:        req.OrderID,
			ShipmentIDs:    req.ShipmentIDs,
			TenantInfo:     req.TenantInfo,
			OffCycleReason: req.OffCycleReason,
		})
	case len(req.ShipmentIDs) == 0:
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrRequired,
			"One shipment is required",
		)
	case len(req.ShipmentIDs) > 1:
		shipmentsReq := &servicesports.CreateInvoiceFromShipmentsRequest{
			ShipmentIDs:    req.ShipmentIDs,
			TenantInfo:     req.TenantInfo,
			OffCycleReason: req.OffCycleReason,
		}
		orderID, err := s.sharedOrderOf(ctx, shipmentsReq)
		if err != nil {
			return nil, err
		}
		return s.planOrderDrafts(ctx, &servicesports.CreateInvoiceFromOrderRequest{
			OrderID:        orderID,
			ShipmentIDs:    req.ShipmentIDs,
			TenantInfo:     req.TenantInfo,
			OffCycleReason: req.OffCycleReason,
		})
	default:
		return s.planShipmentDrafts(ctx, req)
	}
}

func (s *Service) planShipmentDrafts(
	ctx context.Context,
	req *servicesports.PlanCreateInvoicesRequest,
) (*servicesports.CreateInvoicesPlan, error) {
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

	plan := &servicesports.CreateInvoicesPlan{
		Drafts: make([]*invoice.Invoice, 0, len(resolution.Shares)),
	}
	for _, share := range resolution.Shares {
		cus, payerErr := s.checkPayer(ctx, req.TenantInfo, shp, share, req.OffCycleReason)
		if payerErr != nil {
			return nil, payerErr
		}
		anchor := unnumberedAnchor(req.TenantInfo, shp, share.PayerID, share)
		draft, draftErr := s.planDraftFromItem(ctx, &servicesports.CreateInvoiceFromBillingQueueRequest{
			BillingQueueItemID: anchor.ID,
			TenantInfo:         req.TenantInfo,
			OffCycleReason:     offCycleReasonFor(cus, req.OffCycleReason),
		}, anchor)
		if draftErr != nil {
			return nil, draftErr
		}
		if share.PayerID == shp.PayerID() {
			plan.PrimaryIndex = len(plan.Drafts)
		}
		plan.Drafts = append(plan.Drafts, draft.entity)
	}

	return plan, nil
}

func (s *Service) planOrderDrafts(
	ctx context.Context,
	req *servicesports.CreateInvoiceFromOrderRequest,
) (*servicesports.CreateInvoicesPlan, error) {
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

	plan, err := s.planOrderInvoices(ctx, req)
	if err != nil {
		return nil, err
	}

	drafts := &servicesports.CreateInvoicesPlan{
		Drafts: make([]*invoice.Invoice, 0, len(plan.buckets)),
	}
	for idx, bucket := range plan.buckets {
		anchor := unnumberedAnchor(req.TenantInfo, bucket.Legs[0], bucket.PayerID,
			bucket.Shares[bucket.Legs[0].ID])
		entity, _, buildErr := s.buildBucketInvoice(ctx, req, plan, idx, anchor, orderCustomer)
		if buildErr != nil {
			return nil, buildErr
		}
		drafts.Drafts = append(drafts.Drafts, entity)
	}

	return drafts, nil
}

package orderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// OrderChange is an order before and after a change to it; Before is nil for
// one being made. Totals are re-derived when the write lands, so a plan shows
// the fields the write sets and says the total follows.
type OrderChange struct {
	Before *order.Order
	After  *order.Order
}

func (s *Service) PlanCreate(ctx context.Context, entity *order.Order) (*OrderChange, error) {
	entity.Status = order.StatusDraft
	if entity.CurrencyCode == "" {
		entity.CurrencyCode = "USD"
	}
	if entity.OrderNumber == "" {
		if multiErr := s.validateCreateWithoutNumber(ctx, entity); multiErr != nil {
			return nil, multiErr
		}

		return &OrderChange{After: entity}, nil
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	return &OrderChange{After: entity}, nil
}

// validateCreateWithoutNumber checks an order whose number the write will
// issue: the uniqueness rule has nothing to compare yet, so the placeholder
// is dropped from what the validator reports.
func (s *Service) validateCreateWithoutNumber(
	ctx context.Context,
	entity *order.Order,
) *errortypes.MultiError {
	probe := *entity
	probe.OrderNumber = pulid.MustNew("ord_").String()
	multiErr := s.validator.ValidateCreate(ctx, &probe)
	if multiErr == nil {
		return nil
	}

	kept := errortypes.NewMultiError()
	for _, entry := range multiErr.Errors {
		if entry.Field == "orderNumber" {
			continue
		}
		kept.Add(entry.Field, entry.Code, entry.Message, entry.Args...)
	}
	if kept.HasErrors() {
		return kept
	}

	return nil
}

func (s *Service) PlanUpdate(ctx context.Context, entity *order.Order) (*OrderChange, error) {
	original, err := s.planUpdate(ctx, entity)
	if err != nil {
		return nil, err
	}

	return &OrderChange{Before: original, After: entity}, nil
}

func (s *Service) planUpdate(ctx context.Context, entity *order.Order) (*order.Order, error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.GetOrganizationID(),
		BuID:  entity.GetBusinessUnitID(),
	}
	original, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         entity.GetID(),
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity.Status = original.Status
	entity.TotalAmount = original.TotalAmount
	entity.OrderNumber = original.OrderNumber

	if entity.CustomerID != original.CustomerID {
		statuses, legErr := s.repo.GetShipmentStatuses(ctx, tenantInfo, entity.ID)
		if legErr != nil {
			return nil, legErr
		}
		if len(statuses) > 0 {
			return nil, errortypes.NewValidationError(
				"customerId",
				errortypes.ErrInvalid,
				"The customer cannot be changed while the order has legs; detach them first",
			)
		}
	}

	return original, nil
}

// MembershipPlan is what attaching shipments to an order would do: the order,
// the shipments that actually move, and the orders they leave.
type MembershipPlan struct {
	Order          *order.Order
	ShipmentIDs    []pulid.ID
	SourceOrderIDs []pulid.ID
}

func (s *Service) PlanAttachShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentIDs []pulid.ID,
) (*MembershipPlan, error) {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = guardMembershipChange(ord); err != nil {
		return nil, err
	}

	attachIDs, sourceOrderIDs, err := s.validateAttachRefs(ctx, tenantInfo, ord, shipmentIDs)
	if err != nil {
		return nil, err
	}

	sources := make([]pulid.ID, 0, len(sourceOrderIDs))
	for id := range sourceOrderIDs {
		sources = append(sources, id)
	}

	return &MembershipPlan{Order: ord, ShipmentIDs: attachIDs, SourceOrderIDs: sources}, nil
}

// DetachPlan is what detaching a leg would do: the order, the leg, and the
// single-leg order it moves onto, numbered when the write lands.
type DetachPlan struct {
	Order       *order.Order
	Leg         *shipment.Shipment
	Replacement *order.Order
}

func (s *Service) PlanDetachShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentID pulid.ID,
) (*DetachPlan, error) {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenantInfo,
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}
	if err = guardMembershipChange(ord); err != nil {
		return nil, err
	}

	leg, err := detachableLeg(ord, shipmentID)
	if err != nil {
		return nil, err
	}

	return &DetachPlan{
		Order:       ord,
		Leg:         leg,
		Replacement: replacementOrder(ord, leg, ""),
	}, nil
}

func replacementOrder(ord *order.Order, leg *shipment.Shipment, number string) *order.Order {
	return &order.Order{
		OrganizationID: leg.OrganizationID,
		BusinessUnitID: leg.BusinessUnitID,
		CustomerID:     leg.CustomerID,
		OwnerID:        ord.OwnerID,
		EnteredByID:    ord.EnteredByID,
		Status:         order.StatusConfirmed,
		OrderNumber:    number,
		CurrencyCode:   ord.CurrencyCode,
		TotalAmount:    leg.TotalChargeAmount,
	}
}

// ChargePlan is an order charge as a write would leave it; Before is nil for
// one being added.
type ChargePlan struct {
	Order  *order.Order
	Before *order.OrderCharge
	After  *order.OrderCharge
}

func (s *Service) PlanAddCharge(
	ctx context.Context,
	req *AddChargeRequest,
) (*ChargePlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         req.OrderID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = guardMembershipChange(ord); err != nil {
		return nil, err
	}

	charge := &order.OrderCharge{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrderID:        req.OrderID,
		Description:    req.Description,
		Amount:         req.Amount,
		Allocations:    req.Allocations,
	}
	multiErr := errortypes.NewMultiError()
	charge.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if multiErr = s.validateChargeAllocations(
		ctx,
		req.TenantInfo,
		charge,
		req.Allocations,
	); multiErr != nil {
		return nil, multiErr
	}

	return &ChargePlan{Order: ord, After: charge}, nil
}

func (s *Service) PlanUpdateCharge(
	ctx context.Context,
	req *UpdateChargeRequest,
) (*ChargePlan, error) {
	ord, current, err := s.loadCharge(ctx, req.TenantInfo, req.OrderID, req.ChargeID)
	if err != nil {
		return nil, err
	}
	if current.InvoiceID.IsNotNil() || current.IsFullyInvoiced() {
		return nil, errortypes.NewValidationError(
			"chargeId",
			errortypes.ErrInvalidOperation,
			"An invoiced order charge cannot be changed",
		)
	}

	charge := &order.OrderCharge{
		ID:             req.ChargeID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		OrderID:        req.OrderID,
		Description:    req.Description,
		Amount:         req.Amount,
		Version:        req.Version,
		Allocations:    req.Allocations,
	}
	if charge.Allocations == nil {
		charge.Allocations = current.Allocations
	}
	multiErr := errortypes.NewMultiError()
	charge.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if multiErr = s.validateChargeAllocations(
		ctx,
		req.TenantInfo,
		charge,
		req.Allocations,
	); multiErr != nil {
		return nil, multiErr
	}

	return &ChargePlan{Order: ord, Before: current, After: charge}, nil
}

func (s *Service) PlanRemoveCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	chargeID pulid.ID,
) (*ChargePlan, error) {
	ord, current, err := s.loadCharge(ctx, tenantInfo, orderID, chargeID)
	if err != nil {
		return nil, err
	}
	if current.InvoiceID.IsNotNil() || current.IsFullyInvoiced() {
		return nil, errortypes.NewNotFoundError(
			"Order charge not found or already carried on an invoice",
		)
	}

	return &ChargePlan{Order: ord, Before: current}, nil
}

func (s *Service) PlanSetChargeAllocations(
	ctx context.Context,
	req *SetChargeAllocationsRequest,
) (*ChargePlan, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	allocations := req.Allocations
	if allocations == nil {
		allocations = []*shipment.ChargeAllocation{}
	}

	ord, current, err := s.loadCharge(ctx, req.TenantInfo, req.OrderID, req.ChargeID)
	if err != nil {
		return nil, err
	}
	if current.InvoiceID.IsNotNil() || current.IsFullyInvoiced() {
		return nil, errortypes.NewValidationError(
			"chargeId",
			errortypes.ErrInvalidOperation,
			"An invoiced order charge cannot be re-allocated",
		)
	}
	if multiErr := s.validateChargeAllocations(
		ctx,
		req.TenantInfo,
		current,
		allocations,
	); multiErr != nil {
		return nil, multiErr
	}

	after := *current
	after.Allocations = allocations

	return &ChargePlan{Order: ord, Before: current, After: &after}, nil
}

func (s *Service) loadCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	chargeID pulid.ID,
) (*order.Order, *order.OrderCharge, error) {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if err = guardMembershipChange(ord); err != nil {
		return nil, nil, err
	}

	charges, err := s.repo.ListCharges(ctx, tenantInfo, orderID)
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range charges {
		if candidate != nil && candidate.ID == chargeID {
			return ord, candidate, nil
		}
	}

	return nil, nil, errortypes.NewNotFoundError("Order charge not found")
}

func (s *Service) PlanClose(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) (*OrderChange, error) {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if ord.Status != order.StatusBilled {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalidOperation,
			"Only a Billed order can be closed; this order is {0}", ord.Status,
		)
	}

	after := *ord
	after.Status = order.StatusClosed

	return &OrderChange{Before: ord, After: &after}, nil
}

// CancelPlan is an order cancellation before it runs: the order and the legs
// the cancellation cancels with it.
type CancelPlan struct {
	Order *order.Order
	Legs  []*shipment.Shipment
}

func (s *Service) PlanCancel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) (*CancelPlan, error) {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenantInfo,
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}
	if !ord.Status.AllowsMembershipChange() {
		return nil, errortypes.NewValidationError(
			"orderId",
			errortypes.ErrInvalidOperation,
			"A {0} order cannot be canceled", ord.Status,
		)
	}
	legs := make([]*shipment.Shipment, 0, len(ord.Shipments))
	for _, leg := range ord.Shipments {
		if leg == nil {
			continue
		}
		if leg.Status == shipment.StatusInvoiced {
			return nil, errortypes.NewValidationError(
				"orderId",
				errortypes.ErrInvalidOperation,
				"The order has invoiced legs; adjust or credit the invoice before canceling",
			)
		}
		if leg.Status != shipment.StatusCanceled {
			legs = append(legs, leg)
		}
	}

	return &CancelPlan{Order: ord, Legs: legs}, nil
}

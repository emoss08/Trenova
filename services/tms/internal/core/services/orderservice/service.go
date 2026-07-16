package orderservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger          *zap.Logger
	DB              ports.DBConnection
	Repo            repositories.OrderRepository
	Validator       *Validator
	AuditService    services.AuditService
	Generator       seqgen.Generator
	OrderDerivation services.OrderDerivationService
	ShipmentService services.ShipmentService
}

type Service struct {
	l               *zap.Logger
	db              ports.DBConnection
	repo            repositories.OrderRepository
	validator       *Validator
	auditService    services.AuditService
	generator       seqgen.Generator
	orderDerivation services.OrderDerivationService
	shipmentService services.ShipmentService
}

//nolint:gocritic // fx constructor
func New(p Params) *Service {
	return &Service{
		l:               p.Logger.Named("service.order"),
		db:              p.DB,
		repo:            p.Repo,
		validator:       p.Validator,
		auditService:    p.AuditService,
		generator:       p.Generator,
		orderDerivation: p.OrderDerivation,
		shipmentService: p.ShipmentService,
	}
}

func guardMembershipChange(ord *order.Order) error {
	if ord.Status.AllowsMembershipChange() {
		return nil
	}

	return errortypes.NewValidationError(
		"orderId",
		errortypes.ErrInvalidOperation,
		fmt.Sprintf(
			"The legs and charges of a %s order cannot be modified",
			ord.Status,
		),
	)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListOrdersRequest,
) (*pagination.ListResult[*order.Order], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListOrdersConnectionRequest,
) (*pagination.CursorListResult[*order.Order], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetOrderByIDRequest,
) (*order.Order, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.OrderSelectOptionsRequest,
) (*pagination.ListResult[*order.Order], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req repositories.GetOrdersByIDsRequest,
) ([]*order.Order, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *order.Order,
	actor *services.RequestActor,
) (*order.Order, error) {
	auditActor := actor.AuditActor()

	// Orders always begin life as a Draft quote; the status lifecycle is derived from
	// the legs afterwards and is never client-writable.
	entity.Status = order.StatusDraft

	if entity.OrderNumber == "" {
		number, err := s.generator.GenerateOrderNumber(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			"",
			"",
		)
		if err != nil {
			s.l.Error("failed to generate order number", zap.Error(err))
			return nil, err
		}
		entity.OrderNumber = number
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create order", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceOrder,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Order created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) AttachShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentIDs []pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
	var updated *order.Order
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		ord, txErr := s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = guardMembershipChange(ord); txErr != nil {
			return txErr
		}

		attachIDs, sourceOrderIDs, txErr := s.validateAttachRefs(
			txCtx,
			tenantInfo,
			ord,
			shipmentIDs,
		)
		if txErr != nil {
			return txErr
		}

		if len(attachIDs) > 0 {
			if txErr = s.executeAttach(
				txCtx,
				tenantInfo,
				orderID,
				attachIDs,
				sourceOrderIDs,
			); txErr != nil {
				return txErr
			}
		}

		updated, txErr = s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              orderID,
			TenantInfo:      tenantInfo,
			IncludeShipment: true,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Shipments attached to order")
	return updated, nil
}

// executeAttach moves the legs onto the order, recomputes the target, and settles
// every vacated source order (usually single-leg auto-orders): empty leftovers are
// removed, survivors are re-derived.
func (s *Service) executeAttach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	attachIDs []pulid.ID,
	sourceOrderIDs map[pulid.ID]struct{},
) error {
	affected, err := s.repo.AttachShipments(ctx, tenantInfo, orderID, attachIDs)
	if err != nil {
		s.l.Error("failed to attach shipments to order", zap.Error(err))
		return err
	}
	if affected != int64(len(attachIDs)) {
		return errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"One or more shipments changed while attaching; retry the request",
		)
	}

	if err = s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
		return err
	}
	for sourceOrderID := range sourceOrderIDs {
		if _, err = s.repo.DeleteIfEmpty(ctx, tenantInfo, sourceOrderID); err != nil {
			return err
		}
		if err = s.recomputeIfExists(ctx, tenantInfo, sourceOrderID); err != nil {
			return err
		}
	}

	return nil
}

// validateAttachRefs enforces the attach invariants: every requested shipment exists,
// shares the order's customer (invariant #4), is not Canceled/Invoiced, and does not
// belong to a Billed/Closed order. It returns the ids that actually need attaching
// (legs already on the order are skipped) and the distinct source orders being
// vacated.
func (s *Service) validateAttachRefs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ord *order.Order,
	shipmentIDs []pulid.ID,
) ([]pulid.ID, map[pulid.ID]struct{}, error) {
	refs, err := s.repo.GetShipmentAttachRefs(ctx, tenantInfo, shipmentIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(refs) != len(shipmentIDs) {
		return nil, nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"One or more shipments were not found",
		)
	}

	attachIDs := make([]pulid.ID, 0, len(refs))
	sourceOrderIDs := make(map[pulid.ID]struct{}, len(refs))
	for _, ref := range refs {
		if ref.CustomerID != ord.CustomerID {
			return nil, nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"Every shipment attached to an order must belong to the order's customer",
			)
		}
		if ref.Status == shipment.StatusCanceled || ref.Status == shipment.StatusInvoiced {
			return nil, nil, errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				"Canceled or invoiced shipments cannot be attached to an order",
			)
		}
		if ref.OrderID == ord.ID {
			continue
		}
		attachIDs = append(attachIDs, ref.ID)
		if !ref.OrderID.IsNil() {
			sourceOrderIDs[ref.OrderID] = struct{}{}
		}
	}

	if err = s.guardSourceOrders(ctx, tenantInfo, sourceOrderIDs); err != nil {
		return nil, nil, err
	}

	return attachIDs, sourceOrderIDs, nil
}

func (s *Service) guardSourceOrders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sourceOrderIDs map[pulid.ID]struct{},
) error {
	if len(sourceOrderIDs) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(sourceOrderIDs))
	for id := range sourceOrderIDs {
		ids = append(ids, id)
	}
	sources, err := s.repo.GetByIDs(ctx, repositories.GetOrdersByIDsRequest{
		TenantInfo: tenantInfo,
		OrderIDs:   ids,
	})
	if err != nil {
		return err
	}
	for _, source := range sources {
		if source == nil {
			continue
		}
		if !source.Status.AllowsMembershipChange() {
			return errortypes.NewValidationError(
				"shipmentIds",
				errortypes.ErrInvalid,
				fmt.Sprintf(
					"Shipment belongs to order %s (%s) and cannot be moved",
					source.OrderNumber,
					source.Status,
				),
			)
		}
	}

	return nil
}

func (s *Service) recomputeIfExists(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) error {
	err := s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID)
	if err != nil && errortypes.IsNotFoundError(err) {
		return nil
	}
	return err
}

func (s *Service) DetachShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentID pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
	replacementNumber, err := s.generator.GenerateOrderNumber(
		ctx,
		tenantInfo.OrgID,
		tenantInfo.BuID,
		"",
		"",
	)
	if err != nil {
		s.l.Error("failed to generate replacement order number", zap.Error(err))
		return nil, err
	}

	var updated *order.Order
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		ord, txErr := s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              orderID,
			TenantInfo:      tenantInfo,
			IncludeShipment: true,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = guardMembershipChange(ord); txErr != nil {
			return txErr
		}

		leg, txErr := detachableLeg(ord, shipmentID)
		if txErr != nil {
			return txErr
		}

		// Every shipment keeps a commercial parent: the detached leg moves onto a
		// fresh single-leg order seeded from the leg itself.
		replacement := &order.Order{
			OrganizationID: leg.OrganizationID,
			BusinessUnitID: leg.BusinessUnitID,
			CustomerID:     leg.CustomerID,
			OwnerID:        ord.OwnerID,
			EnteredByID:    ord.EnteredByID,
			Status:         order.StatusConfirmed,
			OrderNumber:    replacementNumber,
			CurrencyCode:   ord.CurrencyCode,
			TotalAmount:    leg.TotalChargeAmount,
		}
		if _, txErr = s.repo.Create(txCtx, replacement); txErr != nil {
			return txErr
		}

		affected, txErr := s.repo.DetachShipment(
			txCtx,
			tenantInfo,
			orderID,
			shipmentID,
			replacement.ID,
		)
		if txErr != nil {
			s.l.Error("failed to detach shipment from order", zap.Error(txErr))
			return txErr
		}
		if affected == 0 {
			return errortypes.NewValidationError(
				"shipmentId",
				errortypes.ErrInvalid,
				"Shipment changed while detaching; retry the request",
			)
		}

		if txErr = s.orderDerivation.RecomputeOrder(txCtx, tenantInfo, orderID); txErr != nil {
			return txErr
		}
		if txErr = s.orderDerivation.RecomputeOrder(
			txCtx,
			tenantInfo,
			replacement.ID,
		); txErr != nil {
			return txErr
		}

		updated, txErr = s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              orderID,
			TenantInfo:      tenantInfo,
			IncludeShipment: true,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Shipment detached from order")
	return updated, nil
}

// detachableLeg finds the shipment on the order and enforces the detach guards: the
// leg must exist, must not be invoiced, and must not be the order's only leg.
func detachableLeg(ord *order.Order, shipmentID pulid.ID) (*shipment.Shipment, error) {
	var leg *shipment.Shipment
	for _, candidate := range ord.Shipments {
		if candidate != nil && candidate.ID == shipmentID {
			leg = candidate
			break
		}
	}
	if leg == nil {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Shipment is not a leg of this order",
		)
	}
	if leg.Status == shipment.StatusInvoiced {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Invoiced legs cannot be detached from their order",
		)
	}
	if len(ord.Shipments) <= 1 {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"The only leg of an order cannot be detached",
		)
	}

	return leg, nil
}

func (s *Service) logMembershipChange(
	updated *order.Order,
	actor *services.RequestActor,
	comment string,
) {
	if updated == nil {
		return
	}

	auditActor := actor.AuditActor()
	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceOrder,
		ResourceID:     updated.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) AddCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	description string,
	amount decimal.Decimal,
	actor *services.RequestActor,
) (*order.Order, error) {
	var updated *order.Order
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		ord, txErr := s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = guardMembershipChange(ord); txErr != nil {
			return txErr
		}

		charge := &order.OrderCharge{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			OrderID:        orderID,
			Description:    description,
			Amount:         amount,
		}
		multiErr := errortypes.NewMultiError()
		charge.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		if _, txErr = s.repo.AddCharge(txCtx, charge); txErr != nil {
			s.l.Error("failed to add order charge", zap.Error(txErr))
			return txErr
		}

		if txErr = s.orderDerivation.RecomputeOrder(txCtx, tenantInfo, orderID); txErr != nil {
			return txErr
		}

		updated, txErr = s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              orderID,
			TenantInfo:      tenantInfo,
			IncludeShipment: true,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Order charge added")
	return updated, nil
}

func (s *Service) RemoveCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	chargeID pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
	var updated *order.Order
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		ord, txErr := s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:         orderID,
			TenantInfo: tenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = guardMembershipChange(ord); txErr != nil {
			return txErr
		}

		affected, txErr := s.repo.RemoveCharge(txCtx, &repositories.RemoveOrderChargeRequest{
			TenantInfo: tenantInfo,
			OrderID:    orderID,
			ChargeID:   chargeID,
		})
		if txErr != nil {
			s.l.Error("failed to remove order charge", zap.Error(txErr))
			return txErr
		}
		if affected == 0 {
			return errortypes.NewNotFoundError(
				"Order charge not found or already carried on an invoice",
			)
		}

		if txErr = s.orderDerivation.RecomputeOrder(txCtx, tenantInfo, orderID); txErr != nil {
			return txErr
		}

		updated, txErr = s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              orderID,
			TenantInfo:      tenantInfo,
			IncludeShipment: true,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Order charge removed")
	return updated, nil
}

func (s *Service) UpdateCharge(
	ctx context.Context,
	req *UpdateChargeRequest,
	actor *services.RequestActor,
) (*order.Order, error) {
	var updated *order.Order
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		ord, txErr := s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:         req.OrderID,
			TenantInfo: req.TenantInfo,
		})
		if txErr != nil {
			return txErr
		}
		if txErr = guardMembershipChange(ord); txErr != nil {
			return txErr
		}

		charge := &order.OrderCharge{
			ID:             req.ChargeID,
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			OrderID:        req.OrderID,
			Description:    req.Description,
			Amount:         req.Amount,
			Version:        req.Version,
		}
		multiErr := errortypes.NewMultiError()
		charge.Validate(multiErr)
		if multiErr.HasErrors() {
			return multiErr
		}

		affected, txErr := s.repo.UpdateCharge(txCtx, charge)
		if txErr != nil {
			s.l.Error("failed to update order charge", zap.Error(txErr))
			return txErr
		}
		if affected == 0 {
			return errortypes.NewValidationError(
				"chargeId",
				errortypes.ErrInvalid,
				"Order charge was modified by someone else, is already invoiced, or does not exist",
			)
		}

		if txErr = s.orderDerivation.RecomputeOrder(
			txCtx,
			req.TenantInfo,
			req.OrderID,
		); txErr != nil {
			return txErr
		}

		updated, txErr = s.repo.GetByID(txCtx, repositories.GetOrderByIDRequest{
			ID:              req.OrderID,
			TenantInfo:      req.TenantInfo,
			IncludeShipment: true,
		})
		return txErr
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Order charge updated")
	return updated, nil
}

type UpdateChargeRequest struct {
	TenantInfo  pagination.TenantInfo
	OrderID     pulid.ID
	ChargeID    pulid.ID
	Description string
	Amount      decimal.Decimal
	Version     int64
}

func (s *Service) ListCharges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	return s.repo.ListCharges(ctx, tenantInfo, orderID)
}

// Close settles a Billed order — the terminal step of the AR lifecycle. It is the
// only writer of StatusClosed in the system.
func (s *Service) Close(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
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
			fmt.Sprintf("Only a Billed order can be closed; this order is %s", ord.Status),
		)
	}

	updated, err := s.repo.UpdateStatus(ctx, &repositories.UpdateOrderStatusRequest{
		TenantInfo: tenantInfo,
		OrderID:    orderID,
		Status:     order.StatusClosed,
		Version:    ord.Version,
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Order closed")
	return s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenantInfo,
		IncludeShipment: true,
	})
}

// Cancel cancels every remaining active leg of the order and derives the order to
// Canceled. Billed and Closed orders must go through the invoice-adjustment flow
// instead; an order with invoiced legs cannot be canceled outright.
func (s *Service) Cancel(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	cancelReason string,
	actor *services.RequestActor,
) (*order.Order, error) {
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
			fmt.Sprintf("A %s order cannot be canceled", ord.Status),
		)
	}
	for _, leg := range ord.Shipments {
		if leg != nil && leg.Status == shipment.StatusInvoiced {
			return nil, errortypes.NewValidationError(
				"orderId",
				errortypes.ErrInvalidOperation,
				"The order has invoiced legs; adjust or credit the invoice before canceling",
			)
		}
	}

	now := timeutils.NowUnix()
	for _, leg := range ord.Shipments {
		if leg == nil || leg.Status == shipment.StatusCanceled {
			continue
		}
		if _, err = s.shipmentService.Cancel(ctx, &repositories.CancelShipmentRequest{
			TenantInfo:   tenantInfo,
			ShipmentID:   leg.ID,
			CanceledByID: actor.AuditActor().UserID,
			CanceledAt:   now,
			CancelReason: cancelReason,
		}, actor); err != nil {
			return nil, err
		}
	}

	if err = s.orderDerivation.RecomputeOrder(ctx, tenantInfo, orderID); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:              orderID,
		TenantInfo:      tenantInfo,
		IncludeShipment: true,
	})
	if err != nil {
		return nil, err
	}

	s.logMembershipChange(updated, actor, "Order canceled")
	return updated, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *order.Order,
	actor *services.RequestActor,
) (*order.Order, error) {
	auditActor := actor.AuditActor()

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		s.l.Error("failed to get original order", zap.Error(err))
		return nil, err
	}

	// Status and total are derived; order number is immutable. Whatever the transport
	// bound onto the entity is discarded here.
	entity.Status = original.Status
	entity.TotalAmount = original.TotalAmount
	entity.OrderNumber = original.OrderNumber

	if entity.CustomerID != original.CustomerID {
		statuses, legErr := s.repo.GetShipmentStatuses(
			ctx,
			pagination.TenantInfo{
				OrgID: entity.GetOrganizationID(),
				BuID:  entity.GetBusinessUnitID(),
			},
			entity.ID,
		)
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

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update order", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceOrder,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Order updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

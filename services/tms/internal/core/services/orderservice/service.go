package orderservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.OrderRepository
	Validator    *Validator
	AuditService services.AuditService
	Generator    seqgen.Generator
}

type Service struct {
	l            *zap.Logger
	repo         repositories.OrderRepository
	validator    *Validator
	auditService services.AuditService
	generator    seqgen.Generator
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.order"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		generator:    p.Generator,
	}
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
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	// Invariant #4: every leg must share the order's customer.
	mismatched, err := s.repo.CountShipmentsWithDifferentCustomer(
		ctx,
		tenantInfo,
		ord.CustomerID,
		shipmentIDs,
	)
	if err != nil {
		return nil, err
	}
	if mismatched > 0 {
		return nil, errortypes.NewValidationError(
			"shipmentIds",
			errortypes.ErrInvalid,
			"Every shipment attached to an order must belong to the order's customer",
		)
	}

	if _, err = s.repo.AttachShipments(ctx, tenantInfo, orderID, shipmentIDs); err != nil {
		s.l.Error("failed to attach shipments to order", zap.Error(err))
		return nil, err
	}

	return s.finishMembershipChange(ctx, tenantInfo, orderID, actor, "Shipments attached to order")
}

func (s *Service) DetachShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	shipmentID pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
	if _, err := s.repo.DetachShipment(ctx, tenantInfo, orderID, shipmentID); err != nil {
		s.l.Error("failed to detach shipment from order", zap.Error(err))
		return nil, err
	}

	return s.finishMembershipChange(ctx, tenantInfo, orderID, actor, "Shipment detached from order")
}

// finishMembershipChange re-derives the order status after its set of legs changes
// (attach/detach do not emit shipment events, so the derivation observer would not
// otherwise fire) and returns the refreshed order with its legs.
func (s *Service) finishMembershipChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	actor *services.RequestActor,
	comment string,
) (*order.Order, error) {
	if err := s.recomputeStatus(ctx, tenantInfo, orderID); err != nil {
		s.l.Error("failed to recompute order status", zap.Error(err))
		return nil, err
	}

	// Invariant #1: money rolls up from the legs and order-level charges.
	if err := s.repo.RecalculateTotal(ctx, tenantInfo, orderID); err != nil {
		s.l.Error("failed to recalculate order total", zap.Error(err))
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

	auditActor := actor.AuditActor()
	if err = s.auditService.LogAction(&services.LogActionParams{
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

	return updated, nil
}

func (s *Service) AddCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	description string,
	amount decimal.Decimal,
	actor *services.RequestActor,
) (*order.Order, error) {
	if _, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return nil, err
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
		return nil, multiErr
	}

	if _, err := s.repo.AddCharge(ctx, charge); err != nil {
		s.l.Error("failed to add order charge", zap.Error(err))
		return nil, err
	}

	return s.finishMembershipChange(ctx, tenantInfo, orderID, actor, "Order charge added")
}

func (s *Service) RemoveCharge(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
	chargeID pulid.ID,
	actor *services.RequestActor,
) (*order.Order, error) {
	if _, err := s.repo.RemoveCharge(ctx, tenantInfo, orderID, chargeID); err != nil {
		s.l.Error("failed to remove order charge", zap.Error(err))
		return nil, err
	}

	return s.finishMembershipChange(ctx, tenantInfo, orderID, actor, "Order charge removed")
}

func (s *Service) ListCharges(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) ([]*order.OrderCharge, error) {
	return s.repo.ListCharges(ctx, tenantInfo, orderID)
}

func (s *Service) recomputeStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	orderID pulid.ID,
) error {
	ord, err := s.repo.GetByID(ctx, repositories.GetOrderByIDRequest{
		ID:         orderID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if ord.Status == order.StatusClosed {
		return nil
	}

	statuses, err := s.repo.GetShipmentStatuses(ctx, tenantInfo, orderID)
	if err != nil {
		return err
	}

	next := order.Derive(statuses)
	if next == ord.Status {
		return nil
	}

	_, err = s.repo.UpdateStatus(ctx, &repositories.UpdateOrderStatusRequest{
		TenantInfo: tenantInfo,
		OrderID:    orderID,
		Status:     next,
		Version:    ord.Version,
	})
	return err
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

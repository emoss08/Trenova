package shipmentholdservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.ShipmentHoldRepository
	ShipmentRepo   repositories.ShipmentRepository
	HoldReasonRepo repositories.HoldReasonRepository
	AuditService   services.AuditService
	Realtime       services.RealtimeService
}

type service struct {
	l              *zap.Logger
	repo           repositories.ShipmentHoldRepository
	shipmentRepo   repositories.ShipmentRepository
	holdReasonRepo repositories.HoldReasonRepository
	auditService   services.AuditService
	realtime       services.RealtimeService
}

//nolint:gocritic // service constructor
func New(p Params) services.ShipmentHoldService {
	return &service{
		l:              p.Logger.Named("service.shipment-hold"),
		repo:           p.Repo,
		shipmentRepo:   p.ShipmentRepo,
		holdReasonRepo: p.HoldReasonRepo,
		auditService:   p.AuditService,
		realtime:       p.Realtime,
	}
}

func (s *service) ListByShipmentID(
	ctx context.Context,
	req *repositories.ListShipmentHoldsRequest,
) (*pagination.ListResult[*shipment.ShipmentHold], error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Shipment hold list request is required",
		)
	}
	if req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}

	if err := s.ensureShipmentExists(ctx, req.ShipmentID, req.Filter.TenantInfo); err != nil {
		return nil, err
	}

	return s.repo.ListByShipmentID(ctx, req)
}

func (s *service) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentHoldByIDRequest,
) (*shipment.ShipmentHold, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Shipment hold request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.repo.GetByID(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	req *repositories.CreateShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Shipment hold request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	userID, err := requireHoldUser(actor)
	if err != nil {
		return nil, err
	}

	if err = s.ensureShipmentExists(ctx, req.ShipmentID, req.TenantInfo); err != nil {
		return nil, err
	}

	reason, err := s.getActiveHoldReason(ctx, req.HoldReasonID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	startedAt := timeutils.NowUnix()
	if req.StartedAt != nil {
		startedAt = *req.StartedAt
	}

	entity := &shipment.ShipmentHold{
		ShipmentID:        req.ShipmentID,
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		HoldReasonID:      idPtr(reason.ID),
		Type:              reason.Type,
		Severity:          reason.DefaultSeverity,
		ReasonCode:        reason.Code,
		Notes:             strings.TrimSpace(req.Notes),
		Source:            shipment.HoldSourceUser,
		BlocksDispatch:    reason.DefaultBlocksDispatch,
		BlocksDelivery:    reason.DefaultBlocksDelivery,
		BlocksBilling:     reason.DefaultBlocksBilling,
		VisibleToCustomer: reason.DefaultVisibleToCustomer,
		StartedAt:         startedAt,
		CreatedByID:       idPtr(userID),
	}
	applyCreateOverrides(entity, req)

	if multiErr := validateHold(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logHoldAction(created, auditActor, permission.OpCreate, nil, created, "Shipment hold created")
	s.publishHoldInvalidation(ctx, created, auditActor, "created", created)

	return created, nil
}

func (s *service) Update(
	ctx context.Context,
	req *repositories.UpdateShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Shipment hold request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	if _, err := requireHoldUser(actor); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     req.HoldID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewBusinessError("Only active shipment holds can be updated").
			WithParam("holdId", req.HoldID.String())
	}

	updated := *original
	updated.StartedAt = req.StartedAt
	updated.Severity = req.Severity
	updated.Notes = strings.TrimSpace(req.Notes)
	updated.BlocksDispatch = req.BlocksDispatch
	updated.BlocksDelivery = req.BlocksDelivery
	updated.BlocksBilling = req.BlocksBilling
	updated.VisibleToCustomer = req.VisibleToCustomer
	updated.Version = req.Version

	if multiErr := validateHold(&updated); multiErr != nil {
		return nil, multiErr
	}

	saved, err := s.repo.Update(ctx, &updated)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logHoldAction(
		saved,
		auditActor,
		permission.OpUpdate,
		original,
		saved,
		"Shipment hold updated",
	)
	s.publishHoldInvalidation(ctx, saved, auditActor, "updated", saved)

	return saved, nil
}

func (s *service) Release(
	ctx context.Context,
	req *repositories.ReleaseShipmentHoldRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentHold, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Shipment hold request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	userID, err := requireHoldUser(actor)
	if err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		HoldID:     req.HoldID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !original.IsActive() {
		return nil, errortypes.NewBusinessError("Shipment hold is already released").
			WithParam("holdId", req.HoldID.String())
	}

	releasedAt := timeutils.NowUnix()
	toRelease := *original
	toRelease.ReleasedAt = &releasedAt
	toRelease.ReleasedByID = idPtr(userID)

	released, err := s.repo.Release(ctx, &toRelease)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	s.logHoldAction(
		released,
		auditActor,
		permission.OpUpdate,
		original,
		released,
		"Shipment hold released",
	)
	s.publishHoldInvalidation(ctx, released, auditActor, "released", released)

	return released, nil
}

func (s *service) ensureShipmentExists(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	_, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	return err
}

func (s *service) getActiveHoldReason(
	ctx context.Context,
	holdReasonID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*holdreason.HoldReason, error) {
	reason, err := s.holdReasonRepo.GetByID(ctx, repositories.GetHoldReasonByIDRequest{
		ID:         holdReasonID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !reason.Active {
		return nil, errortypes.NewValidationError(
			"holdReasonId",
			errortypes.ErrInvalid,
			"Hold reason must be active",
		)
	}

	return reason, nil
}

func (s *service) logHoldAction(
	entity *shipment.ShipmentHold,
	actor services.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceShipmentHold,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId": entity.ShipmentID.String(),
			"holdId":     entity.ID.String(),
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log shipment hold action", zap.Error(err))
	}
}

func (s *service) publishHoldInvalidation(
	ctx context.Context,
	entity *shipment.ShipmentHold,
	actor services.AuditActor,
	action string,
	payload any,
) {
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceShipmentHold.String(),
		Action:         action,
		RecordID:       entity.ShipmentID,
		Entity:         payload,
	})
	if err != nil {
		s.l.Warn("failed to publish shipment hold invalidation", zap.Error(err))
	}
}

func applyCreateOverrides(
	entity *shipment.ShipmentHold,
	req *repositories.CreateShipmentHoldRequest,
) {
	if req.Severity != nil {
		entity.Severity = *req.Severity
	}
	if req.BlocksDispatch != nil {
		entity.BlocksDispatch = *req.BlocksDispatch
	}
	if req.BlocksDelivery != nil {
		entity.BlocksDelivery = *req.BlocksDelivery
	}
	if req.BlocksBilling != nil {
		entity.BlocksBilling = *req.BlocksBilling
	}
	if req.VisibleToCustomer != nil {
		entity.VisibleToCustomer = *req.VisibleToCustomer
	}
}

func validateHold(entity *shipment.ShipmentHold) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func requireHoldUser(actor *services.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError(
			"Shipment hold actions require a user actor",
		)
	}

	return actor.UserID, nil
}

func idPtr(id pulid.ID) *pulid.ID {
	if id.IsNil() {
		return nil
	}
	return &id
}

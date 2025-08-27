package shipmenthold

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger              *logger.Logger
	Repo                repositories.ShipmentHoldRepository
	HoldReasonRepo      repositories.HoldReasonRepository
	UserRepo            repositories.UserRepository
	ShipmentRepo        repositories.ShipmentRepository
	PermService         services.PermissionService
	NotificationService services.NotificationService
	AuditService        services.AuditService
	Validator           *shipmentvalidator.ShipmentHoldValidator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentHoldRepository
	hr   repositories.HoldReasonRepository
	ur   repositories.UserRepository
	ps   services.PermissionService
	ns   services.NotificationService
	as   services.AuditService
	sr   repositories.ShipmentRepository
	v    *shipmentvalidator.ShipmentHoldValidator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmenthold").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ur:   p.UserRepo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
		hr:   p.HoldReasonRepo,
		ns:   p.NotificationService,
		sr:   p.ShipmentRepo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListShipmentHoldOptions,
) (*ports.ListResult[*shipment.ShipmentHold], error) {
	log := s.l.With().
		Str("operation", "List").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.Filter.TenantOpts.UserID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionRead,
			BusinessUnitID: req.Filter.TenantOpts.BuID,
			OrganizationID: req.Filter.TenantOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read shipment holds within your organization",
		)
	}

	return s.repo.List(ctx, req)
}

func (s *Service) GetShipmentHoldByShipmentID(
	ctx context.Context,
	req *repositories.GetShipmentHoldByShipmentIDRequest,
) (*ports.ListResult[*shipment.ShipmentHold], error) {
	log := s.l.With().
		Str("operation", "GetShipmentHoldByShipmentID").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionRead,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read shipment holds within your organization",
		)
	}

	return s.repo.GetByShipmentID(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	hold *shipment.ShipmentHold,
	userID pulid.ID,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("holdID", hold.GetID()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionUpdate,
			BusinessUnitID: hold.BusinessUnitID,
			OrganizationID: hold.OrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update shipment holds within your organization",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, hold); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		ID:     hold.ID,
		OrgID:  hold.OrganizationID,
		BuID:   hold.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	updateHold, err := s.repo.Update(ctx, hold)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentHold,
			ResourceID:     updateHold.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updateHold),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updateHold.OrganizationID,
			BusinessUnitID: updateHold.BusinessUnitID,
		},
		audit.WithComment("Shipment hold updated"),
		audit.WithDiff(original, updateHold),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment hold update")
	}

	return updateHold, nil
}

func (s *Service) Create(
	ctx context.Context,
	hold *shipment.ShipmentHold,
	userID pulid.ID,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("shipmentID", hold.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionCreate,
			BusinessUnitID: hold.BusinessUnitID,
			OrganizationID: hold.OrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create shipment holds within your organization",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hold); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hold)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentHold,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment hold created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment hold creation")
	}

	return createdEntity, nil
}

func (s *Service) HoldShipment(
	ctx context.Context,
	req *repositories.HoldShipmentRequest,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "HoldShipment").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	if err := req.Validate(); err != nil {
		return nil, err
	}

	holdReason, err := s.hr.GetByID(ctx, &repositories.GetHoldReasonByIDRequest{
		ID:     req.HoldReasonID,
		OrgID:  req.OrgID,
		BuID:   req.BuID,
		UserID: req.UserID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get hold reason")
		return nil, err
	}

	hold := &shipment.ShipmentHold{
		ShipmentID:        req.ShipmentID,
		BusinessUnitID:    req.BuID,
		OrganizationID:    req.OrgID,
		Type:              req.Type,
		Severity:          req.Severity,
		Notes:             req.Notes,
		Source:            shipment.SourceUser,
		BlocksDispatch:    req.BlocksDispatch,
		BlocksDelivery:    req.BlocksDelivery,
		BlocksBilling:     req.BlocksBilling,
		VisibleToCustomer: req.VisibleToCustomer,
		Metadata:          holdReason.ExternalMap,
		ReasonCode:        holdReason.Code,
		CreatedByID:       &req.UserID,
		StartedAt:         timeutils.NowUnix(),
	}

	createdHold, err := s.Create(ctx, hold, req.UserID)
	if err != nil {
		return nil, err
	}

	return createdHold, nil
}

func (s *Service) ReleaseShipmentHold(
	ctx context.Context,
	req *repositories.ReleaseShipmentHoldRequest,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "ReleaseShipmentHold").
		Interface("request", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionUpdate,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to release shipment holds within your organization",
		)
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	hold, err := s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		ID:     req.HoldID,
		OrgID:  req.OrgID,
		BuID:   req.BuID,
		UserID: req.UserID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment hold")
		return nil, err
	}

	hold.ReleasedAt = timeutils.NowUnixPointer()
	hold.ReleasedByID = &req.UserID

	updatedHold, err := s.repo.Update(ctx, hold)
	if err != nil {
		return nil, err
	}

	if err = s.sendShipmentHoldReleaseNotification(ctx, updatedHold); err != nil {
		log.Error().Err(err).Msg("failed to send shipment hold release notification")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentHold,
			ResourceID:     updatedHold.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(updatedHold),
			PreviousState:  jsonutils.MustToJSON(hold),
			OrganizationID: updatedHold.OrganizationID,
			BusinessUnitID: updatedHold.BusinessUnitID,
		},
		audit.WithComment("Shipment hold released"),
		audit.WithDiff(hold, updatedHold),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment hold release")
	}

	return updatedHold, nil
}

func (s *Service) sendShipmentHoldReleaseNotification(
	ctx context.Context,
	hold *shipment.ShipmentHold,
) error {
	originalShipment, err := s.sr.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    hold.ShipmentID,
		OrgID: hold.OrganizationID,
		BuID:  hold.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return err
	}

	releasedBy, err := s.ur.GetNameByID(ctx, *hold.ReleasedByID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get released by user name")
		return err
	}

	if err = s.ns.SendShipmentHoldReleaseNotification(ctx, &services.ShipmentHoldReleaseNotificationRequest{
		OrgID:          hold.OrganizationID,
		BuID:           hold.BusinessUnitID,
		ProNumber:      originalShipment.ProNumber,
		ReleasedByName: releasedBy,
		TargetUserID:   pulid.ConvertFromPtr(originalShipment.OwnerID),
	}); err != nil {
		log.Error().Err(err).Msg("failed to send shipment hold release notification")
	}

	return nil
}

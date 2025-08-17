package shipmenthold

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger         *logger.Logger
	Repo           repositories.ShipmentHoldRepository
	HoldReasonRepo repositories.HoldReasonRepository
	PermService    services.PermissionService
	AuditService   services.AuditService
	Validator      *shipmentvalidator.ShipmentHoldValidator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentHoldRepository
	ps   services.PermissionService
	hr   repositories.HoldReasonRepository
	as   services.AuditService
	v    *shipmentvalidator.ShipmentHoldValidator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmenthold").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
		hr:   p.HoldReasonRepo,
	}
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

func (s *Service) Update(
	ctx context.Context,
	h *shipment.ShipmentHold,
	userID pulid.ID,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("shipmentID", h.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceShipmentHold,
			Action:         permission.ActionUpdate,
			BusinessUnitID: h.BusinessUnitID,
			OrganizationID: h.OrganizationID,
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
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, h); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
		ID:     h.ID,
		OrgID:  h.OrganizationID,
		BuID:   h.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, h)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment hold")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentHold,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment hold updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment hold update")
	}

	return updatedEntity, nil
}

func (s *Service) HoldShipment(
	ctx context.Context,
	req *repositories.HoldShipmentRequest,
) (*shipment.ShipmentHold, error) {
	log := s.l.With().
		Str("operation", "HoldShipment").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	log.Info().Interface("request", req).Msg("request")

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
		Type:              holdReason.Type,
		Severity:          holdReason.DefaultSeverity,
		ReasonCode:        holdReason.Code,
		Notes:             holdReason.Description,
		Source:            shipment.SourceUser,
		BlocksDispatch:    holdReason.DefaultBlocksDispatch,
		BlocksDelivery:    holdReason.DefaultBlocksDelivery,
		BlocksBilling:     holdReason.DefaultBlocksBilling,
		VisibleToCustomer: holdReason.DefaultVisibleToCustomer,
		Metadata:          holdReason.ExternalMap,
		CreatedByID:       &req.UserID,
		StartedAt:         time.Now().Unix(),
	}

	createdHold, err := s.Create(ctx, hold, req.UserID)
	if err != nil {
		return nil, err
	}

	return createdHold, nil
}

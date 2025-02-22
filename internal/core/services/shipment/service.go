package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/search"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger        *logger.Logger
	Repo          repositories.ShipmentRepository
	ProNumberRepo repositories.ProNumberRepository
	PermService   services.PermissionService
	AuditService  services.AuditService
	SearchService *search.Service
	Validator     *shipmentvalidator.Validator
}

type Service struct {
	l             *zerolog.Logger
	repo          repositories.ShipmentRepository
	proNumberRepo repositories.ProNumberRepository
	ps            services.PermissionService
	as            services.AuditService
	ss            *search.Service
	v             *shipmentvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipment").
		Logger()

	return &Service{
		l:             &log,
		repo:          p.Repo,
		proNumberRepo: p.ProNumberRepo,
		ps:            p.PermService,
		as:            p.AuditService,
		ss:            p.SearchService,
		v:             p.Validator,
	}
}

func (s *Service) SelectOptions(ctx context.Context, opts *repositories.ListShipmentOptions) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, t := range result.Items {
		options[i] = &types.SelectOption{
			Value: t.GetID(),
			Label: t.ProNumber,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *repositories.ListShipmentOptions) (*ports.ListResult[*shipment.Shipment], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read shipments")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list shipments")
		return nil, err
	}

	return &ports.ListResult[*shipment.Shipment]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetShipmentByIDOptions) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this shipment")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, shp *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", shp.ProNumber).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionCreate,
				BusinessUnitID: shp.BusinessUnitID,
				OrganizationID: shp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a shipment")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, shp); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, shp)
	if err != nil {
		return nil, err
	}

	if err = s.ss.Index(ctx, createdEntity); err != nil {
		log.Error().Err(err).Msg("failed to update search index")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, shp *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", shp.ProNumber).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionUpdate,
				BusinessUnitID: shp.BusinessUnitID,
				OrganizationID: shp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this shipment")
	}

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, shp); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetShipmentByIDOptions{
		ID:    shp.ID,
		OrgID: shp.OrganizationID,
		BuID:  shp.BusinessUnitID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, shp)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment")
		return nil, err
	}

	if err = s.ss.Index(ctx, updatedEntity); err != nil {
		log.Error().
			Err(err).
			Interface("shipment", updatedEntity).
			Msg("failed to update search index")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment update")
	}

	return updatedEntity, nil
}

func (s *Service) Cancel(ctx context.Context, req *repositories.CancelShipmentRequest) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.CanceledByID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionCancel,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to cancel this shipment")
	}

	// get the original shipment
	original, err := s.repo.GetByID(ctx, repositories.GetShipmentByIDOptions{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	if err := s.v.ValidateCancellation(original); err != nil {
		return nil, err
	}

	newEntity, err := s.repo.Cancel(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel shipment")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     req.ShipmentID.String(),
			Action:         permission.ActionCancel,
			UserID:         req.CanceledByID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Shipment canceled"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment cancellation")
	}

	return newEntity, nil
}

func (s *Service) Duplicate(ctx context.Context, req *repositories.DuplicateShipmentRequest) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Duplicate").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionDuplicate,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to duplicate this shipment")
	}

	// * Validate the request
	if err := req.Validate(ctx); err != nil {
		return nil, err
	}

	newEntity, err := s.repo.Duplicate(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to duplicate shipment")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     req.ShipmentID.String(),
			Action:         permission.ActionDuplicate,
			UserID:         req.UserID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Shipment duplicated"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment duplication")
	}

	return newEntity, nil
}

func (s *Service) GetNextProNumber(ctx context.Context, orgID pulid.ID) (string, error) {
	return s.proNumberRepo.GetNextProNumber(ctx, orgID)
}

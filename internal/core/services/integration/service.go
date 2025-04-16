package integration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/googlemaps"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams contains the dependencies for the integration service.
type ServiceParams struct {
	fx.In

	Logger           *logger.Logger
	Repo             repositories.IntegrationRepository
	PermService      services.PermissionService
	AuditService     services.AuditService
	PCMilerRepo      repositories.PCMilerConfigurationRepository
	GoogleMapsClient googlemaps.Client
}

// Service implements the IntegrationService interface.
type Service struct {
	l                *zerolog.Logger
	repo             repositories.IntegrationRepository
	ps               services.PermissionService
	as               services.AuditService
	pcMilerRepo      repositories.PCMilerConfigurationRepository
	googleMapsClient googlemaps.Client
}

// NewService creates a new integration service.
func NewService(p ServiceParams) services.IntegrationService {
	log := p.Logger.With().
		Str("service", "integration").
		Logger()

	return &Service{
		l:                &log,
		repo:             p.Repo,
		ps:               p.PermService,
		as:               p.AuditService,
		pcMilerRepo:      p.PCMilerRepo,
		googleMapsClient: p.GoogleMapsClient,
	}
}

// List returns a paginated list of integrations.
func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*integration.Integration], error) {
	log := s.l.With().
		Str("operation", "List").
		Logger()
	// Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceIntegration,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read integrations")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list integrations")
		return nil, err
	}

	return entities, nil
}

// GetByID returns an integration by ID.
func (s *Service) GetByID(ctx context.Context, req repositories.GetIntegrationByIDOptions) (*integration.Integration, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Logger()

	// Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceIntegration,
				Action:         permission.ActionRead,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read this integration")
	}

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get integration by id")
		return nil, err
	}

	return entity, nil
}

func (s *Service) GetByType(ctx context.Context, req repositories.GetIntegrationByTypeRequest) (*integration.Integration, error) {
	log := s.l.With().
		Str("operation", "GetByType").
		Logger()

	// ! We do not check permissions here because we need to allow unauthenticated access to the integration type information

	entity, err := s.repo.GetByType(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get integration by type")
		return nil, err
	}

	return entity, nil
}

// Update updates an integration.
func (s *Service) Update(ctx context.Context, i *integration.Integration, userID pulid.ID) (*integration.Integration, error) {
	log := s.l.With().
		Str("operation", "Update").
		Logger()

	// // Check permissions
	// result, err := s.ps.HasAnyPermissions(ctx,
	// 	[]*services.PermissionCheck{
	// 		{
	// 			UserID:         i.OrganizationID,
	// 			Resource:       permission.ResourceIntegration,
	// 			Action:         permission.ActionUpdate,
	// 			BusinessUnitID: i.BusinessUnitID,
	// 			OrganizationID: i.OrganizationID,
	// 		},
	// 	},
	// )
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to check permissions")
	// 	return nil, err
	// }

	// if !result.Allowed {
	// 	return nil, errors.NewAuthorizationError("You do not have permission to update integrations")
	// }

	// Get the existing integration
	original, err := s.repo.GetByID(ctx, repositories.GetIntegrationByIDOptions{
		ID:    i.ID,
		OrgID: i.OrganizationID,
		BuID:  i.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, i)
	if err != nil {
		log.Error().Err(err).Msg("failed to update integration")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceIntegration,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceIntegration,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}

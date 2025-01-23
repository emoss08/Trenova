package tableconfiguration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"

	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger      *logger.Logger
	Repo        repositories.TableConfigurationRepository
	PermService services.PermissionService
}

type Service struct {
	repo repositories.TableConfigurationRepository
	ps   services.PermissionService
	l    *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().Str("service", "tableconfiguration").Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		l:    &log,
	}
}

func (s *Service) List(ctx context.Context, opts *repositories.TableConfigurationFilters) (*repositories.ListTableConfigurationResult, error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         opts.Base.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionRead,
			BusinessUnitID: opts.Base.BuID,
			OrganizationID: opts.Base.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read table configurations")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list table configurations")
		return nil, eris.Wrap(err, "list table configurations")
	}

	return entities, nil
}

func (s *Service) Create(ctx context.Context, config *tableconfiguration.Configuration) (*tableconfiguration.Configuration, error) {
	log := s.l.With().Str("method", "Create").
		Str("orgID", config.OrganizationID.String()).
		Str("businessUnitID", config.BusinessUnitID.String()).
		Str("userID", config.UserID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         config.UserID,
				Resource:       permission.ResourceTableConfiguration,
				Action:         permission.ActionCreate,
				BusinessUnitID: config.BusinessUnitID,
				OrganizationID: config.OrganizationID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create table configurations")
	}

	// if setting as default, ensure user has permission
	if config.IsDefault {
		defaultPermResult, dprErr := s.ps.HasPermission(ctx,
			&services.PermissionCheck{
				UserID:         config.UserID,
				Resource:       permission.ResourceTableConfiguration,
				Action:         permission.ActionManageDefaults,
				BusinessUnitID: config.BusinessUnitID,
				OrganizationID: config.OrganizationID,
			})
		if dprErr != nil {
			log.Error().Err(dprErr).Msg("failed to check default permission")
			return nil, eris.Wrap(dprErr, "failed to check default permission")
		}

		if !defaultPermResult.Allowed {
			return nil, errors.NewAuthorizationError("You do not have permission to manage default table configurations")
		}
	}

	if err = s.repo.Create(ctx, config); err != nil {
		log.Error().Err(err).Msg("failed to create table configuration")
		return nil, eris.Wrap(err, "create configuration")
	}

	return config, nil
}

func (s *Service) Update(ctx context.Context, config *tableconfiguration.Configuration) error {
	log := s.l.With().
		Str("operation", "Update").
		Str("configID", config.ID.String()).
		Logger()

	existing, err := s.repo.GetByID(ctx, config.ID,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID: config.OrganizationID,
				BuID:  config.BusinessUnitID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing configuration")
		return eris.Wrap(err, "get existing configuration")
	}

	result, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         config.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionUpdate,
			BusinessUnitID: config.BusinessUnitID,
			OrganizationID: config.OrganizationID,
			ResourceID:     config.ID,
			CustomData: map[string]any{
				"userId": existing.UserID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check update permission")
		return eris.Wrap(err, "check update permission")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError("You do not have permission to update this table configuration")
	}

	if err = s.repo.Update(ctx, config); err != nil {
		log.Error().Err(err).Msg("failed to update table configuration")
		return eris.Wrap(err, "update configuration")
	}

	return nil
}

// Delete deletes a table configuration with permission checks
func (s *Service) Delete(ctx context.Context, id pulid.ID, opts *repositories.GetUserByIDOptions) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("configID", id.String()).
		Logger()

	// Get existing configuration
	existing, err := s.repo.GetByID(ctx, id,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID: opts.OrgID,
				BuID:  opts.BuID,
			},
		})
	if err != nil {
		return eris.Wrap(err, "get existing configuration")
	}

	// Check delete permission
	permResult, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         opts.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionDelete,
			BusinessUnitID: opts.BuID,
			OrganizationID: opts.OrgID,
			ResourceID:     id,
			CustomData: map[string]interface{}{
				"userId": existing.UserID,
			},
		})
	if err != nil {
		return eris.Wrap(err, "check permission")
	}
	if !permResult.Allowed {
		return errors.NewAuthorizationError("You don't have permission to delete this configuration")
	}

	if err = s.repo.Delete(ctx, id); err != nil {
		log.Error().Err(err).Msg("failed to delete configuration")
		return eris.Wrap(err, "delete configuration")
	}

	return nil
}

// ShareConfiguration shares a configuration with specified users/roles/teams
func (s *Service) ShareConfiguration(ctx context.Context, share *tableconfiguration.ConfigurationShare, userID pulid.ID) error {
	log := s.l.With().
		Str("operation", "ShareConfiguration").
		Str("configID", share.ConfigurationID.String()).
		Logger()

	// Get existing configuration
	existing, err := s.repo.GetByID(ctx, share.ConfigurationID,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID: share.OrganizationID,
				BuID:  share.BusinessUnitID,
			},
		})
	if err != nil {
		return eris.Wrap(err, "get existing configuration")
	}

	// Check share permission
	permResult, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         userID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionShare,
			BusinessUnitID: existing.BusinessUnitID,
			OrganizationID: existing.OrganizationID,
			ResourceID:     existing.ID,
			CustomData: map[string]interface{}{
				"userId": existing.UserID,
			},
		})
	if err != nil {
		return eris.Wrap(err, "check permission")
	}
	if !permResult.Allowed {
		return errors.NewAuthorizationError("You don't have permission to share this configuration")
	}

	if err = s.repo.ShareConfiguration(ctx, share); err != nil {
		log.Error().Err(err).Msg("failed to share configuration")
		return eris.Wrap(err, "share configuration")
	}

	return nil
}

// GetUserConfigurations retrieves configurations accessible to a user
func (s *Service) GetUserConfigurations(ctx context.Context, tableID string, opts *repositories.GetUserByIDOptions) ([]*tableconfiguration.Configuration, error) {
	log := s.l.With().
		Str("operation", "GetUserConfigurations").
		Str("userID", opts.UserID.String()).
		Logger()

	// Check read permission
	permResult, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         opts.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionRead,
			BusinessUnitID: opts.BuID,
			OrganizationID: opts.OrgID,
		})
	if err != nil {
		return nil, eris.Wrap(err, "check permission")
	}
	if !permResult.Allowed {
		return nil, errors.NewAuthorizationError("You don't have permission to view table configurations")
	}

	configs, err := s.repo.GetUserConfigurations(ctx, tableID,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID: opts.OrgID,
				BuID:  opts.BuID,
			},
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, eris.Wrap(err, "get user configurations")
	}

	return configs, nil
}

package tableconfiguration

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	tcdomain "github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"

	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger              *logger.Logger
	Repo                repositories.TableConfigurationRepository
	UserRepo            repositories.UserRepository
	PermService         services.PermissionService
	AuditService        services.AuditService
	NotificationService services.NotificationService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.TableConfigurationRepository
	ur   repositories.UserRepository
	ps   services.PermissionService
	as   services.AuditService
	ns   services.NotificationService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "tableconfiguration").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		l:    &log,
		ns:   p.NotificationService,
		ur:   p.UserRepo,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*repositories.ListTableConfigurationResult, error) {
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
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read table configurations",
		)
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list table configurations")
		return nil, eris.Wrap(err, "list table configurations")
	}

	return entities, nil
}

func (s *Service) ListPublicConfigurations(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*ports.ListResult[*tcdomain.Configuration], error) {
	log := s.l.With().Str("operation", "ListPublicConfigurations").Logger()

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
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read table configurations",
		)
	}

	entities, err := s.repo.ListPublicConfigurations(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list table configurations")
		return nil, err
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	config *tcdomain.Configuration,
) (*tcdomain.Configuration, error) {
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
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create table configurations",
		)
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
			return nil, errors.NewAuthorizationError(
				"You do not have permission to manage default table configurations",
			)
		}
	}

	createdEntity, err := s.repo.Create(ctx, config)
	if err != nil {
		log.Error().Err(err).Msg("failed to create table configuration")
		return nil, eris.Wrap(err, "create configuration")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         config.UserID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			BusinessUnitID: config.BusinessUnitID,
			OrganizationID: config.OrganizationID,
		},
		audit.WithComment("Table configuration created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log table configuration creation")
	}

	return createdEntity, nil
}

func (s *Service) Copy(
	ctx context.Context,
	req *repositories.CopyTableConfigurationRequest,
) error {
	log := s.l.With().
		Str("operation", "Copy").
		Str("configID", req.ConfigID.String()).
		Logger()

	result, err := s.ps.HasPermission(ctx, &services.PermissionCheck{
		UserID:         req.UserID,
		Resource:       permission.ResourceTableConfiguration,
		Action:         permission.ActionRead,
		BusinessUnitID: req.BuID,
		OrganizationID: req.OrgID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check copy permission")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to copy this configuration",
		)
	}

	if err = s.repo.Copy(ctx, req); err != nil {
		log.Error().Err(err).Msg("failed to copy configuration")
		return err
	}

	existing, err := s.repo.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Base: &ports.FilterQueryOptions{
			OrgID: req.OrgID,
			BuID:  req.BuID,
		},
		IncludeCreator: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing configuration")
		return err
	}

	copiedBy, err := s.ur.GetByID(ctx, repositories.GetUserByIDOptions{
		OrgID:  req.OrgID,
		BuID:   req.BuID,
		UserID: req.UserID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get copied by user")
		return err
	}

	// * We might want to notify the original creator that their configuration has been copied
	notificationReq := &services.ConfigurationCopiedNotificationRequest{
		UserID:         existing.UserID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
		ConfigID:       req.ConfigID,
		ConfigName:     existing.Name,
		ConfigCreator:  existing.Creator.Name,
		ConfigCopiedBy: copiedBy.Name,
	}
	if err = s.ns.SendConfigurationCopiedNotification(ctx, notificationReq); err != nil {
		log.Error().Err(err).Msg("failed to send configuration copied notification")
		// ! we will not return an error here because we want to continue the operation
		// ! even if the notification fails
	}

	return nil
}

func (s *Service) Update(
	ctx context.Context,
	config *tcdomain.Configuration,
) (*tcdomain.Configuration, error) {
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
		return nil, eris.Wrap(err, "get existing configuration")
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
		return nil, eris.Wrap(err, "check update permission")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this table configuration",
		)
	}

	if err = s.repo.Update(ctx, config); err != nil {
		log.Error().Err(err).Msg("failed to update table configuration")
		return nil, eris.Wrap(err, "update configuration")
	}

	return config, nil
}

// Delete deletes a table configuration with permission checks
func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteUserConfigurationRequest,
) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("configID", req.ConfigID.String()).
		Logger()

	// * The deletion can only be done by the user who created the configuration
	result, err := s.ps.HasPermission(ctx, &services.PermissionCheck{
		UserID:         req.UserID,
		Resource:       permission.ResourceTableConfiguration,
		Action:         permission.ActionDelete,
		BusinessUnitID: req.BuID,
		OrganizationID: req.OrgID,
		ResourceID:     req.ConfigID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check delete permission")
		return eris.Wrap(err, "check delete permission")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You don't have permission to delete this configuration",
		)
	}

	existing, err := s.repo.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Base: &ports.FilterQueryOptions{
			OrgID:  req.OrgID,
			BuID:   req.BuID,
			UserID: req.UserID,
		},
	})
	if err != nil {
		return eris.Wrap(err, "get existing configuration")
	}

	// Check delete permission
	permResult, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         req.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionDelete,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
			ResourceID:     req.ConfigID,
			CustomData: map[string]any{
				"userId": existing.UserID,
			},
		})
	if err != nil {
		return eris.Wrap(err, "check permission")
	}
	if !permResult.Allowed {
		return errors.NewAuthorizationError(
			"You don't have permission to delete this configuration",
		)
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error().Err(err).Msg("failed to delete configuration")
		return eris.Wrap(err, "delete configuration")
	}

	return nil
}

// ShareConfiguration shares a configuration with specified users/roles/teams
func (s *Service) ShareConfiguration(
	ctx context.Context,
	share *tcdomain.ConfigurationShare,
	userID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "ShareConfiguration").
		Str("configID", share.ConfigurationID.String()).
		Logger()

	// Get existing configuration
	existing, err := s.repo.GetByID(ctx,
		share.ConfigurationID,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID: share.OrganizationID,
				BuID:  share.BusinessUnitID,
			},
		})
	if err != nil {
		return err
	}

	// Check share permission
	permResult, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceTableConfiguration,
				Action:         permission.ActionShare,
				BusinessUnitID: existing.BusinessUnitID,
				OrganizationID: existing.OrganizationID,
				// ResourceID:     existing.ID,
				CustomData: map[string]any{
					"userId": existing.UserID,
				},
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
		return err
	}

	return nil
}

func (s *Service) ListUserConfigurations(
	ctx context.Context,
	opts *repositories.ListUserConfigurationRequest,
) (*ports.ListResult[*tcdomain.Configuration], error) {
	log := s.l.With().
		Str("operation", "ListUserConfigurations").
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	permResult, err := s.ps.HasPermission(ctx,
		&services.PermissionCheck{
			UserID:         opts.Filter.TenantOpts.UserID,
			Resource:       permission.ResourceTableConfiguration,
			Action:         permission.ActionRead,
			BusinessUnitID: opts.Filter.TenantOpts.BuID,
			OrganizationID: opts.Filter.TenantOpts.OrgID,
		})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permission")
		return nil, eris.Wrap(err, "check permission")
	}

	if !permResult.Allowed {
		return nil, errors.NewAuthorizationError(
			"You don't have permission to view table configurations",
		)
	}

	result, err := s.repo.ListUserConfigurations(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list user configurations")
		return nil, eris.Wrap(err, "list user configurations")
	}

	return result, nil
}

// GetUserConfigurations retrieves configurations accessible to a user
func (s *Service) GetUserConfigurations(
	ctx context.Context,
	tableID string,
	opts *repositories.GetUserByIDOptions,
) ([]*tcdomain.Configuration, error) {
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
		return nil, errors.NewAuthorizationError(
			"You don't have permission to view table configurations",
		)
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

// GetDefaultOrLatestConfiguration retrieves a configuration for the given table identifier and current user.
// If none exists it will create a new one with a minimal default payload so the
// client always receives a valid configuration object.
func (s *Service) GetDefaultOrLatestConfiguration(
	ctx context.Context,
	resource string,
	rCtx *appctx.RequestContext,
) (*tcdomain.Configuration, error) {
	// First attempt to find an existing configuration for this user/org/bu + table
	config, err := s.repo.GetDefaultOrLatestConfiguration(
		ctx,
		resource,
		&repositories.TableConfigurationFilters{
			Base: &ports.FilterQueryOptions{
				OrgID:  rCtx.OrgID,
				BuID:   rCtx.BuID,
				UserID: rCtx.UserID,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return config, nil
}

package tableconfiguration

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/temporaljobs/notificationjobs"

	authCtx "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Repo           repositories.TableConfigurationRepository
	UserRepo       repositories.UserRepository
	AuditService   services.AuditService
	TemporalClient client.Client
}

type Service struct {
	l              *zap.Logger
	repo           repositories.TableConfigurationRepository
	userRepo       repositories.UserRepository
	auditService   services.AuditService
	temporalClient client.Client
}

func NewService(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.tableconfiguration"),
		repo:           p.Repo,
		userRepo:       p.UserRepo,
		auditService:   p.AuditService,
		temporalClient: p.TemporalClient,
	}
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	return s.repo.List(ctx, opts)
}

func (s *Service) ListPublicConfigurations(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	return s.repo.ListPublicConfigurations(ctx, opts)
}

func (s *Service) Create(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) (*tableconfiguration.Configuration, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", config.BusinessUnitID.String()),
		zap.String("userID", config.UserID.String()),
	)

	createdEntity, err := s.repo.Create(ctx, config)
	if err != nil {
		log.Error("failed to create table configuration", zap.Error(err))
		return nil, err
	}

	err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         config.UserID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			BusinessUnitID: config.BusinessUnitID,
			OrganizationID: config.OrganizationID,
		},
		audit.WithComment("Table configuration created"),
	)
	if err != nil {
		log.Error("failed to log table configuration creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Copy(
	ctx context.Context,
	req *repositories.CopyTableConfigurationRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Copy"),
		zap.String("configID", req.ConfigID.String()),
	)

	if err := s.repo.Copy(ctx, req); err != nil {
		log.Error("failed to copy configuration", zap.Error(err))
		return err
	}

	existing, err := s.repo.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Filter: &pagination.QueryOptions{
			TenantOpts: pagination.TenantOptions{
				OrgID: req.OrgID,
				BuID:  req.BuID,
			},
		},
		IncludeCreator: true,
	})
	if err != nil {
		log.Error("failed to get existing configuration", zap.Error(err))
		return err
	}

	copiedBy, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDRequest{
		OrgID:  req.OrgID,
		BuID:   req.BuID,
		UserID: req.UserID,
	})
	if err != nil {
		log.Error("failed to get copied by user", zap.Error(err))
		return err
	}

	if _, err = s.temporalClient.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        fmt.Sprintf("configuration-copied-%s", req.ConfigID.String()),
			TaskQueue: temporaltype.NotificationTaskQueue,
			SearchAttributes: map[string]any{
				"OrganizationId": req.OrgID.String(),
				"BusinessUnitId": req.BuID.String(),
				"UserId":         existing.UserID.String(),
				"ConfigId":       req.ConfigID.String(),
				"WorkflowType":   "SendNotification",
			},
		},
		notificationjobs.SendConfigurationCopiedNotificationWorkflow,
		&notificationjobs.SendConfigurationCopiedNotificationPayload{
			UserID:         existing.UserID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			ConfigID:       req.ConfigID,
			ConfigName:     existing.Name,
			ConfigCreator:  existing.Creator.Name,
			ConfigCopiedBy: copiedBy.Name,
		},
	); err != nil {
		log.Error("failed to send configuration copied notification", zap.Error(err))
		// ! we will not return an error here because we want to continue the operation
		// ! even if the notification fails
	}

	err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpCopy,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(existing),
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
		audit.WithComment("Table configuration copied"),
	)
	if err != nil {
		log.Error("failed to log table configuration copy", zap.Error(err))
	}
	return nil
}

func (s *Service) Update(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) (*tableconfiguration.Configuration, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("configID", config.ID.String()),
	)

	existing, err := s.repo.GetByID(ctx, config.ID,
		&repositories.TableConfigurationFilters{
			Filter: &pagination.QueryOptions{
				TenantOpts: pagination.TenantOptions{
					OrgID: config.OrganizationID,
					BuID:  config.BusinessUnitID,
				},
			},
		})
	if err != nil {
		log.Error("failed to get existing configuration", zap.Error(err))
		return nil, err
	}

	if err = s.repo.Update(ctx, config); err != nil {
		log.Error("failed to update table configuration", zap.Error(err))
		return nil, err
	}

	err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         config.UserID,
			PreviousState:  jsonutils.MustToJSON(config),
			CurrentState:   jsonutils.MustToJSON(existing),
			BusinessUnitID: config.BusinessUnitID,
			OrganizationID: config.OrganizationID,
		},
		audit.WithComment("Table configuration updated"),
		audit.WithDiff(existing, config),
	)
	if err != nil {
		log.Error("failed to log table configuration update", zap.Error(err))
	}

	return config, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteUserConfigurationRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("configID", req.ConfigID.String()),
	)

	existing, err := s.repo.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Filter: &pagination.QueryOptions{
			TenantOpts: pagination.TenantOptions{
				OrgID:  req.OrgID,
				BuID:   req.BuID,
				UserID: req.UserID,
			},
		},
	})
	if err != nil {
		log.Error("failed to get existing configuration", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete configuration", zap.Error(err))
		return err
	}

	err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpDelete,
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		audit.WithComment("Table configuration deleted"),
		audit.WithDiff(existing, nil),
	)
	if err != nil {
		log.Error("failed to log table configuration deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) ShareConfiguration(
	ctx context.Context,
	share *tableconfiguration.ConfigurationShare,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "ShareConfiguration"),
		zap.String("configID", share.ConfigurationID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx,
		share.ConfigurationID,
		&repositories.TableConfigurationFilters{
			Filter: &pagination.QueryOptions{
				TenantOpts: pagination.TenantOptions{
					OrgID: share.OrganizationID,
					BuID:  share.BusinessUnitID,
				},
			},
		})
	if err != nil {
		return err
	}

	if err = s.repo.Share(ctx, share); err != nil {
		log.Error("failed to share configuration", zap.Error(err))
		return err
	}

	err = s.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     existing.GetID(),
			Operation:      permission.OpShare,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(existing),
			OrganizationID: existing.OrganizationID,
			BusinessUnitID: existing.BusinessUnitID,
		},
		audit.WithComment("Table configuration shared"),
	)
	if err != nil {
		log.Error("failed to log table configuration share", zap.Error(err))
	}

	return nil
}

func (s *Service) ListUserConfigurations(
	ctx context.Context,
	opts *repositories.ListUserConfigurationRequest,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	log := s.l.With(
		zap.String("operation", "ListUserConfigurations"),
		zap.Any("opts", opts),
	)

	result, err := s.repo.ListUserConfigurations(ctx, opts)
	if err != nil {
		log.Error("failed to list user configurations", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) GetUserConfigurations(
	ctx context.Context,
	resource string,
	opts *repositories.TableConfigurationFilters,
) ([]*tableconfiguration.Configuration, error) {
	log := s.l.With(
		zap.String("operation", "GetUserConfigurations"),
		zap.String("userID", opts.Filter.TenantOpts.UserID.String()),
	)

	configs, err := s.repo.GetUserConfigurations(ctx, resource,
		&repositories.TableConfigurationFilters{
			Filter: &pagination.QueryOptions{
				TenantOpts: pagination.TenantOptions{
					OrgID: opts.Filter.TenantOpts.OrgID,
					BuID:  opts.Filter.TenantOpts.BuID,
				},
			},
		})
	if err != nil {
		log.Error("failed to get user configurations", zap.Error(err))
		return nil, err
	}

	return configs, nil
}

func (s *Service) GetDefaultOrLatestConfiguration(
	ctx context.Context,
	resource string,
	rCtx *authCtx.AuthContext,
) (*tableconfiguration.Configuration, error) {
	config, err := s.repo.GetDefaultOrLatest(
		ctx,
		resource,
		&repositories.TableConfigurationFilters{
			Filter: &pagination.QueryOptions{
				TenantOpts: pagination.TenantOptions{
					OrgID:  rCtx.OrganizationID,
					BuID:   rCtx.BusinessUnitID,
					UserID: rCtx.UserID,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return config, nil
}

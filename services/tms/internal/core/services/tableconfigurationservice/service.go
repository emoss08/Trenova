package tableconfigurationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.TableConfigurationRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.TableConfigurationRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.tableconfiguration"),
		repo: p.Repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if entity.IsDefault {
		if err := s.repo.ClearDefaultForResource(
			ctx,
			entity.UserID,
			entity.Resource,
			pagination.TenantInfo{
				OrgID:  entity.OrganizationID,
				BuID:   entity.BusinessUnitID,
				UserID: entity.UserID,
			},
		); err != nil {
			log.Error("failed to clear existing default", zap.Error(err))
			return nil, err
		}
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create table configuration", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if entity.IsDefault {
		if err := s.repo.ClearDefaultForResource(
			ctx,
			entity.UserID,
			entity.Resource,
			pagination.TenantInfo{
				OrgID:  entity.OrganizationID,
				BuID:   entity.BusinessUnitID,
				UserID: entity.UserID,
			},
		); err != nil {
			log.Error("failed to clear existing default", zap.Error(err))
			return nil, err
		}
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update table configuration", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetTableConfigurationByIDRequest,
) (*tableconfiguration.TableConfiguration, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTableConfigurationsRequest,
) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", id.String()),
	)

	if err := s.repo.Delete(ctx, id, tenantInfo); err != nil {
		log.Error("failed to delete table configuration", zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) GetDefaultForResource(
	ctx context.Context,
	req repositories.GetDefaultTableConfigurationRequest,
) (*tableconfiguration.TableConfiguration, error) {
	return s.repo.GetDefaultForResource(ctx, req)
}

func (s *Service) SetDefault(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*tableconfiguration.TableConfiguration, error) {
	log := s.l.With(
		zap.String("operation", "SetDefault"),
		zap.String("id", id.String()),
	)

	entity, err := s.repo.GetByID(ctx, repositories.GetTableConfigurationByIDRequest{
		ConfigurationID: id,
		TenantInfo:      tenantInfo,
	})
	if err != nil {
		log.Error("failed to get table configuration", zap.Error(err))
		return nil, err
	}

	if err = s.repo.ClearDefaultForResource(
		ctx,
		entity.UserID,
		entity.Resource,
		tenantInfo,
	); err != nil {
		log.Error("failed to clear existing default", zap.Error(err))
		return nil, err
	}

	entity.IsDefault = true

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to set table configuration as default", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

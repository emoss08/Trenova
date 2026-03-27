package tablechangealertservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	SubRepo       repositories.TCASubscriptionRepository
	AllowlistRepo repositories.TCAAllowlistRepository
}

type Service struct {
	l             *zap.Logger
	subRepo       repositories.TCASubscriptionRepository
	allowlistRepo repositories.TCAAllowlistRepository
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.tablechangealert"),
		subRepo:       p.SubRepo,
		allowlistRepo: p.AllowlistRepo,
	}
}

func (s *Service) ListAllowlistedTables(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*tablechangealert.TCAAllowlistedTable, error) {
	return s.allowlistRepo.List(ctx, tenantInfo)
}

func (s *Service) CreateSubscription(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	log := s.l.With(zap.String("operation", "CreateSubscription"))

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	allowed, err := s.allowlistRepo.IsTableAllowed(ctx, entity.TableName, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to check allowlist", zap.Error(err))
		return nil, err
	}
	if !allowed {
		multiErr.Add("tableName", errortypes.ErrInvalid, "Table is not eligible for change alerts")
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.subRepo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create subscription", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateSubscription(
	ctx context.Context,
	entity *tablechangealert.TCASubscription,
) (*tablechangealert.TCASubscription, error) {
	log := s.l.With(
		zap.String("operation", "UpdateSubscription"),
		zap.String("id", entity.ID.String()),
	)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.subRepo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update subscription", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetSubscriptionByID(
	ctx context.Context,
	req repositories.GetTCASubscriptionByIDRequest,
) (*tablechangealert.TCASubscription, error) {
	return s.subRepo.GetByID(ctx, req)
}

func (s *Service) ListSubscriptions(
	ctx context.Context,
	req *repositories.ListTCASubscriptionsRequest,
) (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
	return s.subRepo.List(ctx, req)
}

func (s *Service) DeleteSubscription(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return s.subRepo.Delete(ctx, id, tenantInfo)
}

func (s *Service) PauseSubscription(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return s.setSubscriptionStatus(ctx, id, tenantInfo, tablechangealert.SubscriptionStatusPaused)
}

func (s *Service) ResumeSubscription(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*tablechangealert.TCASubscription, error) {
	return s.setSubscriptionStatus(ctx, id, tenantInfo, tablechangealert.SubscriptionStatusActive)
}

func (s *Service) setSubscriptionStatus(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	status tablechangealert.SubscriptionStatus,
) (*tablechangealert.TCASubscription, error) {
	log := s.l.With(
		zap.String("operation", "setSubscriptionStatus"),
		zap.String("id", id.String()),
		zap.String("status", status.String()),
	)

	entity, err := s.subRepo.GetByID(ctx, repositories.GetTCASubscriptionByIDRequest{
		SubscriptionID: id,
		TenantInfo:     tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity.Status = status

	updated, err := s.subRepo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update subscription status", zap.Error(err))
		return nil, err
	}

	return updated, nil
}

package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultAccountingControlTTL = 24 * time.Hour
	acKeyPrefix                 = "accounting_control:"
)

type AccountingControlRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type accountingControlRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewAccountingControlRepository(
	p AccountingControlRepositoryParams,
) repositories.AccountingControlCacheRepository {
	return &accountingControlRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.accountingcontrol-repository"),
	}
}

func (ac *accountingControlRepository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*accounting.AccountingControl, error) {
	log := ac.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	accountingControl := new(accounting.AccountingControl)
	key := ac.formatKey(orgID)
	if err := ac.cache.GetJSON(ctx, key, accountingControl); err != nil {
		log.Error("failed to get accounting control from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved accounting control from cache", zap.String("key", key))
	return accountingControl, nil
}

func (ac *accountingControlRepository) Set(
	ctx context.Context,
	accountingControl *accounting.AccountingControl,
) error {
	log := ac.l.With(
		zap.String("operation", "Set"),
		zap.String("orgID", accountingControl.OrganizationID.String()),
	)

	key := ac.formatKey(accountingControl.OrganizationID)
	if err := ac.cache.SetJSON(ctx, key, accountingControl, defaultAccountingControlTTL); err != nil {
		log.Error("failed to set accounting control in cache", zap.Error(err))
		return err
	}

	log.Debug("stored accounting control in cache", zap.String("key", key))
	return nil
}

func (ac *accountingControlRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := ac.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("orgID", orgID.String()),
	)

	key := ac.formatKey(orgID)
	if err := ac.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate accounting control in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated accounting control in cache", zap.String("key", key))
	return nil
}

func (ac *accountingControlRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", acKeyPrefix, orgID)
}

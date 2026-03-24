package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	corerepos "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/bunmarshal"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	customerCachePrefix = "cache:customers"
	documentCachePrefix = "cache:documents"
	shipmentCachePrefix = "cache:shipments"
	workerCachePrefix   = "cache:workers"
)

type syncedCacheBase struct {
	client *redis.Client
	logger *zap.Logger
}

type WorkerCacheRepositoryParams struct {
	fx.In

	Client      *redis.Client
	Logger      *zap.Logger
	UsStateRepo corerepos.UsStateRepository
}

type workerCacheRepository struct {
	*syncedCacheBase
	usStateRepo corerepos.UsStateRepository
}

type CustomerCacheRepositoryParams struct {
	fx.In

	Client      *redis.Client
	Logger      *zap.Logger
	UsStateRepo corerepos.UsStateRepository
}

type customerCacheRepository struct {
	*syncedCacheBase
	usStateRepo corerepos.UsStateRepository
}

type ShipmentCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type shipmentCacheRepository struct {
	*syncedCacheBase
}

type DocumentCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type documentCacheRepository struct {
	*syncedCacheBase
}

func NewWorkerCacheRepository(p WorkerCacheRepositoryParams) corerepos.WorkerCacheRepository {
	return &workerCacheRepository{
		syncedCacheBase: newSyncedCacheBase(p.Client, p.Logger, "worker-cache-repository"),
		usStateRepo:     p.UsStateRepo,
	}
}

func NewCustomerCacheRepository(p CustomerCacheRepositoryParams) corerepos.CustomerCacheRepository {
	return &customerCacheRepository{
		syncedCacheBase: newSyncedCacheBase(p.Client, p.Logger, "customer-cache-repository"),
		usStateRepo:     p.UsStateRepo,
	}
}

func NewShipmentCacheRepository(p ShipmentCacheRepositoryParams) corerepos.ShipmentCacheRepository {
	return &shipmentCacheRepository{
		syncedCacheBase: newSyncedCacheBase(p.Client, p.Logger, "shipment-cache-repository"),
	}
}

func NewDocumentCacheRepository(p DocumentCacheRepositoryParams) corerepos.DocumentCacheRepository {
	return &documentCacheRepository{
		syncedCacheBase: newSyncedCacheBase(p.Client, p.Logger, "document-cache-repository"),
	}
}

func newSyncedCacheBase(client *redis.Client, logger *zap.Logger, component string) *syncedCacheBase {
	return &syncedCacheBase{
		client: client,
		logger: logger.Named("redis." + component),
	}
}

func (r *workerCacheRepository) GetByID(
	ctx context.Context,
	req corerepos.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	if req.IncludeProfile {
		return nil, corerepos.ErrCacheMiss
	}

	entity, err := getCachedEntity[worker.Worker](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(workerCachePrefix, req.ID.String(), req.TenantInfo.OrgID.String(), req.TenantInfo.BuID.String()),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID || entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, corerepos.ErrCacheMiss
	}

	if req.IncludeState {
		state, stateErr := r.usStateRepo.GetByID(ctx, corerepos.GetUsStateByIDRequest{StateID: entity.StateID})
		if stateErr != nil {
			return nil, corerepos.ErrCacheMiss
		}
		entity.State = state
	}

	return entity, nil
}

func (r *customerCacheRepository) GetByID(
	ctx context.Context,
	req corerepos.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	if req.IncludeBillingProfile || req.IncludeEmailProfile {
		return nil, corerepos.ErrCacheMiss
	}

	entity, err := getCachedEntity[customer.Customer](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(customerCachePrefix, req.ID.String(), req.TenantInfo.OrgID.String(), req.TenantInfo.BuID.String()),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID || entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, corerepos.ErrCacheMiss
	}

	state, stateErr := r.usStateRepo.GetByID(ctx, corerepos.GetUsStateByIDRequest{StateID: entity.StateID})
	if stateErr != nil {
		return nil, corerepos.ErrCacheMiss
	}
	entity.State = state

	return entity, nil
}

func (r *shipmentCacheRepository) GetByID(
	ctx context.Context,
	req *corerepos.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	if req.ExpandShipmentDetails {
		return nil, corerepos.ErrCacheMiss
	}

	entity, err := getCachedEntity[shipment.Shipment](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(shipmentCachePrefix, req.ID.String(), req.TenantInfo.OrgID.String(), req.TenantInfo.BuID.String()),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID || entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, corerepos.ErrCacheMiss
	}

	return entity, nil
}

func (r *documentCacheRepository) GetByID(
	ctx context.Context,
	req corerepos.GetDocumentByIDRequest,
) (*document.Document, error) {
	entity, err := getCachedEntity[document.Document](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(documentCachePrefix, req.ID.String(), req.TenantInfo.OrgID.String(), req.TenantInfo.BuID.String()),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID || entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, corerepos.ErrCacheMiss
	}

	return entity, nil
}

func getCachedEntity[T any](
	ctx context.Context,
	base *syncedCacheBase,
	keys []string,
) (*T, error) {
	var lastErr error
	for _, key := range keys {
		raw := make(map[string]any)
		if err := redishelpers.GetJSON(ctx, base.client, key, &raw); err != nil {
			if errors.Is(err, redis.Nil) || redishelpers.IsRedisNil(err) {
				lastErr = corerepos.ErrCacheMiss
				continue
			}
			return nil, err
		}

		entity := new(T)
		if err := bunmarshal.UnmarshalMap(raw, entity); err != nil {
			base.logger.Warn("failed to decode cached entity", zap.String("key", key), zap.Error(err))
			lastErr = corerepos.ErrCacheMiss
			continue
		}

		return entity, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, corerepos.ErrCacheMiss
}

func buildScopedKeys(prefix, id, organizationID, businessUnitID string) []string {
	keys := make([]string, 0, 2)
	if organizationID != "" && businessUnitID != "" {
		keys = append(keys, fmt.Sprintf("%s:%s:%s:%s", prefix, organizationID, businessUnitID, id))
	}
	keys = append(keys, fmt.Sprintf("%s:%s", prefix, id))
	return keys
}

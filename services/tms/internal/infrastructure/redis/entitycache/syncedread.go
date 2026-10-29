package entitycache

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/columnmap"
	"github.com/emoss08/trenova/pkg/readscope"
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
	kind   permission.Resource
}

type WorkerCacheRepositoryParams struct {
	fx.In

	Client      *redis.Client
	Logger      *zap.Logger
	UsStateRepo usstateports.UsStateRepository
}

type workerCacheRepository struct {
	*syncedCacheBase
	usStateRepo usstateports.UsStateRepository
}

type CustomerCacheRepositoryParams struct {
	fx.In

	Client      *redis.Client
	Logger      *zap.Logger
	UsStateRepo usstateports.UsStateRepository
}

type customerCacheRepository struct {
	*syncedCacheBase
	usStateRepo usstateports.UsStateRepository
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

func NewWorkerCacheRepository(p WorkerCacheRepositoryParams) customerports.WorkerCacheRepository {
	return &workerCacheRepository{
		syncedCacheBase: newSyncedCacheBase(
			p.Client,
			p.Logger,
			"worker-cache-repository",
			permission.ResourceWorker,
		),
		usStateRepo: p.UsStateRepo,
	}
}

func NewCustomerCacheRepository(
	p CustomerCacheRepositoryParams,
) customerports.CustomerCacheRepository {
	return &customerCacheRepository{
		syncedCacheBase: newSyncedCacheBase(
			p.Client,
			p.Logger,
			"customer-cache-repository",
			permission.ResourceCustomer,
		),
		usStateRepo: p.UsStateRepo,
	}
}

func NewShipmentCacheRepository(
	p ShipmentCacheRepositoryParams,
) customerports.ShipmentCacheRepository {
	return &shipmentCacheRepository{
		syncedCacheBase: newSyncedCacheBase(
			p.Client,
			p.Logger,
			"shipment-cache-repository",
			permission.ResourceShipment,
		),
	}
}

func NewDocumentCacheRepository(
	p DocumentCacheRepositoryParams,
) customerports.DocumentCacheRepository {
	return &documentCacheRepository{
		syncedCacheBase: newSyncedCacheBase(
			p.Client,
			p.Logger,
			"document-cache-repository",
			permission.ResourceDocument,
		),
	}
}

func newSyncedCacheBase(
	client *redis.Client,
	logger *zap.Logger,
	component string,
	kind permission.Resource,
) *syncedCacheBase {
	return &syncedCacheBase{
		client: client,
		logger: logger.Named("redis." + component),
		kind:   kind,
	}
}

// servable reports whether the cache may answer for the database. The
// cached row is the tenant's, so a request whose read scope narrows the
// kind (an own or team grant) reads through the database's read_scope
// policy instead.
func (b *syncedCacheBase) servable(ctx context.Context) bool {
	narrowed, err := readscope.Narrows(ctx, b.kind.String())
	return err == nil && !narrowed
}

func (r *workerCacheRepository) GetByID(
	ctx context.Context,
	req workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	if !r.servable(ctx) {
		return nil, customerports.ErrCacheMiss
	}
	if req.IncludeProfile || req.IncludeFleetCode || req.IncludeManager {
		return nil, customerports.ErrCacheMiss
	}

	entity, err := getCachedEntity[worker.Worker](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(
			workerCachePrefix,
			req.ID.String(),
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
		),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID ||
		entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, customerports.ErrCacheMiss
	}

	if req.IncludeState {
		state, stateErr := r.usStateRepo.GetByID(
			ctx,
			usstateports.GetUsStateByIDRequest{StateID: entity.StateID},
		)
		if stateErr != nil {
			return nil, customerports.ErrCacheMiss
		}
		entity.State = state
	}

	return entity, nil
}

func (r *customerCacheRepository) GetByID(
	ctx context.Context,
	req customerports.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	if !r.servable(ctx) {
		return nil, customerports.ErrCacheMiss
	}
	if req.IncludeBillingProfile || req.IncludeEmailProfile {
		return nil, customerports.ErrCacheMiss
	}

	entity, err := getCachedEntity[customer.Customer](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(
			customerCachePrefix,
			req.ID.String(),
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
		),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID ||
		entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, customerports.ErrCacheMiss
	}

	state, stateErr := r.usStateRepo.GetByID(
		ctx,
		usstateports.GetUsStateByIDRequest{StateID: entity.StateID},
	)
	if stateErr != nil {
		return nil, customerports.ErrCacheMiss
	}
	entity.State = state

	return entity, nil
}

func (r *shipmentCacheRepository) GetByID(
	ctx context.Context,
	req *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	if !r.servable(ctx) {
		return nil, customerports.ErrCacheMiss
	}
	if req.ExpandShipmentDetails {
		return nil, customerports.ErrCacheMiss
	}

	entity, err := getCachedEntity[shipment.Shipment](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(
			shipmentCachePrefix,
			req.ID.String(),
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
		),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID ||
		entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, customerports.ErrCacheMiss
	}

	return entity, nil
}

func (r *documentCacheRepository) GetByID(
	ctx context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	if !r.servable(ctx) {
		return nil, customerports.ErrCacheMiss
	}
	entity, err := getCachedEntity[document.Document](
		ctx,
		r.syncedCacheBase,
		buildScopedKeys(
			documentCachePrefix,
			req.ID.String(),
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
		),
	)
	if err != nil {
		return nil, err
	}
	if entity.OrganizationID != req.TenantInfo.OrgID ||
		entity.BusinessUnitID != req.TenantInfo.BuID {
		return nil, customerports.ErrCacheMiss
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
				lastErr = customerports.ErrCacheMiss
				continue
			}
			return nil, err
		}

		entity := new(T)
		if err := columnmap.UnmarshalMap(raw, entity); err != nil {
			base.logger.Warn(
				"failed to decode cached entity",
				zap.String("key", key),
				zap.Error(err),
			)
			lastErr = customerports.ErrCacheMiss
			continue
		}

		return entity, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, customerports.ErrCacheMiss
}

func buildScopedKeys(prefix, id, organizationID, businessUnitID string) []string {
	keys := make([]string, 0, 2)
	if organizationID != "" && businessUnitID != "" {
		keys = append(keys, fmt.Sprintf("%s:%s:%s:%s", prefix, organizationID, businessUnitID, id))
	}
	keys = append(keys, fmt.Sprintf("%s:%s", prefix, id))
	return keys
}

package repositories

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func unreachableRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func narrowedTo(ctx context.Context, grants readscope.Grants) context.Context {
	return readscope.With(ctx, func(context.Context) (readscope.Grants, error) {
		return grants, nil
	})
}

type cachedLookup func(ctx context.Context) error

func syncedCacheLookups(t *testing.T) map[permission.Resource]cachedLookup {
	t.Helper()

	client := unreachableRedis(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	workers := NewWorkerCacheRepository(WorkerCacheRepositoryParams{Client: client, Logger: zap.NewNop()})
	customers := NewCustomerCacheRepository(
		CustomerCacheRepositoryParams{Client: client, Logger: zap.NewNop()},
	)
	shipments := NewShipmentCacheRepository(
		ShipmentCacheRepositoryParams{Client: client, Logger: zap.NewNop()},
	)
	documents := NewDocumentCacheRepository(
		DocumentCacheRepositoryParams{Client: client, Logger: zap.NewNop()},
	)

	return map[permission.Resource]cachedLookup{
		permission.ResourceWorker: func(ctx context.Context) error {
			_, err := workers.GetByID(ctx, workerports.GetWorkerByIDRequest{
				ID:         pulid.MustNew("wrk_"),
				TenantInfo: tenant,
			})
			return err
		},
		permission.ResourceCustomer: func(ctx context.Context) error {
			_, err := customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
				ID:         pulid.MustNew("cus_"),
				TenantInfo: tenant,
			})
			return err
		},
		permission.ResourceShipment: func(ctx context.Context) error {
			_, err := shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
				ID:         pulid.MustNew("shp_"),
				TenantInfo: tenant,
			})
			return err
		},
		permission.ResourceDocument: func(ctx context.Context) error {
			_, err := documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
				ID:         pulid.MustNew("doc_"),
				TenantInfo: tenant,
			})
			return err
		},
	}
}

func TestSyncedCaches_NeverAnswerARequestWhoseReadScopeNarrowsTheirKind(t *testing.T) {
	t.Parallel()

	for kind, lookup := range syncedCacheLookups(t) {
		ctx := narrowedTo(t.Context(), readscope.Grants{
			kind.String(): {Scope: readscope.ScopeTeam, Team: []pulid.ID{pulid.MustNew("usr_")}},
		})
		require.ErrorIs(t, lookup(ctx), customerports.ErrCacheMiss, kind)
	}
}

func TestSyncedCaches_StillServeRequestsTheirKindIsNotNarrowedFor(t *testing.T) {
	t.Parallel()

	for kind, lookup := range syncedCacheLookups(t) {
		ctx := narrowedTo(t.Context(), readscope.Grants{
			"order": {Scope: readscope.ScopeOwn},
		})
		err := lookup(ctx)
		require.Error(t, err, kind)
		assert.NotErrorIs(t, err, customerports.ErrCacheMiss, "%s should have asked Redis", kind)
	}
}

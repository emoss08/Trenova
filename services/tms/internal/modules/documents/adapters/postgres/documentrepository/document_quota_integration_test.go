//go:build integration

package documentrepository_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateRefusesADocumentPastTheStorageQuota(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	fixtures := fixturesOf(t.Context(), seedtest.SeedFullTestData(t, ctx, admin))
	conn := dbtest.NewRLSConnection(t, admin, dsn)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := documentrepository.New(documentrepository.Params{
		DB: conn, Logger: zap.NewNop(), Quota: guard, Scopes: managedscope.NewResolver(true),
	})
	entity := createTestDocument(fixtures)
	entity.FileSize = 2048
	tenant := pagination.TenantInfo{OrgID: fixtures.orgID, BuID: fixtures.buID}
	inTx := mock.MatchedBy(func(ctx context.Context) bool {
		return keel.InTx(ctx, conn.Keel(ctx))
	})

	guard.EXPECT().Enforce(inTx, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterDocumentFileBytes,
		Quantity:   2048,
	}).Return(nil).Once()
	guard.EXPECT().Enforce(inTx, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterDocumentUploads,
		Quantity:   1,
	}).Return(nil).Once()
	guard.EXPECT().Enforce(inTx, &platformplanports.QuotaRequest{
		TenantInfo: tenant,
		Meter:      platformcatalog.MeterDocumentStorageBytes,
		Quantity:   2048,
	}).Return(errortypes.NewQuotaExceededError(
		string(platformcatalog.MeterDocumentStorageBytes), 100, 100, "free_demo",
	)).Once()

	created, err := repo.Create(fixtures.ctx, entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))

	d := documentdb.Documents
	exists, err := d.Where(d.ID.Eq(entity.ID)).Exists(fixtures.ctx, admin)
	require.NoError(t, err)
	require.False(t, exists)
}

type storageCeilingGuard struct {
	level   func(ctx context.Context) (int64, error)
	ceiling int64
}

func (g *storageCeilingGuard) Enforce(
	ctx context.Context,
	req *platformplanports.QuotaRequest,
) error {
	if req.Meter != platformcatalog.MeterDocumentStorageBytes {
		return nil
	}
	used, err := g.level(ctx)
	if err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	if used+req.Quantity > g.ceiling {
		return errortypes.NewQuotaExceededError(string(req.Meter), g.ceiling, used, "managed")
	}
	return nil
}

func (*storageCeilingGuard) Check(
	context.Context,
	*platformplanports.QuotaRequest,
) (*platformplanports.QuotaDecision, error) {
	return &platformplanports.QuotaDecision{Allowed: true}, nil
}

func (*storageCeilingGuard) Usage(
	context.Context,
	pagination.TenantInfo,
) (*platformplanports.QuotaUsageSummary, error) {
	return &platformplanports.QuotaUsageSummary{}, nil
}

func TestConcurrentUploadsNeverStorePastTheCeiling(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	fixtures := fixturesOf(t.Context(), seedtest.SeedFullTestData(t, ctx, admin))
	conn := dbtest.NewRLSConnection(t, admin, dsn)
	stored0, err := keel.QueryValue[int64](fixtures.ctx, admin,
		`SELECT COALESCE(sum(bytes), 0)::bigint FROM document_storage_totals WHERE organization_id = $1`,
		fixtures.orgID.String())
	require.NoError(t, err)
	guard := &storageCeilingGuard{
		ceiling: stored0 + 3000,
		level: func(ctx context.Context) (int64, error) {
			return keel.QueryValue[int64](ctx, conn.Keel(ctx),
				`SELECT COALESCE(sum(bytes), 0)::bigint FROM document_storage_totals
                 WHERE organization_id = $1`, fixtures.orgID.String())
		},
	}
	repo := documentrepository.New(documentrepository.Params{
		DB: conn, Logger: zap.NewNop(), Quota: guard, Scopes: managedscope.NewResolver(true),
	})

	const uploads = 3
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		stored  int
		refused int
	)
	for i := range uploads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entity := createTestDocument(fixtures)
			entity.ID = pulid.MustNew("doc_")
			entity.FileSize = 2048
			entity.StoragePath = fixtures.orgID.String() + "/trailer/concurrent-" +
				string(rune('a'+i)) + ".pdf"
			_, err := repo.Create(fixtures.ctx, entity)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				stored++
			case errortypes.IsQuotaExceededError(err):
				refused++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, stored, "only one 2 KB document fits in the 3 KB left under the ceiling")
	assert.Equal(t, uploads-1, refused)
	var total int64
	total, err = keel.QueryValue[int64](fixtures.ctx, admin,
		`SELECT COALESCE(sum(bytes), 0)::bigint FROM document_storage_totals WHERE organization_id = $1`,
		fixtures.orgID.String())
	require.NoError(t, err)
	assert.LessOrEqual(t, total, guard.ceiling, "zero bytes past the ceiling")
}

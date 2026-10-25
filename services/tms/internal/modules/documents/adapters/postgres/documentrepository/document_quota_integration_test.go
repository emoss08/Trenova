//go:build integration

package documentrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	repo := documentrepository.New(documentrepository.Params{DB: conn, Logger: zap.NewNop(), Quota: guard})
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

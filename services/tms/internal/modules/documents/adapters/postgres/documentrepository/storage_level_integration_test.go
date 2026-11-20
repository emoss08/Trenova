//go:build integration

package documentrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentmeter"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentrepository"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestTheStorageLevelFollowsInsertsUpdatesAndDeletes(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	fixtures := fixturesOf(t.Context(), seedtest.SeedFullTestData(t, ctx, admin))
	conn := dbtest.NewRLSConnection(t, admin, dsn)
	repo := documentrepository.New(documentrepository.Params{DB: conn, Logger: zap.NewNop()})
	level := documentmeter.New(documentmeter.Params{DB: conn}).Reader
	tenant := pagination.TenantInfo{OrgID: fixtures.orgID, BuID: fixtures.buID}
	read := func() int64 {
		t.Helper()
		n, err := level.Level(t.Context(), managedscope.Instance())
		require.NoError(t, err)
		return n
	}
	start := read()

	small, err := repo.Create(
		fixtures.ctx,
		createTestDocument(fixtures, func(d *document.Document) {
			d.FileSize = 1024
		}),
	)
	require.NoError(t, err)
	large, err := repo.Create(
		fixtures.ctx,
		createTestDocument(fixtures, func(d *document.Document) {
			d.ID = pulid.MustNew("doc_")
			d.FileSize = 4096
			d.StoragePath = fixtures.orgID.String() + "/trailer/large.pdf"
		}),
	)
	require.NoError(t, err)
	require.Equal(t, start+1024+4096, read())

	large.FileSize = 2048
	large, err = repo.Update(fixtures.ctx, large)
	require.NoError(t, err)
	require.Equal(t, start+1024+2048, read())

	require.NoError(t, repo.Delete(fixtures.ctx, documentports.DeleteDocumentRequest{
		ID:         small.ID,
		TenantInfo: tenant,
	}))
	require.Equal(t, start+2048, read())

	require.NoError(t, repo.BulkDelete(fixtures.ctx, documentports.BulkDeleteDocumentRequest{
		IDs:        []pulid.ID{large.ID},
		TenantInfo: tenant,
	}))
	require.Equal(t, start, read())
}

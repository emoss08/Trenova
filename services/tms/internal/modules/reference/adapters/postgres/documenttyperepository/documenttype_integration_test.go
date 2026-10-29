//go:build integration

package documenttyperepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedDocumentType(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenant *seedtest.TestData,
	code string,
) *documenttype.DocumentType {
	t.Helper()

	entity := &documenttype.DocumentType{Code: code, Name: code + " document"}
	require.NoError(t, documenttypedb.DocumentTypes.Create(tenant.Ctx(ctx), db, entity))

	return entity
}

func TestGetByIDsReturnsOnlyTheTenantsDocumentTypes(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	tenantA := seedtest.SeedFullTestData(t, ctx, db)
	tenantB := seedtest.SeedAdditionalTenant(t, ctx, db, "DQ")
	first := seedDocumentType(t, ctx, db, tenantA, "BYIDPOD")
	second := seedDocumentType(t, ctx, db, tenantA, "BYIDBOL")
	foreign := seedDocumentType(t, ctx, db, tenantB, "BYIDINV")

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})

	entities, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		documenttypeports.GetDocumentTypesByIDsRequest{
			TenantInfo:      tenantA.Info(),
			DocumentTypeIDs: []pulid.ID{second.ID, foreign.ID, first.ID, pulid.MustNew("dt_")},
		},
	)
	require.NoError(t, err)

	codes := make(map[pulid.ID]string, len(entities))
	for _, entity := range entities {
		codes[entity.ID] = entity.Code
	}
	assert.Equal(t, map[pulid.ID]string{first.ID: "BYIDPOD", second.ID: "BYIDBOL"}, codes)

	none, err := repo.GetByIDs(
		tenantA.Ctx(ctx),
		documenttypeports.GetDocumentTypesByIDsRequest{TenantInfo: tenantA.Info()},
	)
	require.NoError(t, err)
	assert.Empty(t, none)
}

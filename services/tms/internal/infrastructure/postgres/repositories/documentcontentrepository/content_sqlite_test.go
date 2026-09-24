package documentcontentrepository_test

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/documentcontentrepository"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

const searchSchema = `
CREATE TABLE documents (
	id TEXT NOT NULL,
	organization_id TEXT NOT NULL,
	business_unit_id TEXT NOT NULL,
	resource_id TEXT NOT NULL,
	resource_type TEXT NOT NULL,
	file_name TEXT NOT NULL,
	original_name TEXT NOT NULL,
	is_current_version BOOLEAN NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE TABLE document_contents (
	id TEXT NOT NULL,
	document_id TEXT NOT NULL,
	organization_id TEXT NOT NULL,
	business_unit_id TEXT NOT NULL,
	content_text TEXT
);`

type searchFixture struct {
	db     *bun.DB
	repo   repositories.DocumentContentRepository
	tenant pagination.TenantInfo
}

func newSearchFixture(t *testing.T) *searchFixture {
	t.Helper()

	sqldb, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	sqldb.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqldb.Close() })

	db := bun.NewDB(sqldb, sqlitedialect.New())
	_, err = db.ExecContext(t.Context(), searchSchema)
	require.NoError(t, err)

	return &searchFixture{
		db: db,
		repo: documentcontentrepository.New(documentcontentrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
}

func (f *searchFixture) insert(t *testing.T, name, text string) pulid.ID {
	t.Helper()

	id := pulid.MustNew("doc_")
	_, err := f.db.ExecContext(t.Context(),
		`INSERT INTO documents VALUES (?, ?, ?, 'shp_1', 'shipment', ?, ?, 1, 1)`,
		id.String(), f.tenant.OrgID.String(), f.tenant.BuID.String(), name, name)
	require.NoError(t, err)
	_, err = f.db.ExecContext(t.Context(),
		`INSERT INTO document_contents VALUES (?, ?, ?, ?, ?)`,
		pulid.MustNew("dc_").String(), id.String(),
		f.tenant.OrgID.String(), f.tenant.BuID.String(), text)
	require.NoError(t, err)

	return id
}

func (f *searchFixture) search(t *testing.T, query string) []pulid.ID {
	t.Helper()

	docs, err := f.repo.SearchByResource(t.Context(), &repositories.DocumentContentSearchRequest{
		TenantInfo:   f.tenant,
		ResourceID:   "shp_1",
		ResourceType: "shipment",
		Query:        query,
	})
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(docs))
	for _, doc := range docs {
		ids = append(ids, doc.ID)
	}

	return ids
}

func TestSearchByResource_LikeFallbackMatchesPercentLiterally(t *testing.T) {
	t.Parallel()

	f := newSearchFixture(t)
	discount := f.insert(t, "fuel-surcharge.pdf", "a 100% fuel surcharge applies")
	f.insert(t, "rate.pdf", "linehaul 1000 dollars")

	assert.Equal(t, []pulid.ID{discount}, f.search(t, "100%"))
}

func TestSearchByResource_LikeFallbackMatchesUnderscoreLiterally(t *testing.T) {
	t.Parallel()

	f := newSearchFixture(t)
	underscored := f.insert(t, "pod_final.pdf", "signed")
	f.insert(t, "podXfinal.pdf", "signed")

	assert.Equal(t, []pulid.ID{underscored}, f.search(t, "pod_final"))
}

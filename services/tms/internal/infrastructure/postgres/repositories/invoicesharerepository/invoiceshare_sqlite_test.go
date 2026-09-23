package invoicesharerepository_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicesharerepository"
	sqlitemigrations "github.com/emoss08/trenova/internal/infrastructure/sqlite/migrations"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

type fixture struct {
	repo      repositories.InvoiceShareRepository
	db        *bun.DB
	tenant    pagination.TenantInfo
	invoiceID pulid.ID
	sharer    *tenant.User
	dana      *tenant.User
	priya     *tenant.User
}

var sqliteDir string

func TestMain(m *testing.M) {
	code := m.Run()
	if db, err := migratedDB(); err == nil && db != nil {
		_ = db.Close()
	}
	if sqliteDir != "" {
		_ = os.RemoveAll(sqliteDir)
	}
	os.Exit(code)
}

var migratedDB = sync.OnceValues(func() (*bun.DB, error) {
	dir, err := os.MkdirTemp("", "invoiceshare-sqlite-*")
	if err != nil {
		return nil, err
	}
	sqliteDir = dir
	sqliteDir = dir

	sqldb, err := sql.Open(
		"sqlite",
		"file:"+filepath.Join(dir, "invoiceshare.db")+
			"?_pragma=foreign_keys(0)&_pragma=busy_timeout(10000)&_txlock=immediate",
	)
	if err != nil {
		return nil, err
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	postgres.NewTestConnection(db)

	migrations, err := sqlitemigrations.Migrations()
	if err != nil {
		return nil, err
	}
	migrator := migrate.NewMigrator(db, migrations)
	ctx := context.Background()
	if err = migrator.Init(ctx); err != nil {
		return nil, err
	}
	if _, err = migrator.Migrate(ctx); err != nil {
		return nil, err
	}

	return db, nil
})

func newFixture(t *testing.T) *fixture {
	t.Helper()

	db, err := migratedDB()
	require.NoError(t, err)

	f := &fixture{
		repo: invoicesharerepository.New(invoicesharerepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		db:        db,
		tenant:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		invoiceID: pulid.MustNew("inv_"),
	}
	f.sharer = f.insertUser(t, "Marcus Bell", "mbell")
	f.dana = f.insertUser(t, "Dana Whitfield", "dwhitfield")
	f.priya = f.insertUser(t, "Priya Nair", "pnair")

	return f
}

func (f *fixture) insertUser(t *testing.T, name, username string) *tenant.User {
	t.Helper()

	id := pulid.MustNew("usr_")
	suffix := strings.ToLower(id.String()[len(id.String())-8:])
	user := &tenant.User{
		ID:                    id,
		BusinessUnitID:        f.tenant.BuID,
		CurrentOrganizationID: f.tenant.OrgID,
		Status:                domaintypes.StatusActive,
		Name:                  name,
		Username:              username + suffix,
		Password:              "hashed",
		EmailAddress:          username + suffix + "@example.com",
		Timezone:              "America/Chicago",
		Locale:                "en",
	}
	_, err := f.db.NewInsert().Model(user).Exec(t.Context())
	require.NoError(t, err)

	return user
}

func (f *fixture) share(
	recipient *tenant.User,
	note string,
	tab invoice.ShareTab,
	at int64,
) *invoice.InvoiceShare {
	return &invoice.InvoiceShare{
		OrganizationID: f.tenant.OrgID,
		BusinessUnitID: f.tenant.BuID,
		InvoiceID:      f.invoiceID,
		SharedWithID:   recipient.ID,
		SharedByID:     f.sharer.ID,
		Note:           note,
		Tab:            tab,
		ShareCount:     1,
		FirstSharedAt:  at,
		LastSharedAt:   at,
		CreatedAt:      at,
		UpdatedAt:      at,
	}
}

func (f *fixture) list(t *testing.T) []*invoice.InvoiceShare {
	t.Helper()

	shares, err := f.repo.ListByInvoiceID(t.Context(), &repositories.ListInvoiceSharesRequest{
		TenantInfo: f.tenant,
		InvoiceID:  f.invoiceID,
	})
	require.NoError(t, err)

	return shares
}

func TestUpsertCreatesOneRowPerRecipient(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "Check the detention line", invoice.ShareTabCharges, 1_000),
		f.share(f.priya, "", invoice.ShareTabOverview, 1_000),
	}))

	shares := f.list(t)
	require.Len(t, shares, 2)
	for _, share := range shares {
		assert.Equal(t, 1, share.ShareCount)
		require.NotNil(t, share.SharedWith)
		require.NotNil(t, share.SharedBy)
		assert.Equal(t, f.sharer.Name, share.SharedBy.Name)
		assert.Empty(t, share.SharedWith.Password)
	}
}

func TestUpsertReshareUpdatesTheExistingRow(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "First look", invoice.ShareTabOverview, 1_000),
	}))
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "Second look at the charges", invoice.ShareTabCharges, 2_000),
	}))

	shares := f.list(t)
	require.Len(t, shares, 1)
	assert.Equal(t, 2, shares[0].ShareCount)
	assert.Equal(t, "Second look at the charges", shares[0].Note)
	assert.Equal(t, invoice.ShareTabCharges, shares[0].Tab)
	assert.Equal(t, int64(1_000), shares[0].FirstSharedAt)
	assert.Equal(t, int64(2_000), shares[0].LastSharedAt)
	assert.Equal(t, f.dana.Name, shares[0].SharedWith.Name)
}

func TestUpsertWithoutANoteClearsThePreviousNote(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "Old note", invoice.ShareTabOverview, 1_000),
	}))
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "", invoice.ShareTabOverview, 2_000),
	}))

	shares := f.list(t)
	require.Len(t, shares, 1)
	assert.Empty(t, shares[0].Note)
}

func TestListOrdersByMostRecentShareAndStaysInTenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.dana, "", invoice.ShareTabOverview, 1_000),
	}))
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{
		f.share(f.priya, "", invoice.ShareTabOverview, 3_000),
	}))

	otherTenant := f.share(f.dana, "", invoice.ShareTabOverview, 5_000)
	otherTenant.OrganizationID = pulid.MustNew("org_")
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{otherTenant}))

	otherInvoice := f.share(f.dana, "", invoice.ShareTabOverview, 6_000)
	otherInvoice.InvoiceID = pulid.MustNew("inv_")
	require.NoError(t, f.repo.Upsert(t.Context(), []*invoice.InvoiceShare{otherInvoice}))

	shares := f.list(t)
	require.Len(t, shares, 2)
	assert.Equal(t, f.priya.ID, shares[0].SharedWithID)
	assert.Equal(t, f.dana.ID, shares[1].SharedWithID)
}

func TestUpsertWithNoSharesIsANoop(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	require.NoError(t, f.repo.Upsert(t.Context(), nil))
	assert.Empty(t, f.list(t))
}

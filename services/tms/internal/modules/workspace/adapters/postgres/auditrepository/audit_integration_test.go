//go:build integration

package auditrepository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	ctx  context.Context
	db   *keel.DB
	conn *postgres.Connection
	repo auditports.AuditRepository
	data *seedtest.TestData
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.WrapDB(db)

	return &fixture{
		ctx:  data.Ctx(ctx),
		db:   db,
		conn: conn,
		repo: New(Params{DB: conn, Logger: zap.NewNop()}),
		data: data,
	}
}

func (f *fixture) entry(resource permission.Resource, resourceID string, category audit.Category) *audit.Entry {
	e := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		OrganizationID: f.data.Organization.ID,
		BusinessUnitID: f.data.BusinessUnit.ID,
		Timestamp:      time.Now().Unix(),
		Resource:       resource,
		ResourceID:     resourceID,
		Operation:      permission.OpUpdate,
		Category:       category,
		PrincipalType:  "session_user",
		PrincipalID:    f.data.User.ID,
		UserID:         f.data.User.ID,
		Changes:        map[string]any{"status": map[string]any{"from": "New", "to": "Assigned"}},
	}
	e.Normalize()
	return e
}

func (f *fixture) list(t *testing.T, req *auditports.ListAuditEntriesRequest) []*audit.Entry {
	t.Helper()

	if req.Filter == nil {
		req.Filter = &pagination.QueryOptions{
			TenantInfo: f.data.Info(),
			Pagination: pagination.Info{Limit: 50},
		}
	}
	result, err := f.repo.ListConnection(f.ctx, req)
	require.NoError(t, err)
	return result.Items
}

func TestInsert_NamesTheActorAsTheyWereWhenTheEntryWasWritten(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	got, err := f.repo.GetByID(f.ctx, &auditports.GetAuditEntryByIDOptions{
		EntryID: e.ID,
		Access:  auditports.EntryAccess{AllResources: true},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, got.ActorName, "the insert trigger names the user")
	assert.Equal(t, "shipment.update", got.Action)
	assert.Equal(t, audit.ChannelWeb, got.Channel)
}

func TestInsert_JoinsTheCallersTransactionAndRollsBackWithIt(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_rolled_back", audit.CategoryUser)
	failure := errors.New("the business write failed after the entry was recorded")
	err := dbtx.WriteErr(f.ctx, f.conn, func(ctx context.Context) error {
		require.NoError(t, f.repo.Insert(ctx, []*audit.Entry{e}))
		return failure
	})
	require.ErrorIs(t, err, failure)

	_, err = f.repo.GetByID(f.ctx, &auditports.GetAuditEntryByIDOptions{
		EntryID: e.ID,
		Access:  auditports.EntryAccess{AllResources: true},
	})
	require.Error(t, err, "an entry written in a transaction that rolled back is not kept")
}

func TestListConnection_ShowsOnlyTheResourcesTheReaderCanReadAndHidesSecurity(t *testing.T) {
	f := newFixture(t)

	shipment := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	invoice := f.entry(permission.ResourceInvoice, "inv_1", audit.CategoryUser)
	role := f.entry(permission.ResourceRole, "rol_1", audit.CategorySecurity)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{shipment, invoice, role}))

	items := f.list(t, &auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{
		Resources: []permission.Resource{permission.ResourceShipment, permission.ResourceRole},
	}})
	require.Len(t, items, 1)
	assert.Equal(t, shipment.ID, items[0].ID)

	items = f.list(t, &auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{
		AllResources:    true,
		IncludeSecurity: true,
	}})
	assert.Len(t, items, 3)

	assert.Empty(t, f.list(t, &auditports.ListAuditEntriesRequest{}), "no readable resource reads nothing")
}

func TestListConnection_LeavesTheStatesOutOfASummaryPage(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	summary := f.list(t, &auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{AllResources: true}})
	require.Len(t, summary, 1)
	assert.Nil(t, summary[0].Changes)
	assert.Equal(t, "shipment.update", summary[0].Action)

	detail := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:        auditports.EntryAccess{AllResources: true},
		IncludeDetail: true,
	})
	require.Len(t, detail, 1)
	assert.Contains(t, detail[0].Changes, "status")
}

func TestListConnection_FieldHistoryKeepsOnlyEntriesThatChangedTheField(t *testing.T) {
	f := newFixture(t)

	status := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	weight := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	weight.Changes = map[string]any{"weight": map[string]any{"from": 1, "to": 2}}
	other := f.entry(permission.ResourceShipment, "shp_2", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{status, weight, other}))

	items := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:       auditports.EntryAccess{AllResources: true},
		ResourceID:   "shp_1",
		ChangedField: "weight",
	})
	require.Len(t, items, 1)
	assert.Equal(t, weight.ID, items[0].ID)
}

func TestPartitionHorizon_ReachesTwoYearsPastTheMigration(t *testing.T) {
	f := newFixture(t)

	horizon, err := f.repo.PartitionHorizon(f.ctx)
	require.NoError(t, err)
	assert.Greater(t, horizon, time.Now().AddDate(1, 11, 0).Unix(),
		"the migration creates partitions 24 months ahead")

	future := f.entry(permission.ResourceShipment, "shp_future", audit.CategoryUser)
	future.Timestamp = time.Now().AddDate(1, 6, 0).Unix()
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{future}))
}

func TestTenantsWithEntries_NamesTheTenantsWithEntriesInTheWindow(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	tenants, err := f.repo.TenantsWithEntries(f.ctx, e.Timestamp-60, e.Timestamp+60)
	require.NoError(t, err)
	assert.Contains(t, tenants, auditports.TenantKey{
		OrganizationID: f.data.Organization.ID,
		BusinessUnitID: f.data.BusinessUnit.ID,
	})

	tenants, err = f.repo.TenantsWithEntries(f.ctx, e.Timestamp+60, e.Timestamp+120)
	require.NoError(t, err)
	assert.Empty(t, tenants)
}

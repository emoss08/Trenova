//go:build integration

package workerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func newTestWorker(data *seedtest.TestData) *worker.Worker {
	return &worker.Worker{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		StateID:        data.State.ID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "John",
		LastName:       "Doe",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "62701",
		Gender:         worker.GenderMale,
		Profile: &worker.WorkerProfile{
			DOB:           timeutils.NowUnix() - 86400*365*30,
			LicenseNumber: "DL123456",
			LicenseExpiry: timeutils.NowUnix() + 86400*365,
			HireDate:      timeutils.NowUnix() - 86400*365,
		},
	}
}

func setupRepo(t *testing.T) (*repository, *seedtest.TestData, func()) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	}).(*repository)

	return repo, data, cleanup
}

func TestCreateAndGetByID(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.ID.IsNotNil())
	assert.Equal(t, "John", created.FirstName)

	fetched, err := repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: data.Organization.ID,
			BuID:  data.BusinessUnit.ID,
		},
		IncludeProfile: true,
		IncludeState:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "John", fetched.FirstName)
	assert.NotNil(t, fetched.Profile)
	assert.NotNil(t, fetched.State)
}

func TestGetByIDNotFoundForWrongTenant(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID: created.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestUpdateReturnsVersionMismatchForStaleEntity(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)

	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	firstCopy, err := repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	secondCopy, err := repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	firstCopy.LastName = "Updated"
	_, err = repo.Update(ctx, firstCopy)
	require.NoError(t, err)

	secondCopy.LastName = "Stale"
	updated, err := repo.Update(ctx, secondCopy)
	require.Nil(t, updated)
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrVersionMismatch, validationErr.Code)
}

func TestListFiltersWorkersByTenant(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	w := newTestWorker(data)
	_, err := repo.Create(ctx, w)
	require.NoError(t, err)

	w2 := newTestWorker(data)
	w2.FirstName = "Jane"
	_, err = repo.Create(ctx, w2)
	require.NoError(t, err)

	result, err := repo.List(ctx, &repositories.ListWorkersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID: data.Organization.ID,
				BuID:  data.BusinessUnit.ID,
			},
			Pagination: pagination.Info{
				Limit:  10,
				Offset: 0,
			},
		},
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Total, 2)
}

func TestGetWorkerSyncReadinessCounts(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	w := newTestWorker(data)
	_, err := repo.Create(ctx, w)
	require.NoError(t, err)

	wSynced := newTestWorker(data)
	wSynced.FirstName = "Synced"
	wSynced.ExternalID = "ext_123"
	_, err = repo.Create(ctx, wSynced)
	require.NoError(t, err)

	counts, err := repo.GetWorkerSyncReadinessCounts(ctx, pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, counts.TotalWorkers, 2)
	assert.GreaterOrEqual(t, counts.ActiveWorkers, 2)
	assert.GreaterOrEqual(t, counts.SyncedActiveWorkers, 1)
}

func TestBuncolgenColumnsMatchDatabase(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)

	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	t.Run("qualified column in where clause", func(t *testing.T) {
		entity := new(worker.Worker)
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(buncolgen.WorkerColumns.ID.Eq(), created.ID).
			Where(buncolgen.WorkerColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.WorkerColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Scan(ctx)
		require.NoError(t, err)
		assert.Equal(t, created.ID, entity.ID)
		assert.Equal(t, "John", entity.FirstName)
	})

	t.Run("order by expression", func(t *testing.T) {
		var entities []*worker.Worker
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(buncolgen.WorkerColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.WorkerColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Order(buncolgen.WorkerColumns.CreatedAt.OrderDesc()).
			Scan(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, entities)
	})

	t.Run("column string in select", func(t *testing.T) {
		entity := new(worker.Worker)
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Column(
				buncolgen.WorkerColumns.ID.String(),
				buncolgen.WorkerColumns.FirstName.String(),
				buncolgen.WorkerColumns.LastName.String(),
				buncolgen.WorkerColumns.Status.String(),
			).
			Where(buncolgen.WorkerColumns.ID.Eq(), created.ID).
			Where(buncolgen.WorkerColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.WorkerColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Scan(ctx)
		require.NoError(t, err)
		assert.Equal(t, "John", entity.FirstName)
		assert.Equal(t, "Doe", entity.LastName)
		assert.Equal(t, domaintypes.StatusActive, entity.Status)
	})

	t.Run("is null expression", func(t *testing.T) {
		var entities []*worker.Worker
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(buncolgen.WorkerColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.WorkerColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Where(buncolgen.WorkerColumns.FleetCodeID.IsNull()).
			Scan(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, entities)
	})

	t.Run("status filter with eq expression", func(t *testing.T) {
		var entities []*worker.Worker
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(buncolgen.WorkerColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.WorkerColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Where(buncolgen.WorkerColumns.Status.Eq(), domaintypes.StatusActive).
			Scan(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, entities)
	})

	t.Run("table info matches database", func(t *testing.T) {
		assert.Equal(t, "workers", buncolgen.WorkerTable.Name)
		assert.Equal(t, "wrk", buncolgen.WorkerTable.Alias)
		assert.Contains(t, buncolgen.WorkerTable.PrimaryKey, "id")
		assert.Contains(t, buncolgen.WorkerTable.PrimaryKey, "business_unit_id")
		assert.Contains(t, buncolgen.WorkerTable.PrimaryKey, "organization_id")
	})

	t.Run("field map matches entity json tags", func(t *testing.T) {
		fm := buncolgen.WorkerFieldMap
		assert.Equal(t, "id", fm["id"])
		assert.Equal(t, "first_name", fm["firstName"])
		assert.Equal(t, "last_name", fm["lastName"])
		assert.Equal(t, "status", fm["status"])
		assert.Equal(t, "organization_id", fm["organizationId"])
		assert.Equal(t, "business_unit_id", fm["businessUnitId"])
	})
}

func TestRelationAndNestedRelationLoading(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := t.Context()

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)

	t.Run("single relation via constant", func(t *testing.T) {
		entity := new(worker.Worker)
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(buncolgen.WorkerRelations.Profile).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WorkerScopeTenant(sq, pagination.TenantInfo{
					OrgID: data.Organization.ID,
					BuID:  data.BusinessUnit.ID,
				}).Where(buncolgen.WorkerColumns.ID.Eq(), created.ID)
			}).
			Scan(ctx)
		require.NoError(t, err)
		assert.NotNil(t, entity.Profile)
	})

	t.Run("nested relation via Rel helper", func(t *testing.T) {
		entity := new(worker.Worker)
		err := repo.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(buncolgen.WorkerRelations.Profile).
			Relation(buncolgen.Rel(buncolgen.WorkerRelations.Profile, "LicenseState")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.WorkerScopeTenant(sq, pagination.TenantInfo{
					OrgID: data.Organization.ID,
					BuID:  data.BusinessUnit.ID,
				}).Where(buncolgen.WorkerColumns.ID.Eq(), created.ID)
			}).
			Scan(ctx)
		require.NoError(t, err)
		assert.NotNil(t, entity.Profile)
	})
}

func TestStaticFieldMapperIntegration(t *testing.T) {
	w := &worker.Worker{}
	fm := w.GetStaticFieldMap()

	require.NotNil(t, fm)
	assert.Equal(t, "first_name", fm["firstName"])
	assert.Equal(t, "organization_id", fm["organizationId"])

	_, hasSearchVector := fm["searchVector"]
	assert.False(t, hasSearchVector, "searchVector should not be in FieldMap (json:\"-\")")
}

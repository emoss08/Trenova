//go:build integration

package workerrepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type fixture struct {
	repo *repository
	data *seedtest.TestData
	db   *keel.DB
	ctx  context.Context
}

func setupRepo(t *testing.T) (*repository, *seedtest.TestData, func()) {
	t.Helper()

	f, cleanup := setupFixture(t)

	return f.repo, f.data, cleanup
}

func setupFixture(t *testing.T) (*fixture, func()) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	data := seedtest.SeedFullTestData(t, ctx, db)
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	}).(*repository)

	return &fixture{repo: repo, data: data, db: db, ctx: ctx}, cleanup
}

func TestCreateAndGetByID(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := data.Ctx(t.Context())

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.ID.IsNotNil())
	assert.Equal(t, "John", created.FirstName)

	fetched, err := repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
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
	f, cleanup := setupFixture(t)
	t.Cleanup(cleanup)
	ctx := f.data.Ctx(t.Context())

	created, err := f.repo.Create(ctx, newTestWorker(f.data))
	require.NoError(t, err)

	other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "WT")
	_, err = f.repo.GetByID(other.Ctx(t.Context()), workerports.GetWorkerByIDRequest{
		ID:         created.ID,
		TenantInfo: other.Info(),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestUpdateReturnsVersionMismatchForStaleEntity(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := data.Ctx(t.Context())

	created, err := repo.Create(ctx, newTestWorker(data))
	require.NoError(t, err)

	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}

	firstCopy, err := repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	secondCopy, err := repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
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
	ctx := data.Ctx(t.Context())

	w := newTestWorker(data)
	_, err := repo.Create(ctx, w)
	require.NoError(t, err)

	w2 := newTestWorker(data)
	w2.FirstName = "Jane"
	_, err = repo.Create(ctx, w2)
	require.NoError(t, err)

	result, err := repo.List(ctx, &workerports.ListWorkersRequest{
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
	assert.GreaterOrEqual(t, len(result.Items), 2)
}

func TestGetWorkerSyncReadinessCounts(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := data.Ctx(t.Context())

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

func TestStaticFieldMapperIntegration(t *testing.T) {
	w := &worker.Worker{}
	fm := w.GetStaticFieldMap()

	require.NotNil(t, fm)
	assert.Equal(t, "first_name", fm["firstName"])
	assert.Equal(t, "organization_id", fm["organizationId"])

	_, hasSearchVector := fm["searchVector"]
	assert.False(t, hasSearchVector, "searchVector should not be in FieldMap (json:\"-\")")
}

func TestGetByID_LoadsTheManagerFleetAndLicenseStateItIsAskedFor(t *testing.T) {
	repo, data, cleanup := setupRepo(t)
	t.Cleanup(cleanup)
	ctx := data.Ctx(t.Context())

	entity := newTestWorker(data)
	entity.ManagerID = data.User.ID
	entity.Profile.LicenseStateID = data.State.ID
	created, err := repo.Create(ctx, entity)
	require.NoError(t, err)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}

	full, err := repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:                  created.ID,
		TenantInfo:          tenant,
		IncludeProfile:      true,
		IncludeFleetCode:    true,
		IncludeManager:      true,
		IncludeLicenseState: true,
	})
	require.NoError(t, err)
	require.NotNil(t, full.Manager)
	assert.Equal(t, data.User.ID, full.Manager.ID)
	require.NotNil(t, full.Profile)
	require.NotNil(t, full.Profile.LicenseState)
	assert.Equal(t, data.State.ID, full.Profile.LicenseState.ID)
	assert.Nil(t, full.FleetCode)

	bare, err := repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         created.ID,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Nil(t, bare.Manager)
	assert.Nil(t, bare.Profile)
}

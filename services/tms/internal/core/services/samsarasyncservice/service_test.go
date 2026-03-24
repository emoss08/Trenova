package samsarasyncservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeSamsaraDriverService struct {
	listAllFunc func(params drivers.ListParams) ([]drivers.Driver, error)
	createFunc  func(req drivers.CreateRequest) (drivers.Driver, error)
	updateFunc  func(id string, req drivers.UpdateRequest) (drivers.Driver, error)
}

func (f *fakeSamsaraDriverService) List(
	_ context.Context,
	params drivers.ListParams,
) (drivers.ListResponse, error) {
	items, err := f.ListAll(context.Background(), params)
	if err != nil {
		return drivers.ListResponse{}, err
	}
	return drivers.ListResponse{Data: &items}, nil
}

func (f *fakeSamsaraDriverService) ListAll(
	_ context.Context,
	params drivers.ListParams,
) ([]drivers.Driver, error) {
	if f.listAllFunc == nil {
		return []drivers.Driver{}, nil
	}
	return f.listAllFunc(params)
}

func (f *fakeSamsaraDriverService) Create(
	_ context.Context,
	req drivers.CreateRequest,
) (drivers.Driver, error) {
	if f.createFunc == nil {
		return drivers.Driver{}, nil
	}
	return f.createFunc(req)
}

func (f *fakeSamsaraDriverService) Update(
	_ context.Context,
	id string,
	req drivers.UpdateRequest,
) (drivers.Driver, error) {
	if f.updateFunc == nil {
		return drivers.Driver{}, nil
	}
	return f.updateFunc(id, req)
}

func setupTestService(t *testing.T) (*Service, *mocks.MockWorkerRepository) {
	t.Helper()
	repo := mocks.NewMockWorkerRepository(t)
	return &Service{
		l:    zap.NewNop(),
		repo: repo,
	}, repo
}

func newTestWorker() *worker.Worker {
	return &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		StateID:        pulid.MustNew("st_"),
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeOTR,
		FirstName:      "John",
		LastName:       "Doe",
		AddressLine1:   "123 Main St",
		City:           "Springfield",
		PostalCode:     "12345",
		Gender:         worker.GenderMale,
		Version:        1,
		Profile: &worker.WorkerProfile{
			LicenseNumber: "DL123456",
		},
	}
}

func TestSyncWorkersToSamsara(t *testing.T) {
	t.Parallel()

	t.Run("returns error when samsara is not configured", func(t *testing.T) {
		t.Parallel()
		svc, _ := setupTestService(t)

		result, err := svc.SyncWorkersToSamsara(t.Context(), pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, strings.ToLower(err.Error()), "samsara")
	})

	t.Run("maps worker from samsara external ids", func(t *testing.T) {
		t.Parallel()
		svc, repo := setupTestService(t)
		w := newTestWorker()
		w.ExternalID = ""

		driverID := "drv-42"
		externalIDs := map[string]any{samsaraWorkerExternalIDKey: w.ID.String()}
		svc.samsaraClient = &sharedsamsara.Client{
			Drivers: &fakeSamsaraDriverService{
				listAllFunc: func(params drivers.ListParams) ([]drivers.Driver, error) {
					assert.Equal(t, 512, params.Limit)
					return []drivers.Driver{{Id: &driverID, ExternalIds: &externalIDs}}, nil
				},
			},
		}

		repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{w},
			Total: 1,
		}, nil).Once()
		repo.On("Update", mock.Anything, mock.Anything).Return(
			func(_ context.Context, entity *worker.Worker) *worker.Worker { return entity },
			nil,
		).Once()

		result, err := svc.SyncWorkersToSamsara(t.Context(), pagination.TenantInfo{
			OrgID: w.OrganizationID,
			BuID:  w.BusinessUnitID,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.MappedFromExternalIDs)
		assert.Equal(t, 1, result.UpdatedMappings)
		assert.Equal(t, 0, result.CreatedDrivers)
	})

	t.Run("updates external ids for already mapped driver", func(t *testing.T) {
		t.Parallel()
		svc, repo := setupTestService(t)
		w := newTestWorker()
		w.ExternalID = "drv-77"

		svc.samsaraClient = &sharedsamsara.Client{
			Drivers: &fakeSamsaraDriverService{
				listAllFunc: func(params drivers.ListParams) ([]drivers.Driver, error) {
					assert.Equal(t, 512, params.Limit)
					return []drivers.Driver{{Id: &w.ExternalID}}, nil
				},
				updateFunc: func(id string, req drivers.UpdateRequest) (drivers.Driver, error) {
					assert.Equal(t, w.ExternalID, id)
					require.NotNil(t, req.ExternalIds)
					assert.Equal(t, w.ID.String(), (*req.ExternalIds)[samsaraWorkerExternalIDKey])
					return drivers.Driver{Id: &id}, nil
				},
			},
		}

		repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{w},
			Total: 1,
		}, nil).Once()

		result, err := svc.SyncWorkersToSamsara(t.Context(), pagination.TenantInfo{
			OrgID: w.OrganizationID,
			BuID:  w.BusinessUnitID,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.AlreadyMapped)
		assert.Equal(t, 1, result.UpdatedRemoteDrivers)
	})

	t.Run("creates missing samsara driver and persists mapping", func(t *testing.T) {
		t.Parallel()
		svc, repo := setupTestService(t)
		w := newTestWorker()
		w.Email = "driver.one@trenova.app"
		w.PhoneNumber = "(555) 123-4567"
		w.ExternalID = ""

		createdDriverID := "drv-1001"
		var capturedCreateReq drivers.CreateRequest
		svc.samsaraClient = &sharedsamsara.Client{
			Drivers: &fakeSamsaraDriverService{
				listAllFunc: func(params drivers.ListParams) ([]drivers.Driver, error) {
					assert.Equal(t, 512, params.Limit)
					return []drivers.Driver{}, nil
				},
				createFunc: func(req drivers.CreateRequest) (drivers.Driver, error) {
					capturedCreateReq = req
					return drivers.Driver{Id: &createdDriverID}, nil
				},
			},
		}

		repo.On("List", mock.Anything, mock.Anything).Return(&pagination.ListResult[*worker.Worker]{
			Items: []*worker.Worker{w},
			Total: 1,
		}, nil).Once()
		repo.On("Update", mock.Anything, mock.Anything).Return(
			func(_ context.Context, entity *worker.Worker) *worker.Worker { return entity },
			nil,
		).Once()

		result, err := svc.SyncWorkersToSamsara(t.Context(), pagination.TenantInfo{
			OrgID: w.OrganizationID,
			BuID:  w.BusinessUnitID,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 1, result.CreatedDrivers)
		assert.Equal(t, 1, result.UpdatedMappings)
		assert.Equal(t, w.FullName(), capturedCreateReq.Name)
		require.NotNil(t, capturedCreateReq.ExternalIds)
		assert.Equal(t, w.ID.String(), (*capturedCreateReq.ExternalIds)[samsaraWorkerExternalIDKey])
		require.NotNil(t, capturedCreateReq.Phone)
		assert.Equal(t, "+15551234567", *capturedCreateReq.Phone)
	})
}

func TestGetWorkerSyncReadiness(t *testing.T) {
	t.Parallel()

	t.Run("returns readiness metrics from worker counts", func(t *testing.T) {
		t.Parallel()

		svc, repo := setupTestService(t)
		tenantInfo := pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		}

		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			&repositories.WorkerSyncReadinessCounts{
				TotalWorkers:        12,
				ActiveWorkers:       10,
				SyncedActiveWorkers: 7,
			},
			nil,
		).Once()

		result, err := svc.GetWorkerSyncReadiness(t.Context(), tenantInfo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 12, result.TotalWorkers)
		assert.Equal(t, 10, result.ActiveWorkers)
		assert.Equal(t, 7, result.SyncedActiveWorkers)
		assert.Equal(t, 3, result.UnsyncedActiveWorkers)
		assert.False(t, result.AllActiveWorkersSynced)
		assert.Positive(t, result.LastCalculatedAt)
	})

	t.Run("marks all active workers synced when none are unsynced", func(t *testing.T) {
		t.Parallel()

		svc, repo := setupTestService(t)
		tenantInfo := pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		}

		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			&repositories.WorkerSyncReadinessCounts{
				TotalWorkers:        4,
				ActiveWorkers:       0,
				SyncedActiveWorkers: 0,
			},
			nil,
		).Once()

		result, err := svc.GetWorkerSyncReadiness(t.Context(), tenantInfo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 0, result.UnsyncedActiveWorkers)
		assert.True(t, result.AllActiveWorkersSynced)
	})

	t.Run("returns business error when repository fails", func(t *testing.T) {
		t.Parallel()

		svc, repo := setupTestService(t)
		tenantInfo := pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		}

		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			(*repositories.WorkerSyncReadinessCounts)(nil),
			errors.New("db failure"),
		).Once()

		result, err := svc.GetWorkerSyncReadiness(t.Context(), tenantInfo)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errortypes.IsBusinessError(err))
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockWorkerRepository(t)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo})
	require.NotNil(t, svc)
}

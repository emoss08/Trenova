package samsarasyncservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"

	"io"
	"net/http"
	"net/http/httptest"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/integrationconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/samsara/drivers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeSamsaraDriverService struct {
	listAllFunc func(params drivers.ListParams) ([]drivers.Driver, error)
	createFunc  func(req drivers.CreateRequest) (drivers.Driver, error)
	updateFunc  func(id string, req drivers.UpdateRequest) (drivers.Driver, error)
}

func (f *fakeSamsaraDriverService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, _ := strings.CutPrefix(strings.TrimPrefix(r.URL.Path, "/fleet/drivers"), "/")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var payload any
	switch r.Method {
	case http.MethodGet:
		payload, err = f.list(r)
	case http.MethodPost:
		payload, err = f.create(body)
	case http.MethodPatch:
		payload, err = f.update(id, body)
	default:
		err = errors.New("unexpected method " + r.Method)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	encoded, err := sonic.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encoded)
}

func (f *fakeSamsaraDriverService) list(r *http.Request) (any, error) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return nil, err
	}

	items := []drivers.Driver{}
	if f.listAllFunc != nil {
		items, err = f.listAllFunc(drivers.ListParams{
			Limit:                  limit,
			DriverActivationStatus: r.URL.Query().Get("driverActivationStatus"),
		})
		if err != nil {
			return nil, err
		}
	}

	return drivers.ListResponse{Data: &items}, nil
}

func (f *fakeSamsaraDriverService) create(body []byte) (any, error) {
	var req drivers.CreateRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	if f.createFunc == nil {
		return samsaraDriverEnvelope{Data: drivers.Driver{}}, nil
	}

	driver, err := f.createFunc(req)
	if err != nil {
		return nil, err
	}

	return samsaraDriverEnvelope{Data: driver}, nil
}

func (f *fakeSamsaraDriverService) update(id string, body []byte) (any, error) {
	var req drivers.UpdateRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	if f.updateFunc == nil {
		return samsaraDriverEnvelope{Data: drivers.Driver{}}, nil
	}

	driver, err := f.updateFunc(id, req)
	if err != nil {
		return nil, err
	}

	return samsaraDriverEnvelope{Data: driver}, nil
}

type samsaraDriverEnvelope struct {
	Data drivers.Driver `json:"data"`
}

type fakeIntegrationRepo struct {
	integrationports.IntegrationRepository
	record *integration.Integration
}

func (f *fakeIntegrationRepo) GetByType(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*integration.Integration, error) {
	if f.record == nil {
		return nil, errortypes.NewNotFoundError("integration not found")
	}
	return f.record, nil
}

func samsaraIntegration(
	t *testing.T,
	fake *fakeSamsaraDriverService,
) *integrationservice.Service {
	t.Helper()

	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)

	encryption := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("unit-test-encryption-key-with-at-least-32-bytes"),
	)
	token, err := encryption.EncryptString("samsara-test-token")
	require.NoError(t, err)

	return integrationservice.New(integrationservice.Params{
		Logger: zap.NewNop(),
		Repo: &fakeIntegrationRepo{record: &integration.Integration{
			Type:    integration.TypeSamsara,
			Enabled: true,
			Configuration: map[string]any{
				"token":   token,
				"baseUrl": server.URL,
			},
		}},
		Encryption: encryption,
		Config:     &integrationconfig.Config{AllowPrivateNetworks: true},
	})
}

func activeDrivers(
	t *testing.T,
	active []drivers.Driver,
) func(params drivers.ListParams) ([]drivers.Driver, error) {
	t.Helper()
	return func(params drivers.ListParams) ([]drivers.Driver, error) {
		assert.Equal(t, drivers.MaxListLimit, params.Limit)
		assert.NoError(t, params.Validate())
		if params.DriverActivationStatus == drivers.ActivationStatusActive {
			return active, nil
		}
		assert.Equal(t, drivers.ActivationStatusDeactivated, params.DriverActivationStatus)
		return []drivers.Driver{}, nil
	}
}

func setupTestService(t *testing.T) (*Service, *workerportsmock.MockWorkerRepository) {
	t.Helper()
	repo := workerportsmock.NewMockWorkerRepository(t)
	return New(Params{
		Logger: zap.NewNop(),
		Repo:   repo,
		IntegrationService: integrationservice.New(integrationservice.Params{
			Logger: zap.NewNop(),
			Repo:   &fakeIntegrationRepo{},
		}),
	}), repo
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

		driverID := "1654973"
		externalIDs := map[string]any{samsaraWorkerExternalIDKey: w.ID.String()}
		svc.integrationService = samsaraIntegration(t, &fakeSamsaraDriverService{
			listAllFunc: activeDrivers(t, []drivers.Driver{
				{Id: &driverID, ExternalIds: &externalIDs},
			}),
		})

		total := 1
		repo.On("List", mock.Anything, mock.Anything).
			Return(&pagination.CursorListResult[*worker.Worker]{
				Items:      []*worker.Worker{w},
				TotalCount: &total,
			}, nil).
			Once()
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
		w.ExternalID = "1654977"

		svc.integrationService = samsaraIntegration(t, &fakeSamsaraDriverService{
			listAllFunc: activeDrivers(t, []drivers.Driver{{Id: &w.ExternalID}}),
			updateFunc: func(id string, req drivers.UpdateRequest) (drivers.Driver, error) {
				assert.Equal(t, w.ExternalID, id)
				if assert.NotNil(t, req.ExternalIds) {
					assert.Equal(
						t,
						w.ID.String(),
						(*req.ExternalIds)[samsaraWorkerExternalIDKey],
					)
				}
				return drivers.Driver{Id: &id}, nil
			},
		})

		total := 1
		repo.On("List", mock.Anything, mock.Anything).
			Return(&pagination.CursorListResult[*worker.Worker]{
				Items:      []*worker.Worker{w},
				TotalCount: &total,
			}, nil).
			Once()

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

		createdDriverID := "1655001"
		var capturedCreateReq drivers.CreateRequest
		svc.integrationService = samsaraIntegration(t, &fakeSamsaraDriverService{
			listAllFunc: activeDrivers(t, []drivers.Driver{}),
			createFunc: func(req drivers.CreateRequest) (drivers.Driver, error) {
				capturedCreateReq = req
				return drivers.Driver{Id: &createdDriverID}, nil
			},
		})

		total := 1
		repo.On("List", mock.Anything, mock.Anything).
			Return(&pagination.CursorListResult[*worker.Worker]{
				Items:      []*worker.Worker{w},
				TotalCount: &total,
			}, nil).
			Once()
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
		assert.Equal(
			t,
			map[string]string{samsaraWorkerExternalIDKey: w.ID.String()},
			*capturedCreateReq.ExternalIds,
		)
		require.NotNil(t, capturedCreateReq.Phone)
		assert.Equal(t, "+15551234567", *capturedCreateReq.Phone)
	})
}

func TestSyncWorkersToSamsaraMapsDeactivatedDriverInsteadOfCreating(t *testing.T) {
	t.Parallel()

	svc, repo := setupTestService(t)
	w := newTestWorker()
	w.ExternalID = ""

	driverID := "1654999"
	externalIDs := map[string]any{samsaraWorkerExternalIDKey: w.ID.String()}
	statuses := make([]string, 0, 2)
	svc.integrationService = samsaraIntegration(t, &fakeSamsaraDriverService{
		listAllFunc: func(params drivers.ListParams) ([]drivers.Driver, error) {
			assert.NoError(t, params.Validate())
			statuses = append(statuses, params.DriverActivationStatus)
			if params.DriverActivationStatus == drivers.ActivationStatusDeactivated {
				return []drivers.Driver{{Id: &driverID, ExternalIds: &externalIDs}}, nil
			}
			return []drivers.Driver{}, nil
		},
		createFunc: func(drivers.CreateRequest) (drivers.Driver, error) {
			t.Error("a driver already carrying the worker external id must not be recreated")
			return drivers.Driver{}, nil
		},
	})

	total := 1
	repo.On("List", mock.Anything, mock.Anything).
		Return(&pagination.CursorListResult[*worker.Worker]{
			Items:      []*worker.Worker{w},
			TotalCount: &total,
		}, nil).
		Once()
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
	assert.ElementsMatch(
		t,
		[]string{drivers.ActivationStatusActive, drivers.ActivationStatusDeactivated},
		statuses,
	)
	assert.Equal(t, 1, result.RemoteDrivers)
	assert.Equal(t, 1, result.MappedFromExternalIDs)
	assert.Equal(t, 0, result.CreatedDrivers)
	assert.Equal(t, driverID, w.ExternalID)
}

func TestBuildSamsaraDriverCreateRequestSatisfiesSpec(t *testing.T) {
	t.Parallel()

	w := newTestWorker()
	w.Email = "John Doe+ops@trenova.app"
	req := buildSamsaraDriverCreateRequest(w)
	require.NoError(t, drivers.ValidateCreateRequest(req))
	assert.NotContains(t, req.Username, "@")
	assert.NotContains(t, req.Username, " ")

	w.Email = ""
	w.FirstName = ""
	w.LastName = ""
	req = buildSamsaraDriverCreateRequest(w)
	require.NoError(t, drivers.ValidateCreateRequest(req))
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
			&workerports.WorkerSyncReadinessCounts{
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
			&workerports.WorkerSyncReadinessCounts{
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
			(*workerports.WorkerSyncReadinessCounts)(nil),
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

	repo := workerportsmock.NewMockWorkerRepository(t)
	svc := New(Params{Logger: zap.NewNop(), Repo: repo})
	require.NotNil(t, svc)
}

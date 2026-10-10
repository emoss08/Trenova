package telematicsservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/telematics/samsaraprovider"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const driverBatchTemplateID = "8c0a7f4e-6d0b-4f7a-9d63-4f0d7a1b2c3d"

type driverBatchRepo struct {
	repositories.TelematicsRepository
	mappings    []repositories.WorkerTelematicsMapping
	syncs       []*repositories.SyncWorkerHOSLogsRequest
	submissions []*telematics.FormSubmission
}

func (r *driverBatchRepo) ListWorkerMappings(
	context.Context,
	pagination.TenantInfo,
) ([]repositories.WorkerTelematicsMapping, error) {
	return r.mappings, nil
}

func (r *driverBatchRepo) SyncWorkerHOSLogs(
	_ context.Context,
	req *repositories.SyncWorkerHOSLogsRequest,
) (int, error) {
	r.syncs = append(r.syncs, req)
	return len(req.Logs), nil
}

func (r *driverBatchRepo) ListFormMappings(
	context.Context,
	*repositories.ListFormMappingsRequest,
) ([]*telematics.FormMapping, error) {
	return nil, nil
}

func (r *driverBatchRepo) UpsertFormSubmission(
	_ context.Context,
	submission *telematics.FormSubmission,
) (bool, error) {
	r.submissions = append(r.submissions, submission)
	return true, nil
}

type driverBatchProvider struct {
	services.TelematicsProvider
	hosLogCalls  [][]string
	formCalls    [][]string
	hosLogs      map[string][]services.ProviderHOSLogEntry
	submissions  []services.ProviderFormSubmission
	failedChunk  []string
	totalFailure error
}

func (p *driverBatchProvider) Type() integration.Type {
	return integration.TypeSamsara
}

func (p *driverBatchProvider) partialFailure() error {
	if len(p.failedChunk) > 0 {
		return &services.ProviderDriverBatchError{
			DriverIDs: p.failedChunk,
			Err:       errors.New("upstream rejected the chunk"),
		}
	}
	return nil
}

func (p *driverBatchProvider) ListHOSLogs(
	_ context.Context,
	driverIDs []string,
	_ int64,
	_ int64,
) (map[string][]services.ProviderHOSLogEntry, error) {
	p.hosLogCalls = append(p.hosLogCalls, driverIDs)
	if p.totalFailure != nil {
		return nil, p.totalFailure
	}
	return p.hosLogs, p.partialFailure()
}

func (p *driverBatchProvider) ListFormSubmissions(
	_ context.Context,
	driverIDs []string,
	_ int64,
	_ int64,
) ([]services.ProviderFormSubmission, error) {
	p.formCalls = append(p.formCalls, driverIDs)
	if p.totalFailure != nil {
		return nil, p.totalFailure
	}
	return p.submissions, p.partialFailure()
}

type driverBatchFixture struct {
	tenant          pagination.TenantInfo
	externalIDs     []string
	workerByDriver  map[string]pulid.ID
	driverByWorker  map[pulid.ID]string
	repo            *driverBatchRepo
	workerMappings  []repositories.WorkerTelematicsMapping
	expectedWorkers []pulid.ID
}

func newDriverBatchFixture(n int) *driverBatchFixture {
	fixture := &driverBatchFixture{
		tenant: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		externalIDs:     make([]string, 0, n),
		workerByDriver:  make(map[string]pulid.ID, n),
		driverByWorker:  make(map[pulid.ID]string, n),
		workerMappings:  make([]repositories.WorkerTelematicsMapping, 0, n),
		expectedWorkers: make([]pulid.ID, 0, n),
	}
	for i := range n {
		externalID := fmt.Sprintf("drv-%04d", i)
		workerID := pulid.MustNew("wrk_")
		fixture.externalIDs = append(fixture.externalIDs, externalID)
		fixture.workerByDriver[externalID] = workerID
		fixture.driverByWorker[workerID] = externalID
		fixture.expectedWorkers = append(fixture.expectedWorkers, workerID)
		fixture.workerMappings = append(
			fixture.workerMappings,
			repositories.WorkerTelematicsMapping{
				WorkerID:   workerID,
				ExternalID: externalID,
				FirstName:  "Driver",
				LastName:   externalID,
			},
		)
	}
	fixture.repo = &driverBatchRepo{mappings: fixture.workerMappings}
	return fixture
}

func (f *driverBatchFixture) service(provider services.TelematicsProvider) *Service {
	return &Service{
		repo:             f.repo,
		providerOverride: provider,
		l:                zap.NewNop(),
	}
}

type samsaraRequestCounter struct {
	mu        sync.Mutex
	byPath    map[string]int
	maxBatch  map[string]int
	driverIDs map[string][]string
}

func newSamsaraRequestCounter() *samsaraRequestCounter {
	return &samsaraRequestCounter{
		byPath:    make(map[string]int),
		maxBatch:  make(map[string]int),
		driverIDs: make(map[string][]string),
	}
}

func (c *samsaraRequestCounter) record(r *http.Request) []string {
	var driverIDs []string
	if raw := r.URL.Query().Get("driverIds"); raw != "" {
		driverIDs = strings.Split(raw, ",")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byPath[r.URL.Path]++
	c.maxBatch[r.URL.Path] = max(c.maxBatch[r.URL.Path], len(driverIDs))
	c.driverIDs[r.URL.Path] = append(c.driverIDs[r.URL.Path], driverIDs...)
	return driverIDs
}

func (c *samsaraRequestCounter) requests(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.byPath[path]
}

func newSamsaraTestProvider(
	t *testing.T,
	handler func(w http.ResponseWriter, r *http.Request, driverIDs []string) any,
) (*samsaraprovider.Provider, *samsaraRequestCounter) {
	t.Helper()

	counter := newSamsaraRequestCounter()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		driverIDs := counter.record(r)
		body, err := sonic.Marshal(handler(w, r, driverIDs))
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	client, err := sharedsamsara.New(
		"test-token",
		sharedsamsara.WithBaseURL(server.URL),
		sharedsamsara.WithRateLimiting(false),
		sharedsamsara.WithRetry(sharedsamsara.RetryConfig{
			Enabled:        false,
			MaxAttempts:    1,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     time.Millisecond,
		}),
	)
	require.NoError(t, err)
	return samsaraprovider.New(client), counter
}

func lastPage() map[string]any {
	return map[string]any{"endCursor": "", "hasNextPage": false}
}

func TestSyncHOSLogsSendsOneSamsaraRequestPerChunkNotPerDriver(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(1500)
	provider, counter := newSamsaraTestProvider(
		t,
		func(_ http.ResponseWriter, _ *http.Request, driverIDs []string) any {
			data := make([]map[string]any, 0, len(driverIDs))
			for _, driverID := range driverIDs {
				data = append(data, map[string]any{
					"driver": map[string]any{"id": driverID},
					"hosLogs": []map[string]any{{
						"hosStatusType": "driving",
						"logStartTime":  "2026-03-01T14:00:00Z",
						"remark":        driverID,
					}},
				})
			}
			return map[string]any{"data": data, "pagination": lastPage()}
		},
	)
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	require.NoError(t, svc.syncHOSLogs(t.Context(), fixture.tenant, provider, drivers, result))

	assert.Equal(t, 15, counter.requests("/fleet/hos/logs"))
	assert.Equal(t, 100, counter.maxBatch["/fleet/hos/logs"])
	assert.ElementsMatch(t, fixture.externalIDs, counter.driverIDs["/fleet/hos/logs"])

	require.Len(t, fixture.repo.syncs, 1)
	synced := fixture.repo.syncs[0]
	assert.ElementsMatch(t, fixture.expectedWorkers, synced.WorkerIDs)
	require.Len(t, synced.Logs, 1500)
	for _, entry := range synced.Logs {
		assert.Equal(t, fixture.driverByWorker[entry.WorkerID], entry.Remark)
		assert.Equal(t, telematics.DutyStatusDriving, entry.DutyStatus)
		assert.Equal(t, string(integration.TypeSamsara), entry.Provider)
	}
	assert.Equal(t, 1500, result.HOSLogsUpserted)
}

func TestSyncHOSLogsSyncsOnlyDriversOutsideFailedChunks(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(6)
	failed := fixture.externalIDs[:2]
	provider := &driverBatchProvider{
		failedChunk: failed,
		hosLogs: map[string][]services.ProviderHOSLogEntry{
			fixture.externalIDs[2]: {{HosStatusType: "driving", LogStartAt: 100}},
			fixture.externalIDs[3]: {
				{HosStatusType: "mystery", LogStartAt: 200},
				{HosStatusType: "driving", LogStartAt: 0},
			},
		},
	}
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	require.NoError(t, svc.syncHOSLogs(t.Context(), fixture.tenant, provider, drivers, result))

	require.Len(t, provider.hosLogCalls, 1)
	assert.Equal(t, fixture.externalIDs, provider.hosLogCalls[0])

	require.Len(t, fixture.repo.syncs, 1)
	synced := fixture.repo.syncs[0]
	assert.ElementsMatch(t, fixture.expectedWorkers[2:], synced.WorkerIDs)
	require.Len(t, synced.Logs, 2)
	assert.Equal(t, fixture.workerByDriver[fixture.externalIDs[2]], synced.Logs[0].WorkerID)
	assert.Equal(t, fixture.workerByDriver[fixture.externalIDs[3]], synced.Logs[1].WorkerID)
	assert.Equal(t, telematics.DutyStatusOnDuty, synced.Logs[1].DutyStatus)
	assert.Equal(t, 2, result.HOSLogsUpserted)
}

func TestSyncHOSLogsLeavesStoredLogsAloneWhenProviderFails(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(3)
	provider := &driverBatchProvider{totalFailure: errors.New("samsara unavailable")}
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	require.NoError(t, svc.syncHOSLogs(t.Context(), fixture.tenant, provider, drivers, result))

	assert.Len(t, provider.hosLogCalls, 1)
	assert.Empty(t, fixture.repo.syncs)
	assert.Zero(t, result.HOSLogsUpserted)
}

func TestSyncHOSLogsWithoutMappedDriversCallsNothing(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(0)
	provider := &driverBatchProvider{}
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	require.NoError(t, svc.syncHOSLogs(t.Context(), fixture.tenant, provider, drivers, nil))
	require.NoError(t, svc.syncForms(t.Context(), fixture.tenant, provider, drivers, nil))

	assert.Empty(t, provider.hosLogCalls)
	assert.Empty(t, provider.formCalls)
	assert.Empty(t, fixture.repo.syncs)
}

func TestSyncFormsSendsOneSamsaraRequestPerChunkNotPerDriver(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(120)
	provider, counter := newSamsaraTestProvider(
		t,
		func(_ http.ResponseWriter, r *http.Request, driverIDs []string) any {
			if r.URL.Path == "/form-templates" {
				return map[string]any{
					"data": []map[string]any{
						{"id": driverBatchTemplateID, "title": "Proof of delivery"},
					},
					"pagination": lastPage(),
				}
			}
			data := make([]map[string]any, 0, len(driverIDs))
			for _, driverID := range driverIDs {
				data = append(data, map[string]any{
					"id": "sub-" + driverID,
					"formTemplate": map[string]any{
						"id":         driverBatchTemplateID,
						"revisionId": driverBatchTemplateID,
					},
					"submittedBy":     map[string]any{"id": driverID, "type": "driver"},
					"submittedAtTime": "2026-03-01T14:00:00Z",
					"createdAtTime":   "2026-03-01T14:00:00Z",
					"updatedAtTime":   "2026-03-01T14:00:00Z",
					"status":          "completed",
					"fields":          []map[string]any{},
				})
			}
			return map[string]any{"data": data, "pagination": lastPage()}
		},
	)
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	require.NoError(t, svc.syncForms(t.Context(), fixture.tenant, provider, drivers, result))

	assert.Equal(t, 3, counter.requests("/form-submissions/stream"))
	assert.Equal(t, 50, counter.maxBatch["/form-submissions/stream"])
	assert.Equal(t, 1, counter.requests("/form-templates"))

	require.Len(t, fixture.repo.submissions, 120)
	for _, submission := range fixture.repo.submissions {
		driverID := fixture.driverByWorker[submission.WorkerID]
		require.NotEmpty(t, driverID)
		assert.Equal(t, "sub-"+driverID, submission.ProviderSubmissionID)
		assert.Equal(t, "Proof of delivery", submission.TemplateName)
	}
	assert.Equal(t, 120, result.FormsUpserted)
}

func TestSyncFormsIngestsFetchedChunksAndReportsFailedChunk(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(4)
	provider := &driverBatchProvider{
		failedChunk: fixture.externalIDs[2:],
		submissions: []services.ProviderFormSubmission{
			{ID: "sub-a", TemplateID: "tpl", DriverID: fixture.externalIDs[0]},
			{ID: "sub-b", TemplateID: "tpl", DriverID: fixture.externalIDs[1]},
		},
	}
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	err = svc.syncForms(t.Context(), fixture.tenant, provider, drivers, result)
	require.Error(t, err)
	batchErr, ok := errors.AsType[*services.ProviderDriverBatchError](err)
	require.True(t, ok)
	assert.Equal(t, fixture.externalIDs[2:], batchErr.DriverIDs)

	require.Len(t, provider.formCalls, 1)
	require.Len(t, fixture.repo.submissions, 2)
	assert.Equal(t, fixture.expectedWorkers[0], fixture.repo.submissions[0].WorkerID)
	assert.Equal(t, fixture.expectedWorkers[1], fixture.repo.submissions[1].WorkerID)
	assert.Equal(t, 2, result.FormsUpserted)
}

func TestSyncFormsReturnsTotalProviderFailure(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(2)
	failure := errors.New("samsara unavailable")
	provider := &driverBatchProvider{totalFailure: failure}
	svc := fixture.service(provider)

	drivers, err := svc.loadSweepDrivers(t.Context(), fixture.tenant)
	require.NoError(t, err)
	result := new(TenantSweepResult)
	err = svc.syncForms(t.Context(), fixture.tenant, provider, drivers, result)
	require.ErrorIs(t, err, failure)
	assert.Empty(t, fixture.repo.submissions)
}

func TestGetHOSCertificationSummaryBatchesDailyLogRequests(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(150)
	shared := fixture.externalIDs[5]
	sharedWorker := pulid.MustNew("wrk_")
	fixture.repo.mappings = append(fixture.repo.mappings, repositories.WorkerTelematicsMapping{
		WorkerID:   sharedWorker,
		ExternalID: shared,
		FirstName:  "Second",
		LastName:   "Mapping",
	})

	provider, counter := newSamsaraTestProvider(
		t,
		func(_ http.ResponseWriter, r *http.Request, driverIDs []string) any {
			assert.Equal(t, "2026-03-01", r.URL.Query().Get("startDate"))
			assert.Equal(t, "2026-03-07", r.URL.Query().Get("endDate"))
			data := make([]map[string]any, 0, len(driverIDs)*2)
			for i, driverID := range driverIDs {
				data = append(data,
					map[string]any{
						"driver":      map[string]any{"id": driverID, "name": driverID},
						"startTime":   "2026-03-01T00:00:00Z",
						"endTime":     "2026-03-02T00:00:00Z",
						"logMetaData": map[string]any{"isCertified": true},
					},
					map[string]any{
						"driver":      map[string]any{"id": driverID, "name": driverID},
						"startTime":   "2026-03-02T00:00:00Z",
						"endTime":     "2026-03-03T00:00:00Z",
						"logMetaData": map[string]any{"isCertified": i%2 == 0},
					},
				)
			}
			return map[string]any{"data": data, "pagination": lastPage()}
		},
	)
	svc := fixture.service(provider)

	summaries, err := svc.GetHOSCertificationSummary(
		t.Context(),
		fixture.tenant,
		"2026-03-01",
		"2026-03-07",
	)
	require.NoError(t, err)

	assert.Equal(t, 2, counter.requests("/fleet/hos/daily-logs"))
	assert.Equal(t, 100, counter.maxBatch["/fleet/hos/daily-logs"])
	assert.ElementsMatch(t, fixture.externalIDs, counter.driverIDs["/fleet/hos/daily-logs"])

	byWorker := make(map[pulid.ID]*HOSCertificationSummary, len(summaries))
	for _, summary := range summaries {
		byWorker[summary.WorkerID] = summary
	}
	assert.Len(t, summaries, 76)
	for i, externalID := range fixture.externalIDs {
		summary, ok := byWorker[fixture.workerByDriver[externalID]]
		if i%100%2 == 0 {
			assert.False(t, ok, externalID)
			continue
		}
		require.True(t, ok, externalID)
		assert.Equal(t, 1, summary.UncertifiedDays)
		assert.Equal(t, 2, summary.TotalDays)
		assert.Equal(t, "Driver "+externalID, summary.WorkerName)
	}

	second, ok := byWorker[sharedWorker]
	require.True(t, ok)
	assert.Equal(t, "Second Mapping", second.WorkerName)
	assert.Equal(t, 1, second.UncertifiedDays)
}

func TestGetWorkerHOSLogsRequestsOnlyThatDriver(t *testing.T) {
	t.Parallel()

	fixture := newDriverBatchFixture(3)
	target := fixture.externalIDs[1]
	provider := &driverBatchProvider{
		hosLogs: map[string][]services.ProviderHOSLogEntry{
			target: {{HosStatusType: "driving", LogStartAt: 100}},
		},
	}
	svc := fixture.service(provider)

	entries, err := svc.GetWorkerHOSLogs(
		t.Context(),
		fixture.tenant,
		fixture.workerByDriver[target],
		1,
		2,
	)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, int64(100), entries[0].LogStartAt)
	assert.Equal(t, [][]string{{target}}, provider.hosLogCalls)
}

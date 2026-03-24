package auditjobs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type mockAuditRepository struct {
	mock.Mock
}

func (m *mockAuditRepository) InsertAuditEntries(
	ctx context.Context,
	entries []*audit.Entry,
) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *mockAuditRepository) List(
	ctx context.Context,
	req *repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*audit.Entry]), args.Error(1)
}

func (m *mockAuditRepository) ListByResourceID(
	ctx context.Context,
	req *repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*audit.Entry]), args.Error(1)
}

func (m *mockAuditRepository) GetByID(
	ctx context.Context,
	req repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.Entry), args.Error(1)
}

func (m *mockAuditRepository) GetByResourceAndOperation(
	ctx context.Context,
	req *repositories.GetAuditByResourceRequest,
) ([]*audit.Entry, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Entry), args.Error(1)
}

func (m *mockAuditRepository) GetRecentEntries(
	ctx context.Context,
	req *repositories.GetRecentEntriesRequest,
) ([]*audit.Entry, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Entry), args.Error(1)
}

func (m *mockAuditRepository) DeleteAuditEntries(
	ctx context.Context,
	timestamp int64,
) (int64, error) {
	args := m.Called(ctx, timestamp)
	return args.Get(0).(int64), args.Error(1)
}

type mockDataRetentionRepository struct {
	mock.Mock
}

func (m *mockDataRetentionRepository) List(
	ctx context.Context,
) (*pagination.ListResult[*tenant.DataRetention], error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*tenant.DataRetention]), args.Error(1)
}

func (m *mockDataRetentionRepository) Get(
	ctx context.Context,
	req repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.DataRetention), args.Error(1)
}

func (m *mockDataRetentionRepository) Update(
	ctx context.Context,
	entity *tenant.DataRetention,
) (*tenant.DataRetention, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tenant.DataRetention), args.Error(1)
}

type mockAuditBufferRepository struct {
	mock.Mock
}

func (m *mockAuditBufferRepository) Push(ctx context.Context, entry *audit.Entry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockAuditBufferRepository) PushBatch(
	ctx context.Context,
	entries []*audit.Entry,
) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *mockAuditBufferRepository) Pop(
	ctx context.Context,
	count int,
) ([]*audit.Entry, error) {
	args := m.Called(ctx, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Entry), args.Error(1)
}

func (m *mockAuditBufferRepository) Size(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type mockAuditDLQRepository struct {
	mock.Mock
}

func (m *mockAuditDLQRepository) Insert(ctx context.Context, entry *audit.DLQEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockAuditDLQRepository) InsertBatch(
	ctx context.Context,
	entries []*audit.DLQEntry,
) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *mockAuditDLQRepository) GetPendingEntries(
	ctx context.Context,
	limit int,
) ([]*audit.DLQEntry, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.DLQEntry), args.Error(1)
}

func (m *mockAuditDLQRepository) MarkAsRecovered(
	ctx context.Context,
	ids []pulid.ID,
) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *mockAuditDLQRepository) MarkAsFailed(
	ctx context.Context,
	id pulid.ID,
	errMsg string,
) error {
	args := m.Called(ctx, id, errMsg)
	return args.Error(0)
}

func (m *mockAuditDLQRepository) Update(ctx context.Context, entry *audit.DLQEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockAuditDLQRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockAuditDLQRepository) CountByStatus(
	ctx context.Context,
	status audit.DLQStatus,
) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockAuditDLQRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*audit.DLQEntry, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.DLQEntry), args.Error(1)
}

func (m *mockAuditDLQRepository) DeleteRecovered(
	ctx context.Context,
	olderThan int64,
) (int64, error) {
	args := m.Called(ctx, olderThan)
	return args.Get(0).(int64), args.Error(1)
}

func TestProcessAuditBatchActivity_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDataRetentionRepo := new(mockDataRetentionRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		abr:  mockBufferRepo,
		adlq: mockDLQRepo,
		dr:   mockDataRetentionRepo,
	}

	batchID := pulid.MustNew("aeb_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	entries := []*audit.Entry{
		{ID: pulid.MustNew("ael_")},
		{ID: pulid.MustNew("ael_")},
	}

	payload := &ProcessAuditBatchPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Timestamp:      time.Now().Unix(),
		},
		Entries: entries,
		BatchID: batchID,
	}

	mockAuditRepo.On("InsertAuditEntries", mock.Anything, entries).Return(nil)

	env.RegisterActivity(activities.ProcessAuditBatchActivity)
	result, err := env.ExecuteActivity(activities.ProcessAuditBatchActivity, payload)

	require.NoError(t, err)

	var response *ProcessAuditBatchResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 2, response.ProcessedCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Equal(t, batchID, response.BatchID)

	mockAuditRepo.AssertExpectations(t)
}

func TestProcessAuditBatchActivity_EmptyEntries(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDataRetentionRepo := new(mockDataRetentionRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		abr:  mockBufferRepo,
		adlq: mockDLQRepo,
		dr:   mockDataRetentionRepo,
	}

	batchID := pulid.MustNew("aeb_")
	payload := &ProcessAuditBatchPayload{
		Entries: []*audit.Entry{},
		BatchID: batchID,
	}

	env.RegisterActivity(activities.ProcessAuditBatchActivity)
	result, err := env.ExecuteActivity(activities.ProcessAuditBatchActivity, payload)

	require.NoError(t, err)

	var response *ProcessAuditBatchResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 0, response.ProcessedCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Equal(t, "No entries to process", response.Metadata["message"])
}

func TestFlushFromRedisActivity_WithEntries(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockBufferRepo := new(mockAuditBufferRepository)

	activities := &Activities{
		abr: mockBufferRepo,
	}

	entries := []*audit.Entry{
		{ID: pulid.MustNew("ael_")},
		{ID: pulid.MustNew("ael_")},
	}

	mockBufferRepo.On("Pop", mock.Anything, defaultBatchSize).Return(entries, nil).Once()
	mockBufferRepo.On("Pop", mock.Anything, defaultBatchSize).Return([]*audit.Entry{}, nil).Once()

	env.RegisterActivity(activities.FlushFromRedisActivity)
	result, err := env.ExecuteActivity(activities.FlushFromRedisActivity)

	require.NoError(t, err)

	var response *FlushFromRedisResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 2, response.EntryCount)
	assert.Len(t, response.Batches, 1)

	mockBufferRepo.AssertExpectations(t)
}

func TestFlushFromRedisActivity_Empty(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockBufferRepo := new(mockAuditBufferRepository)

	activities := &Activities{
		abr: mockBufferRepo,
	}

	mockBufferRepo.On("Pop", mock.Anything, defaultBatchSize).Return([]*audit.Entry{}, nil)

	env.RegisterActivity(activities.FlushFromRedisActivity)
	result, err := env.ExecuteActivity(activities.FlushFromRedisActivity)

	require.NoError(t, err)

	var response *FlushFromRedisResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 0, response.EntryCount)
	assert.Empty(t, response.Batches)

	mockBufferRepo.AssertExpectations(t)
}

func TestDeleteAuditEntriesActivity_NoRetentionConfig(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDataRetentionRepo := new(mockDataRetentionRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		abr:  mockBufferRepo,
		adlq: mockDLQRepo,
		dr:   mockDataRetentionRepo,
	}

	mockDataRetentionRepo.On("List", mock.Anything).Return(
		&pagination.ListResult[*tenant.DataRetention]{
			Items: []*tenant.DataRetention{},
			Total: 0,
		}, nil)

	env.RegisterActivity(activities.DeleteAuditEntriesActivity)
	result, err := env.ExecuteActivity(activities.DeleteAuditEntriesActivity)

	require.NoError(t, err)

	var response *DeleteAuditEntriesResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 0, response.TotalDeleted)
	assert.Contains(t, response.Result, "No data retention entities configured")

	mockDataRetentionRepo.AssertExpectations(t)
}

func TestDeleteAuditEntriesActivity_WithRetention(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDataRetentionRepo := new(mockDataRetentionRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		abr:  mockBufferRepo,
		adlq: mockDLQRepo,
		dr:   mockDataRetentionRepo,
	}

	orgID := pulid.MustNew("org_")

	mockDataRetentionRepo.On("List", mock.Anything).Return(
		&pagination.ListResult[*tenant.DataRetention]{
			Items: []*tenant.DataRetention{
				{
					OrganizationID:       orgID,
					AuditRetentionPeriod: 30,
				},
			},
			Total: 1,
		}, nil)

	expectedTimestamp := time.Now().AddDate(0, 0, -30).Unix()
	mockAuditRepo.On("DeleteAuditEntries", mock.Anything, mock.MatchedBy(func(ts int64) bool {
		return ts <= expectedTimestamp+60 && ts >= expectedTimestamp-60
	})).Return(int64(100), nil)

	env.RegisterActivity(activities.DeleteAuditEntriesActivity)
	result, err := env.ExecuteActivity(activities.DeleteAuditEntriesActivity)

	require.NoError(t, err)

	var response *DeleteAuditEntriesResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 100, response.TotalDeleted)

	mockDataRetentionRepo.AssertExpectations(t)
	mockAuditRepo.AssertExpectations(t)
}

func TestGetBufferStatusActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		abr:  mockBufferRepo,
		adlq: mockDLQRepo,
	}

	mockBufferRepo.On("Size", mock.Anything).Return(int64(3), nil)
	mockDLQRepo.On("Count", mock.Anything).Return(int64(1), nil)

	env.RegisterActivity(activities.GetBufferStatusActivity)
	result, err := env.ExecuteActivity(activities.GetBufferStatusActivity)

	require.NoError(t, err)

	var status *AuditBufferStatus
	require.NoError(t, result.Get(&status))

	assert.Equal(t, 3, status.BufferedEntries)
	assert.Equal(t, 1, status.DLQEntries)

	mockBufferRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_NoPendingEntries(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		adlq: mockDLQRepo,
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return([]*audit.DLQEntry{}, nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 0, response.RetryCount)
	assert.Equal(t, 0, response.SuccessCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Equal(t, 0, response.ExhaustedCount)

	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_GetPendingEntriesError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		adlq: mockDLQRepo,
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).
		Return(nil, fmt.Errorf("db connection failed"))

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	_, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.Error(t, err)

	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_SuccessfulRetry(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		adlq: mockDLQRepo,
	}

	entryID := pulid.MustNew("dlq_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	entryData := map[string]any{
		"id":             pulid.MustNew("ael_").String(),
		"organizationId": orgID.String(),
		"businessUnitId": buID.String(),
	}

	dlqEntries := []*audit.DLQEntry{
		{
			ID:             entryID,
			RetryCount:     1,
			EntryData:      entryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil)
	mockDLQRepo.On("MarkAsRecovered", mock.Anything, []pulid.ID{entryID}).Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.RetryCount)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Equal(t, 0, response.ExhaustedCount)
	assert.Len(t, response.RecoveredIDs, 1)

	mockAuditRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_ExhaustedMaxRetries(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		adlq: mockDLQRepo,
	}

	entryID := pulid.MustNew("dlq_")

	dlqEntries := []*audit.DLQEntry{
		{
			ID:         entryID,
			RetryCount: defaultDLQMaxRetry,
			EntryData:  map[string]any{"id": "test"},
			Status:     audit.DLQStatusRetrying,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockDLQRepo.On("MarkAsFailed", mock.Anything, entryID, "Max retries exhausted").Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.RetryCount)
	assert.Equal(t, 0, response.SuccessCount)
	assert.Equal(t, 0, response.FailedCount)
	assert.Equal(t, 1, response.ExhaustedCount)

	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_InvalidEntryData(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		adlq: mockDLQRepo,
	}

	entryID := pulid.MustNew("dlq_")

	dlqEntries := []*audit.DLQEntry{
		{
			ID:         entryID,
			RetryCount: 0,
			EntryData:  map[string]any{"id": func() {}},
			Status:     audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockDLQRepo.On("MarkAsFailed", mock.Anything, entryID, mock.MatchedBy(func(msg string) bool {
		return len(msg) > 0
	})).Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.RetryCount)
	assert.Equal(t, 0, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Len(t, response.FailedIDs, 1)

	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_InsertFails(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		adlq: mockDLQRepo,
	}

	entryID := pulid.MustNew("dlq_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	entryData := map[string]any{
		"id":             pulid.MustNew("ael_").String(),
		"organizationId": orgID.String(),
		"businessUnitId": buID.String(),
	}

	dlqEntries := []*audit.DLQEntry{
		{
			ID:             entryID,
			RetryCount:     1,
			EntryData:      entryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(fmt.Errorf("insert failed"))
	mockDLQRepo.On("Update", mock.Anything, mock.MatchedBy(func(entry *audit.DLQEntry) bool {
		return entry.RetryCount == 2 && entry.LastError == "insert failed"
	})).Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.RetryCount)
	assert.Equal(t, 0, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Len(t, response.FailedIDs, 1)

	mockAuditRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_MixedResults(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		adlq: mockDLQRepo,
	}

	successID := pulid.MustNew("dlq_")
	exhaustedID := pulid.MustNew("dlq_")
	failID := pulid.MustNew("dlq_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	validEntryData := map[string]any{
		"id":             pulid.MustNew("ael_").String(),
		"organizationId": orgID.String(),
		"businessUnitId": buID.String(),
	}

	dlqEntries := []*audit.DLQEntry{
		{
			ID:             successID,
			RetryCount:     1,
			EntryData:      validEntryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
		{
			ID:         exhaustedID,
			RetryCount: defaultDLQMaxRetry,
			EntryData:  map[string]any{"id": "test"},
			Status:     audit.DLQStatusRetrying,
		},
		{
			ID:             failID,
			RetryCount:     0,
			EntryData:      validEntryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil).Once()
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(fmt.Errorf("db error")).
		Once()
	mockDLQRepo.On("MarkAsFailed", mock.Anything, exhaustedID, "Max retries exhausted").Return(nil)
	mockDLQRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockDLQRepo.On("MarkAsRecovered", mock.Anything, []pulid.ID{successID}).Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 3, response.RetryCount)
	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)
	assert.Equal(t, 1, response.ExhaustedCount)

	mockAuditRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_MarkAsRecoveredError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		ar:   mockAuditRepo,
		adlq: mockDLQRepo,
	}

	entryID := pulid.MustNew("dlq_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	entryData := map[string]any{
		"id":             pulid.MustNew("ael_").String(),
		"organizationId": orgID.String(),
		"businessUnitId": buID.String(),
	}

	dlqEntries := []*audit.DLQEntry{
		{
			ID:             entryID,
			RetryCount:     1,
			EntryData:      entryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil)
	mockDLQRepo.On("MarkAsRecovered", mock.Anything, []pulid.ID{entryID}).
		Return(fmt.Errorf("mark recovered failed"))

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.SuccessCount)
	assert.Len(t, response.RecoveredIDs, 1)

	mockAuditRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestRetryDLQEntriesActivity_WithMetrics(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockAuditRepo := new(mockAuditRepository)
	mockDLQRepo := new(mockAuditDLQRepository)

	promRegistry := prometheus.NewRegistry()
	auditMetrics := metrics.NewAudit(promRegistry, zap.NewNop(), true)
	metricsRegistry := &metrics.Registry{
		Audit: auditMetrics,
	}

	activities := &Activities{
		ar:      mockAuditRepo,
		adlq:    mockDLQRepo,
		metrics: metricsRegistry,
	}

	successID := pulid.MustNew("dlq_")
	failID := pulid.MustNew("dlq_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	validEntryData := map[string]any{
		"id":             pulid.MustNew("ael_").String(),
		"organizationId": orgID.String(),
		"businessUnitId": buID.String(),
	}

	dlqEntries := []*audit.DLQEntry{
		{
			ID:             successID,
			RetryCount:     1,
			EntryData:      validEntryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
		{
			ID:             failID,
			RetryCount:     0,
			EntryData:      validEntryData,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Status:         audit.DLQStatusPending,
		},
	}

	mockDLQRepo.On("GetPendingEntries", mock.Anything, 10).Return(dlqEntries, nil)
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil).Once()
	mockAuditRepo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(fmt.Errorf("fail")).
		Once()
	mockDLQRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockDLQRepo.On("MarkAsRecovered", mock.Anything, []pulid.ID{successID}).Return(nil)

	env.RegisterActivity(activities.RetryDLQEntriesActivity)
	result, err := env.ExecuteActivity(activities.RetryDLQEntriesActivity, 10)

	require.NoError(t, err)

	var response *DLQRetryResult
	require.NoError(t, result.Get(&response))

	assert.Equal(t, 1, response.SuccessCount)
	assert.Equal(t, 1, response.FailedCount)

	mockAuditRepo.AssertExpectations(t)
	mockDLQRepo.AssertExpectations(t)
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	mockAuditRepo := new(mockAuditRepository)
	mockBufferRepo := new(mockAuditBufferRepository)
	mockDLQRepo := new(mockAuditDLQRepository)
	mockDataRetentionRepo := new(mockDataRetentionRepository)

	activities := NewActivities(ActivitiesParams{
		AuditRepository:         mockAuditRepo,
		AuditBufferRepository:   mockBufferRepo,
		AuditDLQRepository:      mockDLQRepo,
		DataRetentionRepository: mockDataRetentionRepo,
	})

	logger := zap.NewNop()

	registry := NewRegistry(RegistryParams{
		Activities: activities,
		Logger:     logger,
	})

	assert.NotNil(t, registry)
	assert.Equal(t, "audit-worker", registry.GetName())
	assert.Equal(t, string(temporaltype.AuditTaskQueue), registry.GetTaskQueue())
}

func TestMoveToDLQActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	mockDLQRepo := new(mockAuditDLQRepository)

	activities := &Activities{
		adlq: mockDLQRepo,
	}

	entries := []*audit.Entry{
		{
			ID:             pulid.MustNew("ael_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}

	payload := &MoveToDLQPayload{
		Entries:      entries,
		ErrorMessage: "test error",
	}

	mockDLQRepo.On("InsertBatch", mock.Anything, mock.Anything).Return(nil)

	env.RegisterActivity(activities.MoveToDLQActivity)
	_, err := env.ExecuteActivity(activities.MoveToDLQActivity, payload)

	require.NoError(t, err)

	mockDLQRepo.AssertExpectations(t)
}

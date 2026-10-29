package auditservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/auditmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	req *auditports.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*audit.Entry]), args.Error(1)
}

func (m *mockAuditRepository) ListConnection(
	ctx context.Context,
	req *auditports.ListAuditEntriesConnectionRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.CursorListResult[*audit.Entry]), args.Error(1)
}

func (m *mockAuditRepository) ListByResourceID(
	ctx context.Context,
	req *auditports.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*audit.Entry]), args.Error(1)
}

func (m *mockAuditRepository) GetByID(
	ctx context.Context,
	req auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*audit.Entry), args.Error(1)
}

func (m *mockAuditRepository) DeleteAuditEntries(
	ctx context.Context,
	req auditports.DeleteAuditEntriesRequest,
) (int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(int64), args.Error(1)
}

type mockBufferRepository struct {
	mock.Mock
}

func (m *mockBufferRepository) Push(ctx context.Context, entry *audit.Entry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *mockBufferRepository) PushBatch(ctx context.Context, entries []*audit.Entry) error {
	args := m.Called(ctx, entries)
	return args.Error(0)
}

func (m *mockBufferRepository) Pop(ctx context.Context, count int) ([]*audit.Entry, error) {
	args := m.Called(ctx, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*audit.Entry), args.Error(1)
}

func (m *mockBufferRepository) PopTenantBatches(
	ctx context.Context,
	batchSize int,
	totalLimit int,
) ([][]*audit.Entry, error) {
	args := m.Called(ctx, batchSize, totalLimit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]*audit.Entry), args.Error(1)
}

func (m *mockBufferRepository) Size(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

type noopRealtimeService struct{}

func (s *noopRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *realtimeports.PublishResourceInvalidationRequest,
) error {
	return nil
}

func newTestService(
	repo *mockAuditRepository,
	bufferRepo *mockBufferRepository,
	realtime realtimeports.RealtimeService,
) *service {
	logger := zap.NewNop()
	cfg := &config.Config{
		App: appconfig.Config{Env: "test"},
		Security: securityconfig.Config{
			Encryption: securityconfig.EncryptionConfig{Key: "test-key-for-unit-tests-only"},
		},
	}
	auditMetrics := auditmetrics.NewAudit(nil, logger, false)
	metricsRegistry := &metrics.Registry{Audit: auditMetrics}

	srv := &service{
		repo:       repo,
		bufferRepo: bufferRepo,
		realtime:   realtime,
		logger:     logger.Named("service.audit"),
		config:     cfg,
		sdm:        NewSensitiveDataManager(cfg.Security.Encryption),
		metrics:    metricsRegistry,
	}
	srv.configureSensitiveDataManager("test")
	return srv
}

func validLogActionParams() *auditports.LogActionParams {
	return &auditports.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     "usr_01ABC",
		Operation:      permission.OpCreate,
		CurrentState:   map[string]any{"name": "test"},
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Critical:       false,
	}
}

func TestLogAction_NonCritical_BufferSuccess(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	err := srv.LogAction(validLogActionParams())

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
	repo.AssertNotCalled(t, "InsertAuditEntries", mock.Anything, mock.Anything)
}

func TestLogAction_Critical_DirectInsert(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil)

	params := validLogActionParams()
	params.Critical = true

	err := srv.LogAction(params)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	bufferRepo.AssertNotCalled(t, "Push", mock.Anything, mock.Anything)
}

func TestLogAction_Critical_InsertError_Buffered(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db error"))
	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1 && entries[0].Critical
	})).Return(nil)

	params := validLogActionParams()
	params.Critical = true

	require.NoError(t, srv.LogAction(params))
	bufferRepo.AssertExpectations(t)
}

func TestLogAction_Critical_InsertAndBufferFail(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db error"))
	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	params := validLogActionParams()
	params.Critical = true

	err := srv.LogAction(params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record critical audit entry")
	assert.Contains(t, err.Error(), "db error")
	assert.Contains(t, err.Error(), "redis error")
}

func TestLogAction_NonCritical_BufferFails_FallbackSuccess(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("Push", mock.Anything, mock.Anything).
		Return(errors.New("redis down"))
	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil)

	err := srv.LogAction(validLogActionParams())

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestLogAction_NonCritical_BufferFails_FallbackFails(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("Push", mock.Anything, mock.Anything).
		Return(errors.New("redis down"))
	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db also down"))

	err := srv.LogAction(validLogActionParams())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert audit entry")
}

func TestLogAction_InvalidEntry(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	params := &auditports.LogActionParams{
		Resource:   "",
		ResourceID: "",
		Operation:  "",
	}

	err := srv.LogAction(params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid audit entry")
}

func TestLogAction_WithOptions(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("Push", mock.Anything, mock.MatchedBy(func(entry *audit.Entry) bool {
		return entry.Comment == "test comment" && entry.IPAddress == "10.0.0.1"
	})).Return(nil)

	err := srv.LogAction(validLogActionParams(), WithComment("test comment"), WithIP("10.0.0.1"))

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestLogAction_OptionReturnsError(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	badOpt := func(_ *audit.Entry) error {
		return errors.New("option error")
	}

	err := srv.LogAction(validLogActionParams(), badOpt)

	require.Error(t, err)
	assert.Equal(t, "option error", err.Error())
}

func TestLogActions_EmptyEntries(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	err := srv.LogActions(nil)

	require.NoError(t, err)
}

func TestLogActions_AllCritical(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 2
	})).Return(nil)

	params1 := validLogActionParams()
	params1.Critical = true
	params2 := validLogActionParams()
	params2.Critical = true

	entries := []auditports.BulkLogEntry{
		{Params: params1},
		{Params: params2},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	bufferRepo.AssertNotCalled(t, "PushBatch", mock.Anything, mock.Anything)
}

func TestLogActions_AllNonCritical_BufferSuccess(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 2
	})).Return(nil)

	entries := []auditports.BulkLogEntry{
		{Params: validLogActionParams()},
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestLogActions_MixedCriticalAndNonCritical(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1 && entries[0].Critical
	})).Return(nil)

	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1 && !entries[0].Critical
	})).Return(nil)

	criticalParams := validLogActionParams()
	criticalParams.Critical = true

	entries := []auditports.BulkLogEntry{
		{Params: criticalParams},
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	repo.AssertExpectations(t)
	bufferRepo.AssertExpectations(t)
}

func TestLogActions_NonCritical_BufferFails_FallbackSuccess(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).
		Return(errors.New("redis down"))
	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).Return(nil)

	entries := []auditports.BulkLogEntry{
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestLogActions_NonCritical_BufferFails_FallbackFails(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).
		Return(errors.New("redis down"))
	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db also down"))

	entries := []auditports.BulkLogEntry{
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
}

func TestLogActions_CriticalInsertError_Buffered(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db error"))
	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).Return(nil)

	params := validLogActionParams()
	params.Critical = true

	require.NoError(t, srv.LogActions([]auditports.BulkLogEntry{{Params: params}}))
	bufferRepo.AssertExpectations(t)
}

func TestLogActions_CriticalInsertAndBufferFail(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	repo.On("InsertAuditEntries", mock.Anything, mock.Anything).
		Return(errors.New("db error"))
	bufferRepo.On("PushBatch", mock.Anything, mock.Anything).
		Return(errors.New("redis error"))

	params := validLogActionParams()
	params.Critical = true

	require.Error(t, srv.LogActions([]auditports.BulkLogEntry{{Params: params}}))
}

func TestLogActions_InvalidEntry_Skipped(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1
	})).Return(nil)

	invalidParams := &auditports.LogActionParams{
		Resource:   "",
		ResourceID: "",
	}

	entries := []auditports.BulkLogEntry{
		{Params: invalidParams},
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestLogActions_AllInvalid(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	invalidParams := &auditports.LogActionParams{
		Resource:   "",
		ResourceID: "",
	}

	entries := []auditports.BulkLogEntry{
		{Params: invalidParams},
		{Params: invalidParams},
	}

	err := srv.LogActions(entries)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 audit entries failed validation/sanitization")
}

func TestLogActions_OptionError_Skipped(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1
	})).Return(nil)

	badOpt := func(_ *audit.Entry) error {
		return errors.New("option failed")
	}

	entries := []auditports.BulkLogEntry{
		{Params: validLogActionParams(), Options: []auditports.LogOption{badOpt}},
		{Params: validLogActionParams()},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestList_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	expected := &pagination.ListResult[*audit.Entry]{
		Items:      []*audit.Entry{{ID: pulid.MustNew("ae_")}},
		Total:      1,
		TotalExact: true,
	}

	req := &auditports.ListAuditEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
		},
	}

	repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := srv.List(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestList_Error(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	req := &auditports.ListAuditEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
		},
	}

	repo.On("List", mock.Anything, req).Return(nil, errors.New("db error"))

	result, err := srv.List(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list audit entries")
}

func TestList_NormalizesNumericOperationFilter(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	expected := &pagination.ListResult[*audit.Entry]{
		Items:      []*audit.Entry{{ID: pulid.MustNew("ae_")}},
		Total:      1,
		TotalExact: true,
	}

	req := &auditports.ListAuditEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
			FieldFilters: []domaintypes.FieldFilter{
				{
					Field:    "operation",
					Operator: dbtype.OpEqual,
					Value:    int64(permission.ClientOpRead),
				},
			},
		},
	}

	repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := srv.List(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "read", req.Filter.FieldFilters[0].Value)
	repo.AssertExpectations(t)
}

func TestList_NormalizesNumericOperationFilterGroupValues(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	expected := &pagination.ListResult[*audit.Entry]{
		Items:      []*audit.Entry{{ID: pulid.MustNew("ae_")}},
		Total:      1,
		TotalExact: true,
	}

	req := &auditports.ListAuditEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
			FilterGroups: []domaintypes.FilterGroup{
				{
					Filters: []domaintypes.FieldFilter{
						{
							Field:    "operation",
							Operator: dbtype.OpIn,
							Value: []int64{
								int64(permission.ClientOpRead),
								int64(permission.ClientOpUpdate),
							},
						},
					},
				},
			},
		},
	}

	repo.On("List", mock.Anything, req).Return(expected, nil)

	result, err := srv.List(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(
		t,
		[]string{"read", "update"},
		req.Filter.FilterGroups[0].Filters[0].Value,
	)
	repo.AssertExpectations(t)
}

func TestListByResourceID_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	resourceID := pulid.MustNew("res_")
	expected := &pagination.ListResult[*audit.Entry]{
		Items:      []*audit.Entry{{ID: pulid.MustNew("ae_")}},
		Total:      1,
		TotalExact: true,
	}

	req := &auditports.ListByResourceIDRequest{
		ResourceID: resourceID,
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
		},
	}

	repo.On("ListByResourceID", mock.Anything, req).Return(expected, nil)

	result, err := srv.ListByResourceID(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestListByResourceID_Error(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	req := &auditports.ListByResourceIDRequest{
		ResourceID: pulid.MustNew("res_"),
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  pulid.MustNew("org_"),
				BuID:   pulid.MustNew("bu_"),
				UserID: pulid.MustNew("usr_"),
			},
		},
	}

	repo.On("ListByResourceID", mock.Anything, req).Return(nil, errors.New("db error"))

	result, err := srv.ListByResourceID(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list audit entries by resource id")
}

func TestGetByID_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	entryID := pulid.MustNew("ae_")
	expected := &audit.Entry{ID: entryID}

	req := auditports.GetAuditEntryByIDOptions{
		EntryID: entryID,
	}

	repo.On("GetByID", mock.Anything, req).Return(expected, nil)

	result, err := srv.GetByID(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetByID_Error(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	req := auditports.GetAuditEntryByIDOptions{
		EntryID: pulid.MustNew("ae_"),
	}

	repo.On("GetByID", mock.Anything, req).Return(nil, errors.New("not found"))

	result, err := srv.GetByID(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get audit entry by id")
}

func TestService_RegisterSensitiveFields(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	fields := []auditports.SensitiveField{
		{Name: "password", Action: auditports.SensitiveFieldOmit},
	}

	err := srv.RegisterSensitiveFields(permission.ResourceUser, fields)

	require.NoError(t, err)
}

func TestConfigureSensitiveDataManager_Production(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("production")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyStrict), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Staging(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("staging")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyDefault), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Development(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("development")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyPartial), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Test(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("test")

	assert.False(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyPartial), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Unknown(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("something-else")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyDefault), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Prod(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("prod")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyStrict), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Stage(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("stage")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyDefault), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Dev(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("dev")

	assert.True(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyPartial), srv.sdm.strategy.Load())
}

func TestConfigureSensitiveDataManager_Testing(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.configureSensitiveDataManager("testing")

	assert.False(t, srv.sdm.autoDetect.Load())
	assert.Equal(t, int32(MaskStrategyPartial), srv.sdm.strategy.Load())
}

func TestLogAction_WithSensitiveDataSanitization(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	srv.sdm.SetAutoDetect(true)

	bufferRepo.On("Push", mock.Anything, mock.Anything).Return(nil)

	params := validLogActionParams()
	params.CurrentState = map[string]any{
		"name":     "visible",
		"password": "secret123",
	}

	err := srv.LogAction(params)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestLogActions_WithOptions(t *testing.T) {
	t.Parallel()

	repo := new(mockAuditRepository)
	bufferRepo := new(mockBufferRepository)
	srv := newTestService(repo, bufferRepo, &noopRealtimeService{})

	bufferRepo.On("PushBatch", mock.Anything, mock.MatchedBy(func(entries []*audit.Entry) bool {
		return len(entries) == 1 && entries[0].Comment == "bulk comment"
	})).Return(nil)

	entries := []auditports.BulkLogEntry{
		{
			Params:  validLogActionParams(),
			Options: []auditports.LogOption{WithComment("bulk comment")},
		},
	}

	err := srv.LogActions(entries)

	require.NoError(t, err)
	bufferRepo.AssertExpectations(t)
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates service with all dependencies", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "my-test-encryption-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		require.NotNil(t, svc)
		concrete, ok := svc.(*service)
		require.True(t, ok)
		assert.Same(t, repo, concrete.repo)
		assert.Same(t, bufferRepo, concrete.bufferRepo)
		assert.NotNil(t, concrete.logger)
		assert.Same(t, cfg, concrete.config)
		assert.NotNil(t, concrete.sdm)
		assert.Same(t, metricsRegistry, concrete.metrics)
	})

	t.Run("configures sensitive data manager for test environment", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "test-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)
		assert.False(t, concrete.sdm.autoDetect.Load())
		assert.Equal(t, int32(MaskStrategyPartial), concrete.sdm.strategy.Load())
	})

	t.Run("configures sensitive data manager for production environment", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "production"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "prod-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)
		assert.True(t, concrete.sdm.autoDetect.Load())
		assert.Equal(t, int32(MaskStrategyStrict), concrete.sdm.strategy.Load())
	})

	t.Run("sets up encryption key in sensitive data manager", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "my-encryption-key-for-testing"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)
		assert.NotEmpty(t, concrete.sdm.encryptionKey)
		assert.Len(t, concrete.sdm.encryptionKey, 32)
	})

	t.Run("handles empty encryption key", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: ""},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)
		assert.Empty(t, concrete.sdm.encryptionKey)
	})

	t.Run("returns AuditService interface", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		var _ auditports.AuditService = svc
	})
}

func TestRegisterDefaultSensitiveFields(t *testing.T) {
	t.Parallel()

	t.Run("registers user sensitive fields", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "test-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)

		fields, ok := concrete.sdm.registeredFields(permission.ResourceUser)
		require.True(t, ok)

		passwordCfg, exists := fields["password"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldOmit, passwordCfg.Action)

		hashedPasswordCfg, exists := fields["hashedPassword"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldOmit, hashedPasswordCfg.Action)

		emailCfg, exists := fields["emailAddress"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, emailCfg.Action)

		addressCfg, exists := fields["address"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, addressCfg.Action)
	})

	t.Run("registers organization sensitive fields", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "test-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)

		fields, ok := concrete.sdm.registeredFields(permission.ResourceOrganization)
		require.True(t, ok)

		logoCfg, exists := fields["logoUrl"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, logoCfg.Action)

		taxIdCfg, exists := fields["taxId"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, taxIdCfg.Action)
	})

	t.Run("registers worker sensitive fields", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "test-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)

		fields, ok := concrete.sdm.registeredFields(permission.ResourceWorker)
		require.True(t, ok)

		licenseCfg, exists := fields["licenseNumber"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, licenseCfg.Action)

		dobCfg, exists := fields["dateOfBirth"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, dobCfg.Action)

		profileLicenseCfg, exists := fields["profile.licenseNumber"]
		assert.True(t, exists)
		assert.Equal(t, auditports.SensitiveFieldMask, profileLicenseCfg.Action)
		assert.Equal(t, "profile", profileLicenseCfg.Path)
	})

	t.Run("registered fields are applied during sanitization", func(t *testing.T) {
		t.Parallel()

		repo := new(mockAuditRepository)
		bufferRepo := new(mockBufferRepository)
		logger := zap.NewNop()
		cfg := &config.Config{
			App: appconfig.Config{Env: "test"},
			Security: securityconfig.Config{
				Encryption: securityconfig.EncryptionConfig{Key: "test-key"},
			},
		}
		auditMetrics := auditmetrics.NewAudit(nil, logger, false)
		metricsRegistry := &metrics.Registry{Audit: auditMetrics}

		svc := New(Params{
			AuditRepository:       repo,
			AuditBufferRepository: bufferRepo,
			Realtime:              &noopRealtimeService{},
			Logger:                logger,
			Config:                cfg,
			Metrics:               metricsRegistry,
		})

		concrete := svc.(*service)

		entry := &audit.Entry{
			Resource: permission.ResourceUser,
			CurrentState: map[string]any{
				"name":           "John Doe",
				"password":       "supersecret",
				"hashedPassword": "bcrypt-hash-value",
				"emailAddress":   "john@example.com",
			},
		}

		err := concrete.sdm.SanitizeEntry(entry)
		require.NoError(t, err)

		assert.Nil(t, entry.CurrentState["password"])
		assert.Nil(t, entry.CurrentState["hashedPassword"])
		assert.NotEqual(t, "john@example.com", entry.CurrentState["emailAddress"])
		assert.Equal(t, "John Doe", entry.CurrentState["name"])
	})
}

package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

type mockScheduleHandle struct {
	id           string
	updateErr    error
	deleteErr    error
	invokeUpdate bool
	updateCalled bool
}

func (m *mockScheduleHandle) GetID() string { return m.id }
func (m *mockScheduleHandle) Delete(_ context.Context) error {
	return m.deleteErr
}
func (m *mockScheduleHandle) Backfill(_ context.Context, _ client.ScheduleBackfillOptions) error {
	return nil
}
func (m *mockScheduleHandle) Update(_ context.Context, opts client.ScheduleUpdateOptions) error {
	m.updateCalled = true
	if m.invokeUpdate && opts.DoUpdate != nil {
		input := client.ScheduleUpdateInput{
			Description: client.ScheduleDescription{
				Schedule: client.Schedule{},
			},
		}
		_, _ = opts.DoUpdate(input)
	}
	return m.updateErr
}
func (m *mockScheduleHandle) Describe(_ context.Context) (*client.ScheduleDescription, error) {
	return nil, nil
}
func (m *mockScheduleHandle) Trigger(_ context.Context, _ client.ScheduleTriggerOptions) error {
	return nil
}
func (m *mockScheduleHandle) Pause(_ context.Context, _ client.SchedulePauseOptions) error {
	return nil
}
func (m *mockScheduleHandle) Unpause(_ context.Context, _ client.ScheduleUnpauseOptions) error {
	return nil
}

type mockScheduleListIterator struct {
	entries []*client.ScheduleListEntry
	index   int
	err     error
}

func (m *mockScheduleListIterator) HasNext() bool {
	return m.index < len(m.entries)
}

func (m *mockScheduleListIterator) Next() (*client.ScheduleListEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	entry := m.entries[m.index]
	m.index++
	return entry, nil
}

type mockScheduleClient struct {
	createHandle client.ScheduleHandle
	createErr    error
	listIter     client.ScheduleListIterator
	listErr      error
	handles      map[string]*mockScheduleHandle
}

func (m *mockScheduleClient) Create(
	_ context.Context,
	_ client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createHandle, nil
}

func (m *mockScheduleClient) List(
	_ context.Context,
	_ client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listIter, nil
}

func (m *mockScheduleClient) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	if h, ok := m.handles[id]; ok {
		return h
	}
	return &mockScheduleHandle{id: id}
}

type retryMockScheduleClient struct {
	callCount     int
	failUntilCall int
	listErr       error
	successIter   client.ScheduleListIterator
	createHandle  client.ScheduleHandle
}

func (m *retryMockScheduleClient) Create(
	_ context.Context,
	_ client.ScheduleOptions,
) (client.ScheduleHandle, error) {
	return m.createHandle, nil
}

func (m *retryMockScheduleClient) List(
	_ context.Context,
	_ client.ScheduleListOptions,
) (client.ScheduleListIterator, error) {
	m.callCount++
	if m.callCount < m.failUntilCall {
		return nil, m.listErr
	}
	return m.successIter, nil
}

func (m *retryMockScheduleClient) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	return &mockScheduleHandle{id: id}
}

type mockTemporalClient struct {
	client.Client
	scheduleClient client.ScheduleClient
}

func (m *mockTemporalClient) ScheduleClient() client.ScheduleClient {
	return m.scheduleClient
}

func TestReconcileResult_HasErrors_NoErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Errors: []error{},
	}

	assert.False(t, result.HasErrors())
}

func TestReconcileResult_HasErrors_WithErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Errors: []error{errors.New("something failed")},
	}

	assert.True(t, result.HasErrors())
}

func TestReconcileResult_Summary_AllZeros(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{},
		Updated: []string{},
		Deleted: []string{},
		Skipped: []string{},
		Errors:  []error{},
	}

	assert.Equal(t, "created=0 updated=0 deleted=0 skipped=0 errors=0", result.Summary())
}

func TestReconcileResult_Summary_MixedValues(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{"sched-1", "sched-2"},
		Updated: []string{"sched-3"},
		Deleted: []string{"sched-4", "sched-5", "sched-6"},
		Skipped: []string{"sched-7"},
		Errors:  []error{},
	}

	assert.Equal(t, "created=2 updated=1 deleted=3 skipped=1 errors=0", result.Summary())
}

func TestReconcileResult_Summary_WithErrors(t *testing.T) {
	t.Parallel()

	result := &ReconcileResult{
		Created: []string{"sched-1"},
		Updated: []string{},
		Deleted: []string{},
		Skipped: []string{"sched-2"},
		Errors:  []error{errors.New("err1"), errors.New("err2")},
	}

	assert.Equal(t, "created=1 updated=0 deleted=0 skipped=1 errors=2", result.Summary())
}

func TestNewReconciler(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	reconciler := NewReconciler(nil, registry, logger)

	require.NotNil(t, reconciler)
	assert.Equal(t, registry, reconciler.registry)
}

func TestReconcile_CreatesNewSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "new-schedule-1",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
			{
				ID:        "new-schedule-2",
				Spec:      Cron("0 9 * * *"),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "new-schedule-1"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 2)
	assert.Contains(t, result.Created, "new-schedule-1")
	assert.Contains(t, result.Created, "new-schedule-2")
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Deleted)
	assert.Empty(t, result.Skipped)
	assert.Empty(t, result.Errors)
}

func TestReconcile_UpdatesExistingSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "existing-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	handle := &mockScheduleHandle{id: "existing-schedule", invokeUpdate: true}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "existing-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte("different-hash")},
							},
						},
					},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"existing-schedule": handle,
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Len(t, result.Updated, 1)
	assert.Contains(t, result.Updated, "existing-schedule")
	assert.Empty(t, result.Deleted)
	assert.Empty(t, result.Skipped)
	assert.Empty(t, result.Errors)
	assert.True(t, handle.updateCalled)
}

func TestReconcile_SkipsUnchangedSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "unchanged-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "unchanged-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte(sched.Hash())},
							},
						},
					},
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Deleted)
	assert.Len(t, result.Skipped, 1)
	assert.Contains(t, result.Skipped, "unchanged-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_DeletesOrphanSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "keep-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "orphan-schedule"},
				},
			},
			createHandle: &mockScheduleHandle{id: "keep-schedule"},
			handles: map[string]*mockScheduleHandle{
				"orphan-schedule": {id: "orphan-schedule"},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Contains(t, result.Created, "keep-schedule")
	assert.Len(t, result.Deleted, 1)
	assert.Contains(t, result.Deleted, "orphan-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_ErrorInCollectSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:   "",
				Spec: Every(30 * time.Minute),
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to collect schedules")
}

func TestReconcile_ErrorInListExistingSchedules(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "test-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listErr: errors.New("connection refused"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list existing schedules")
}

func TestReconcile_ErrorInCreateSchedule(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "fail-create",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("permission denied"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Created)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "create fail-create")
	assert.Contains(t, result.Errors[0].Error(), "permission denied")
}

func TestReconcile_AlreadyExistsErrorInCreate_FallsBackToUpdate(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "already-exists-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	handle := &mockScheduleHandle{id: "already-exists-schedule", invokeUpdate: true}

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: serviceerror.NewAlreadyExists("schedule already exists"),
			handles: map[string]*mockScheduleHandle{
				"already-exists-schedule": handle,
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Contains(t, result.Created, "already-exists-schedule")
	assert.Empty(t, result.Errors)
	assert.True(t, handle.updateCalled)
}

func TestReconcile_DeleteSchedule_NotFoundError_Succeeds(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "already-gone-schedule"},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"already-gone-schedule": {
					id:        "already-gone-schedule",
					deleteErr: serviceerror.NewNotFound("schedule not found"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Len(t, result.Deleted, 1)
	assert.Contains(t, result.Deleted, "already-gone-schedule")
	assert.Empty(t, result.Errors)
}

func TestReconcile_DeleteSchedule_OtherError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "fail-delete-schedule"},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"fail-delete-schedule": {
					id:        "fail-delete-schedule",
					deleteErr: errors.New("internal error"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Deleted)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "delete fail-delete-schedule")
}

func TestReconcile_ListIteratorError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "test-schedule",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{ID: "entry-1"},
				},
				err: errors.New("iteration failed"),
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to iterate schedules")
}

func TestReconcile_UpdateScheduleError(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	sched := &Schedule{
		ID:        "update-fail-schedule",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	provider := &testProvider{
		schedules: []*Schedule{sched},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{
					{
						ID: "update-fail-schedule",
						Memo: &commonpb.Memo{
							Fields: map[string]*commonpb.Payload{
								"scheduleHash": {Data: []byte("different-hash")},
							},
						},
					},
				},
			},
			handles: map[string]*mockScheduleHandle{
				"update-fail-schedule": {
					id:        "update-fail-schedule",
					updateErr: errors.New("update failed"),
				},
			},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.Reconcile(t.Context())

	require.NoError(t, err)
	assert.Empty(t, result.Updated)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "update update-fail-schedule")
}

func TestNeedsUpdate_NoMemo(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID:   "test",
		Memo: nil,
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_NoFields(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: nil,
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_NoHashField(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"otherField": {Data: []byte("value")},
			},
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_HashDiffers(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"scheduleHash": {Data: []byte("old-hash-value")},
			},
		},
	}
	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	assert.True(t, reconciler.needsUpdate(existing, desired))
}

func TestNeedsUpdate_HashMatches(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)
	reconciler := NewReconciler(nil, registry, logger)

	desired := &Schedule{
		ID:        "test",
		Spec:      Every(30 * time.Minute),
		Workflow:  dummyWorkflow,
		TaskQueue: "test-queue",
	}

	existing := &client.ScheduleListEntry{
		ID: "test",
		Memo: &commonpb.Memo{
			Fields: map[string]*commonpb.Payload{
				"scheduleHash": {Data: []byte(desired.Hash())},
			},
		},
	}

	assert.False(t, reconciler.needsUpdate(existing, desired))
}

func TestReconcileWithRetry_SuccessOnFirstTry(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "retry-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createHandle: &mockScheduleHandle{id: "retry-test"},
		},
	}

	reconciler := NewReconciler(mc, registry, logger)
	result, err := reconciler.ReconcileWithRetry(t.Context(), 3)

	require.NoError(t, err)
	assert.Len(t, result.Created, 1)
	assert.Empty(t, result.Errors)
}

func TestReconcileWithRetry_SuccessOnRetry(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "retry-success",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	sc := &retryMockScheduleClient{
		failUntilCall: 2,
		listErr:       errors.New("temporary failure"),
		successIter: &mockScheduleListIterator{
			entries: []*client.ScheduleListEntry{},
		},
		createHandle: &mockScheduleHandle{id: "retry-success"},
	}

	mc := &mockTemporalClient{
		scheduleClient: sc,
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 3)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Created, 1)
}

func TestReconcileWithRetry_ContextCancelled(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "cancel-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("always fails"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 3)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotNil(t, result)
}

func TestReconcileWithRetry_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "exhaust-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listErr: errors.New("persistent failure"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 0)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "persistent failure")
}

func TestReconcileWithRetry_MaxRetriesExhausted_WithResultErrors(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	registry := NewRegistry(logger)

	provider := &testProvider{
		schedules: []*Schedule{
			{
				ID:        "error-result-test",
				Spec:      Every(30 * time.Minute),
				Workflow:  dummyWorkflow,
				TaskQueue: "test-queue",
			},
		},
	}
	registry.RegisterProvider(provider)

	mc := &mockTemporalClient{
		scheduleClient: &mockScheduleClient{
			listIter: &mockScheduleListIterator{
				entries: []*client.ScheduleListEntry{},
			},
			createErr: errors.New("create always fails"),
		},
	}

	reconciler := NewReconciler(mc, registry, logger)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	result, err := reconciler.ReconcileWithRetry(ctx, 0)

	require.Error(t, err)
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

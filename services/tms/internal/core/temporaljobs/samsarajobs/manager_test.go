package samsarajobs

import (
	"context"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeWorkflowRun struct {
	client.WorkflowRun
	workflowID string
	runID      string
	getFunc    func(ctx context.Context, value any) error
}

func (f *fakeWorkflowRun) GetID() string {
	return f.workflowID
}

func (f *fakeWorkflowRun) GetRunID() string {
	return f.runID
}

func (f *fakeWorkflowRun) Get(ctx context.Context, value any) error {
	if f.getFunc == nil {
		return nil
	}
	return f.getFunc(ctx, value)
}

type fakeTemporalClient struct {
	client.Client
	executeWorkflowFunc        func(ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any) (client.WorkflowRun, error)
	describeWorkflowStatusFunc func(ctx context.Context, workflowID, runID string) (*workflowservicepb.DescribeWorkflowExecutionResponse, error)
	getWorkflowFunc            func(ctx context.Context, workflowID, runID string) client.WorkflowRun
}

func (f *fakeTemporalClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	if f.executeWorkflowFunc == nil {
		return nil, nil
	}
	return f.executeWorkflowFunc(ctx, options, workflow, args...)
}

func (f *fakeTemporalClient) DescribeWorkflowExecution(
	ctx context.Context,
	workflowID string,
	runID string,
) (*workflowservicepb.DescribeWorkflowExecutionResponse, error) {
	if f.describeWorkflowStatusFunc == nil {
		return nil, nil
	}
	return f.describeWorkflowStatusFunc(ctx, workflowID, runID)
}

func (f *fakeTemporalClient) GetWorkflow(
	ctx context.Context,
	workflowID string,
	runID string,
) client.WorkflowRun {
	if f.getWorkflowFunc == nil {
		return &fakeWorkflowRun{}
	}
	return f.getWorkflowFunc(ctx, workflowID, runID)
}

func setupManager(
	t *testing.T,
	temporalClient client.Client,
) (*Manager, *mocks.MockWorkerRepository) {
	t.Helper()
	repo := mocks.NewMockWorkerRepository(t)
	svc := samsarasyncservice.New(samsarasyncservice.Params{
		Logger: zap.NewNop(),
		Repo:   repo,
	})
	return NewManager(ManagerParams{
		TemporalClient: temporalClient,
		WorkflowStarter: workflowstarter.New(workflowstarter.Params{
			TemporalClient: temporalClient,
		}),
		SyncService: svc,
	}), repo
}

func TestManager_StartWorkersSyncWorkflow(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		manager, repo := setupManager(t, &fakeTemporalClient{
			executeWorkflowFunc: func(_ context.Context, options client.StartWorkflowOptions, workflow any, _ ...any) (client.WorkflowRun, error) {
				assert.Equal(t, temporaltype.IntegrationTaskQueue, options.TaskQueue)
				assert.Equal(t, SyncWorkersToSamsaraWorkflowName, workflow)
				return &fakeWorkflowRun{workflowID: options.ID, runID: "run-1"}, nil
			},
		})

		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			&repositories.WorkerSyncReadinessCounts{
				TotalWorkers:        10,
				ActiveWorkers:       8,
				SyncedActiveWorkers: 7,
			},
			nil,
		).Once()

		result, err := manager.StartWorkersSyncWorkflow(t.Context(), tenantInfo)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "run-1", result.RunID)
	})

	t.Run("already started", func(t *testing.T) {
		t.Parallel()
		manager, repo := setupManager(t, &fakeTemporalClient{
			executeWorkflowFunc: func(_ context.Context, _ client.StartWorkflowOptions, _ any, _ ...any) (client.WorkflowRun, error) {
				return nil, serviceerror.NewWorkflowExecutionAlreadyStarted(
					"already running",
					"start-1",
					"run-2",
				)
			},
		})
		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			&repositories.WorkerSyncReadinessCounts{
				TotalWorkers:        10,
				ActiveWorkers:       8,
				SyncedActiveWorkers: 7,
			},
			nil,
		).Once()

		_, err := manager.StartWorkersSyncWorkflow(t.Context(), tenantInfo)
		require.Error(t, err)
		assert.True(t, errortypes.IsConflictError(err))
	})

	t.Run("not configured", func(t *testing.T) {
		t.Parallel()
		manager, repo := setupManager(t, nil)
		repo.On("GetWorkerSyncReadinessCounts", mock.Anything, tenantInfo).Return(
			&repositories.WorkerSyncReadinessCounts{
				TotalWorkers:        10,
				ActiveWorkers:       8,
				SyncedActiveWorkers: 7,
			},
			nil,
		).Once()

		_, err := manager.StartWorkersSyncWorkflow(t.Context(), tenantInfo)
		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
	})
}

func TestManager_GetWorkersSyncWorkflowStatus(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	workflowID := buildWorkersSyncWorkflowID(tenantInfo)
	startedAt := time.Unix(1710000000, 0)

	manager, _ := setupManager(t, &fakeTemporalClient{
		describeWorkflowStatusFunc: func(_ context.Context, requestedWorkflowID, runID string) (*workflowservicepb.DescribeWorkflowExecutionResponse, error) {
			assert.Equal(t, workflowID, requestedWorkflowID)
			assert.Equal(t, "", runID)
			return &workflowservicepb.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
					Execution: &commonpb.WorkflowExecution{
						WorkflowId: workflowID,
						RunId:      "run-1",
					},
					Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
					TaskQueue: temporaltype.IntegrationTaskQueue,
					StartTime: timestamppb.New(startedAt),
				},
			}, nil
		},
	})

	result, err := manager.GetWorkersSyncWorkflowStatus(t.Context(), tenantInfo, workflowID, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(), result.Status)
}

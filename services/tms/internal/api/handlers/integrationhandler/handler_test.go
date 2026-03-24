package integrationhandler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/emoss08/trenova/internal/api/handlers/integrationhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/testutil"
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

func setupIntegrationHandler(
	t *testing.T,
	temporalClient client.Client,
) (*integrationhandler.Handler, *mocks.MockWorkerRepository) {
	t.Helper()

	logger := zap.NewNop()
	repo := mocks.NewMockWorkerRepository(t)
	syncService := samsarasyncservice.New(samsarasyncservice.Params{
		Logger: logger,
		Repo:   repo,
	})
	jobsManager := samsarajobs.NewManager(samsarajobs.ManagerParams{
		TemporalClient: temporalClient,
		SyncService:    syncService,
	})

	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: logger,
		Config: cfg,
	})

	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})

	return integrationhandler.New(integrationhandler.Params{
		SyncService:          syncService,
		JobsManager:          jobsManager,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	}), repo
}

func TestIntegrationHandler_StartWorkerSync_Success(t *testing.T) {
	t.Parallel()

	h, repo := setupIntegrationHandler(t, &fakeTemporalClient{
		executeWorkflowFunc: func(_ context.Context, options client.StartWorkflowOptions, workflow any, args ...any) (client.WorkflowRun, error) {
			assert.Equal(t, temporaltype.IntegrationTaskQueue, options.TaskQueue)
			assert.Equal(t, samsarajobs.SyncWorkersToSamsaraWorkflowName, workflow)
			return &fakeWorkflowRun{workflowID: options.ID, runID: "run-1"}, nil
		},
	})
	repo.On("GetWorkerSyncReadinessCounts", mock.Anything, mock.Anything).Return(
		&repositories.WorkerSyncReadinessCounts{
			TotalWorkers:        10,
			ActiveWorkers:       8,
			SyncedActiveWorkers: 7,
		},
		nil,
	).Once()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/integrations/samsara/workers/sync/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusAccepted, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "run-1", resp["runId"])
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(), resp["status"])
}

func TestIntegrationHandler_StartWorkerSync_NotConfigured(t *testing.T) {
	t.Parallel()

	h, repo := setupIntegrationHandler(t, nil)
	repo.On("GetWorkerSyncReadinessCounts", mock.Anything, mock.Anything).Return(
		&repositories.WorkerSyncReadinessCounts{
			TotalWorkers:        10,
			ActiveWorkers:       8,
			SyncedActiveWorkers: 7,
		},
		nil,
	).Once()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/integrations/samsara/workers/sync/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusUnprocessableEntity, ginCtx.ResponseCode())
}

func TestIntegrationHandler_GetWorkerSyncStatus_Success(t *testing.T) {
	t.Parallel()

	startTime := time.Unix(1710000000, 0)
	h, _ := setupIntegrationHandler(t, &fakeTemporalClient{
		describeWorkflowStatusFunc: func(_ context.Context, workflowID, runID string) (*workflowservicepb.DescribeWorkflowExecutionResponse, error) {
			return &workflowservicepb.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
					Execution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: "run-1"},
					Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
					TaskQueue: temporaltype.IntegrationTaskQueue,
					StartTime: timestamppb.New(startTime),
				},
			}, nil
		},
	})

	workflowID := fmt.Sprintf(
		"samsara-worker-sync-%s-%s",
		testutil.TestOrgID.String(),
		testutil.TestBuID.String(),
	)
	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/integrations/samsara/workers/sync/" + workflowID + "/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(), resp["status"])
}

func TestIntegrationHandler_GetWorkerSyncReadiness_Success(t *testing.T) {
	t.Parallel()

	h, repo := setupIntegrationHandler(t, &fakeTemporalClient{})
	repo.On("GetWorkerSyncReadinessCounts", mock.Anything, mock.Anything).Return(
		&repositories.WorkerSyncReadinessCounts{
			TotalWorkers:        10,
			ActiveWorkers:       8,
			SyncedActiveWorkers: 5,
		},
		nil,
	).Once()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/integrations/samsara/workers/sync/readiness/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.EqualValues(t, 10, resp["totalWorkers"])
	assert.EqualValues(t, 8, resp["activeWorkers"])
	assert.EqualValues(t, 5, resp["syncedActiveWorkers"])
	assert.EqualValues(t, 3, resp["unsyncedActiveWorkers"])
	assert.Equal(t, false, resp["allActiveWorkersSynced"])
}

func TestIntegrationHandler_GetWorkerSyncDrift_Success(t *testing.T) {
	t.Parallel()

	h, repo := setupIntegrationHandler(t, &fakeTemporalClient{})
	repo.On("ListWorkerSyncDrifts", mock.Anything, mock.Anything).Return(
		[]repositories.WorkerSyncDriftRecord{
			{
				WorkerID:   "wrk_1",
				WorkerName: "Casey Nguyen",
				DriftType:  repositories.WorkerSyncDriftTypeMissingMapping,
				Message:    "active worker is missing a Samsara driver mapping",
				DetectedAt: time.Now().Unix(),
			},
		},
		nil,
	).Once()

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/integrations/samsara/workers/sync/drift/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.EqualValues(t, 1, resp["totalDrifts"])
}

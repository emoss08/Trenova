package samsaracontract

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	workflowstarterservice "github.com/emoss08/trenova/internal/core/services/workflowstarter"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type ManagerParams struct {
	fx.In

	TemporalClient  client.Client
	WorkflowStarter kernelports.WorkflowStarter
	SyncService     *samsarasyncservice.Service
}

type Manager struct {
	temporalClient  client.Client
	workflowStarter kernelports.WorkflowStarter
	syncService     *samsarasyncservice.Service
}

func NewManager(p ManagerParams) *Manager {
	workflowStarter := p.WorkflowStarter
	if workflowStarter == nil {
		workflowStarter = workflowstarterservice.New(workflowstarterservice.Params{
			TemporalClient: p.TemporalClient,
		})
	}

	return &Manager{
		temporalClient:  p.TemporalClient,
		workflowStarter: workflowStarter,
		syncService:     p.SyncService,
	}
}

func (m *Manager) StartWorkersSyncWorkflow(
	ctx context.Context,
	TenantInfo pagination.TenantInfo,
) (*SyncWorkflowStartResponse, error) {
	readiness, err := m.syncService.GetWorkerSyncReadiness(ctx, TenantInfo)
	if err != nil {
		return nil, err
	}
	if readiness.AllActiveWorkersSynced {
		return nil, errortypes.NewBusinessError(
			"all active workers are already synced to Samsara",
		)
	}

	workflowID := BuildWorkersSyncWorkflowID(TenantInfo)
	payload := &WorkersSyncWorkflowPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: TenantInfo.OrgID,
			BusinessUnitID: TenantInfo.BuID,
			UserID:         TenantInfo.UserID,
			Timestamp:      timeutils.NowUnix(),
			Metadata: map[string]any{
				"trigger": "api",
			},
		},
		RequestedBy: TenantInfo.UserID,
	}

	if !m.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError("Samsara worker sync is not configured")
	}

	run, err := m.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       workflowID,
			TaskQueue:                                temporaltype.IntegrationTaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Sync workers to Samsara for business unit %s",
				TenantInfo.BuID.String(),
			),
		},
		SyncWorkersToSamsaraWorkflowName,
		payload,
	)
	if err != nil {
		var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStartedErr) {
			return nil, errortypes.NewConflictError(
				"Samsara worker sync is already running",
			).WithUsageStats(map[string]any{
				"workflowId": workflowID,
				"runId":      alreadyStartedErr.RunId,
			})
		}

		return nil, errortypes.NewBusinessError(
			"failed to start Samsara worker sync workflow",
		).WithInternal(err)
	}

	return &SyncWorkflowStartResponse{
		WorkflowID:  run.GetID(),
		RunID:       run.GetRunID(),
		TaskQueue:   temporaltype.IntegrationTaskQueue,
		Status:      enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(),
		SubmittedAt: timeutils.NowUnix(),
	}, nil
}

func (m *Manager) GetWorkersSyncWorkflowStatus(
	ctx context.Context,
	TenantInfo pagination.TenantInfo,
	workflowID string,
	runID string,
) (*SyncWorkflowStatusResponse, error) {
	requestedWorkflowID := strings.TrimSpace(workflowID)
	if requestedWorkflowID == "" {
		return nil, errortypes.NewBusinessError("workflow id is required")
	}

	expectedWorkflowID := BuildWorkersSyncWorkflowID(TenantInfo)
	if requestedWorkflowID != expectedWorkflowID {
		return nil, errortypes.NewNotFoundError("Samsara worker sync workflow not found")
	}

	description, err := m.temporalClient.DescribeWorkflowExecution(
		ctx,
		requestedWorkflowID,
		strings.TrimSpace(runID),
	)
	if err != nil {
		if notFoundErr, ok := errors.AsType[*serviceerror.NotFound](err); ok {
			return nil, errortypes.NewNotFoundError(
				"Samsara worker sync workflow not found",
			).WithInternal(notFoundErr)
		}

		return nil, errortypes.NewBusinessError(
			"failed to describe Samsara worker sync workflow",
		).WithInternal(err)
	}

	info := description.GetWorkflowExecutionInfo()
	if info == nil || info.GetExecution() == nil {
		return nil, errortypes.NewNotFoundError("Samsara worker sync workflow not found")
	}

	status := info.GetStatus()
	response := &SyncWorkflowStatusResponse{
		WorkflowID: requestedWorkflowID,
		RunID:      info.GetExecution().GetRunId(),
		TaskQueue:  info.GetTaskQueue(),
		Status:     status.String(),
	}
	if response.RunID == "" {
		response.RunID = strings.TrimSpace(runID)
	}
	if response.TaskQueue == "" {
		response.TaskQueue = temporaltype.IntegrationTaskQueue
	}

	if startTime := info.GetStartTime(); startTime != nil {
		response.StartedAt = startTime.AsTime().Unix()
	}
	if closeTime := info.GetCloseTime(); closeTime != nil {
		response.ClosedAt = closeTime.AsTime().Unix()
	}

	switch status {
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		workflowRun := m.temporalClient.GetWorkflow(ctx, requestedWorkflowID, response.RunID)
		workflowResult := new(WorkersSyncWorkflowResult)
		if getErr := workflowRun.Get(ctx, workflowResult); getErr != nil {
			return nil, errortypes.NewBusinessError(
				"failed to retrieve Samsara worker sync workflow result",
			).WithInternal(getErr)
		}
		response.Result = workflowResult.Result
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED,
		enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED,
		enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		workflowRun := m.temporalClient.GetWorkflow(ctx, requestedWorkflowID, response.RunID)
		if getErr := workflowRun.Get(ctx, nil); getErr != nil {
			response.Error = getErr.Error()
		}
	case enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW,
		enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED:
	default:
	}

	return response, nil
}

func BuildWorkersSyncWorkflowID(TenantInfo pagination.TenantInfo) string {
	return fmt.Sprintf(
		syncWorkersWorkflowIDFormat,
		TenantInfo.OrgID.String(),
		TenantInfo.BuID.String(),
	)
}

const SyncWorkersToSamsaraWorkflowName = "SyncWorkersToSamsaraWorkflow"

const syncWorkersWorkflowIDFormat = "samsara-worker-sync-%s-%s"

type WorkersSyncWorkflowPayload struct {
	temporaltype.BasePayload
	RequestedBy pulid.ID `json:"requestedBy"`
}

type WorkersSyncWorkflowResult struct {
	Result *telematicsports.SamsaraWorkerSyncResult `json:"result,omitempty"`
}

type SyncWorkflowStartResponse struct {
	WorkflowID  string `json:"workflowId"`
	RunID       string `json:"runId"`
	TaskQueue   string `json:"taskQueue"`
	Status      string `json:"status"`
	SubmittedAt int64  `json:"submittedAt"`
}

type SyncWorkflowStatusResponse struct {
	WorkflowID string                                   `json:"workflowId"`
	RunID      string                                   `json:"runId"`
	TaskQueue  string                                   `json:"taskQueue"`
	Status     string                                   `json:"status"`
	StartedAt  int64                                    `json:"startedAt,omitempty"`
	ClosedAt   int64                                    `json:"closedAt,omitempty"`
	Result     *telematicsports.SamsaraWorkerSyncResult `json:"result,omitempty"`
	Error      string                                   `json:"error,omitempty"`
}

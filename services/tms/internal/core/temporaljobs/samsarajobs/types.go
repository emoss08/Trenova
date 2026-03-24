package samsarajobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	SyncWorkersToSamsaraWorkflowName = "SyncWorkersToSamsaraWorkflow"
	syncWorkersWorkflowIDFormat      = "samsara-worker-sync-%s-%s"
)

type WorkersSyncWorkflowPayload struct {
	temporaltype.BasePayload
	RequestedBy pulid.ID `json:"requestedBy"`
}

type WorkersSyncWorkflowResult struct {
	Result *services.SamsaraWorkerSyncResult `json:"result,omitempty"`
}

type SyncWorkflowStartResponse struct {
	WorkflowID  string `json:"workflowId"`
	RunID       string `json:"runId"`
	TaskQueue   string `json:"taskQueue"`
	Status      string `json:"status"`
	SubmittedAt int64  `json:"submittedAt"`
}

type SyncWorkflowStatusResponse struct {
	WorkflowID string                            `json:"workflowId"`
	RunID      string                            `json:"runId"`
	TaskQueue  string                            `json:"taskQueue"`
	Status     string                            `json:"status"`
	StartedAt  int64                             `json:"startedAt,omitempty"`
	ClosedAt   int64                             `json:"closedAt,omitempty"`
	Result     *services.SamsaraWorkerSyncResult `json:"result,omitempty"`
	Error      string                            `json:"error,omitempty"`
}

package extractionevalcontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

const ExtractionEvalRunWorkflowName = "ExtractionEvalRunWorkflow"

type RunPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	RunID          pulid.ID `json:"runId"`
	AfterOrdinal   int      `json:"afterOrdinal,omitempty"`
	Evaluated      int      `json:"evaluated,omitempty"`
}

func EvaluationPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflowcontract.PriorityEvaluation,
		FairnessKey: orgID.String(),
	}
}

var _ extractionevalports.ExtractionEvalRunStarter = (*RunStarter)(nil)

type RunStarterParams struct {
	fx.In

	Client client.Client
}

type RunStarter struct {
	client client.Client
}

func NewRunStarter(p RunStarterParams) *RunStarter {
	return &RunStarter{client: p.Client}
}

func (s *RunStarter) StartExtractionEvalRun(
	ctx context.Context,
	start *extractionevalports.ExtractionEvalRunStart,
) (string, error) {
	workflowID := extractioneval.RunWorkflowID(start.RunID)
	if _, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.DocumentExtractionTaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		Priority:              EvaluationPriority(start.TenantInfo.OrgID),
	}, ExtractionEvalRunWorkflowName, &RunPayload{
		OrganizationID: start.TenantInfo.OrgID,
		BusinessUnitID: start.TenantInfo.BuID,
		RunID:          start.RunID,
	}); err != nil {
		return "", fmt.Errorf("start extraction evaluation workflow: %w", err)
	}

	return workflowID, nil
}

const (
	RefreezeWorkflowName  = "RefreezeExtractionEvalCasesWorkflow"
	RefreezeProgressQuery = "progress"
)

type RefreezePayload struct {
	OrganizationID pulid.ID   `json:"organizationId"`
	BusinessUnitID pulid.ID   `json:"businessUnitId"`
	UserID         pulid.ID   `json:"userId"`
	CaseIDs        []pulid.ID `json:"caseIds"`
}

type RefreezeProgress struct {
	Total   int `json:"total"`
	Frozen  int `json:"frozen"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

func RefreezeWorkflowID(tenant pagination.TenantInfo) string {
	return "extraction-eval-refreeze:" + tenant.OrgID.String() + ":" + tenant.BuID.String()
}

var _ extractionevalports.ExtractionEvalRefreezeStarter = (*RunStarter)(nil)

func (s *RunStarter) StartExtractionEvalRefreeze(
	ctx context.Context,
	start *extractionevalports.ExtractionEvalRefreezeStart,
) (*extractionevalports.ExtractionEvalRefreezeStarted, error) {
	workflowID := RefreezeWorkflowID(start.TenantInfo)
	_, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       workflowID,
		TaskQueue:                                temporaltype.DocumentExtractionTaskQueue,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy:                 enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		Priority:                                 EvaluationPriority(start.TenantInfo.OrgID),
		StaticSummary: fmt.Sprintf(
			"Freeze %d evaluation cases through Docling",
			len(start.CaseIDs),
		),
	}, RefreezeWorkflowName, &RefreezePayload{
		OrganizationID: start.TenantInfo.OrgID,
		BusinessUnitID: start.TenantInfo.BuID,
		UserID:         start.TenantInfo.UserID,
		CaseIDs:        start.CaseIDs,
	})
	if _, running := errors.AsType[*serviceerror.WorkflowExecutionAlreadyStarted](err); running {
		return &extractionevalports.ExtractionEvalRefreezeStarted{
			WorkflowID:     workflowID,
			AlreadyRunning: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("start extraction evaluation re-freeze workflow: %w", err)
	}

	return &extractionevalports.ExtractionEvalRefreezeStarted{WorkflowID: workflowID}, nil
}

func (s *RunStarter) ExtractionEvalRefreezeProgress(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionevalports.ExtractionEvalRefreezeProgress, error) {
	workflowID := RefreezeWorkflowID(tenant)
	described, err := s.client.DescribeWorkflowExecution(ctx, workflowID, "")
	if _, missing := errors.AsType[*serviceerror.NotFound](err); missing {
		return nil, nil //nolint:nilnil // no re-freeze has run for this organization
	}
	if err != nil {
		return nil, fmt.Errorf("describe extraction evaluation re-freeze: %w", err)
	}

	progress := &extractionevalports.ExtractionEvalRefreezeProgress{
		WorkflowID: workflowID,
		Status: RefreezeStatusOf(
			described.GetWorkflowExecutionInfo().GetStatus(),
		),
	}
	encoded, err := s.client.QueryWorkflow(ctx, workflowID, "", RefreezeProgressQuery)
	if err != nil {
		if progress.Status == extractioneval.RefreezeStatusFailed {
			return progress, nil
		}
		return nil, fmt.Errorf("query extraction evaluation re-freeze: %w", err)
	}

	var counts RefreezeProgress
	if err = encoded.Get(&counts); err != nil {
		return nil, fmt.Errorf("decode extraction evaluation re-freeze progress: %w", err)
	}
	progress.Total = counts.Total
	progress.Frozen = counts.Frozen
	progress.Skipped = counts.Skipped
	progress.Failed = counts.Failed

	return progress, nil
}

func RefreezeStatusOf(status enums.WorkflowExecutionStatus) extractioneval.RefreezeStatus {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW,
		enums.WORKFLOW_EXECUTION_STATUS_PAUSED:
		return extractioneval.RefreezeStatusRunning
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return extractioneval.RefreezeStatusCompleted
	case enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED,
		enums.WORKFLOW_EXECUTION_STATUS_FAILED,
		enums.WORKFLOW_EXECUTION_STATUS_CANCELED,
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return extractioneval.RefreezeStatusFailed
	default:
		return extractioneval.RefreezeStatusFailed
	}
}

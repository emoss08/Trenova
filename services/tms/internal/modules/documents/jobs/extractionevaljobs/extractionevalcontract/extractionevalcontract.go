package extractionevalcontract

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
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

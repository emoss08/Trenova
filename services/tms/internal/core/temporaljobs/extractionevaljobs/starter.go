package extractionevaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

var _ services.ExtractionEvalRunStarter = (*RunStarter)(nil)

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
	start *services.ExtractionEvalRunStart,
) (string, error) {
	workflowID := extractioneval.RunWorkflowID(start.RunID)
	if _, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.DocumentIntelligenceTaskQueue,
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		Priority:              evaluationPriority(start.TenantInfo.OrgID),
	}, ExtractionEvalRunWorkflowName, &RunPayload{
		OrganizationID: start.TenantInfo.OrgID,
		BusinessUnitID: start.TenantInfo.BuID,
		RunID:          start.RunID,
	}); err != nil {
		return "", fmt.Errorf("start extraction evaluation workflow: %w", err)
	}

	return workflowID, nil
}

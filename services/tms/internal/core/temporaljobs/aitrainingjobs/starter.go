package aitrainingjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

var _ services.AITrainingExportStarter = (*ExportStarter)(nil)

type ExportStarterParams struct {
	fx.In

	Client client.Client
}

type ExportStarter struct {
	client client.Client
}

func NewExportStarter(p ExportStarterParams) *ExportStarter {
	return &ExportStarter{client: p.Client}
}

func AsExportStarter(s *ExportStarter) services.AITrainingExportStarter { return s }

func (s *ExportStarter) StartAITrainingExport(ctx context.Context, exportID pulid.ID) (string, error) {
	workflowID := aitraining.ExportWorkflowID(exportID)
	if _, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, AITrainingExportWorkflowName, &ExportPayload{ExportID: exportID}); err != nil {
		return "", fmt.Errorf("start training export workflow: %w", err)
	}

	return workflowID, nil
}

package extractionshadowjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

var _ services.ExtractionShadowStarter = (*Starter)(nil)

type StarterParams struct {
	fx.In

	Client client.Client
}

type Starter struct {
	client client.Client
}

func NewStarter(p StarterParams) *Starter {
	return &Starter{client: p.Client}
}

func (s *Starter) StartExtractionShadow(
	ctx context.Context,
	start *services.ExtractionShadowStart,
) (string, error) {
	workflowID := extractionshadow.WorkflowID(start.ResultID)
	_, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       workflowID,
		TaskQueue:                                temporaltype.DocumentIntelligenceTaskQueue,
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		Priority:                                 shadowPriority(start.TenantInfo.OrgID),
		StaticSummary:                            "Shadow extraction " + start.ResultID.String(),
	}, ExtractionShadowWorkflowName, &ShadowPayload{
		OrganizationID: start.TenantInfo.OrgID,
		BusinessUnitID: start.TenantInfo.BuID,
		ResultID:       start.ResultID,
	})
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return workflowID, nil
		}

		return "", fmt.Errorf("start extraction shadow workflow: %w", err)
	}

	return workflowID, nil
}

package portalcontract

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const WorkflowName = "PortalRunWorkflow"

func WorkflowIDFor(runID pulid.ID) string { return "agent-portal-run:" + runID.String() }

type Payload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
}

func (p *Payload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type PollResult struct {
	Finished     bool `json:"finished"`
	AwaitingCode bool `json:"awaitingCode"`
}

type StopInput struct {
	Payload *Payload                       `json:"payload"`
	Status  agentextension.PortalRunStatus `json:"status"`
	Reason  string                         `json:"reason"`
}

type StartParams struct {
	Workflows kernelports.WorkflowStarter
	Tenant    pagination.TenantInfo
	RunID     pulid.ID
	SiteName  string
	Now       int64
}

// Start starts the workflow that follows a portal action to its end. A run
// already followed is not followed twice: an approval retried finds the
// workflow running and leaves it.
func Start(ctx context.Context, p *StartParams) error {
	_, err := p.Workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    WorkflowIDFor(p.RunID),
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		StaticSummary:         "Portal task on " + p.SiteName,
	}, WorkflowName, &Payload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: p.Tenant.OrgID,
			BusinessUnitID: p.Tenant.BuID,
			UserID:         p.Tenant.UserID,
			Timestamp:      p.Now,
		},
		RunID: p.RunID,
	})
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return nil
	}

	return err
}

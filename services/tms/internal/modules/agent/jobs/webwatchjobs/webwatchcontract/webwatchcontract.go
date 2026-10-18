package webwatchcontract

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	WorkflowName = "WebWatchWorkflow"

	// SignalCheck asks the workflow to check the watch now rather than at
	// its next look: the vendor reported a run, or the watch was stopped.
	SignalCheck = "check"
)

func WorkflowIDFor(watchID pulid.ID) string { return "agent-web-watch:" + watchID.String() }

type Payload struct {
	temporaltype.BasePayload

	WatchID pulid.ID `json:"watchId"`
}

func (p *Payload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type CheckResult struct {
	Ended bool `json:"ended"`
}

type StartParams struct {
	Workflows kernelports.WorkflowStarter
	Tenant    pagination.TenantInfo
	WatchID   pulid.ID
	Summary   string
	Now       int64
}

// Start starts the workflow that follows a watch until it ends. A watch
// already followed is not followed twice.
func Start(ctx context.Context, p *StartParams) error {
	_, err := p.Workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    WorkflowIDFor(p.WatchID),
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		StaticSummary:         "Web watch: " + p.Summary,
	}, WorkflowName, &Payload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: p.Tenant.OrgID,
			BusinessUnitID: p.Tenant.BuID,
			UserID:         p.Tenant.UserID,
			Timestamp:      p.Now,
		},
		WatchID: p.WatchID,
	})
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &started) {
		return nil
	}

	return err
}

// Check asks the workflow following a watch to look now. One that has
// already ended is not an error: there is nothing left to check.
func Check(ctx context.Context, workflows kernelports.WorkflowStarter, watchID pulid.ID) error {
	err := workflows.SignalWorkflow(ctx, WorkflowIDFor(watchID), "", SignalCheck, nil)
	var missing *serviceerror.NotFound
	if errors.As(err, &missing) {
		return nil
	}

	return err
}

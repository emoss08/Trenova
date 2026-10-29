package accountingsynccontract

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type DriftCheckerParams struct {
	fx.In

	Workflows kernelports.WorkflowStarter
}

type DriftChecker struct {
	workflows kernelports.WorkflowStarter
}

var _ accountingdriftports.AccountingDriftChecker = (*DriftChecker)(nil)

func NewDriftChecker(p DriftCheckerParams) *DriftChecker {
	return &DriftChecker{workflows: p.Workflows}
}

func (c *DriftChecker) CheckNow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := c.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       DriftWorkflowID(connectionID),
		TaskQueue:                temporaltype.IntegrationTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StaticSummary: "Compare the accounting system with Trenova for connection " +
			connectionID.String(),
	}, ReconcileAccountingDriftWorkflowName, &DriftPayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ConnectionID:   connectionID,
	})
	if errors.Is(err, kernelports.ErrWorkflowStarterDisabled) {
		return errortypes.NewBusinessError(
			"Background work is not available on this server, so the books cannot be checked now",
		).WithInternal(err)
	}
	return err
}

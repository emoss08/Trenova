package accountingsyncjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type RefresherParams struct {
	fx.In

	Workflows services.WorkflowStarter
}

type ReferenceRefresher struct {
	workflows services.WorkflowStarter
}

var _ services.AccountingReferenceRefresher = (*ReferenceRefresher)(nil)

func NewReferenceRefresher(p RefresherParams) *ReferenceRefresher {
	return &ReferenceRefresher{workflows: p.Workflows}
}

func (r *ReferenceRefresher) RequestReferenceRefresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := r.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       ReferenceWorkflowID(connectionID),
		TaskQueue:                temporaltype.IntegrationTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}, RefreshAccountingReferenceWorkflowName, &RefreshReferencePayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ConnectionID:   connectionID,
	})
	if errors.Is(err, services.ErrWorkflowStarterDisabled) {
		return errortypes.NewBusinessError(
			"Background work is not available on this server, so the reference data cannot be refreshed",
		).WithInternal(err)
	}
	return err
}

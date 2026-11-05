package scimcontract

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ReconcileWorkflowName = "ReconcileSCIMDirectoryGrantsWorkflow"
	ReconcileSignalName   = "scim-directory-reconcile"
	workflowIDPrefix      = "scim-directory-reconcile-"
)

type ReconcilePayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	DirectoryID    pulid.ID `json:"directoryId"`
	Iterations     int      `json:"iterations"`
}

func (p *ReconcilePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type ReconcileSignal struct{}

func ReconcileWorkflowID(directoryID pulid.ID) string {
	return workflowIDPrefix + directoryID.String()
}

type RequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

type Requester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ scimports.ReconcileRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) scimports.ReconcileRequester {
	return &Requester{signals: p.Signals, l: p.Logger.Named("scim-reconcile-requester")}
}

func (r *Requester) RequestReconcile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	directoryID pulid.ID,
) {
	if directoryID.IsNil() || tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		_, err := r.signals.SignalWithStartWorkflow(
			committed,
			ReconcileWorkflowID(directoryID),
			ReconcileSignalName,
			ReconcileSignal{},
			client.StartWorkflowOptions{
				ID:            ReconcileWorkflowID(directoryID),
				TaskQueue:     temporaltype.TaskQueueSystem.String(),
				StaticSummary: "Bring the roles a SCIM directory's groups grant up to date",
				Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
			},
			ReconcileWorkflowName,
			&ReconcilePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				DirectoryID:    directoryID,
			},
		)
		if err != nil {
			r.l.Warn("could not request a SCIM directory reconcile",
				zap.String("directoryId", directoryID.String()),
				zap.Error(err))
		}
	})
}

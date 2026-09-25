package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var _ serviceports.ToolPreviewer = (*resolveBankReceiptWorkItemTool)(nil)

var closedWorkItemFields = []string{"status", "resolutionType", "resolutionNote", "resolvedAt"}

func (a resolveWorkItemArgs) close(
	item *bankreceiptworkitem.WorkItem,
	tenant pagination.TenantInfo,
	userID pulid.ID,
	now int64,
) error {
	if a.dismisses() {
		return bankreceiptworkitemservice.DismissWorkItem(item, a.dismissal(tenant), userID, now)
	}

	return bankreceiptworkitemservice.ResolveWorkItem(item, a.resolved(tenant), userID, now)
}

func (t *resolveBankReceiptWorkItemTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	args, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	tenant := tenantFrom(params)
	item, err := t.items.Get(ctx, &serviceports.GetBankReceiptWorkItemRequest{
		WorkItemID: args.id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	plan, err := planArchive(toolpreview.Record{
		Resource: permission.ResourceBankReceiptWorkItem,
		ID:       item.ID,
		Label:    "Bank receipt reconciliation",
		Version:  previewVersion(item.Version),
	}, item, func(closed *bankreceiptworkitem.WorkItem) error {
		return args.close(closed, tenant, params.Actor.UserID, now)
	}, toolpreview.Only(closedWorkItemFields...), toolpreview.Volatile("resolvedAt"))
	if err != nil {
		return nil, err
	}

	return plan.preview(fmt.Sprintf(
		"Would close this bank receipt's reconciliation work item as %s without matching it; "+
			"no money moves.",
		args.resolution,
	)), nil
}

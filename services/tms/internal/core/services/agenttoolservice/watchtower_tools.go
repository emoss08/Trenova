package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramWatchtowerItemID = "watchtowerItemId"

type watchtowerDismisser interface {
	PlanDismiss(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
		actor *serviceports.RequestActor,
	) (*serviceports.RecordChange[watchtower.Item], error)
	Dismiss(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
		actor *serviceports.RequestActor,
	) (*watchtower.Item, error)
}

type watchtowerItemView struct {
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	Open       bool   `json:"open"`
	ResolvedAt *int64 `json:"resolvedAt,omitempty"`
}

func watchtowerItemViewOf(item *watchtower.Item) *watchtowerItemView {
	return &watchtowerItemView{
		Kind:       item.SourceKind.Label(),
		Severity:   string(item.Severity),
		Open:       !item.IsResolved(),
		ResolvedAt: item.ResolvedAt,
	}
}

func newDismissWatchtowerItemTool(items watchtowerDismisser) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "dismiss_watchtower_item",
		description: "Take an item off the watchtower feed once it has been dealt with; the " +
			"record behind it is left alone. Use it for a weather alert that has passed or a " +
			"message already answered. Items about agents' own runs, proposals and " +
			"exceptions stay for a person to clear.",
		resource:    permission.ResourceWatchtower,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		rationale: "Takes an item off the shared attention feed inside Trenova; nothing is " +
			"sent and the record behind it is unchanged.",
		properties: map[string]any{
			paramWatchtowerItemID: stringProperty("The item, from list_watchtower_items. "+
				"Never guess one.", 0),
		},
		required:    []string{paramWatchtowerItemID},
		searchTerms: []string{"dismiss watchtower", "clear feed item", "mark handled"},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramWatchtowerItemID, permission.ResourceWatchtower)
		},
	}, receivablePlan[pulid.ID, *serviceports.RecordChange[watchtower.Item]]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramWatchtowerItemID)
		},
		settle: func(ctx context.Context, id pulid.ID, params *serviceports.ToolExecuteParams) error {
			_, err := plannedDismissal(ctx, items, id, params)

			return err
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.RecordChange[watchtower.Item], error) {
			return plannedDismissal(ctx, items, id, params)
		},
		refused: func(pulid.ID) string { return "Would take an item off the watchtower feed." },
		render: func(
			_ pulid.ID,
			change *serviceports.RecordChange[watchtower.Item],
		) (*agent.ToolPreview, error) {
			built, err := toolpreview.Changed(toolpreview.Record{
				Resource: permission.ResourceWatchtower,
				ID:       change.Before.ID,
				Label:    change.Before.Title,
				Version:  pinnedVersion(change.Before.Version),
			}, watchtowerItemViewOf(change.Before), watchtowerItemViewOf(change.After),
				toolpreview.Volatile("resolvedAt"))
			if err != nil {
				return nil, err
			}
			built.Operation = agent.PreviewOperationArchive

			return toolpreview.Build(fmt.Sprintf(
				"Would take %q off the watchtower feed; the record behind it is unchanged.",
				change.Before.Title,
			), built), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			dismissed, err := items.Dismiss(ctx, tenantFrom(*params), id, params.Actor)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "dismissed",
				Kind:   "watchtower item",
				Name:   dismissed.Title,
				IDs:    map[string]string{paramWatchtowerItemID: id.String()},
			}, nil
		},
	})
}

func plannedDismissal(
	ctx context.Context,
	items watchtowerDismisser,
	id pulid.ID,
	params *serviceports.ToolExecuteParams,
) (*serviceports.RecordChange[watchtower.Item], error) {
	change, err := items.PlanDismiss(ctx, tenantFrom(*params), id, params.Actor)
	if err != nil {
		return nil, err
	}
	switch {
	case change.Before.SourceKind.OverseesAgents():
		return nil, errortypes.NewValidationError(paramWatchtowerItemID,
			errortypes.ErrInvalid, "An item about an agent's own work stays on the feed for "+
				"a person to clear")
	case change.Before.IsResolved():
		return nil, errortypes.NewValidationError(paramWatchtowerItemID,
			errortypes.ErrInvalid, "This item is already off the feed")
	}

	return change, nil
}

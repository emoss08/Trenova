package exceptiontools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramWatchtowerItemID = "watchtowerItemId"

type WatchtowerDismisser interface {
	PlanDismiss(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
		actor *permissionports.RequestActor,
	) (*kernelports.RecordChange[watchtower.Item], error)
	Dismiss(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
		actor *permissionports.RequestActor,
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

func NewDismissWatchtowerItemTool(items WatchtowerDismisser) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:   "exception_dismiss_watchtower_item",
		Recipe: []string{toolListWatchtowerItems, "exception_dismiss_watchtower_item"},
		Description: "Take an item off the watchtower feed once it has been dealt with; the " +
			"record behind it is left alone. Use it for a weather alert that has passed or a " +
			"message already answered. Items about agents' own runs, proposals and " +
			"exceptions stay for a person to clear.",
		Resource:    permission.ResourceWatchtower,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale: "Takes an item off the shared attention feed inside Trenova; nothing is " +
			"sent and the record behind it is unchanged.",
		Properties: map[string]any{
			paramWatchtowerItemID: agenttoolschema.OfKinds(
				toolparams.StringProperty("The item, from "+
					"exception_list_watchtower_items. Never guess one.", 0),
				permission.KindWatchtowerItem,
			),
		},
		Required:    []string{paramWatchtowerItemID},
		SearchTerms: []string{"dismiss watchtower", "clear feed item", "mark handled"},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramWatchtowerItemID, permission.ResourceWatchtower)
		},
	}, toolspec.ReceivablePlan[pulid.ID, *kernelports.RecordChange[watchtower.Item]]{
		Request: func(params *agenttoolports.ToolExecuteParams) (pulid.ID, error) {
			return toolparams.RequirePulid(params.Params, paramWatchtowerItemID)
		},
		Settle: func(ctx context.Context, id pulid.ID, params *agenttoolports.ToolExecuteParams) error {
			_, err := plannedDismissal(ctx, items, id, params)

			return err
		},
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*kernelports.RecordChange[watchtower.Item], error) {
			return plannedDismissal(ctx, items, id, params)
		},
		Refused: func(pulid.ID) string { return "Would take an item off the watchtower feed." },
		Render: func(
			_ pulid.ID,
			change *kernelports.RecordChange[watchtower.Item],
		) (*agentpreview.ToolPreview, error) {
			built, err := toolpreview.Changed(toolpreview.Record{
				Resource: permission.ResourceWatchtower,
				ID:       change.Before.ID,
				Label:    change.Before.Title,
				Version:  changepreview.PinnedVersion(change.Before.Version),
			}, watchtowerItemViewOf(change.Before), watchtowerItemViewOf(change.After),
				toolpreview.Volatile("resolvedAt"))
			if err != nil {
				return nil, err
			}
			built.Operation = agentpreview.PreviewOperationArchive

			return toolpreview.Build(fmt.Sprintf(
				"Would take %q off the watchtower feed; the record behind it is unchanged.",
				change.Before.Title,
			), built), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			dismissed, err := items.Dismiss(ctx, toolparams.TenantFrom(*params), id, params.Actor)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
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
	items WatchtowerDismisser,
	id pulid.ID,
	params *agenttoolports.ToolExecuteParams,
) (*kernelports.RecordChange[watchtower.Item], error) {
	change, err := items.PlanDismiss(ctx, toolparams.TenantFrom(*params), id, params.Actor)
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

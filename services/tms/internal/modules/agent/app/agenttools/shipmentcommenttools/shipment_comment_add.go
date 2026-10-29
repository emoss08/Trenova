package shipmentcommenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/shared/timeutils"
)

type AddShipmentCommentTool struct {
	comments shipmentcommentports.ShipmentCommentService
}

func NewAddShipmentCommentTool(
	comments shipmentcommentports.ShipmentCommentService,
) agenttoolports.AgentTool {
	return &AddShipmentCommentTool{comments: comments}
}

func (t *AddShipmentCommentTool) Preview(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	entity, err := t.comment(params)
	if err != nil {
		return nil, err
	}

	created, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipmentComment, Label: "Shipment comment"},
		entity,
		toolpreview.Only(
			changepreview.PreviewFieldShipmentID,
			"comment",
			"type",
			"visibility",
			"priority",
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.PreviewFieldShipmentID: permission.ResourceShipment,
		}),
	)
	if err != nil {
		return nil, err
	}

	shared := toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       entity.ShipmentID,
			Label:    commentAudience(entity.Visibility),
		},
		&agentpreview.MessagePreview{
			Channel:    agentpreview.MessageChannelComment,
			To:         []string{commentAudience(entity.Visibility)},
			Body:       entity.Comment,
			Visibility: string(entity.Visibility),
		},
	)

	preview := toolpreview.Build(
		fmt.Sprintf("Would leave a %s note on the shipment, read by %s.",
			strings.ToLower(string(entity.Priority)),
			strings.ToLower(commentAudience(entity.Visibility))),
		created,
		shared,
	)
	if entity.Visibility == shipmentvocab.CommentVisibilityCustomer ||
		entity.Visibility == shipmentvocab.CommentVisibilityDriver {
		outboundpreview.WarnSensitiveContent(preview, entity.Comment)
	}

	return preview, nil
}

func commentAudience(visibility shipmentvocab.CommentVisibility) string {
	switch visibility {
	case shipmentvocab.CommentVisibilityCustomer:
		return "The customer"
	case shipmentvocab.CommentVisibilityDriver:
		return "The driver"
	case shipmentvocab.CommentVisibilityAccounting:
		return "Accounting"
	case shipmentvocab.CommentVisibilityOperations:
		return "Operations"
	case shipmentvocab.CommentVisibilityInternal:
		return "Internal staff"
	default:
		return "Internal staff"
	}
}

func (t *AddShipmentCommentTool) Name() string { return "shipment_add_comment" }

func (t *AddShipmentCommentTool) Description() string {
	return "Leave a note on a shipment's comment thread, where dispatch will read it. " +
		"Use it to record what you found or what you did — a carrier's confirmation, a " +
		"discrepancy, why something was held. The note is internal unless you say " +
		"otherwise, and is marked as written by the assistant."
}

func (t *AddShipmentCommentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to comment on, from the page or "+
					"shipment_list."),
			"comment": map[string]any{
				"type":        "string",
				"description": "What to say, in the words a dispatcher would use.",
			},
			changepreview.FieldVisibility: agenttoolschema.Enum(
				"Who sees it. Defaults to Internal; only widen it when the "+
					"person asked for the note to reach a customer or a driver.",
				agenttoolschema.CommentVisibilities,
			),
			"priority": agenttoolschema.Enum(
				"How urgently dispatch should read it. Defaults to Normal; use "+
					"High or Urgent only for something that changes what happens to the "+
					"load today.",
				agenttoolschema.CommentPriorities,
			),
		},
		"required":             []string{toolparams.ParamShipmentID, "comment"},
		"additionalProperties": false,
	}
}

func (t *AddShipmentCommentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:        t.Name(),
		Kind:        agenttooltrust.ToolKindAction,
		Resource:    permission.ResourceShipmentComment,
		Operation:   permission.OpCreate,
		Scope:       agenttooltrust.ToolScopeTenant,
		DefaultTier: agent.TierAutoExecute,
		MaxTier:     agent.TierAutoExecute,
		Egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		Classify:      shipmentpreview.ClassifyCommentVisibility,
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		CarriesTaint:  true,
		Rationale: "An internal note stays inside the organization; a customer or driver " +
			"note is read outside it, so its visibility argument decides. A note written " +
			"on its own after the run read outside text is marked as drawn from it.",
	}
}

func (t *AddShipmentCommentTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	entity, err := t.comment(params)
	if err != nil {
		return err
	}

	_, err = t.comments.Create(ctx, entity, params.Actor)

	return err
}

// comment is the note the preview shows and the write creates.
func (t *AddShipmentCommentTool) comment(
	params *agenttoolports.ToolExecuteParams,
) (*shipment.ShipmentComment, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	body, err := toolparams.RequireString(params.Params, "comment")
	if err != nil {
		return nil, err
	}

	// Widening a note's audience on a typo is the one mistake here that
	// reaches a customer: a visibility the domain does not have is refused,
	// never read as one it does.
	visibility, given, err := toolparams.OptionalEnum(
		params.Params, "visibility", agenttoolschema.CommentVisibilities.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		visibility = shipmentvocab.CommentVisibilityInternal
	}

	priority, given, err := toolparams.OptionalEnum(
		params.Params, "priority", agenttoolschema.CommentPriorities.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		priority = shipmentvocab.CommentPriorityNormal
	}

	// Source is the assistant's, always. A note that reads as hand-typed but
	// was not carries a colleague's authority without a colleague's check.
	return &shipment.ShipmentComment{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		ShipmentID:     shipmentID,
		UserID:         params.Actor.UserID,
		Comment:        body,
		Type:           commentType(visibility),
		Visibility:     visibility,
		Priority:       priority,
		Source:         shipmentvocab.CommentSourceAI,
		Metadata:       commentTaintMetadata(params),
	}, nil
}

// Validate builds the note the preview builds, so a call the comment
// service would refuse is refused to the model before it runs.
func (t *AddShipmentCommentTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	return toolspec.PreviewValidates(ctx, t, params)
}

func commentTaintMetadata(params *agenttoolports.ToolExecuteParams) map[string]any {
	if params.ApprovedFromProposal() || !params.CarriedTaint(timeutils.NowUnix()).Tainted() {
		return nil
	}

	return map[string]any{shipment.CommentMetadataTainted: true}
}

// commentType follows the audience. Type and visibility are separate axes in the
// domain, and a note filed as Internal but shown to the customer contradicts
// itself — anyone later filtering the thread by type would miss it.
func commentType(visibility shipmentvocab.CommentVisibility) shipmentvocab.CommentType {
	switch visibility {
	case shipmentvocab.CommentVisibilityCustomer:
		return shipmentvocab.CommentTypeCustomerUpdate
	case shipmentvocab.CommentVisibilityDriver:
		return shipmentvocab.CommentTypeDriverUpdate
	case shipmentvocab.CommentVisibilityAccounting:
		return shipmentvocab.CommentTypeBilling
	case shipmentvocab.CommentVisibilityOperations:
		return shipmentvocab.CommentTypeDispatch
	case shipmentvocab.CommentVisibilityInternal:
		return shipmentvocab.CommentTypeInternal
	default:
		return shipmentvocab.CommentTypeInternal
	}
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *AddShipmentCommentTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

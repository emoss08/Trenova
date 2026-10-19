package shipmenttools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/timeutils"
)

type AddShipmentCommentTool struct {
	comments shipmentports.ShipmentCommentService
}

func NewAddShipmentCommentTool(
	comments shipmentports.ShipmentCommentService,
) agentports.AgentTool {
	return &AddShipmentCommentTool{comments: comments}
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
			common.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to comment on, from the page, shipment_list or "+
					"shipment_list."),
			"comment": map[string]any{
				"type":        "string",
				"description": "What to say, in the words a dispatcher would use.",
			},
			common.FieldVisibility: agenttoolschema.Enum(
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
		"required":             []string{common.ParamShipmentID, "comment"},
		"additionalProperties": false,
	}
}

func (t *AddShipmentCommentTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:        t.Name(),
		Kind:        agent.ToolKindAction,
		Resource:    permission.ResourceShipmentComment,
		Operation:   permission.OpCreate,
		Scope:       agent.ToolScopeTenant,
		DefaultTier: agent.TierAutoExecute,
		MaxTier:     agent.TierAutoExecute,
		Egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
			agent.EgressDriverVisible,
		},
		Classify:      common.ClassifyCommentVisibility,
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		CarriesTaint:  true,
		Rationale: "An internal note stays inside the organization; a customer or driver " +
			"note is read outside it, so its visibility argument decides. A note written " +
			"on its own after the run read outside text is marked as drawn from it.",
	}
}

func (t *AddShipmentCommentTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	entity, err := t.comment(&params)
	if err != nil {
		return err
	}

	_, err = t.comments.Create(ctx, entity, params.Actor)

	return err
}

// comment is the note the preview shows and the write creates.
func (t *AddShipmentCommentTool) comment(
	params *agentports.ToolExecuteParams,
) (*shipment.ShipmentComment, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	body, err := common.RequireString(params.Params, "comment")
	if err != nil {
		return nil, err
	}

	// Widening a note's audience on a typo is the one mistake here that
	// reaches a customer: a visibility the domain does not have is refused,
	// never read as one it does.
	visibility, given, err := common.OptionalEnum(
		params.Params, "visibility", agenttoolschema.CommentVisibilities.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		visibility = shipment.CommentVisibilityInternal
	}

	priority, given, err := common.OptionalEnum(
		params.Params, "priority", agenttoolschema.CommentPriorities.Values,
	)
	if err != nil {
		return nil, err
	}
	if !given {
		priority = shipment.CommentPriorityNormal
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
		Source:         shipment.CommentSourceAI,
		Metadata:       commentTaintMetadata(*params),
	}, nil
}

// Validate builds the note the preview builds, so a call the comment
// service would refuse is refused to the model before it runs.
func (t *AddShipmentCommentTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return common.PreviewValidates(ctx, t, &params)
}

func commentTaintMetadata(params agentports.ToolExecuteParams) map[string]any {
	if params.ApprovedFromProposal() || !params.CarriedTaint(timeutils.NowUnix()).Tainted() {
		return nil
	}

	return map[string]any{shipment.CommentMetadataTainted: true}
}

// commentType follows the audience. Type and visibility are separate axes in the
// domain, and a note filed as Internal but shown to the customer contradicts
// itself — anyone later filtering the thread by type would miss it.
func commentType(visibility shipment.CommentVisibility) shipment.CommentType {
	switch visibility {
	case shipment.CommentVisibilityCustomer:
		return shipment.CommentTypeCustomerUpdate
	case shipment.CommentVisibilityDriver:
		return shipment.CommentTypeDriverUpdate
	case shipment.CommentVisibilityAccounting:
		return shipment.CommentTypeBilling
	case shipment.CommentVisibilityOperations:
		return shipment.CommentTypeDispatch
	case shipment.CommentVisibilityInternal:
		return shipment.CommentTypeInternal
	default:
		return shipment.CommentTypeInternal
	}
}

type PlaceShipmentHoldTool struct {
	holds shipmentports.ShipmentHoldService
}

func NewPlaceShipmentHoldTool(holds shipmentports.ShipmentHoldService) agentports.AgentTool {
	return &PlaceShipmentHoldTool{holds: holds}
}

func (t *PlaceShipmentHoldTool) Name() string { return "shipment_place_hold" }

func (t *PlaceShipmentHoldTool) Recipe() []string {
	return []string{toolShipmentList, "masterdata_list_hold_reasons", "shipment_place_hold"}
}

func (t *PlaceShipmentHoldTool) Description() string {
	return "Put a shipment on hold so it stops moving, billing or delivering until " +
		"someone clears it. The reason decides what the hold blocks, so call " +
		"masterdata_list_hold_reasons first and use a real reason id rather than describing one."
}

func (t *PlaceShipmentHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			common.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to hold, from the page, shipment_list."),
			"holdReasonId": agenttoolschema.RecordIDText(permission.ResourceHoldReason,
				"A hold reason id from masterdata_list_hold_reasons."),
			common.ParamNote: map[string]any{
				"type": "string",
				"description": "What prompted the hold, for whoever releases it. " +
					"Say what would have to be true to clear it.",
			},
		},
		"required":             []string{common.ParamShipmentID, "holdReasonId"},
		"additionalProperties": false,
	}
}

func (t *PlaceShipmentHoldTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentHold,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Places a hold on a shipment inside Trenova; no customer or EDI notice is " +
			"sent.",
	}
}

func (t *PlaceShipmentHoldTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.holds.Create(ctx, request, params.Actor)

	return err
}

func (t *PlaceShipmentHoldTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.CreateShipmentHoldRequest, error) {
	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	reasonID, err := common.RequirePulid(params.Params, "holdReasonId")
	if err != nil {
		return nil, err
	}

	// The blocking flags and severity are left unset on purpose: they default
	// from the hold reason, which is policy the organization already decided.
	// An agent choosing them would be quietly overriding that.
	return &shipmentports.CreateShipmentHoldRequest{
		TenantInfo:   common.TenantFrom(*params),
		ShipmentID:   shipmentID,
		HoldReasonID: reasonID,
		Notes:        common.OptionalString(params.Params, common.ParamNote),
	}, nil
}

type ReleaseShipmentHoldTool struct {
	holds shipmentports.ShipmentHoldService
}

func NewReleaseShipmentHoldTool(holds shipmentports.ShipmentHoldService) agentports.AgentTool {
	return &ReleaseShipmentHoldTool{holds: holds}
}

func (t *ReleaseShipmentHoldTool) Name() string { return "shipment_release_hold" }

func (t *ReleaseShipmentHoldTool) Description() string {
	return "Clear a hold so the shipment can move again. Only release a hold whose " +
		"reason you have confirmed is resolved — releasing one that still applies is " +
		"how a shipment leaves without the paperwork it was waiting on."
}

func (t *ReleaseShipmentHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			common.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment the hold is on, from the page, shipment_list or "+
					"shipment_list."),
			"holdId": agenttoolschema.RecordIDText(permission.ResourceShipmentHold,
				"The hold to release, from the holds shipment_get lists, the page or the "+
					"event that started this run. Never guess one."),
		},
		"required":             []string{common.ParamShipmentID, "holdId"},
		"additionalProperties": false,
	}
}

func (t *ReleaseShipmentHoldTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipmentHold,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Releases a hold on a shipment inside Trenova; no customer or EDI notice " +
			"is sent.",
	}
}

func (t *ReleaseShipmentHoldTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.holds.Release(ctx, request, params.Actor)

	return err
}

func (t *ReleaseShipmentHoldTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.ReleaseShipmentHoldRequest, error) {
	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	holdID, err := common.RequirePulid(params.Params, "holdId")
	if err != nil {
		return nil, err
	}

	return &shipmentports.ReleaseShipmentHoldRequest{
		TenantInfo: common.TenantFrom(*params),
		ShipmentID: shipmentID,
		HoldID:     holdID,
	}, nil
}

type CancelShipmentTool struct {
	shipments shipmentports.ShipmentService
	partners  common.ShipmentPartnerNotices
	tenders   liveTenderReader
}

func NewCancelShipmentTool(
	shipments shipmentports.ShipmentService,
	partners common.ShipmentPartnerNotices,
	tenders liveTenderReader,
) agentports.AgentTool {
	return &CancelShipmentTool{shipments: shipments, partners: partners, tenders: tenders}
}

func (t *CancelShipmentTool) Name() string { return "shipment_cancel" }

func (t *CancelShipmentTool) Description() string {
	return "Cancel a shipment the person says is off. It releases its assignments, withdraws " +
		"live tenders, sends the reason to a linked trading partner over EDI and stops it " +
		"being billed. shipment_uncancel reopens it if the freight comes back. Never cancel " +
		"a record that merely looks wrong; correct it."
}

func (t *CancelShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			common.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to cancel, from the page, shipment_list."),
			common.ParamReason: map[string]any{
				"type": "string",
				"description": "Why it is being cancelled, in the customer's or " +
					"dispatcher's own terms. This is kept on the record.",
			},
		},
		"required":             []string{common.ParamShipmentID, common.ParamReason},
		"additionalProperties": false,
	}
}

func (t *CancelShipmentTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpCancel,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Cancelling withdraws live tenders from carriers and sends the model's " +
			"cancel reason to a linked partner over EDI.",
	}
}

func (t *CancelShipmentTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.shipments.Cancel(ctx, request, params.Actor)

	return err
}

func (t *CancelShipmentTool) request(
	params *agentports.ToolExecuteParams,
) (*shipmentports.CancelShipmentRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	reason, err := common.RequireString(params.Params, common.ParamReason)
	if err != nil {
		return nil, err
	}

	return &shipmentports.CancelShipmentRequest{
		TenantInfo:   common.TenantFrom(*params),
		ShipmentID:   shipmentID,
		CanceledByID: params.Actor.UserID,
		CanceledAt:   timeutils.NowUnix(),
		CancelReason: reason,
	}, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *AddShipmentCommentTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamShipmentID, permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *PlaceShipmentHoldTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamShipmentID, permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *ReleaseShipmentHoldTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamShipmentID, permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *CancelShipmentTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamShipmentID, permission.ResourceShipment)
}

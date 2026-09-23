package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
)

/*
The dispatch actions an assistant is asked to take.

Every one goes through its service rather than a repository, so the rules that
govern a person doing the same thing govern the agent: a hold's blocking
behaviour comes from its reason, a cancellation runs the same side effects, and
a comment lands in the same thread a dispatcher reads.

Each authorizes against the thing it writes rather than the shipment it hangs
off. Granting an agent shipment:update should not quietly grant it the power to
cancel one.
*/

// --- add_shipment_comment ---------------------------------------------------

type addShipmentCommentTool struct {
	comments serviceports.ShipmentCommentService
}

func newAddShipmentCommentTool(
	comments serviceports.ShipmentCommentService,
) serviceports.AgentTool {
	return &addShipmentCommentTool{comments: comments}
}

func (t *addShipmentCommentTool) Name() string { return "add_shipment_comment" }

func (t *addShipmentCommentTool) Description() string {
	return "Leave a note on a shipment's comment thread, where dispatch will read it. " +
		"Use it to record what you found or what you did — a carrier's confirmation, a " +
		"discrepancy, why something was held. The note is internal unless you say " +
		"otherwise, and is marked as written by the assistant."
}

func (t *addShipmentCommentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment to comment on, from the page, list_shipments or " +
					"search_shipments.",
			},
			"comment": map[string]any{
				"type":        "string",
				"description": "What to say, in the words a dispatcher would use.",
			},
			"visibility": map[string]any{
				"type": "string",
				"enum": []string{"Internal", "Operations", "Customer", "Driver", "Accounting"},
				"description": "Who sees it. Defaults to Internal; only widen it when the " +
					"person asked for the note to reach a customer or a driver.",
			},
			"priority": map[string]any{
				"type": "string",
				"enum": []string{"Low", "Normal", "High"},
				"description": "How urgently dispatch should read it. Defaults to Normal; use " +
					"High only for something that changes what happens to the load today.",
			},
		},
		"required":             []string{"shipmentId", "comment"},
		"additionalProperties": false,
	}
}

func (t *addShipmentCommentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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
		Classify:      classifyCommentVisibility,
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "An internal note stays inside the organization; a customer or driver " +
			"note is read outside it, so its visibility argument decides.",
	}
}

func (t *addShipmentCommentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	body, err := requireString(params.Params, "comment")
	if err != nil {
		return err
	}

	visibility := commentVisibility(optionalString(params.Params, "visibility"))

	// Source is the assistant's, always. A note that reads as hand-typed but
	// was not carries a colleague's authority without a colleague's check.
	entity := &shipment.ShipmentComment{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		ShipmentID:     shipmentID,
		UserID:         params.Actor.UserID,
		Comment:        body,
		Type:           commentType(visibility),
		Visibility:     visibility,
		Priority:       commentPriority(optionalString(params.Params, "priority")),
		Source:         shipment.CommentSourceAI,
	}

	_, err = t.comments.Create(ctx, entity, params.Actor)

	return err
}

// commentVisibility defaults to Internal for anything it does not recognise.
// Widening a note's audience on a typo is the one mistake here that reaches a
// customer.
func commentVisibility(raw string) shipment.CommentVisibility {
	switch shipment.CommentVisibility(raw) {
	case shipment.CommentVisibilityOperations:
		return shipment.CommentVisibilityOperations
	case shipment.CommentVisibilityCustomer:
		return shipment.CommentVisibilityCustomer
	case shipment.CommentVisibilityDriver:
		return shipment.CommentVisibilityDriver
	case shipment.CommentVisibilityAccounting:
		return shipment.CommentVisibilityAccounting
	case shipment.CommentVisibilityInternal:
		return shipment.CommentVisibilityInternal
	default:
		return shipment.CommentVisibilityInternal
	}
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

func commentPriority(raw string) shipment.CommentPriority {
	switch shipment.CommentPriority(raw) {
	case shipment.CommentPriorityLow:
		return shipment.CommentPriorityLow
	case shipment.CommentPriorityHigh:
		return shipment.CommentPriorityHigh
	case shipment.CommentPriorityNormal:
		return shipment.CommentPriorityNormal
	default:
		return shipment.CommentPriorityNormal
	}
}

// --- place_shipment_hold ----------------------------------------------------

type placeShipmentHoldTool struct {
	holds serviceports.ShipmentHoldService
}

func newPlaceShipmentHoldTool(holds serviceports.ShipmentHoldService) serviceports.AgentTool {
	return &placeShipmentHoldTool{holds: holds}
}

func (t *placeShipmentHoldTool) Name() string { return "place_shipment_hold" }

func (t *placeShipmentHoldTool) Description() string {
	return "Put a shipment on hold so it stops moving, billing or delivering until " +
		"someone clears it. The reason decides what the hold blocks, so call " +
		"list_hold_reasons first and use a real reason id rather than describing one."
}

func (t *placeShipmentHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment to hold, from the page, list_shipments or " +
					"search_shipments.",
			},
			"holdReasonId": map[string]any{
				"type":        "string",
				"description": "A hold reason id from list_hold_reasons.",
			},
			"notes": map[string]any{
				"type": "string",
				"description": "What prompted the hold, for whoever releases it. " +
					"Say what would have to be true to clear it.",
			},
		},
		"required":             []string{"shipmentId", "holdReasonId"},
		"additionalProperties": false,
	}
}

func (t *placeShipmentHoldTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

func (t *placeShipmentHoldTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	reasonID, err := requirePulid(params.Params, "holdReasonId")
	if err != nil {
		return err
	}

	// The blocking flags and severity are left unset on purpose: they default
	// from the hold reason, which is policy the organization already decided.
	// An agent choosing them would be quietly overriding that.
	_, err = t.holds.Create(ctx, &repositories.CreateShipmentHoldRequest{
		TenantInfo:   tenantFrom(params),
		ShipmentID:   shipmentID,
		HoldReasonID: reasonID,
		Notes:        optionalString(params.Params, "notes"),
	}, params.Actor)

	return err
}

// --- release_shipment_hold --------------------------------------------------

type releaseShipmentHoldTool struct {
	holds serviceports.ShipmentHoldService
}

func newReleaseShipmentHoldTool(holds serviceports.ShipmentHoldService) serviceports.AgentTool {
	return &releaseShipmentHoldTool{holds: holds}
}

func (t *releaseShipmentHoldTool) Name() string { return "release_shipment_hold" }

func (t *releaseShipmentHoldTool) Description() string {
	return "Clear a hold so the shipment can move again. Only release a hold whose " +
		"reason you have confirmed is resolved — releasing one that still applies is " +
		"how a shipment leaves without the paperwork it was waiting on."
}

func (t *releaseShipmentHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment the hold is on, from the page, list_shipments or " +
					"search_shipments.",
			},
			"holdId": map[string]any{
				"type": "string",
				"description": "The hold to release. No tool lists a shipment's holds, so it " +
					"comes from the page or the event that started this run; never guess one.",
			},
		},
		"required":             []string{"shipmentId", "holdId"},
		"additionalProperties": false,
	}
}

func (t *releaseShipmentHoldTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

func (t *releaseShipmentHoldTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	holdID, err := requirePulid(params.Params, "holdId")
	if err != nil {
		return err
	}

	_, err = t.holds.Release(ctx, &repositories.ReleaseShipmentHoldRequest{
		TenantInfo: tenantFrom(params),
		ShipmentID: shipmentID,
		HoldID:     holdID,
	}, params.Actor)

	return err
}

// --- cancel_shipment --------------------------------------------------------

type cancelShipmentTool struct {
	shipments serviceports.ShipmentService
}

func newCancelShipmentTool(shipments serviceports.ShipmentService) serviceports.AgentTool {
	return &cancelShipmentTool{shipments: shipments}
}

func (t *cancelShipmentTool) Name() string { return "cancel_shipment" }

func (t *cancelShipmentTool) Description() string {
	return "Cancel a shipment. This releases its assignments and stops it being billed, " +
		"and it is not an undo — a cancelled shipment is not re-opened, it is rebooked. " +
		"Only propose this when the person has said the load is dead, never to tidy up " +
		"a record that merely looks wrong."
}

func (t *cancelShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment to cancel, from the page, list_shipments or " +
					"search_shipments.",
			},
			"cancelReason": map[string]any{
				"type": "string",
				"description": "Why it is being cancelled, in the customer's or " +
					"dispatcher's own terms. This is kept on the record.",
			},
		},
		"required":             []string{"shipmentId", "cancelReason"},
		"additionalProperties": false,
	}
}

func (t *cancelShipmentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

func (t *cancelShipmentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	reason, err := requireString(params.Params, "cancelReason")
	if err != nil {
		return err
	}

	_, err = t.shipments.Cancel(ctx, &repositories.CancelShipmentRequest{
		TenantInfo:   tenantFrom(params),
		ShipmentID:   shipmentID,
		CanceledByID: params.Actor.UserID,
		CanceledAt:   timeutils.NowUnix(),
		CancelReason: reason,
	}, params.Actor)

	return err
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *addShipmentCommentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *placeShipmentHoldTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *releaseShipmentHoldTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *cancelShipmentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

package shipmenttools

import (
	"context"

	"github.com/emoss08/trenova/pkg/toolschema"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/shared/timeutils"
)

type PlaceShipmentHoldTool struct {
	holds shipmentholdports.ShipmentHoldService
}

func NewPlaceShipmentHoldTool(
	holds shipmentholdports.ShipmentHoldService,
) agenttoolports.AgentTool {
	return &PlaceShipmentHoldTool{holds: holds}
}

func (t *PlaceShipmentHoldTool) Name() string { return "shipment_place_hold" }

func (t *PlaceShipmentHoldTool) Recipe() []string {
	return []string{ToolShipmentList, "masterdata_list_hold_reasons", "shipment_place_hold"}
}

func (t *PlaceShipmentHoldTool) Description() string {
	return "Put a shipment on hold so it stops moving, billing or delivering until " +
		"someone clears it. The reason decides what the hold blocks, so call " +
		"masterdata_list_hold_reasons first and use a real reason id rather than describing one."
}

func (t *PlaceShipmentHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to hold, from the page, shipment_list."),
			"holdReasonId": agenttoolschema.RecordIDText(permission.ResourceHoldReason,
				"A hold reason id from masterdata_list_hold_reasons."),
			toolparams.ParamNote: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "What prompted the hold, for whoever releases it. " +
					"Say what would have to be true to clear it.",
			},
		},
		toolschema.KeyRequired:             []string{toolparams.ParamShipmentID, "holdReasonId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *PlaceShipmentHoldTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentHold,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Places a hold on a shipment inside Trenova; no customer or EDI notice is " +
			"sent.",
	}
}

func (t *PlaceShipmentHoldTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
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
	params *agenttoolports.ToolExecuteParams,
) (*shipmentholdports.CreateShipmentHoldRequest, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	reasonID, err := toolparams.RequirePulid(params.Params, "holdReasonId")
	if err != nil {
		return nil, err
	}

	// The blocking flags and severity are left unset on purpose: they default
	// from the hold reason, which is policy the organization already decided.
	// An agent choosing them would be quietly overriding that.
	return &shipmentholdports.CreateShipmentHoldRequest{
		TenantInfo:   toolparams.TenantFrom(params),
		ShipmentID:   shipmentID,
		HoldReasonID: reasonID,
		Notes:        toolparams.OptionalString(params.Params, toolparams.ParamNote),
	}, nil
}

type ReleaseShipmentHoldTool struct {
	holds shipmentholdports.ShipmentHoldService
}

func NewReleaseShipmentHoldTool(
	holds shipmentholdports.ShipmentHoldService,
) agenttoolports.AgentTool {
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
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment the hold is on, from the page or "+
					"shipment_list."),
			"holdId": agenttoolschema.RecordIDText(permission.ResourceShipmentHold,
				"The hold to release, from the holds shipment_get lists, the page or the "+
					"event that started this run. Never guess one."),
		},
		toolschema.KeyRequired:             []string{toolparams.ParamShipmentID, "holdId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *ReleaseShipmentHoldTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipmentHold,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Releases a hold on a shipment inside Trenova; no customer or EDI notice " +
			"is sent.",
	}
}

func (t *ReleaseShipmentHoldTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, &params); err != nil {
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
	params *agenttoolports.ToolExecuteParams,
) (*shipmentholdports.ReleaseShipmentHoldRequest, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	holdID, err := toolparams.RequirePulid(params.Params, "holdId")
	if err != nil {
		return nil, err
	}

	return &shipmentholdports.ReleaseShipmentHoldRequest{
		TenantInfo: toolparams.TenantFrom(params),
		ShipmentID: shipmentID,
		HoldID:     holdID,
	}, nil
}

type CancelShipmentTool struct {
	shipments shipmentserviceports.ShipmentLifecycleService
	partners  shipmentpreview.ShipmentPartnerNotices
	tenders   liveTenderReader
}

func NewCancelShipmentTool(
	shipments shipmentserviceports.ShipmentLifecycleService,
	partners shipmentpreview.ShipmentPartnerNotices,
	tenders liveTenderReader,
) agenttoolports.AgentTool {
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
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to cancel, from the page, shipment_list."),
			toolparams.ParamReason: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Why it is being cancelled, in the customer's or " +
					"dispatcher's own terms. This is kept on the record.",
			},
		},
		toolschema.KeyRequired: []string{
			toolparams.ParamShipmentID,
			toolparams.ParamReason,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *CancelShipmentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpCancel,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Cancelling withdraws live tenders from carriers and sends the model's " +
			"cancel reason to a linked partner over EDI.",
	}
}

func (t *CancelShipmentTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.shipments.Cancel(ctx, request, params.Actor)

	return err
}

func (t *CancelShipmentTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*shipmentports.CancelShipmentRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	reason, err := toolparams.RequireString(params.Params, toolparams.ParamReason)
	if err != nil {
		return nil, err
	}

	return &shipmentports.CancelShipmentRequest{
		TenantInfo:   toolparams.TenantFrom(params),
		ShipmentID:   shipmentID,
		CanceledByID: params.Actor.UserID,
		CanceledAt:   timeutils.NowUnix(),
		CancelReason: reason,
	}, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *PlaceShipmentHoldTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *ReleaseShipmentHoldTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *CancelShipmentTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

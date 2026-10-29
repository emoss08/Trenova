package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
)

const ParamHoldID = "holdId"

const ParamHoldSeverity = "severity"

const paramBlocksDispatch = "blocksDispatch"

const ParamBlocksDelivery = "blocksDelivery"

const ParamBlocksBilling = "blocksBilling"

const ParamVisibleToCustomer = "visibleToCustomer"

const maxHoldNoteChars = 2000

var holdSeverities = agenttoolschema.Source(
	"holdReason.severity",
	holdreason.HoldSeverityValues(),
)

var updatedHoldFields = []string{
	changepreview.FieldSeverity,
	changepreview.FieldNotes,
	paramBlocksDispatch,
	ParamBlocksDelivery,
	ParamBlocksBilling,
	ParamVisibleToCustomer,
}

type holdUpdater interface {
	GetByID(
		ctx context.Context,
		req *shipmentholdports.GetShipmentHoldByIDRequest,
	) (*shipmenthold.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *shipmentholdports.UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
	PreviewUpdate(
		ctx context.Context,
		req *shipmentholdports.UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
}

type holdPatch struct {
	get               shipmentholdports.GetShipmentHoldByIDRequest
	severity          *holdreason.HoldSeverity
	notes             *string
	blocksDispatch    *bool
	blocksDelivery    *bool
	blocksBilling     *bool
	visibleToCustomer *bool
}

type holdUpdatePlan struct {
	before *shipmenthold.ShipmentHold
	after  *shipmenthold.ShipmentHold
}

func (p *holdPatch) request(
	current *shipmenthold.ShipmentHold,
) *shipmentholdports.UpdateShipmentHoldRequest {
	req := &shipmentholdports.UpdateShipmentHoldRequest{
		TenantInfo:        p.get.TenantInfo,
		ShipmentID:        p.get.ShipmentID,
		HoldID:            p.get.HoldID,
		Severity:          current.Severity,
		Notes:             current.Notes,
		BlocksDispatch:    current.BlocksDispatch,
		BlocksDelivery:    current.BlocksDelivery,
		BlocksBilling:     current.BlocksBilling,
		VisibleToCustomer: current.VisibleToCustomer,
		StartedAt:         current.StartedAt,
		Version:           current.Version,
	}
	if p.severity != nil {
		req.Severity = *p.severity
	}
	if p.notes != nil {
		req.Notes = *p.notes
	}
	if p.blocksDispatch != nil {
		req.BlocksDispatch = *p.blocksDispatch
	}
	if p.blocksDelivery != nil {
		req.BlocksDelivery = *p.blocksDelivery
	}
	if p.blocksBilling != nil {
		req.BlocksBilling = *p.blocksBilling
	}
	if p.visibleToCustomer != nil {
		req.VisibleToCustomer = *p.visibleToCustomer
	}

	return req
}

func readHoldPatch(params *agenttoolports.ToolExecuteParams) (*holdPatch, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	holdID, err := toolparams.RequirePulid(params.Params, ParamHoldID)
	if err != nil {
		return nil, err
	}

	patch := &holdPatch{get: shipmentholdports.GetShipmentHoldByIDRequest{
		HoldID:     holdID,
		ShipmentID: shipmentID,
		TenantInfo: toolparams.TenantFrom(*params),
	}}
	severity, given, err := toolparams.OptionalEnum(
		params.Params,
		ParamHoldSeverity,
		holdSeverities.Values,
	)
	if err != nil {
		return nil, err
	}
	if given {
		patch.severity = &severity
	}
	if patch.notes, err = toolparams.OptionalBoundedText(
		params.Params,
		toolparams.ParamNote,
		maxHoldNoteChars,
	); err != nil {
		return nil, err
	}
	flags := []struct {
		key string
		out **bool
	}{
		{paramBlocksDispatch, &patch.blocksDispatch},
		{ParamBlocksDelivery, &patch.blocksDelivery},
		{ParamBlocksBilling, &patch.blocksBilling},
		{ParamVisibleToCustomer, &patch.visibleToCustomer},
	}
	for _, flag := range flags {
		if *flag.out, err = toolparams.OptionalBoolPointer(params.Params, flag.key); err != nil {
			return nil, err
		}
	}
	if patch.severity == nil && patch.notes == nil && patch.blocksDispatch == nil &&
		patch.blocksDelivery == nil && patch.blocksBilling == nil &&
		patch.visibleToCustomer == nil {
		return nil, toolparams.ErrNothingToChange
	}

	return patch, nil
}

func classifyHoldVisibility(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	if toolparams.OptionalBool(params.Params, ParamVisibleToCustomer) {
		return agenttoolports.CallPolicy{Egress: agent.EgressCustomerVisible}
	}

	return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
}

func NewUpdateShipmentHoldTool(holds holdUpdater) agenttoolports.AgentTool {
	base := toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_update_hold",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Change an active hold on a shipment: its severity, its note, what it " +
			"blocks (dispatch, delivery, billing) and whether the customer sees it. Send only " +
			"what changes; everything else stays as it is. Use shipment_release_hold to clear " +
			"a hold rather than unblocking everything. Making a hold visible to the customer " +
			"is shown to them, so only do it when the person asked.",
		Resource:    permission.ResourceShipmentHold,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Changes a hold inside Trenova; one made visible to the customer is shown " +
			"to them, so that call is held to the customer-visible ceiling.",
		Properties: holdUpdateProperties(),
		Required:   []string{toolparams.ParamShipmentID, ParamHoldID},
		Target:     TargetShipment,
	}, toolspec.ReceivablePlan[*holdPatch, *holdUpdatePlan]{
		Request: readHoldPatch,
		Plan: func(
			ctx context.Context,
			patch *holdPatch,
			params *agenttoolports.ToolExecuteParams,
		) (*holdUpdatePlan, error) {
			current, err := holds.GetByID(ctx, &patch.get)
			if err != nil {
				return nil, err
			}
			after, err := holds.PreviewUpdate(ctx, patch.request(current), params.Actor)
			if err != nil {
				return nil, err
			}

			return &holdUpdatePlan{before: current, after: after}, nil
		},
		Refused: func(*holdPatch) string { return "Would change a hold on the shipment." },
		Render:  renderHoldUpdate,
		Run: func(
			ctx context.Context,
			patch *holdPatch,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			current, err := holds.GetByID(ctx, &patch.get)
			if err != nil {
				return nil, err
			}
			saved, err := holds.Update(ctx, patch.request(current), params.Actor)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "updated",
				Kind:   "shipment hold",
				Name:   holdLabel(saved),
				IDs: map[string]string{
					ParamHoldID:                saved.ID.String(),
					toolparams.ParamShipmentID: saved.ShipmentID.String(),
				},
				Record: changepreview.RecordOf(
					changepreview.ShipmentRecordEntity,
					saved.ShipmentID,
				),
			}, nil
		},
	})

	return toolspec.ClassifiedTool[*holdPatch, *holdUpdatePlan]{
		ReportingReceivableTool: base,
		Egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
		},
		Classify: classifyHoldVisibility,
	}
}

func holdUpdateProperties() map[string]any {
	return map[string]any{
		toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty(
			"The shipment the hold is on",
		),
		ParamHoldID: agenttoolschema.RecordIDText(
			permission.ResourceShipmentHold,
			"The hold to change, from the holds shipment_get lists or "+
				"the page you are on. Never guess one.",
		),
		ParamHoldSeverity: agenttoolschema.Enum("How strongly it applies.", holdSeverities),
		toolparams.ParamNote: toolparams.StringProperty(
			"What the hold waits on, replacing the note on it.",
			maxHoldNoteChars,
		),
		paramBlocksDispatch: toolparams.BooleanProperty(
			"Whether it stops the shipment being dispatched.",
		),
		ParamBlocksDelivery: toolparams.BooleanProperty(
			"Whether it stops the shipment being delivered.",
		),
		ParamBlocksBilling: toolparams.BooleanProperty(
			"Whether it stops the shipment being billed.",
		),
		ParamVisibleToCustomer: toolparams.BooleanProperty(
			"Whether the customer sees the hold.",
		),
	}
}

func renderHoldUpdate(_ *holdPatch, plan *holdUpdatePlan) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		ID:       plan.before.ID,
		Label:    holdLabel(plan.before),
		Version:  changepreview.PreviewVersion(plan.before.Version),
	}, plan.before, plan.after, toolpreview.Only(updatedHoldFields...))
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would change the %s so that it %s.",
		holdLabel(plan.before),
		holdBlocks(plan.after),
	)
	if plan.after.VisibleToCustomer && !plan.before.VisibleToCustomer {
		summary += " The customer would see it."
	}

	return toolpreview.Build(strings.TrimSpace(summary), change), nil
}

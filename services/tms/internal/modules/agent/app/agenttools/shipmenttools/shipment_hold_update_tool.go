package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
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
	common.FieldSeverity,
	common.FieldNotes,
	paramBlocksDispatch,
	ParamBlocksDelivery,
	ParamBlocksBilling,
	ParamVisibleToCustomer,
}

type holdUpdater interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *shipmentports.UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
	PreviewUpdate(
		ctx context.Context,
		req *shipmentports.UpdateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipment.ShipmentHold, error)
}

type holdPatch struct {
	get               shipmentports.GetShipmentHoldByIDRequest
	severity          *holdreason.HoldSeverity
	notes             *string
	blocksDispatch    *bool
	blocksDelivery    *bool
	blocksBilling     *bool
	visibleToCustomer *bool
}

type holdUpdatePlan struct {
	before *shipment.ShipmentHold
	after  *shipment.ShipmentHold
}

func (p *holdPatch) request(
	current *shipment.ShipmentHold,
) *shipmentports.UpdateShipmentHoldRequest {
	req := &shipmentports.UpdateShipmentHoldRequest{
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

func readHoldPatch(params *agentports.ToolExecuteParams) (*holdPatch, error) {
	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	holdID, err := common.RequirePulid(params.Params, ParamHoldID)
	if err != nil {
		return nil, err
	}

	patch := &holdPatch{get: shipmentports.GetShipmentHoldByIDRequest{
		HoldID:     holdID,
		ShipmentID: shipmentID,
		TenantInfo: common.TenantFrom(*params),
	}}
	severity, given, err := common.OptionalEnum(
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
	if patch.notes, err = common.OptionalBoundedText(
		params.Params,
		common.ParamNote,
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
		if *flag.out, err = common.OptionalBoolPointer(params.Params, flag.key); err != nil {
			return nil, err
		}
	}
	if patch.severity == nil && patch.notes == nil && patch.blocksDispatch == nil &&
		patch.blocksDelivery == nil && patch.blocksBilling == nil &&
		patch.visibleToCustomer == nil {
		return nil, common.ErrNothingToChange
	}

	return patch, nil
}

func classifyHoldVisibility(
	params agentports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agentports.CallPolicy {
	if common.OptionalBool(params.Params, ParamVisibleToCustomer) {
		return agentports.CallPolicy{Egress: agent.EgressCustomerVisible}
	}

	return agentports.CallPolicy{Egress: agent.EgressInternal}
}

func NewUpdateShipmentHoldTool(holds holdUpdater) agentports.AgentTool {
	base := common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "shipment_update_hold",
		Artifact: common.ShipmentRecordEntity,
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
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment the hold is on"),
			ParamHoldID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentHold,
				"The hold to change, from the holds shipment_get lists or "+
					"the page you are on. Never guess one.",
			),
			ParamHoldSeverity: agenttoolschema.Enum("How strongly it applies.", holdSeverities),
			common.ParamNote: common.StringProperty(
				"What the hold waits on, replacing the note on it.",
				maxHoldNoteChars,
			),
			paramBlocksDispatch: common.BooleanProperty(
				"Whether it stops the shipment being dispatched.",
			),
			ParamBlocksDelivery: common.BooleanProperty(
				"Whether it stops the shipment being delivered.",
			),
			ParamBlocksBilling: common.BooleanProperty(
				"Whether it stops the shipment being billed.",
			),
			ParamVisibleToCustomer: common.BooleanProperty("Whether the customer sees the hold."),
		},
		Required: []string{common.ParamShipmentID, ParamHoldID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*holdPatch, *holdUpdatePlan]{
		Request: readHoldPatch,
		Plan: func(
			ctx context.Context,
			patch *holdPatch,
			params *agentports.ToolExecuteParams,
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
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			current, err := holds.GetByID(ctx, &patch.get)
			if err != nil {
				return nil, err
			}
			saved, err := holds.Update(ctx, patch.request(current), params.Actor)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "updated",
				Kind:   "shipment hold",
				Name:   holdLabel(saved),
				IDs: map[string]string{
					ParamHoldID:            saved.ID.String(),
					common.ParamShipmentID: saved.ShipmentID.String(),
				},
				Record: common.RecordOf(common.ShipmentRecordEntity, saved.ShipmentID),
			}, nil
		},
	})

	return common.ClassifiedTool[*holdPatch, *holdUpdatePlan]{
		ReportingReceivableTool: base,
		Egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
		},
		Classify: classifyHoldVisibility,
	}
}

func renderHoldUpdate(_ *holdPatch, plan *holdUpdatePlan) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		ID:       plan.before.ID,
		Label:    holdLabel(plan.before),
		Version:  common.PreviewVersion(plan.before.Version),
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

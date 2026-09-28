package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

const (
	paramHoldID            = "holdId"
	paramHoldSeverity      = "severity"
	paramBlocksDispatch    = "blocksDispatch"
	paramBlocksDelivery    = "blocksDelivery"
	paramBlocksBilling     = "blocksBilling"
	paramVisibleToCustomer = "visibleToCustomer"
	maxHoldNoteChars       = 2000
)

var holdSeverities = agenttoolschema.Source(
	"holdReason.severity",
	holdreason.HoldSeverityValues(),
)

var updatedHoldFields = []string{
	fieldSeverity, fieldNotes, paramBlocksDispatch, paramBlocksDelivery, paramBlocksBilling,
	paramVisibleToCustomer,
}

type classifiedTool[R, P any] struct {
	reportingReceivableTool[R, P]
	egress   []agent.EgressClass
	classify func(serviceports.ToolExecuteParams) serviceports.CallPolicy
}

func (t classifiedTool[R, P]) Policy() serviceports.ToolPolicy {
	policy := t.reportingReceivableTool.Policy()
	policy.Egress = t.egress
	policy.Classify = t.classify

	return policy
}

type holdUpdater interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Update(
		ctx context.Context,
		req *repositories.UpdateShipmentHoldRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentHold, error)
	PreviewUpdate(
		ctx context.Context,
		req *repositories.UpdateShipmentHoldRequest,
		actor *serviceports.RequestActor,
	) (*shipment.ShipmentHold, error)
}

type holdPatch struct {
	get               repositories.GetShipmentHoldByIDRequest
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
) *repositories.UpdateShipmentHoldRequest {
	req := &repositories.UpdateShipmentHoldRequest{
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

func readHoldPatch(params *serviceports.ToolExecuteParams) (*holdPatch, error) {
	shipmentID, err := requirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}
	holdID, err := requirePulid(params.Params, paramHoldID)
	if err != nil {
		return nil, err
	}

	patch := &holdPatch{get: repositories.GetShipmentHoldByIDRequest{
		HoldID:     holdID,
		ShipmentID: shipmentID,
		TenantInfo: tenantFrom(*params),
	}}
	severity, given, err := optionalEnum(params.Params, paramHoldSeverity, holdSeverities.Values)
	if err != nil {
		return nil, err
	}
	if given {
		patch.severity = &severity
	}
	if patch.notes, err = optionalBoundedText(
		params.Params,
		fieldNotes,
		maxHoldNoteChars,
	); err != nil {
		return nil, err
	}
	flags := []struct {
		key string
		out **bool
	}{
		{paramBlocksDispatch, &patch.blocksDispatch},
		{paramBlocksDelivery, &patch.blocksDelivery},
		{paramBlocksBilling, &patch.blocksBilling},
		{paramVisibleToCustomer, &patch.visibleToCustomer},
	}
	for _, flag := range flags {
		if *flag.out, err = optionalBoolPointer(params.Params, flag.key); err != nil {
			return nil, err
		}
	}
	if patch.severity == nil && patch.notes == nil && patch.blocksDispatch == nil &&
		patch.blocksDelivery == nil && patch.blocksBilling == nil &&
		patch.visibleToCustomer == nil {
		return nil, errNothingToChange
	}

	return patch, nil
}

func classifyHoldVisibility(
	params serviceports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) serviceports.CallPolicy {
	if optionalBool(params.Params, paramVisibleToCustomer) {
		return serviceports.CallPolicy{Egress: agent.EgressCustomerVisible}
	}

	return serviceports.CallPolicy{Egress: agent.EgressInternal}
}

func newUpdateShipmentHoldTool(holds holdUpdater) serviceports.AgentTool {
	base := newReportingReceivableTool(&receivableSpec{
		name:     "update_shipment_hold",
		artifact: shipmentRecordEntity,
		description: "Change an active hold on a shipment: its severity, its notes, what it " +
			"blocks (dispatch, delivery, billing) and whether the customer sees it. Send only " +
			"what changes; everything else stays as it is. Use release_shipment_hold to clear " +
			"a hold rather than unblocking everything. Making a hold visible to the customer " +
			"is shown to them, so only do it when the person asked.",
		resource:    permission.ResourceShipmentHold,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Changes a hold inside Trenova; one made visible to the customer is shown " +
			"to them, so that call is held to the customer-visible ceiling.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment the hold is on"),
			paramHoldID: idProperty("The hold to change, from the holds get_shipment lists or " +
				"the page you are on. Never guess one."),
			paramHoldSeverity: agenttoolschema.Enum("How strongly it applies.", holdSeverities),
			fieldNotes: stringProperty("What the hold waits on, replacing the notes on it.",
				maxHoldNoteChars),
			paramBlocksDispatch: booleanProperty(
				"Whether it stops the shipment being dispatched.",
			),
			paramBlocksDelivery: booleanProperty(
				"Whether it stops the shipment being delivered.",
			),
			paramBlocksBilling:     booleanProperty("Whether it stops the shipment being billed."),
			paramVisibleToCustomer: booleanProperty("Whether the customer sees the hold."),
		},
		required: []string{paramShipmentID, paramHoldID},
		target:   targetShipment,
	}, receivablePlan[*holdPatch, *holdUpdatePlan]{
		request: readHoldPatch,
		plan: func(
			ctx context.Context,
			patch *holdPatch,
			params *serviceports.ToolExecuteParams,
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
		refused: func(*holdPatch) string { return "Would change a hold on the shipment." },
		render:  renderHoldUpdate,
		run: func(
			ctx context.Context,
			patch *holdPatch,
			params *serviceports.ToolExecuteParams,
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
					paramHoldID:     saved.ID.String(),
					paramShipmentID: saved.ShipmentID.String(),
				},
				Record: recordOf(shipmentRecordEntity, saved.ShipmentID),
			}, nil
		},
	})

	return classifiedTool[*holdPatch, *holdUpdatePlan]{
		reportingReceivableTool: base,
		egress: []agent.EgressClass{
			agent.EgressInternal,
			agent.EgressCustomerVisible,
		},
		classify: classifyHoldVisibility,
	}
}

func renderHoldUpdate(_ *holdPatch, plan *holdUpdatePlan) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		ID:       plan.before.ID,
		Label:    holdLabel(plan.before),
		Version:  previewVersion(plan.before.Version),
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

package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramShipmentID     = "shipmentId"
	paramOwnerID        = "ownerId"
	paramCopyCount      = "count"
	paramFirstPickupAt  = "firstPickupAt"
	maxDuplicateCopies  = 20
	kindShipment        = "shipment"
	actionUncanceled    = "uncanceled"
	actionTransferred   = "transferred"
	actionRerated       = "re-rated"
	actionRecalculated  = "recalculated"
	actionDuplicateSent = "duplication started"
)

type shipmentOperator interface {
	Uncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewUncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
	) (*shipmentports.ShipmentChangePreview, error)
	TransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewTransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*shipmentports.ShipmentChangePreview, error)
	AutoRate(
		ctx context.Context,
		req *shipmentports.AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, *shipmentports.ContractRateApplication, error)
	PreviewAutoRate(
		ctx context.Context,
		req *shipmentports.AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipmentports.ShipmentAutoRatePreview, error)
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentports.DistanceCalculationResponse, error)
	PreviewRecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentports.ShipmentDistancePreview, error)
	Duplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*shipmentports.ShipmentDuplicateWorkflowResponse, error)
	PreviewDuplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*shipmentports.ShipmentDuplicatePreview, error)
}

func targetShipment(params map[string]any) (agentports.ToolTarget, bool) {
	return targetOf(params, paramShipmentID, permission.ResourceShipment)
}

func shipmentIDProperty(what string) map[string]any {
	return agenttoolschema.RecordID(permission.ResourceShipment, what,
		"get_shipment, search_shipments or the page you are on")
}

func shipmentResult(action string, entity *shipment.Shipment) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindShipment}
	if entity == nil {
		return result
	}
	result.Name = entity.ProNumber
	result.IDs = map[string]string{paramShipmentID: entity.ID.String()}
	result.Record = recordOf(shipmentRecordEntity, entity.ID)

	return result
}

func newUncancelShipmentTool(shipments shipmentOperator) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "uncancel_shipment",
		artifact: shipmentRecordEntity,
		description: "Reopen a canceled shipment as New, with its moves and stops back to New, " +
			"when the person says the freight is back on. Nothing is sent to the customer, a " +
			"carrier or a trading partner, and nobody is assigned: tender or assign it again " +
			"afterwards. Read it with get_shipment first to be sure it is the canceled one.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Puts a canceled shipment back to New inside Trenova; nothing is sent, and " +
			"canceling it again undoes it.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The canceled shipment"),
		},
		required: []string{paramShipmentID},
		target:   targetShipment,
	}, receivablePlan[*shipmentports.UncancelShipmentRequest, *shipmentports.ShipmentChangePreview]{
		request: func(params *agentports.ToolExecuteParams) (*shipmentports.UncancelShipmentRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentports.UncancelShipmentRequest{
				TenantInfo: tenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentChangePreview, error) {
			return shipments.PreviewUncancel(ctx, req)
		},
		refused: func(*shipmentports.UncancelShipmentRequest) string {
			return "Would reopen a canceled shipment."
		},
		render: renderUncancel,
		run: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.Uncancel(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentResult(actionUncanceled, entity), nil
		},
	})
}

type ownershipRequest = shipmentports.TransferOwnershipRequest

func newTransferShipmentOwnershipTool(shipments shipmentOperator) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "transfer_shipment_ownership",
		artifact: shipmentRecordEntity,
		description: "Make another person the owner of a shipment, the one it is filed under " +
			"and who is asked about it. Take the new owner from the person who asked or the " +
			"ownerId get_shipment shows on another of the customer's shipments; never guess a " +
			"user. Only the current owner or someone allowed to change the shipment may hand " +
			"it on.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes who owns a shipment inside Trenova; nothing is sent, and it is " +
			"transferred back the same way.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment"),
			paramOwnerID: agenttoolschema.RecordIDText(
				permission.ResourceUser,
				"The user who takes it over: the person who asked, or "+
					"an ownerId get_shipment shows. Never guess one.",
			),
		},
		required: []string{paramShipmentID, paramOwnerID},
		target:   targetShipment,
	}, receivablePlan[*ownershipRequest, *shipmentports.ShipmentChangePreview]{
		request: func(params *agentports.ToolExecuteParams) (*ownershipRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}
			ownerID, err := requirePulid(params.Params, paramOwnerID)
			if err != nil {
				return nil, err
			}

			return &ownershipRequest{
				TenantInfo: tenantFrom(*params),
				ShipmentID: shipmentID,
				OwnerID:    ownerID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentChangePreview, error) {
			return shipments.PreviewTransferOwnership(ctx, req, params.Actor)
		},
		refused: func(*ownershipRequest) string {
			return "Would hand the shipment to a new owner."
		},
		render: renderOwnershipTransfer,
		run: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.TransferOwnership(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentResult(actionTransferred, entity), nil
		},
	})
}

func newRerateShipmentTool(shipments shipmentOperator) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "rerate_shipment",
		artifact: shipmentRecordEntity,
		description: "Price a saved shipment again from its customer's rate agreement, " +
			"replacing its rating method, base rate and contract accessorials. Use it when the lane, the stops or the agreement changed and " +
			"the person wants the contract price back; read explain_rate first to see it. A " +
			"shipment whose rate is locked or that is billed is refused, and one no contract " +
			"covers is left as it is.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes what the customer will be billed for the shipment; nothing is " +
			"invoiced or sent, and a person can edit the rate back.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment to re-rate"),
		},
		required: []string{paramShipmentID},
		target:   targetShipment,
	}, receivablePlan[*shipmentports.AutoRateShipmentRequest, *shipmentports.ShipmentAutoRatePreview]{
		request: func(
			params *agentports.ToolExecuteParams,
		) (*shipmentports.AutoRateShipmentRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentports.AutoRateShipmentRequest{
				TenantInfo: tenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *shipmentports.AutoRateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentAutoRatePreview, error) {
			return shipments.PreviewAutoRate(ctx, req, params.Actor)
		},
		refused: func(*shipmentports.AutoRateShipmentRequest) string {
			return "Would price the shipment again from its contract."
		},
		render: renderRerate,
		run: func(
			ctx context.Context,
			req *shipmentports.AutoRateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, application, err := shipments.AutoRate(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			result := shipmentResult(actionRerated, entity)
			if application != nil && !application.Applied {
				result.Action = "left unchanged: no agreement prices it (" +
					string(application.Outcome) + ")"
			}

			return result, nil
		},
	})
}

type distanceRequest struct {
	shipmentID pulid.ID
	tenant     pagination.TenantInfo
}

func newRecalculateShipmentDistanceTool(shipments shipmentOperator) agentports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:        "recalculate_shipment_distance",
		searchTerms: []string{"recalculate miles", "shipment mileage", "rerun distance"},
		artifact:    shipmentRecordEntity,
		description: "Work out a shipment's move miles again from its stops through the " +
			"organization's distance profile and routing provider, and save them. Use it " +
			"after its stops changed and the miles on the board look wrong. It " +
			"changes the miles only; the charges follow them the next time the shipment " +
			"is rated.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		rationale: "Re-derives the shipment's move distances inside Trenova from its stops; " +
			"nothing is sent and running it again gives the same miles.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment"),
		},
		required: []string{paramShipmentID},
		target:   targetShipment,
	}, receivablePlan[*distanceRequest, *shipmentports.ShipmentDistancePreview]{
		request: func(params *agentports.ToolExecuteParams) (*distanceRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &distanceRequest{shipmentID: shipmentID, tenant: tenantFrom(*params)}, nil
		},
		plan: func(
			ctx context.Context,
			req *distanceRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentDistancePreview, error) {
			return shipments.PreviewRecalculateDistance(ctx, req.shipmentID, req.tenant)
		},
		refused: func(*distanceRequest) string {
			return "Would work out the shipment's move miles again."
		},
		render: renderDistance,
		run: func(
			ctx context.Context,
			req *distanceRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			response, err := shipments.RecalculateDistance(ctx, req.shipmentID, req.tenant)
			if err != nil {
				return nil, err
			}
			result := &agent.ToolExecutionResult{
				Action: actionRecalculated,
				Kind:   kindShipment,
				IDs:    map[string]string{paramShipmentID: req.shipmentID.String()},
				Record: recordOf(shipmentRecordEntity, req.shipmentID),
			}
			if response != nil {
				result.Name = fmt.Sprintf("%.1f miles", response.TotalDistance)
			}

			return result, nil
		},
	})
}

type shipmentSourceReader interface {
	Get(ctx context.Context, req *shipmentports.GetShipmentByIDRequest) (*shipment.Shipment, error)
}

type duplicateShipmentDeps struct {
	Shipments shipmentOperator
	Sources   shipmentSourceReader
	Locations locationZoneReader
}

func newDuplicateShipmentTool(deps duplicateShipmentDeps) agentports.AgentTool {
	shipments := deps.Shipments

	return newReportingReceivableTool(&receivableSpec{
		name:     "duplicate_shipment",
		artifact: shipmentRecordEntity,
		description: "Copy an existing shipment into new ones on the server, carrying its " +
			"customer, stops, commodities, charges, freight terms and rating method exactly " +
			"as saved. Prefer it to create_shipment whenever the freight repeats one already " +
			"booked, since nothing is retyped. Give firstPickupAt to move every stop window " +
			"so the first pickup starts then, in local time at that pickup; leave it out to " +
			"keep the source's dates. A shipment nothing prices cannot be copied. Each " +
			"copy gets its own pro number and order; they appear in search_shipments within " +
			"a minute.",
		resource:    permission.ResourceShipment,
		operation:   permission.OpDuplicate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Creates new shipments inside Trenova from one already saved; nothing is " +
			"sent, and a copy made in error is canceled.",
		properties: map[string]any{
			paramShipmentID: shipmentIDProperty("The shipment to copy"),
			paramCopyCount: integerProperty(
				"How many copies to make, 1 to 20. Defaults to 1.", 1, maxDuplicateCopies,
			),
			paramFirstPickupAt: agenttoolschema.LocalDateTime(
				"When the copies' first pickup window " +
					"starts; every other stop keeps its distance from it. Only from what the " +
					"person asked for, never invented.",
			),
		},
		required: []string{paramShipmentID},
		target:   targetShipment,
	}, receivablePlan[*shipmentports.BulkDuplicateShipmentRequest, *shipmentports.ShipmentDuplicatePreview]{
		request: duplicateRequest,
		settle: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) error {
			return settleFirstPickup(ctx, &deps, req, params)
		},
		plan: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentDuplicatePreview, error) {
			return shipments.PreviewDuplicate(ctx, req)
		},
		refused: func(req *shipmentports.BulkDuplicateShipmentRequest) string {
			return fmt.Sprintf(
				"Would copy the shipment into %s.",
				countOf(req.Count, "new shipment"),
			)
		},
		render: renderDuplicate,
		run: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			started, err := shipments.Duplicate(ctx, req)
			if err != nil {
				return nil, err
			}
			result := &agent.ToolExecutionResult{
				Action: actionDuplicateSent,
				Kind:   kindShipment,
				Name:   countOf(req.Count, "copy"),
				IDs:    map[string]string{paramShipmentID: req.ShipmentID.String()},
			}
			if started != nil {
				result.IDs["workflowId"] = started.WorkflowID
			}

			return result, nil
		},
	})
}

func duplicateRequest(
	params *agentports.ToolExecuteParams,
) (*shipmentports.BulkDuplicateShipmentRequest, error) {
	shipmentID, err := requirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}

	count := 1
	if _, given := params.Params[paramCopyCount]; given {
		if count, err = requireIntInRange(
			params.Params, paramCopyCount, 1, maxDuplicateCopies,
		); err != nil {
			return nil, err
		}
	}

	return &shipmentports.BulkDuplicateShipmentRequest{
		TenantInfo: tenantFrom(*params),
		ShipmentID: shipmentID,
		Count:      count,
	}, nil
}

func settleFirstPickup(
	ctx context.Context,
	deps *duplicateShipmentDeps,
	req *shipmentports.BulkDuplicateShipmentRequest,
	params *agentports.ToolExecuteParams,
) error {
	if _, given := params.Params[paramFirstPickupAt]; !given {
		return nil
	}

	zones := locationZones{fallback: params.Timezone}
	pickup := pulid.Nil
	if deps.Sources != nil {
		source, err := deps.Sources.Get(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:              req.ShipmentID,
			TenantInfo:      req.TenantInfo,
			ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
		})
		if err != nil {
			return err
		}
		if pickup = firstStopLocation(source); pickup.IsNotNil() {
			if zones, err = readLocationZones(
				ctx, deps.Locations, req.TenantInfo, params.Timezone, []pulid.ID{pickup},
			); err != nil {
				return err
			}
		}
	}

	firstPickup, err := optionalLocalTime(params.Params, paramFirstPickupAt, zones.at(pickup))
	if err != nil {
		return err
	}
	req.FirstPickupAt = firstPickup

	return nil
}

func firstStopLocation(source *shipment.Shipment) pulid.ID {
	var first *shipment.Stop
	firstMove := int64(-1)
	for _, move := range source.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			if first == nil || move.Sequence < firstMove ||
				(move.Sequence == firstMove && stop.Sequence < first.Sequence) {
				first, firstMove = stop, move.Sequence
			}
		}
	}
	if first == nil {
		return pulid.Nil
	}

	return first.LocationID
}

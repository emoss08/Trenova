package shipmenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamCopyCount = "count"

const ParamFirstPickupAt = "firstPickupAt"

const maxDuplicateCopies = 20

const actionUncanceled = "uncanceled"

const actionTransferred = "transferred"

const actionRerated = "re-rated"

const actionRecalculated = "recalculated"

const actionDuplicateSent = "duplication started"

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
	return common.TargetOf(params, common.ParamShipmentID, permission.ResourceShipment)
}

func NewUncancelShipmentTool(shipments shipmentOperator) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "uncancel_shipment",
		Artifact: common.ShipmentRecordEntity,
		Description: "Reopen a canceled shipment as New, with its moves and stops back to New, " +
			"when the person says the freight is back on. Nothing is sent to the customer, a " +
			"carrier or a trading partner, and nobody is assigned: tender or assign it again " +
			"afterwards. Read it with get_shipment first to be sure it is the canceled one.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Puts a canceled shipment back to New inside Trenova; nothing is sent, and " +
			"canceling it again undoes it.",
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The canceled shipment"),
		},
		Required: []string{common.ParamShipmentID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*shipmentports.UncancelShipmentRequest, *shipmentports.ShipmentChangePreview]{
		Request: func(params *agentports.ToolExecuteParams) (*shipmentports.UncancelShipmentRequest, error) {
			shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentports.UncancelShipmentRequest{
				TenantInfo: common.TenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentChangePreview, error) {
			return shipments.PreviewUncancel(ctx, req)
		},
		Refused: func(*shipmentports.UncancelShipmentRequest) string {
			return "Would reopen a canceled shipment."
		},
		Render: renderUncancel,
		Run: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.Uncancel(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return common.ShipmentResult(actionUncanceled, entity), nil
		},
	})
}

type ownershipRequest = shipmentports.TransferOwnershipRequest

func NewTransferShipmentOwnershipTool(shipments shipmentOperator) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "transfer_shipment_ownership",
		Artifact: common.ShipmentRecordEntity,
		Description: "Make another person the owner of a shipment, the one it is filed under " +
			"and who is asked about it. Take the new owner from the person who asked or the " +
			"ownerId get_shipment shows on another of the customer's shipments; never guess a " +
			"user. Only the current owner or someone allowed to change the shipment may hand " +
			"it on.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes who owns a shipment inside Trenova; nothing is sent, and it is " +
			"transferred back the same way.",
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment"),
			common.ParamOwnerID: agenttoolschema.RecordIDText(
				permission.ResourceUser,
				"The user who takes it over: the person who asked, or "+
					"an ownerId get_shipment shows. Never guess one.",
			),
		},
		Required: []string{common.ParamShipmentID, common.ParamOwnerID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*ownershipRequest, *shipmentports.ShipmentChangePreview]{
		Request: func(params *agentports.ToolExecuteParams) (*ownershipRequest, error) {
			shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
			if err != nil {
				return nil, err
			}
			ownerID, err := common.RequirePulid(params.Params, common.ParamOwnerID)
			if err != nil {
				return nil, err
			}

			return &ownershipRequest{
				TenantInfo: common.TenantFrom(*params),
				ShipmentID: shipmentID,
				OwnerID:    ownerID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentChangePreview, error) {
			return shipments.PreviewTransferOwnership(ctx, req, params.Actor)
		},
		Refused: func(*ownershipRequest) string {
			return "Would hand the shipment to a new owner."
		},
		Render: renderOwnershipTransfer,
		Run: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.TransferOwnership(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return common.ShipmentResult(actionTransferred, entity), nil
		},
	})
}

func NewRerateShipmentTool(shipments shipmentOperator) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "rerate_shipment",
		Artifact: common.ShipmentRecordEntity,
		Description: "Price a saved shipment again from its customer's rate agreement, " +
			"replacing its rating method, base rate and contract accessorials. Use it when the lane, the stops or the agreement changed and " +
			"the person wants the contract price back; read explain_rate first to see it. A " +
			"shipment whose rate is locked or that is billed is refused, and one no contract " +
			"covers is left as it is.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes what the customer will be billed for the shipment; nothing is " +
			"invoiced or sent, and a person can edit the rate back.",
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment to re-rate"),
		},
		Required: []string{common.ParamShipmentID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*shipmentports.AutoRateShipmentRequest, *shipmentports.ShipmentAutoRatePreview]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*shipmentports.AutoRateShipmentRequest, error) {
			shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentports.AutoRateShipmentRequest{
				TenantInfo: common.TenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *shipmentports.AutoRateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentAutoRatePreview, error) {
			return shipments.PreviewAutoRate(ctx, req, params.Actor)
		},
		Refused: func(*shipmentports.AutoRateShipmentRequest) string {
			return "Would price the shipment again from its contract."
		},
		Render: renderRerate,
		Run: func(
			ctx context.Context,
			req *shipmentports.AutoRateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, application, err := shipments.AutoRate(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			result := common.ShipmentResult(actionRerated, entity)
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

func NewRecalculateShipmentDistanceTool(shipments shipmentOperator) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:        "recalculate_shipment_distance",
		SearchTerms: []string{"recalculate miles", "shipment mileage", "rerun distance"},
		Artifact:    common.ShipmentRecordEntity,
		Description: "Work out a shipment's move miles again from its stops through the " +
			"organization's distance profile and routing provider, and save them. Use it " +
			"after its stops changed and the miles on the board look wrong. It " +
			"changes the miles only; the charges follow them the next time the shipment " +
			"is rated.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Rationale: "Re-derives the shipment's move distances inside Trenova from its stops; " +
			"nothing is sent and running it again gives the same miles.",
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment"),
		},
		Required: []string{common.ParamShipmentID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*distanceRequest, *shipmentports.ShipmentDistancePreview]{
		Request: func(params *agentports.ToolExecuteParams) (*distanceRequest, error) {
			shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &distanceRequest{shipmentID: shipmentID, tenant: common.TenantFrom(*params)}, nil
		},
		Plan: func(
			ctx context.Context,
			req *distanceRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentDistancePreview, error) {
			return shipments.PreviewRecalculateDistance(ctx, req.shipmentID, req.tenant)
		},
		Refused: func(*distanceRequest) string {
			return "Would work out the shipment's move miles again."
		},
		Render: renderDistance,
		Run: func(
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
				Kind:   common.KindShipment,
				IDs:    map[string]string{common.ParamShipmentID: req.shipmentID.String()},
				Record: common.RecordOf(common.ShipmentRecordEntity, req.shipmentID),
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

type DuplicateShipmentDeps struct {
	Shipments shipmentOperator
	Sources   shipmentSourceReader
	Locations common.LocationZoneReader
}

func NewDuplicateShipmentTool(deps DuplicateShipmentDeps) agentports.AgentTool {
	shipments := deps.Shipments

	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "duplicate_shipment",
		Artifact: common.ShipmentRecordEntity,
		Description: "Copy an existing shipment into new ones on the server, carrying its " +
			"customer, stops, commodities, charges, freight terms and rating method exactly " +
			"as saved. Prefer it to create_shipment whenever the freight repeats one already " +
			"booked, since nothing is retyped. Give firstPickupAt to move every stop window " +
			"so the first pickup starts then, in local time at that pickup; leave it out to " +
			"keep the source's dates. A shipment nothing prices cannot be copied. Each " +
			"copy gets its own pro number and order; they appear in search_shipments within " +
			"a minute.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpDuplicate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Creates new shipments inside Trenova from one already saved; nothing is " +
			"sent, and a copy made in error is canceled.",
		Properties: map[string]any{
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment to copy"),
			ParamCopyCount: common.IntegerProperty(
				"How many copies to make, 1 to 20. Defaults to 1.", 1, maxDuplicateCopies,
			),
			ParamFirstPickupAt: agenttoolschema.LocalDateTime(
				"When the copies' first pickup window " +
					"starts; every other stop keeps its distance from it. Only from what the " +
					"person asked for, never invented.",
			),
		},
		Required: []string{common.ParamShipmentID},
		Target:   targetShipment,
	}, common.ReceivablePlan[*shipmentports.BulkDuplicateShipmentRequest, *shipmentports.ShipmentDuplicatePreview]{
		Request: duplicateRequest,
		Settle: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			params *agentports.ToolExecuteParams,
		) error {
			return settleFirstPickup(ctx, &deps, req, params)
		},
		Plan: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			_ *agentports.ToolExecuteParams,
		) (*shipmentports.ShipmentDuplicatePreview, error) {
			return shipments.PreviewDuplicate(ctx, req)
		},
		Refused: func(req *shipmentports.BulkDuplicateShipmentRequest) string {
			return fmt.Sprintf(
				"Would copy the shipment into %s.",
				common.CountOf(req.Count, "new shipment"),
			)
		},
		Render: renderDuplicate,
		Run: func(
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
				Kind:   common.KindShipment,
				Name:   common.CountOf(req.Count, "copy"),
				IDs:    map[string]string{common.ParamShipmentID: req.ShipmentID.String()},
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
	shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	count := 1
	if _, given := params.Params[ParamCopyCount]; given {
		if count, err = common.RequireIntInRange(
			params.Params, ParamCopyCount, 1, maxDuplicateCopies,
		); err != nil {
			return nil, err
		}
	}

	return &shipmentports.BulkDuplicateShipmentRequest{
		TenantInfo: common.TenantFrom(*params),
		ShipmentID: shipmentID,
		Count:      count,
	}, nil
}

func settleFirstPickup(
	ctx context.Context,
	deps *DuplicateShipmentDeps,
	req *shipmentports.BulkDuplicateShipmentRequest,
	params *agentports.ToolExecuteParams,
) error {
	if _, given := params.Params[ParamFirstPickupAt]; !given {
		return nil
	}

	zones := common.LocationZones{Fallback: params.Timezone}
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
			if zones, err = common.ReadLocationZones(
				ctx, deps.Locations, req.TenantInfo, params.Timezone, []pulid.ID{pickup},
			); err != nil {
				return err
			}
		}
	}

	firstPickup, err := common.OptionalLocalTime(
		params.Params,
		ParamFirstPickupAt,
		zones.At(pickup),
	)
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

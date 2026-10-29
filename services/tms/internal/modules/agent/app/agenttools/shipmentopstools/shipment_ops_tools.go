package shipmentopstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
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

type shipmentLifecycleOperator interface {
	Uncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewUncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
	) (*shipmentserviceports.ShipmentChangePreview, error)
	TransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewTransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*shipmentserviceports.ShipmentChangePreview, error)
	Duplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*shipmentports.ShipmentDuplicateWorkflowResponse, error)
	PreviewDuplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*shipmentserviceports.ShipmentDuplicatePreview, error)
}

type shipmentRerater interface {
	AutoRate(
		ctx context.Context,
		req *shipmentserviceports.AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, *shipmentserviceports.ContractRateApplication, error)
	PreviewAutoRate(
		ctx context.Context,
		req *shipmentserviceports.AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipmentserviceports.ShipmentAutoRatePreview, error)
}

type shipmentDistanceRecalculator interface {
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentdistanceports.DistanceCalculationResponse, error)
	PreviewRecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentdistanceports.ShipmentDistancePreview, error)
}

func NewUncancelShipmentTool(shipments shipmentLifecycleOperator) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_uncancel",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Reopen a canceled shipment as New, with its moves and stops back to New, " +
			"when the person says the freight is back on. Nothing is sent to the customer, a " +
			"carrier or a trading partner, and nobody is assigned: tender or assign it again " +
			"afterwards. Read it with shipment_get first to be sure it is the canceled one.",
		Resource:    permission.ResourceShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Puts a canceled shipment back to New inside Trenova; nothing is sent, and " +
			"canceling it again undoes it.",
		Properties: map[string]any{
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty("The canceled shipment"),
		},
		Required: []string{toolparams.ParamShipmentID},
		Target:   shipmenttools.TargetShipment,
	}, toolspec.ReceivablePlan[*shipmentports.UncancelShipmentRequest, *shipmentserviceports.ShipmentChangePreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*shipmentports.UncancelShipmentRequest, error) {
			shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentports.UncancelShipmentRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*shipmentserviceports.ShipmentChangePreview, error) {
			return shipments.PreviewUncancel(ctx, req)
		},
		Refused: func(*shipmentports.UncancelShipmentRequest) string {
			return "Would reopen a canceled shipment."
		},
		Render: renderUncancel,
		Run: func(
			ctx context.Context,
			req *shipmentports.UncancelShipmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, err := shipments.Uncancel(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentpreview.ShipmentResult(actionUncanceled, entity), nil
		},
	})
}

type ownershipRequest = shipmentports.TransferOwnershipRequest

func NewTransferShipmentOwnershipTool(
	shipments shipmentLifecycleOperator,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_transfer_ownership",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Make another person the owner of a shipment, the one it is filed under " +
			"and who is asked about it. Take the new owner from the person who asked or the " +
			"ownerId shipment_get shows on another of the customer's shipments; never guess a " +
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
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty("The shipment"),
			toolparams.ParamOwnerID: agenttoolschema.RecordIDText(
				permission.ResourceUser,
				"The user who takes it over: the person who asked, or "+
					"an ownerId shipment_get shows. Never guess one.",
			),
		},
		Required: []string{toolparams.ParamShipmentID, toolparams.ParamOwnerID},
		Target:   shipmenttools.TargetShipment,
	}, toolspec.ReceivablePlan[*ownershipRequest, *shipmentserviceports.ShipmentChangePreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*ownershipRequest, error) {
			shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
			if err != nil {
				return nil, err
			}
			ownerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamOwnerID)
			if err != nil {
				return nil, err
			}

			return &ownershipRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				ShipmentID: shipmentID,
				OwnerID:    ownerID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*shipmentserviceports.ShipmentChangePreview, error) {
			return shipments.PreviewTransferOwnership(ctx, req, params.Actor)
		},
		Refused: func(*ownershipRequest) string {
			return "Would hand the shipment to a new owner."
		},
		Render: renderOwnershipTransfer,
		Run: func(
			ctx context.Context,
			req *ownershipRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, err := shipments.TransferOwnership(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentpreview.ShipmentResult(actionTransferred, entity), nil
		},
	})
}

func NewRerateShipmentTool(shipments shipmentRerater) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_rerate",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Price a saved shipment again from its customer's rate agreement, " +
			"replacing its rating method, base rate and contract accessorials. Use it when the lane, the stops or the agreement changed and " +
			"the person wants the contract price back; read rate_explain first to see it. A " +
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
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty(
				"The shipment to re-rate",
			),
		},
		Required: []string{toolparams.ParamShipmentID},
		Target:   shipmenttools.TargetShipment,
	}, toolspec.ReceivablePlan[*shipmentserviceports.AutoRateShipmentRequest, *reratePlan]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*shipmentserviceports.AutoRateShipmentRequest, error) {
			shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &shipmentserviceports.AutoRateShipmentRequest{
				TenantInfo: toolparams.TenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *shipmentserviceports.AutoRateShipmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*reratePlan, error) {
			preview, err := shipments.PreviewAutoRate(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			plan := &reratePlan{ShipmentAutoRatePreview: preview}
			if preview.Before != nil && preview.After != nil {
				if plan.money, err = invoicepreview.ChargeEditMoney(
					ctx,
					preview.Before,
					preview.After,
				); err != nil {
					return nil, err
				}
			}

			return plan, nil
		},
		Refused: func(*shipmentserviceports.AutoRateShipmentRequest) string {
			return "Would price the shipment again from its contract."
		},
		Render: renderRerate,
		Run: func(
			ctx context.Context,
			req *shipmentserviceports.AutoRateShipmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, application, err := shipments.AutoRate(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}
			result := shipmentpreview.ShipmentResult(actionRerated, entity)
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

func NewRecalculateShipmentDistanceTool(
	shipments shipmentDistanceRecalculator,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "shipment_recalculate_distance",
		SearchTerms: []string{"recalculate miles", "shipment mileage", "rerun distance"},
		Artifact:    changepreview.ShipmentRecordEntity,
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
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty("The shipment"),
		},
		Required: []string{toolparams.ParamShipmentID},
		Target:   shipmenttools.TargetShipment,
	}, toolspec.ReceivablePlan[*distanceRequest, *shipmentdistanceports.ShipmentDistancePreview]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*distanceRequest, error) {
			shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &distanceRequest{
				shipmentID: shipmentID,
				tenant:     toolparams.TenantFrom(*params),
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *distanceRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*shipmentdistanceports.ShipmentDistancePreview, error) {
			return shipments.PreviewRecalculateDistance(ctx, req.shipmentID, req.tenant)
		},
		Refused: func(*distanceRequest) string {
			return "Would work out the shipment's move miles again."
		},
		Render: renderDistance,
		Run: func(
			ctx context.Context,
			req *distanceRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			response, err := shipments.RecalculateDistance(ctx, req.shipmentID, req.tenant)
			if err != nil {
				return nil, err
			}
			result := &agentrecord.ToolExecutionResult{
				Action: actionRecalculated,
				Kind:   shipmentpreview.KindShipment,
				IDs:    map[string]string{toolparams.ParamShipmentID: req.shipmentID.String()},
				Record: changepreview.RecordOf(changepreview.ShipmentRecordEntity, req.shipmentID),
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
	Shipments shipmentLifecycleOperator
	Sources   shipmentSourceReader
	Locations shipmentpreview.LocationZoneReader
}

func NewDuplicateShipmentTool(deps DuplicateShipmentDeps) agenttoolports.AgentTool {
	shipments := deps.Shipments

	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:     "shipment_duplicate",
		Artifact: changepreview.ShipmentRecordEntity,
		Description: "Copy an existing shipment into new ones on the server, carrying its " +
			"customer, stops, commodities, charges, freight terms and rating method exactly " +
			"as saved. Prefer it to shipment_create whenever the freight repeats one already " +
			"booked, since nothing is retyped. Give firstPickupAt to move every stop window " +
			"so the first pickup starts then, in local time at that pickup; leave it out to " +
			"keep the source's dates. A shipment nothing prices cannot be copied. Each " +
			"copy gets its own pro number and order; they appear in shipment_list within " +
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
			toolparams.ParamShipmentID: shipmentpreview.ShipmentIDProperty("The shipment to copy"),
			ParamCopyCount: toolparams.IntegerProperty(
				"How many copies to make, 1 to 20. Defaults to 1.", 1, maxDuplicateCopies,
			),
			ParamFirstPickupAt: agenttoolschema.LocalDateTime(
				"When the copies' first pickup window " +
					"starts; every other stop keeps its distance from it. Only from what the " +
					"person asked for, never invented.",
			),
		},
		Required: []string{toolparams.ParamShipmentID},
		Target:   shipmenttools.TargetShipment,
	}, toolspec.ReceivablePlan[*shipmentports.BulkDuplicateShipmentRequest, *shipmentserviceports.ShipmentDuplicatePreview]{
		Request: duplicateRequest,
		Settle: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			params *agenttoolports.ToolExecuteParams,
		) error {
			return settleFirstPickup(ctx, &deps, req, params)
		},
		Plan: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*shipmentserviceports.ShipmentDuplicatePreview, error) {
			return shipments.PreviewDuplicate(ctx, req)
		},
		Refused: func(req *shipmentports.BulkDuplicateShipmentRequest) string {
			return fmt.Sprintf(
				"Would copy the shipment into %s.",
				changepreview.CountOf(req.Count, "new shipment"),
			)
		},
		Render: renderDuplicate,
		Run: func(
			ctx context.Context,
			req *shipmentports.BulkDuplicateShipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			started, err := shipments.Duplicate(ctx, req)
			if err != nil {
				return nil, err
			}
			result := &agentrecord.ToolExecutionResult{
				Action: actionDuplicateSent,
				Kind:   shipmentpreview.KindShipment,
				Name:   changepreview.CountOf(req.Count, "copy"),
				IDs:    map[string]string{toolparams.ParamShipmentID: req.ShipmentID.String()},
			}
			if started != nil {
				result.IDs["workflowId"] = started.WorkflowID
			}

			return result, nil
		},
	})
}

func duplicateRequest(
	params *agenttoolports.ToolExecuteParams,
) (*shipmentports.BulkDuplicateShipmentRequest, error) {
	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}

	count := 1
	if _, given := params.Params[ParamCopyCount]; given {
		if count, err = toolparams.RequireIntInRange(
			params.Params, ParamCopyCount, 1, maxDuplicateCopies,
		); err != nil {
			return nil, err
		}
	}

	return &shipmentports.BulkDuplicateShipmentRequest{
		TenantInfo: toolparams.TenantFrom(*params),
		ShipmentID: shipmentID,
		Count:      count,
	}, nil
}

func settleFirstPickup(
	ctx context.Context,
	deps *DuplicateShipmentDeps,
	req *shipmentports.BulkDuplicateShipmentRequest,
	params *agenttoolports.ToolExecuteParams,
) error {
	if _, given := params.Params[ParamFirstPickupAt]; !given {
		return nil
	}

	zones := shipmentpreview.LocationZones{Fallback: params.Timezone}
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
			if zones, err = shipmentpreview.ReadLocationZones(
				ctx, deps.Locations, req.TenantInfo, params.Timezone, []pulid.ID{pickup},
			); err != nil {
				return err
			}
		}
	}

	firstPickup, err := toolparams.OptionalLocalTime(
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

package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
		req *repositories.UncancelShipmentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewUncancel(
		ctx context.Context,
		req *repositories.UncancelShipmentRequest,
	) (*serviceports.ShipmentChangePreview, error)
	TransferOwnership(
		ctx context.Context,
		req *repositories.TransferOwnershipRequest,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewTransferOwnership(
		ctx context.Context,
		req *repositories.TransferOwnershipRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ShipmentChangePreview, error)
	AutoRate(
		ctx context.Context,
		req *serviceports.AutoRateShipmentRequest,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, *serviceports.ContractRateApplication, error)
	PreviewAutoRate(
		ctx context.Context,
		req *serviceports.AutoRateShipmentRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ShipmentAutoRatePreview, error)
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*serviceports.DistanceCalculationResponse, error)
	PreviewRecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*serviceports.ShipmentDistancePreview, error)
	Duplicate(
		ctx context.Context,
		req *repositories.BulkDuplicateShipmentRequest,
	) (*repositories.ShipmentDuplicateWorkflowResponse, error)
	PreviewDuplicate(
		ctx context.Context,
		req *repositories.BulkDuplicateShipmentRequest,
	) (*serviceports.ShipmentDuplicatePreview, error)
}

func targetShipment(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramShipmentID, permission.ResourceShipment)
}

func shipmentIDProperty(what string) map[string]any {
	return idProperty(what + ", from get_shipment, search_shipments or the page you are on. " +
		"Never guess one.")
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

func newUncancelShipmentTool(shipments shipmentOperator) serviceports.AgentTool {
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
	}, receivablePlan[*repositories.UncancelShipmentRequest, *serviceports.ShipmentChangePreview]{
		request: func(params *serviceports.ToolExecuteParams) (*repositories.UncancelShipmentRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &repositories.UncancelShipmentRequest{
				TenantInfo: tenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *repositories.UncancelShipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentChangePreview, error) {
			return shipments.PreviewUncancel(ctx, req)
		},
		refused: func(*repositories.UncancelShipmentRequest) string {
			return "Would reopen a canceled shipment."
		},
		render: renderUncancel,
		run: func(
			ctx context.Context,
			req *repositories.UncancelShipmentRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.Uncancel(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentResult(actionUncanceled, entity), nil
		},
	})
}

type ownershipRequest = repositories.TransferOwnershipRequest

func newTransferShipmentOwnershipTool(shipments shipmentOperator) serviceports.AgentTool {
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
			paramOwnerID: idProperty("The user who takes it over: the person who asked, or " +
				"an ownerId get_shipment shows. Never guess one."),
		},
		required: []string{paramShipmentID, paramOwnerID},
		target:   targetShipment,
	}, receivablePlan[*ownershipRequest, *serviceports.ShipmentChangePreview]{
		request: func(params *serviceports.ToolExecuteParams) (*ownershipRequest, error) {
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
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentChangePreview, error) {
			return shipments.PreviewTransferOwnership(ctx, req, params.Actor)
		},
		refused: func(*ownershipRequest) string {
			return "Would hand the shipment to a new owner."
		},
		render: renderOwnershipTransfer,
		run: func(
			ctx context.Context,
			req *ownershipRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := shipments.TransferOwnership(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return shipmentResult(actionTransferred, entity), nil
		},
	})
}

func newRerateShipmentTool(shipments shipmentOperator) serviceports.AgentTool {
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
	}, receivablePlan[*serviceports.AutoRateShipmentRequest, *serviceports.ShipmentAutoRatePreview]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*serviceports.AutoRateShipmentRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &serviceports.AutoRateShipmentRequest{
				TenantInfo: tenantFrom(*params),
				ShipmentID: shipmentID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *serviceports.AutoRateShipmentRequest,
			params *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentAutoRatePreview, error) {
			return shipments.PreviewAutoRate(ctx, req, params.Actor)
		},
		refused: func(*serviceports.AutoRateShipmentRequest) string {
			return "Would price the shipment again from its contract."
		},
		render: renderRerate,
		run: func(
			ctx context.Context,
			req *serviceports.AutoRateShipmentRequest,
			params *serviceports.ToolExecuteParams,
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

func newRecalculateShipmentDistanceTool(shipments shipmentOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "recalculate_shipment_distance",
		artifact: shipmentRecordEntity,
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
	}, receivablePlan[*distanceRequest, *serviceports.ShipmentDistancePreview]{
		request: func(params *serviceports.ToolExecuteParams) (*distanceRequest, error) {
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &distanceRequest{shipmentID: shipmentID, tenant: tenantFrom(*params)}, nil
		},
		plan: func(
			ctx context.Context,
			req *distanceRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentDistancePreview, error) {
			return shipments.PreviewRecalculateDistance(ctx, req.shipmentID, req.tenant)
		},
		refused: func(*distanceRequest) string {
			return "Would work out the shipment's move miles again."
		},
		render: renderDistance,
		run: func(
			ctx context.Context,
			req *distanceRequest,
			_ *serviceports.ToolExecuteParams,
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

func newDuplicateShipmentTool(shipments shipmentOperator) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "duplicate_shipment",
		artifact: shipmentRecordEntity,
		description: "Copy an existing shipment into new ones on the server, carrying its " +
			"customer, stops, commodities, charges, freight terms and rating method exactly " +
			"as saved. Prefer it to create_shipment whenever the freight repeats one already " +
			"booked, since nothing is retyped. Give firstPickupAt to move every stop window " +
			"so the first pickup starts then; leave it out to keep the source's dates. Each " +
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
			paramFirstPickupAt: dateTimeProperty("When the copies' first pickup window starts; " +
				"every other stop keeps its distance from it. Only from what the person asked " +
				"for, never invented."),
		},
		required: []string{paramShipmentID},
		target:   targetShipment,
	}, receivablePlan[*repositories.BulkDuplicateShipmentRequest, *serviceports.ShipmentDuplicatePreview]{
		request: duplicateRequest,
		plan: func(
			ctx context.Context,
			req *repositories.BulkDuplicateShipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*serviceports.ShipmentDuplicatePreview, error) {
			return shipments.PreviewDuplicate(ctx, req)
		},
		refused: func(req *repositories.BulkDuplicateShipmentRequest) string {
			return fmt.Sprintf(
				"Would copy the shipment into %s.",
				countOf(req.Count, "new shipment"),
			)
		},
		render: renderDuplicate,
		run: func(
			ctx context.Context,
			req *repositories.BulkDuplicateShipmentRequest,
			_ *serviceports.ToolExecuteParams,
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
	params *serviceports.ToolExecuteParams,
) (*repositories.BulkDuplicateShipmentRequest, error) {
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

	firstPickup, err := optionalDateTime(params.Params, paramFirstPickupAt)
	if err != nil {
		return nil, err
	}

	return &repositories.BulkDuplicateShipmentRequest{
		TenantInfo:    tenantFrom(*params),
		ShipmentID:    shipmentID,
		Count:         count,
		FirstPickupAt: firstPickup,
	}, nil
}

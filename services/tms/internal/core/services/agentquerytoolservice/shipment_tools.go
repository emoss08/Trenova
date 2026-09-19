package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 25
)

type getShipmentTool struct {
	repo repositories.ShipmentRepository
}

func newGetShipmentTool(repo repositories.ShipmentRepository) serviceports.AgentQueryTool {
	return &getShipmentTool{repo: repo}
}

func (t *getShipmentTool) Name() string { return "get_shipment" }

func (t *getShipmentTool) Description() string {
	return "Retrieve one shipment by its id, including its stops, moves, and assignments. " +
		"Use search_shipments first when you only have a pro number or customer name."
}

func (t *getShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type":        "string",
				"description": "The shipment's id",
			},
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *getShipmentTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *getShipmentTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}

	return t.repo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID: shipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
			IncludeCustomer:       true,
		},
	})
}

type searchShipmentsTool struct {
	repo repositories.ShipmentRepository
}

func newSearchShipmentsTool(repo repositories.ShipmentRepository) serviceports.AgentQueryTool {
	return &searchShipmentsTool{repo: repo}
}

func (t *searchShipmentsTool) Name() string { return "search_shipments" }

func (t *searchShipmentsTool) Description() string {
	return "List shipments, optionally narrowed by free text such as a pro number, " +
		"BOL, customer name, or city, and by status. Call it with no query to see the " +
		"most recent shipments. Returns matches with their ids, which get_shipment " +
		"can then expand."
}

func (t *searchShipmentsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text to match. Omit to list shipments unfiltered.",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Optional status filter, such as New, Assigned, InTransit, Delivered, or Canceled",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"additionalProperties": false,
	}
}

func (t *searchShipmentsTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *searchShipmentsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	// Optional for the same reason as on search_worker: the repository applies a
	// text search only when the term is non-empty, so requiring one here refused
	// a call the layer beneath would have served.
	query := optionalString(params.Params, "query")

	// A model asked for "all shipments" will happily request a limit of 10000,
	// which would blow the context window and the query budget at once.
	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}

	status := optionalString(params.Params, "status")

	criteria := newSearchCriteria("shipments")
	criteria.text(query)
	criteria.field("status", status)

	result, err := t.repo.List(ctx, &repositories.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
		ShipmentOptions: repositories.ShipmentOptions{
			Status:          status,
			IncludeCustomer: true,
		},
	})
	if err != nil {
		return nil, err
	}

	return criteria.result(result.Items, len(result.Items)), nil
}

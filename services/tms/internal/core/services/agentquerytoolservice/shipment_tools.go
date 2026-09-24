package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
				"type": "string",
				"description": "The shipment's id, from search_shipments or list_shipments, " +
					"the page you are on, or this run's subject.",
			},
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *getShipmentTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
	})
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
				"type": "string",
				"enum": shipmentStatuses,
				"description": "Optional status filter. There is no Delivered: a " +
					"delivered load is Completed. Omit to include every status.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"additionalProperties": false,
	}
}

func (t *searchShipmentsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
	})
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

	// Unvalidated, this passed whatever the model sent straight to the
	// repository, which matched nothing and returned an empty page the model
	// reported as "there are no shipments in that state". The schema used to
	// name "Delivered" here, which is not one of the statuses, so the tool was
	// instructing its caller to produce exactly that.
	status, err := shipmentStatusFilter(params.Params)
	if err != nil {
		return nil, err
	}

	criteria := filtercatalog.NewCriteria("shipments").At(clockFor(params))
	criteria.Text(query)
	criteria.Field("status", status)

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

	rows := make([]shipmentRow, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, toShipmentRow(item))
	}

	return searchResult(criteria, rows, len(rows)), nil
}

// shipmentStatusFilter refuses a status outside the set and names the set.
//
// A refusal that lists the alternatives is a correction the model can act on;
// an empty result is one it reports as fact.
func shipmentStatusFilter(params map[string]any) (string, error) {
	status := optionalString(params, "status")
	if status == "" {
		return "", nil
	}

	for _, candidate := range shipmentStatuses {
		if candidate == status {
			return status, nil
		}
	}

	return "", fmt.Errorf(
		"parameter %q does not accept %q; a delivered load is Completed. Use one of: %s",
		"status", status, strings.Join(shipmentStatuses, ", "),
	)
}

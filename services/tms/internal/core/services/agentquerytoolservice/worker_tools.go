package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type getWorkerTool struct {
	repo repositories.WorkerRepository
}

func newGetWorkerTool(repo repositories.WorkerRepository) serviceports.AgentQueryTool {
	return &getWorkerTool{repo: repo}
}

func (t *getWorkerTool) Name() string { return "get_worker" }

func (t *getWorkerTool) Description() string {
	return "Retrieve one worker (driver) by id, including their profile and current state. " +
		"Use search_worker first when you only have a name."
}

func (t *getWorkerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The worker's id",
			},
		},
		"required":             []string{"workerId"},
		"additionalProperties": false,
	}
}

func (t *getWorkerTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *getWorkerTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}

	return t.repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID: workerID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		IncludeProfile: true,
		IncludeState:   true,
	})
}

type searchWorkerTool struct {
	repo repositories.WorkerRepository
}

func newSearchWorkerTool(repo repositories.WorkerRepository) serviceports.AgentQueryTool {
	return &searchWorkerTool{repo: repo}
}

func (t *searchWorkerTool) Name() string { return "search_worker" }

func (t *searchWorkerTool) Description() string {
	return "List workers (drivers), optionally narrowed by a name or code. " +
		"Call it with no query to see who is on the roster; pass a query only when " +
		"you already have a name. Returns matches with their ids, which get_worker " +
		"can then expand. To find drivers by a licence or medical card date, use " +
		"list_expiring_credentials instead — this tool does not filter on dates."
}

func (t *searchWorkerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional name or code to match. Omit to list workers unfiltered.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"additionalProperties": false,
	}
}

func (t *searchWorkerTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *searchWorkerTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	// The term is optional because the repository never needed it:
	// ApplyCursorFilters only applies a text search when Query is non-empty, so
	// an empty one already means "the most recent N". Requiring it here made
	// the tool reject a call the layer beneath it would have served.
	query := optionalString(params.Params, "query")

	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}

	criteria := newSearchCriteria("workers")
	criteria.text(query)

	result, err := t.repo.List(ctx, &repositories.ListWorkersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
	})
	if err != nil {
		return nil, err
	}

	return criteria.result(result.Items, len(result.Items)), nil
}

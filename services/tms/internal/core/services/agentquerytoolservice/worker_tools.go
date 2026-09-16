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
		"Use search_workers first when you only have a name."
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

type searchWorkersTool struct {
	repo repositories.WorkerRepository
}

func newSearchWorkersTool(repo repositories.WorkerRepository) serviceports.AgentQueryTool {
	return &searchWorkersTool{repo: repo}
}

func (t *searchWorkersTool) Name() string { return "search_workers" }

func (t *searchWorkersTool) Description() string {
	return "Search workers (drivers) by name or code. Returns matches with their ids, " +
		"which get_worker can then expand."
}

func (t *searchWorkersTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Name or code to search for",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}

func (t *searchWorkersTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *searchWorkersTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query, err := requireString(params.Params, "query")
	if err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}

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

	return result.Items, nil
}

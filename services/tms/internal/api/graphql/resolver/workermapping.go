package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
)

func applyWorkerPatch(entity *worker.Worker, input gqlmodel.WorkerPatchInput) {
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.Type != nil {
		entity.Type = *input.Type
	}
	if input.DriverType != nil {
		entity.DriverType = *input.DriverType
	}
}

func workerListConnectionToModel(
	result *pagination.ListResult[*worker.Worker],
	offset int,
) *gqlmodel.WorkerConnection {
	hasNextPage := offset+len(result.Items) < result.Total
	edges := make([]*gqlmodel.WorkerEdge, 0, len(result.Items))
	for i, entity := range result.Items {
		edges = append(edges, &gqlmodel.WorkerEdge{
			Node:   entity,
			Cursor: offsetCursor(offset + i + 1),
		})
	}

	return &gqlmodel.WorkerConnection{
		Edges: edges,
		PageInfo: &gqlmodel.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   offsetEndCursor(offset, len(result.Items)),
		},
		TotalCount: &result.Total,
	}
}

func workerPTOListConnectionToModel(
	result *pagination.ListResult[*worker.WorkerPTO],
	offset int,
) *gqlmodel.WorkerPTOConnection {
	hasNextPage := offset+len(result.Items) < result.Total
	edges := make([]*gqlmodel.WorkerPTOEdge, 0, len(result.Items))
	for i, entity := range result.Items {
		edges = append(edges, &gqlmodel.WorkerPTOEdge{
			Node:   entity,
			Cursor: offsetCursor(offset + i + 1),
		})
	}

	return &gqlmodel.WorkerPTOConnection{
		Edges: edges,
		PageInfo: &gqlmodel.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   offsetEndCursor(offset, len(result.Items)),
		},
		TotalCount: &result.Total,
	}
}

func offsetEndCursor(offset, count int) *string {
	if count == 0 {
		return nil
	}

	cursor := offsetCursor(offset + count)
	return &cursor
}

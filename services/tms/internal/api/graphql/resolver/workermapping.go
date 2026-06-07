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

func workerCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.Worker],
) (*gqlmodel.WorkerConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.Worker, cursor string) *gqlmodel.WorkerEdge {
			return &gqlmodel.WorkerEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerConnection{
		Edges:      edges,
		PageInfo:   pageInfo(result.HasNextPage, lastEdgeCursor(edges, func(edge *gqlmodel.WorkerEdge) string { return edge.Cursor })),
		TotalCount: result.TotalCount,
	}, nil
}

func workerPTOCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerPTO],
) (*gqlmodel.WorkerPTOConnection, error) {
	edges, err := entityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *worker.WorkerPTO, cursor string) *gqlmodel.WorkerPTOEdge {
			return &gqlmodel.WorkerPTOEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.WorkerPTOConnection{
		Edges:      edges,
		PageInfo:   pageInfo(result.HasNextPage, lastEdgeCursor(edges, func(edge *gqlmodel.WorkerPTOEdge) string { return edge.Cursor })),
		TotalCount: result.TotalCount,
	}, nil
}

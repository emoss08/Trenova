package workerresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func workerCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.Worker],
) (*gqlmodel.WorkerConnection, error) {
	edges, err := base.EntityCursorEdges(
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
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(edges, func(edge *gqlmodel.WorkerEdge) string {
				return edge.Cursor
			}),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func workerPTOCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerPTO],
) (*gqlmodel.WorkerPTOConnection, error) {
	edges, err := base.EntityCursorEdges(
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
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(edges, func(edge *gqlmodel.WorkerPTOEdge) string {
				return edge.Cursor
			}),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func ptoBulkActionOperation(action workerports.PTOBulkActionType) (permission.Operation, error) {
	switch action {
	case workerports.PTOBulkActionApprove:
		return permission.OpApprove, nil
	case workerports.PTOBulkActionReject:
		return permission.OpReject, nil
	case workerports.PTOBulkActionCancel:
		return permission.OpCancel, nil
	default:
		return "", errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Bulk action is invalid",
		)
	}
}

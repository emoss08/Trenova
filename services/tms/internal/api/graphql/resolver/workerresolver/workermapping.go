package workerresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func workerCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.Worker],
) (*gqlmodel.WorkerConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
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
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func workerPTOCursorConnectionToModel(
	result *pagination.CursorListResult[*worker.WorkerPTO],
) (*gqlmodel.WorkerPTOConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
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
		TotalCount:      result.TotalCount,
		TotalCountExact: result.TotalCountExact,
	}, nil
}

func ptoBulkActionOperation(action workerptoports.PTOBulkActionType) (permission.Operation, error) {
	switch action {
	case workerptoports.PTOBulkActionApprove:
		return permission.OpApprove, nil
	case workerptoports.PTOBulkActionReject:
		return permission.OpReject, nil
	case workerptoports.PTOBulkActionCancel:
		return permission.OpCancel, nil
	default:
		return "", errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Bulk action is invalid",
		)
	}
}

func workerByIDRequest(
	fieldRequested func(string) bool,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) workerports.GetWorkerByIDRequest {
	includeProfile := fieldRequested("profile")
	return workerports.GetWorkerByIDRequest{
		ID:                  id,
		TenantInfo:          tenantInfo,
		IncludeProfile:      includeProfile,
		IncludeState:        fieldRequested("state"),
		IncludeFleetCode:    fieldRequested("fleetCode"),
		IncludeManager:      fieldRequested("manager"),
		IncludeLicenseState: includeProfile && fieldRequested("profile.licenseState"),
	}
}

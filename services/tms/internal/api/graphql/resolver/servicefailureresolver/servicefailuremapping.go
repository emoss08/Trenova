package servicefailureresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/loaderfields"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func serviceFailureConnectionToModel(
	result *pagination.CursorListResult[*servicefailure.ServiceFailure],
) (*gqlmodel.ServiceFailureConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *servicefailure.ServiceFailure, cursor string) *gqlmodel.ServiceFailureEdge {
			return &gqlmodel.ServiceFailureEdge{
				Node:   node,
				Cursor: cursor,
			}
		},
		func(edge *gqlmodel.ServiceFailureEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.ServiceFailureConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func failureUser(ctx context.Context, loaded *tenant.User, id *pulid.ID) (*tenant.User, error) {
	if loaded != nil {
		return loaded, nil
	}

	return loaderfields.LoadUser(ctx, pulid.ConvertFromPtr(id))
}

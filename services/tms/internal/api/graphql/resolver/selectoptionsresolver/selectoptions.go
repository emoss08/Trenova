package selectoptionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *QueryResolver) resolveSelectOptions(
	ctx context.Context,
	input gqlmodel.SelectOptionsInput,
	registry map[gqlmodel.SelectOptionResource]base.SelectOptionRegistryEntry,
) (*gqlmodel.SelectOptionConnection, error) {
	entry, ok := registry[input.Resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"Select option resource is not supported",
		)
	}

	authCtx, err := r.RequireAuthContext(ctx)
	if err != nil {
		return nil, err
	}

	req, err := selectOptionsRequestFromInput(input, authCtx, entry)
	if err != nil {
		return nil, err
	}

	return entry.Resolve(ctx, req)
}

func selectOptionsRequestFromInput(
	input gqlmodel.SelectOptionsInput,
	authCtx *authctx.AuthContext,
	entry base.SelectOptionRegistryEntry,
) (*base.SelectOptionsRequest, error) {
	ids, err := entry.IDs(input.Ids)
	if err != nil {
		return nil, err
	}

	first := pagination.DefaultLimit
	if input.First != nil {
		first = pagination.ClampLimit(*input.First)
	}

	offset := pagination.DefaultOffset
	if input.Offset != nil {
		offset = pagination.ClampOffset(*input.Offset)
	}

	tenant := base.TenantInfo(authCtx)
	return &base.SelectOptionsRequest{
		TenantInfo: tenant,
		IDs:        ids,
		Filters:    selectOptionFilters(input.Filters),
		SelectQuery: &pagination.SelectQueryRequest{
			TenantInfo: tenant,
			Pagination: pagination.Info{
				Limit:  first,
				Offset: offset,
			},
			Query: base.StringValue(input.Query),
		},
	}, nil
}

func selectOptionFilters(filters map[string]any) map[string]any {
	if len(filters) == 0 {
		return map[string]any{}
	}

	return filters
}

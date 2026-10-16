package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
)

type connectionEntity interface {
	domaintypes.PostgresSearchable
	pagination.CursorValueCarrier
}

type connectionTable struct {
	name     string
	alias    string
	all      string
	orgScope func(*bun.SelectQuery, pagination.TenantInfo) *bun.SelectQuery
}

func scopeMFAAuthenticatorOrganization(
	q *bun.SelectQuery,
	tenant pagination.TenantInfo,
) *bun.SelectQuery {
	return q.Where(iamcol.MFAAuthenticatorColumns.OrganizationID.Eq(), tenant.OrgID)
}

func applyConnectionFilters[T connectionEntity](
	q *bun.SelectQuery,
	table connectionTable,
	filter *pagination.QueryOptions,
	model T,
) *bun.SelectQuery {
	if table.orgScope == nil {
		return querybuilder.ApplyFiltersWithoutSort(q, table.alias, filter, model)
	}

	q = querybuilder.ApplyFiltersWithoutSortOrTenantScope(q, table.alias, filter, model)
	return table.orgScope(q, filter.TenantInfo)
}

func applyConnectionCursorFilters[T connectionEntity](
	q *bun.SelectQuery,
	table connectionTable,
	req *iamports.ListIAMConnectionRequest,
	model T,
) (*bun.SelectQuery, error) {
	if table.orgScope == nil {
		return querybuilder.ApplyCursorFilters(q, table.alias, req.Filter, req.Cursor, model)
	}

	q, err := querybuilder.ApplyCursorFiltersWithoutTenantScope(
		q,
		table.alias,
		req.Filter,
		req.Cursor,
		model,
	)
	if err != nil {
		return nil, err
	}
	return table.orgScope(q, req.Filter.TenantInfo), nil
}

func listIAMConnection[T connectionEntity](
	ctx context.Context,
	db bun.IDB,
	table connectionTable,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[T], error) {
	var model T

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := db.NewSelect().
			Model(model).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return applyConnectionFilters(sq, table, req.Filter, model)
			}).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", table.name, err)
		}
		totalCount = new(int(total))
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[T]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]T) *bun.SelectQuery {
			return db.NewSelect().Model(entities).ColumnExpr(table.all)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return applyConnectionCursorFilters(sq, table, req, model)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table.name, err)
	}

	return result, nil
}

func (r *repository) ListAuthEventsConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.AuthEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.AuthEvent], error) {
			return listIAMConnection[*iam.AuthEvent](
				ctx,
				r.db.DBForContext(ctx),
				connectionTable{
					name:  "auth events",
					alias: iamcol.AuthEventTable.Alias,
					all:   iamcol.AuthEventTable.All(),
				},
				req,
			)
		},
	)
}

func (r *repository) ListRiskDecisionsConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.RiskDecision], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.RiskDecision], error) {
			return listIAMConnection[*iam.RiskDecision](
				ctx,
				r.db.DBForContext(ctx),
				connectionTable{
					name:  "risk decisions",
					alias: iamcol.RiskDecisionTable.Alias,
					all:   iamcol.RiskDecisionTable.All(),
				},
				req,
			)
		},
	)
}

func (r *repository) ListExternalIdentitiesConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.ExternalIdentity], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.ExternalIdentity], error) {
			return listIAMConnection[*iam.ExternalIdentity](
				ctx,
				r.db.DBForContext(ctx),
				connectionTable{
					name:  "external identities",
					alias: iamcol.ExternalIdentityTable.Alias,
					all:   iamcol.ExternalIdentityTable.All(),
				},
				req,
			)
		},
	)
}

func (r *repository) ListMFAAuthenticatorsConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.MFAAuthenticator], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.MFAAuthenticator], error) {
			return listIAMConnection[*iam.MFAAuthenticator](
				ctx,
				r.db.DBForContext(ctx),
				connectionTable{
					name:     "mfa authenticators",
					alias:    iamcol.MFAAuthenticatorTable.Alias,
					all:      iamcol.MFAAuthenticatorTable.All(),
					orgScope: scopeMFAAuthenticatorOrganization,
				},
				req,
			)
		},
	)
}

func (r *repository) ListSCIMDirectoriesConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.SCIMDirectory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.SCIMDirectory], error) {
			return listIAMConnection[*iam.SCIMDirectory](
				ctx,
				r.db.DBForContext(ctx),
				connectionTable{
					name:  "scim directories",
					alias: iamcol.SCIMDirectoryTable.Alias,
					all:   iamcol.SCIMDirectoryTable.All(),
				},
				req,
			)
		},
	)
}

package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
)

func listIAMConnection[T any](
	ctx context.Context,
	db *keel.DB,
	name string,
	q *keel.Query[T],
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*T], error) {
	result, err := listquery.Connection(ctx, db, q, &listquery.Request[T]{
		Filter: req.Filter,
		Cursor: req.Cursor,
	})
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", name, err)
	}

	return result, nil
}

func requestTenant(req *iamports.ListIAMConnectionRequest) pagination.TenantInfo {
	if req.Filter == nil {
		return pagination.TenantInfo{}
	}

	return req.Filter.TenantInfo
}

func (r *repository) ListAuthEventsConnection(
	ctx context.Context,
	req *iamports.ListIAMConnectionRequest,
) (*pagination.CursorListResult[*iam.AuthEvent], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.AuthEvent], error) {
			ae := iamdb.AuthEvents
			tenant := requestTenant(req)
			return listIAMConnection(
				ctx,
				r.db.Keel(ctx),
				"auth events",
				ae.Where(
					ae.OrganizationID.Eq(tenant.OrgID),
					ae.BusinessUnitID.Eq(tenant.BuID),
				),
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
			rd := iamdb.RiskDecisions
			tenant := requestTenant(req)
			return listIAMConnection(
				ctx,
				r.db.Keel(ctx),
				"risk decisions",
				rd.Where(
					rd.OrganizationID.Eq(tenant.OrgID),
					rd.BusinessUnitID.Eq(tenant.BuID),
				),
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
			return listIAMConnection(
				ctx,
				r.db.Keel(ctx),
				"external identities",
				iamdb.ExternalIdentities.Query(),
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
			return listIAMConnection(
				ctx,
				r.db.Keel(ctx),
				"mfa authenticators",
				iamdb.MFAAuthenticators.Query(),
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
			return listIAMConnection(
				ctx,
				r.db.Keel(ctx),
				"scim directories",
				iamdb.SCIMDirectories.Query(),
				req,
			)
		},
	)
}

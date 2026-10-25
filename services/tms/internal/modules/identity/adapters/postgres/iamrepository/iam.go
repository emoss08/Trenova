package iamrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultListLimit = 250

var (
	identityProviderErrors = dberror.Mapping{Entity: "Identity provider"}
	scimDirectoryErrors    = dberror.Mapping{Entity: "SCIM directory"}
	scimTokenErrors        = dberror.Mapping{Entity: "SCIM token"}
	accessPolicyErrors     = dberror.Mapping{
		Entity: "Access policy",
		Unique: []dberror.Constraint{
			{
				Name:    "idx_access_policies_name_tenant",
				Field:   "name",
				Message: "Access policy with this name already exists",
			},
			{
				Name:    "idx_access_policies_enabled_scope_tenant",
				Field:   "resource",
				Message: "An enabled access policy with the same resource, operation, and conditions already exists",
			},
		},
	}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) iamports.IAMRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.iam-repository"),
	}
}

func safeLimit(limit int) int {
	if limit <= 0 || limit > defaultListLimit {
		return defaultListLimit
	}
	return limit
}

func forTenant[T any](
	ctx context.Context,
	q *keel.Query[T],
	preds ...keel.Pred,
) *keel.Query[T] {
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

func (r *repository) ListIdentityProviders(
	ctx context.Context,
	req iamports.ListIAMRequest,
) ([]*iam.IdentityProvider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.IdentityProvider, error) {
		idp := iamdb.IdentityProviders
		return idp.Where(idp.Protocol.Eq(iam.IdentityProviderProtocolOIDC)).
			Order(idp.Name.Asc()).
			Limit(safeLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetIdentityProvider(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*iam.IdentityProvider, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.IdentityProvider, error) {
		entity, err := iamdb.IdentityProviders.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, identityProviderErrors.Map(err, id.String())
		}
		return entity, nil
	})
}

func (r *repository) CreateIdentityProvider(
	ctx context.Context,
	entity *iam.IdentityProvider,
) (*iam.IdentityProvider, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.IdentityProvider, error) {
		if err := iamdb.IdentityProviders.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, identityProviderErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) UpdateIdentityProvider(
	ctx context.Context,
	entity *iam.IdentityProvider,
) (*iam.IdentityProvider, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.IdentityProvider, error) {
		idp := iamdb.IdentityProviders
		if err := idp.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			idp.Name,
			idp.Slug,
			idp.Protocol,
			idp.Enabled,
			idp.EnforceSSO,
			idp.AutoProvision,
			idp.AllowFederatedMFA,
			idp.FederatedMFAACRs,
			idp.AllowedDomains,
			idp.AttributeMap,
			idp.OIDCIssuerURL,
			idp.OIDCClientID,
			idp.OIDCClientSecret,
			idp.OIDCRedirectURL,
			idp.OIDCScopes,
		); err != nil {
			return nil, identityProviderErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) DeleteIdentityProvider(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		idp := iamdb.IdentityProviders
		deleted, err := idp.Where(idp.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return identityProviderErrors.Map(err, id.String())
		}
		return dberror.CheckAffected(deleted, identityProviderErrors.Entity, id.String())
	})
}

func (r *repository) ListSCIMDirectories(
	ctx context.Context,
	req *iamports.ListSCIMDirectoryRequest,
) (*pagination.ListResult[*iam.SCIMDirectory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*iam.SCIMDirectory], error) {
			sd := iamdb.SCIMDirectories
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				sd.Query(),
				&listquery.Request[iam.SCIMDirectory]{
					Filter: &pagination.QueryOptions{
						TenantInfo: req.Filter.TenantInfo,
						Pagination: req.Filter.Pagination,
					},
					Then: []keel.Order{sd.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *repository) GetSCIMDirectory(
	ctx context.Context,
	req iamports.GetSCIMDirectoryRequest,
) (*iam.SCIMDirectory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SCIMDirectory, error) {
		entity, err := iamdb.SCIMDirectories.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, scimDirectoryErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) CreateSCIMDirectory(
	ctx context.Context,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMDirectory, error) {
		if err := iamdb.SCIMDirectories.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, scimDirectoryErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) UpdateSCIMDirectory(
	ctx context.Context,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMDirectory, error) {
		sd := iamdb.SCIMDirectories
		if err := sd.Save(ctx, r.db.Keel(ctx), entity, sd.TenantSlug, sd.Enabled); err != nil {
			return nil, scimDirectoryErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) DeleteSCIMDirectory(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		sd := iamdb.SCIMDirectories
		deleted, err := sd.Where(sd.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return scimDirectoryErrors.Map(err, id.String())
		}
		return dberror.CheckAffected(deleted, scimDirectoryErrors.Entity, id.String())
	})
}

func (r *repository) ListSCIMTokens(
	ctx context.Context,
	orgID, directoryID pulid.ID,
) ([]*iam.SCIMToken, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.SCIMToken, error) {
		st := iamdb.SCIMTokens
		return st.Where(st.OrganizationID.Eq(orgID), st.DirectoryID.Eq(directoryID)).
			Order(st.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) CreateSCIMToken(
	ctx context.Context,
	entity *iam.SCIMToken,
) (*iam.SCIMToken, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMToken, error) {
		if err := iamdb.SCIMTokens.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, scimTokenErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) RevokeSCIMToken(
	ctx context.Context,
	orgID, tokenID pulid.ID,
) (*iam.SCIMToken, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMToken, error) {
		st := iamdb.SCIMTokens
		rows, err := st.Where(st.ID.Eq(tokenID), st.OrganizationID.Eq(orgID)).
			UpdateReturning(ctx, r.db.Keel(ctx), st.Status.To(iam.SCIMTokenStatusRevoked))
		if err != nil {
			return nil, scimTokenErrors.Map(err, tokenID.String())
		}
		if err = dberror.CheckAffected(
			int64(len(rows)),
			scimTokenErrors.Entity,
			tokenID.String(),
		); err != nil {
			return nil, err
		}
		return rows[0], nil
	})
}

func (r *repository) ListProvisioningAuditRecords(
	ctx context.Context,
	orgID pulid.ID,
	directoryID pulid.ID,
	limit int,
) ([]*iam.ProvisioningAuditRecord, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.ProvisioningAuditRecord, error) {
		par := iamdb.ProvisioningAuditRecords
		return par.Where(
			par.OrganizationID.Eq(orgID),
			keel.When(directoryID.IsNotNil(), par.DirectoryID.Eq(directoryID)),
		).
			Order(par.CreatedAt.Desc()).
			Limit(safeLimit(limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListAccessPolicies(
	ctx context.Context,
	req iamports.ListIAMRequest,
) ([]*iam.AccessPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AccessPolicy, error) {
		ap := iamdb.AccessPolicies
		return ap.Query().
			Order(ap.Priority.Desc(), ap.CreatedAt.Desc()).
			Limit(safeLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListEnabledAccessPolicies(
	ctx context.Context,
	req iamports.IAMPolicyLookupRequest,
) ([]*iam.AccessPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AccessPolicy, error) {
		ap := iamdb.AccessPolicies
		return forTenant(
			ctx,
			ap.Query(),
			ap.OrganizationID.Eq(req.OrganizationID),
			ap.BusinessUnitID.Eq(req.BusinessUnitID),
			ap.Resource.Eq(req.Resource),
			ap.Operation.Eq(string(req.Operation)),
			ap.Enabled.IsTrue(),
		).
			Order(ap.Priority.Desc(), ap.Effect.Desc(), ap.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListEnabledTenantAccessPolicies(
	ctx context.Context,
	req iamports.IAMTenantPolicyLookupRequest,
) ([]*iam.AccessPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AccessPolicy, error) {
		ap := iamdb.AccessPolicies
		return forTenant(
			ctx,
			ap.Query(),
			ap.OrganizationID.Eq(req.OrganizationID),
			ap.BusinessUnitID.Eq(req.BusinessUnitID),
			ap.Enabled.IsTrue(),
		).
			Order(ap.Priority.Desc(), ap.Effect.Desc(), ap.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) CreateAccessPolicy(
	ctx context.Context,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.AccessPolicy, error) {
		if err := iamdb.AccessPolicies.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, accessPolicyErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) UpdateAccessPolicy(
	ctx context.Context,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.AccessPolicy, error) {
		ap := iamdb.AccessPolicies
		if err := ap.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			ap.Name,
			ap.Resource,
			ap.Operation,
			ap.Effect,
			ap.Priority,
			ap.Conditions,
			ap.Enabled,
		); err != nil {
			return nil, accessPolicyErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) DeleteAccessPolicy(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ap := iamdb.AccessPolicies
		deleted, err := ap.Where(ap.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return accessPolicyErrors.Map(err, id.String())
		}
		return dberror.CheckAffected(deleted, accessPolicyErrors.Entity, id.String())
	})
}

func (r *repository) ListAuthEvents(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.AuthEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AuthEvent, error) {
		ae := iamdb.AuthEvents
		return ae.Where(ae.OrganizationID.Eq(orgID)).
			Order(ae.OccurredAt.Desc()).
			Limit(safeLimit(limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListRiskDecisions(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.RiskDecision, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.RiskDecision, error) {
		rd := iamdb.RiskDecisions
		return rd.Where(rd.OrganizationID.Eq(orgID)).
			Order(rd.CreatedAt.Desc()).
			Limit(safeLimit(limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListExternalIdentities(
	ctx context.Context,
	req iamports.ListIAMRequest,
) ([]*iam.ExternalIdentity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.ExternalIdentity, error) {
		ei := iamdb.ExternalIdentities
		return ei.Query().
			Order(ei.UpdatedAt.Desc()).
			Limit(safeLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListMFAAuthenticators(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.MFAAuthenticator, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.MFAAuthenticator, error) {
		mfa := iamdb.MFAAuthenticators
		return mfa.Where(mfa.OrganizationID.Eq(orgID)).
			Order(mfa.UpdatedAt.Desc()).
			Limit(safeLimit(limit)).
			List(ctx, r.db.Keel(ctx))
	})
}

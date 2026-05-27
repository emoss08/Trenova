package iamrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultListLimit = 250

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.IAMRepository {
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

func (r *repository) ListIdentityProviders(
	ctx context.Context,
	req repositories.ListIAMRequest,
) ([]*iam.IdentityProvider, error) {
	entities := make([]*iam.IdentityProvider, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("idp.organization_id = ?", req.TenantInfo.OrgID).
		Where("idp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("idp.protocol = ?", iam.IdentityProviderProtocolOIDC).
		Order("idp.name ASC").
		Limit(safeLimit(req.Limit)).
		Scan(ctx)
	return entities, err
}

func (r *repository) GetIdentityProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*iam.IdentityProvider, error) {
	entity := new(iam.IdentityProvider)
	if err := r.db.DB().NewSelect().
		Model(entity).
		Where("idp.id = ?", id).
		Where("idp.organization_id = ?", tenantInfo.OrgID).
		Where("idp.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Identity provider")
	}
	return entity, nil
}

func (r *repository) CreateIdentityProvider(
	ctx context.Context,
	entity *iam.IdentityProvider,
) (*iam.IdentityProvider, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateIdentityProvider(
	ctx context.Context,
	entity *iam.IdentityProvider,
) (*iam.IdentityProvider, error) {
	entity.Version++
	res, err := r.db.DB().NewUpdate().
		Model(entity).
		Column(
			"name",
			"slug",
			"protocol",
			"enabled",
			"enforce_sso",
			"auto_provision",
			"allow_federated_mfa",
			"allowed_domains",
			"attribute_map",
			"oidc_issuer_url",
			"oidc_client_id",
			"oidc_client_secret",
			"oidc_redirect_url",
			"oidc_scopes",
			"version",
		).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "Identity provider", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteIdentityProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	res, err := r.db.DB().NewDelete().
		Model((*iam.IdentityProvider)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(res, "Identity provider", id.String())
}

func (r *repository) ListSCIMDirectories(
	ctx context.Context,
	req *repositories.ListSCIMDirectoryRequest,
) (*pagination.ListResult[*iam.SCIMDirectory], error) {
	entities := make([]*iam.SCIMDirectory, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.SCIMDirectoryColumns

	total, err := r.db.DB().NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SCIMDirectoryApplyTenant(req.Filter.TenantInfo)(sq)
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)

	return &pagination.ListResult[*iam.SCIMDirectory]{
		Items: entities,
		Total: total,
	}, err
}

func (r *repository) GetSCIMDirectory(
	ctx context.Context,
	req repositories.GetSCIMDirectoryRequest,
) (*iam.SCIMDirectory, error) {
	entity := new(iam.SCIMDirectory)
	cols := buncolgen.SCIMDirectoryColumns

	if err := r.db.DB().NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SCIMDirectoryScopeTenant(sq, req.TenantInfo).Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SCIM directory")
	}
	return entity, nil
}

func (r *repository) CreateSCIMDirectory(
	ctx context.Context,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateSCIMDirectory(
	ctx context.Context,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	res, err := r.db.DB().NewUpdate().
		Model(entity).
		Column("tenant_slug", "enabled").
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "SCIM directory", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteSCIMDirectory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	res, err := r.db.DB().NewDelete().
		Model((*iam.SCIMDirectory)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(res, "SCIM directory", id.String())
}

func (r *repository) ListSCIMTokens(
	ctx context.Context,
	orgID, directoryID pulid.ID,
) ([]*iam.SCIMToken, error) {
	entities := make([]*iam.SCIMToken, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("st.organization_id = ?", orgID).
		Where("st.directory_id = ?", directoryID).
		Order("st.created_at DESC").
		Scan(ctx)
	return entities, err
}

func (r *repository) CreateSCIMToken(
	ctx context.Context,
	entity *iam.SCIMToken,
) (*iam.SCIMToken, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) RevokeSCIMToken(
	ctx context.Context,
	orgID, tokenID pulid.ID,
) (*iam.SCIMToken, error) {
	entity := &iam.SCIMToken{ID: tokenID, OrganizationID: orgID, Status: iam.SCIMTokenStatusRevoked}
	res, err := r.db.DB().NewUpdate().
		Model(entity).
		Column("status").
		Where("id = ?", tokenID).
		Where("organization_id = ?", orgID).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "SCIM token", tokenID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListSCIMGroupRoleMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	directoryID pulid.ID,
) ([]*iam.SCIMGroupRoleMapping, error) {
	entities := make([]*iam.SCIMGroupRoleMapping, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("sgrm.organization_id = ?", tenantInfo.OrgID).
		Where("sgrm.business_unit_id = ?", tenantInfo.BuID).
		Where("sgrm.directory_id = ?", directoryID).
		Order("sgrm.display_name ASC").
		Scan(ctx)
	return entities, err
}

func (r *repository) CreateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	res, err := r.db.DB().NewUpdate().
		Model(entity).
		Column("external_group_id", "display_name", "role_id").
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		res,
		"SCIM group role mapping",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteSCIMGroupRoleMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	res, err := r.db.DB().NewDelete().
		Model((*iam.SCIMGroupRoleMapping)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(res, "SCIM group role mapping", id.String())
}

func (r *repository) ListProvisioningAuditRecords(
	ctx context.Context,
	orgID pulid.ID,
	directoryID pulid.ID,
	limit int,
) ([]*iam.ProvisioningAuditRecord, error) {
	entities := make([]*iam.ProvisioningAuditRecord, 0)
	query := r.db.DB().NewSelect().
		Model(&entities).
		Where("par.organization_id = ?", orgID).
		Order("par.created_at DESC").
		Limit(safeLimit(limit))
	if directoryID.IsNotNil() {
		query = query.Where("par.directory_id = ?", directoryID)
	}
	err := query.Scan(ctx)
	return entities, err
}

func (r *repository) ListAccessPolicies(
	ctx context.Context,
	req repositories.ListIAMRequest,
) ([]*iam.AccessPolicy, error) {
	entities := make([]*iam.AccessPolicy, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("ap.organization_id = ?", req.TenantInfo.OrgID).
		Where("ap.business_unit_id = ?", req.TenantInfo.BuID).
		Order("ap.priority DESC", "ap.created_at DESC").
		Limit(safeLimit(req.Limit)).
		Scan(ctx)
	return entities, err
}

func (r *repository) ListEnabledAccessPolicies(
	ctx context.Context,
	req repositories.IAMPolicyLookupRequest,
) ([]*iam.AccessPolicy, error) {
	entities := make([]*iam.AccessPolicy, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("ap.organization_id = ?", req.OrganizationID).
		Where("ap.business_unit_id = ?", req.BusinessUnitID).
		Where("ap.resource = ?", req.Resource).
		Where("ap.operation = ?", string(req.Operation)).
		Where("ap.enabled = TRUE").
		Order("ap.priority DESC", "ap.effect DESC", "ap.created_at ASC").
		Scan(ctx)
	return entities, err
}

func (r *repository) CreateAccessPolicy(
	ctx context.Context,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateAccessPolicy(
	ctx context.Context,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	res, err := r.db.DB().NewUpdate().
		Model(entity).
		Column("name", "resource", "operation", "effect", "priority", "conditions", "enabled").
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(res, "Access policy", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) DeleteAccessPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	res, err := r.db.DB().NewDelete().
		Model((*iam.AccessPolicy)(nil)).
		Where("id = ?", id).
		Where("organization_id = ?", tenantInfo.OrgID).
		Where("business_unit_id = ?", tenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(res, "Access policy", id.String())
}

func (r *repository) ListAuthEvents(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.AuthEvent, error) {
	entities := make([]*iam.AuthEvent, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("ae.organization_id = ?", orgID).
		Order("ae.occurred_at DESC").
		Limit(safeLimit(limit)).
		Scan(ctx)
	return entities, err
}

func (r *repository) ListRiskDecisions(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.RiskDecision, error) {
	entities := make([]*iam.RiskDecision, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("rd.organization_id = ?", orgID).
		Order("rd.created_at DESC").
		Limit(safeLimit(limit)).
		Scan(ctx)
	return entities, err
}

func (r *repository) ListExternalIdentities(
	ctx context.Context,
	req repositories.ListIAMRequest,
) ([]*iam.ExternalIdentity, error) {
	entities := make([]*iam.ExternalIdentity, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("extid.organization_id = ?", req.TenantInfo.OrgID).
		Where("extid.business_unit_id = ?", req.TenantInfo.BuID).
		Order("extid.updated_at DESC").
		Limit(safeLimit(req.Limit)).
		Scan(ctx)
	return entities, err
}

func (r *repository) ListMFAAuthenticators(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.MFAAuthenticator, error) {
	entities := make([]*iam.MFAAuthenticator, 0)
	err := r.db.DB().NewSelect().
		Model(&entities).
		Where("mfa.organization_id = ?", orgID).
		Order("mfa.updated_at DESC").
		Limit(safeLimit(limit)).
		Scan(ctx)
	return entities, err
}

var _ bun.IDB

package ssoconfigrepository

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const resolveProviderScopeReason = "resolve an SSO provider named in a public URL before its tenant is known"

var ssoConfigErrors = dberror.Mapping{Entity: "SSOConfig"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.SSOConfigRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sso-config-repository"),
	}
}

func forTenant[T any](
	ctx context.Context,
	q *keel.Query[*T],
	preds ...keel.Pred,
) *keel.Query[*T] {
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

func enabledOIDC() keel.Pred {
	idp := iamdb.IdentityProviders
	return keel.And(idp.Enabled.IsTrue(), idp.Protocol.Eq(iam.IdentityProviderProtocolOIDC))
}

func (r *repository) ListEnabledByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
) ([]*tenant.SSOConfig, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.SSOConfig, error) {
		idp := iamdb.IdentityProviders
		providers, err := forTenant(
			ctx,
			idp.Query(),
			idp.OrganizationID.Eq(organizationID),
			enabledOIDC(),
		).
			Order(idp.Name.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		configs := make([]*tenant.SSOConfig, 0, len(providers))
		for _, provider := range providers {
			configs = append(configs, identityProviderToSSOConfig(provider))
		}
		return configs, nil
	})
}

func (r *repository) GetEnabledByID(
	ctx context.Context,
	providerID pulid.ID,
) (*tenant.SSOConfig, error) {
	ctx = dbscope.WithSystem(ctx, resolveProviderScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SSOConfig, error) {
		idp := iamdb.IdentityProviders
		provider, err := idp.AllTenants().
			Where(idp.ID.Eq(providerID), enabledOIDC()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, ssoConfigErrors.Map(err, providerID.String())
		}

		return identityProviderToSSOConfig(provider), nil
	})
}

func (r *repository) GetByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
	provider tenant.SSOProvider,
) (*tenant.SSOConfig, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SSOConfig, error) {
		entity, err := r.find(ctx, organizationID, provider)
		if err != nil {
			return nil, ssoConfigErrors.Map(err, organizationID.String())
		}

		return entity, nil
	})
}

func (r *repository) find(
	ctx context.Context,
	organizationID pulid.ID,
	provider tenant.SSOProvider,
) (*tenant.SSOConfig, error) {
	sc := tenantdb.SSOConfigs
	return forTenant(
		ctx,
		sc.Query(),
		sc.OrganizationID.Eq(organizationID),
		sc.Provider.Eq(provider),
	).One(ctx, r.db.Keel(ctx))
}

func (r *repository) GetEnabledByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
	provider tenant.SSOProvider,
) (*tenant.SSOConfig, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SSOConfig, error) {
		idp := iamdb.IdentityProviders
		entity, err := forTenant(
			ctx,
			idp.Query(),
			idp.OrganizationID.Eq(organizationID),
			idp.Slug.Eq(providerSlug(provider)),
			enabledOIDC(),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, ssoConfigErrors.Map(err, organizationID.String())
		}

		return identityProviderToSSOConfig(entity), nil
	})
}

func (r *repository) Save(
	ctx context.Context,
	entity *tenant.SSOConfig,
) (*tenant.SSOConfig, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.SSOConfig, error) {
		sc := tenantdb.SSOConfigs
		db := r.db.Keel(ctx)

		existing, err := r.find(ctx, entity.OrganizationID, entity.Provider)
		switch {
		case dberror.IsNotFoundError(err):
			if err = sc.Create(ctx, db, entity); err != nil {
				return nil, ssoConfigErrors.Map(err, entity.ID.String())
			}
		case err != nil:
			return nil, err
		default:
			entity.ID = existing.ID
			entity.Version = existing.Version
			if err = sc.Save(
				ctx,
				db,
				entity,
				sc.Name,
				sc.Provider,
				sc.Protocol,
				sc.Enabled,
				sc.EnforceSSO,
				sc.AutoProvision,
				sc.AllowedDomains,
				sc.AttributeMap,
				sc.OIDCIssuerURL,
				sc.OIDCClientID,
				sc.OIDCClientSecret,
				sc.OIDCRedirectURL,
				sc.OIDCScopes,
			); err != nil {
				return nil, ssoConfigErrors.Map(err, entity.ID.String())
			}
		}

		if err = r.upsertIdentityProvider(ctx, entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) upsertIdentityProvider(ctx context.Context, cfg *tenant.SSOConfig) error {
	idp := iamdb.IdentityProviders
	provider := &iam.IdentityProvider{
		ID:                cfg.ID,
		OrganizationID:    cfg.OrganizationID,
		BusinessUnitID:    cfg.BusinessUnitID,
		Name:              cfg.Name,
		Slug:              providerSlug(cfg.Provider),
		Protocol:          iam.IdentityProviderProtocolOIDC,
		Enabled:           cfg.Enabled,
		EnforceSSO:        cfg.EnforceSSO,
		AutoProvision:     cfg.AutoProvision,
		AllowFederatedMFA: true,
		AllowedDomains:    cfg.AllowedDomains,
		AttributeMap:      cfg.AttributeMap,
		OIDCIssuerURL:     cfg.OIDCIssuerURL,
		OIDCClientID:      cfg.OIDCClientID,
		OIDCClientSecret:  cfg.OIDCClientSecret,
		OIDCRedirectURL:   cfg.OIDCRedirectURL,
		OIDCScopes:        cfg.OIDCScopes,
		Version:           cfg.Version,
	}

	_, err := idp.Upsert(
		ctx,
		r.db.Keel(ctx),
		[]*iam.IdentityProvider{provider},
		keel.OnConflict(idp.OrganizationID, idp.Slug).
			Update(
				idp.Name,
				idp.Protocol,
				idp.Enabled,
				idp.EnforceSSO,
				idp.AutoProvision,
				idp.AllowedDomains,
				idp.AttributeMap,
				idp.OIDCIssuerURL,
				idp.OIDCClientID,
				idp.OIDCClientSecret,
				idp.OIDCRedirectURL,
				idp.OIDCScopes,
			).
			Set(idp.Version.ToExpr(idp.Version.Excluded())),
	)
	return err
}

func identityProviderToSSOConfig(provider *iam.IdentityProvider) *tenant.SSOConfig {
	return &tenant.SSOConfig{
		ID:                provider.ID,
		OrganizationID:    provider.OrganizationID,
		BusinessUnitID:    provider.BusinessUnitID,
		Name:              provider.Name,
		Provider:          providerFromSlug(provider.Slug),
		Protocol:          tenant.SSOProtocolOIDC,
		Enabled:           provider.Enabled,
		EnforceSSO:        provider.EnforceSSO,
		AutoProvision:     provider.AutoProvision,
		AllowedDomains:    provider.AllowedDomains,
		AttributeMap:      provider.AttributeMap,
		OIDCIssuerURL:     provider.OIDCIssuerURL,
		OIDCClientID:      provider.OIDCClientID,
		OIDCClientSecret:  provider.OIDCClientSecret,
		OIDCRedirectURL:   provider.OIDCRedirectURL,
		OIDCScopes:        provider.OIDCScopes,
		AllowFederatedMFA: provider.AllowFederatedMFA,
		FederatedMFAACRs:  provider.FederatedMFAACRs,
		Version:           provider.Version,
		CreatedAt:         provider.CreatedAt,
		UpdatedAt:         provider.UpdatedAt,
	}
}

func providerSlug(provider tenant.SSOProvider) string {
	return strings.ToLower(string(provider))
}

func providerFromSlug(slug string) tenant.SSOProvider {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case providerSlug(tenant.SSOProviderAzureAD):
		return tenant.SSOProviderAzureAD
	case providerSlug(tenant.SSOProviderOkta):
		return tenant.SSOProviderOkta
	default:
		return tenant.SSOProvider(slug)
	}
}

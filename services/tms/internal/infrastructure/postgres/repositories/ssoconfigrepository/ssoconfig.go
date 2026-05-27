package ssoconfigrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.SSOConfigRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sso-config-repository"),
	}
}

func (r *repository) ListEnabledByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
) ([]*tenant.SSOConfig, error) {
	providers := make([]*iam.IdentityProvider, 0)
	err := r.db.DB().
		NewSelect().
		Model(&providers).
		Where("idp.organization_id = ?", organizationID).
		Where("idp.enabled = TRUE").
		Where("idp.protocol = ?", iam.IdentityProviderProtocolOIDC).
		Order("idp.name ASC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	configs := make([]*tenant.SSOConfig, 0, len(providers))
	for _, provider := range providers {
		configs = append(configs, identityProviderToSSOConfig(provider))
	}
	return configs, nil
}

func (r *repository) GetEnabledByID(
	ctx context.Context,
	providerID pulid.ID,
) (*tenant.SSOConfig, error) {
	provider := new(iam.IdentityProvider)
	if err := r.db.DB().
		NewSelect().
		Model(provider).
		Where("idp.id = ?", providerID).
		Where("idp.enabled = TRUE").
		Where("idp.protocol = ?", iam.IdentityProviderProtocolOIDC).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSOConfig")
	}

	return identityProviderToSSOConfig(provider), nil
}

func (r *repository) GetByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
	provider tenant.SSOProvider,
) (*tenant.SSOConfig, error) {
	entity := new(tenant.SSOConfig)
	if err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("ssoc.organization_id = ?", organizationID).
		Where("ssoc.provider = ?", provider).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSOConfig")
	}

	return entity, nil
}

func (r *repository) GetEnabledByOrganizationID(
	ctx context.Context,
	organizationID pulid.ID,
	provider tenant.SSOProvider,
) (*tenant.SSOConfig, error) {
	entity := new(iam.IdentityProvider)
	if err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("idp.organization_id = ?", organizationID).
		Where("idp.slug = ?", providerSlug(provider)).
		Where("idp.enabled = TRUE").
		Where("idp.protocol = ?", iam.IdentityProviderProtocolOIDC).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSOConfig")
	}

	return identityProviderToSSOConfig(entity), nil
}

func (r *repository) Save(
	ctx context.Context,
	entity *tenant.SSOConfig,
) (*tenant.SSOConfig, error) {
	existing := new(tenant.SSOConfig)
	err := r.db.DB().
		NewSelect().
		Model(existing).
		Where("ssoc.organization_id = ?", entity.OrganizationID).
		Where("ssoc.provider = ?", entity.Provider).
		Scan(ctx)
	if err != nil && !dberror.IsNotFoundError(err) {
		return nil, err
	}

	if dberror.IsNotFoundError(err) {
		if _, err = r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
			return nil, err
		}
		if err = r.upsertIdentityProvider(ctx, entity); err != nil {
			return nil, err
		}
		return entity, nil
	}

	entity.ID = existing.ID
	entity.Version = existing.Version + 1

	if _, err = r.db.DB().
		NewUpdate().
		Model(entity).
		Column(
			"name",
			"provider",
			"protocol",
			"enabled",
			"enforce_sso",
			"auto_provision",
			"allowed_domains",
			"attribute_map",
			"oidc_issuer_url",
			"oidc_client_id",
			"oidc_client_secret",
			"oidc_redirect_url",
			"oidc_scopes",
			"version",
		).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("id = ?", entity.ID).
		Exec(ctx); err != nil {
		return nil, err
	}

	if err = r.upsertIdentityProvider(ctx, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) upsertIdentityProvider(ctx context.Context, cfg *tenant.SSOConfig) error {
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
		CreatedAt:         cfg.CreatedAt,
		UpdatedAt:         cfg.UpdatedAt,
	}

	_, err := r.db.DB().NewInsert().
		Model(provider).
		On(`CONFLICT ("organization_id", "slug") DO UPDATE`).
		Set("name = EXCLUDED.name").
		Set("protocol = EXCLUDED.protocol").
		Set("enabled = EXCLUDED.enabled").
		Set("enforce_sso = EXCLUDED.enforce_sso").
		Set("auto_provision = EXCLUDED.auto_provision").
		Set("allow_federated_mfa = EXCLUDED.allow_federated_mfa").
		Set("allowed_domains = EXCLUDED.allowed_domains").
		Set("attribute_map = EXCLUDED.attribute_map").
		Set("oidc_issuer_url = EXCLUDED.oidc_issuer_url").
		Set("oidc_client_id = EXCLUDED.oidc_client_id").
		Set("oidc_client_secret = EXCLUDED.oidc_client_secret").
		Set("oidc_redirect_url = EXCLUDED.oidc_redirect_url").
		Set("oidc_scopes = EXCLUDED.oidc_scopes").
		Set("version = EXCLUDED.version").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func identityProviderToSSOConfig(provider *iam.IdentityProvider) *tenant.SSOConfig {
	return &tenant.SSOConfig{
		ID:               provider.ID,
		OrganizationID:   provider.OrganizationID,
		BusinessUnitID:   provider.BusinessUnitID,
		Name:             provider.Name,
		Provider:         providerFromSlug(provider.Slug),
		Protocol:         tenant.SSOProtocolOIDC,
		Enabled:          provider.Enabled,
		EnforceSSO:       provider.EnforceSSO,
		AutoProvision:    provider.AutoProvision,
		AllowedDomains:   provider.AllowedDomains,
		AttributeMap:     provider.AttributeMap,
		OIDCIssuerURL:    provider.OIDCIssuerURL,
		OIDCClientID:     provider.OIDCClientID,
		OIDCClientSecret: provider.OIDCClientSecret,
		OIDCRedirectURL:  provider.OIDCRedirectURL,
		OIDCScopes:       provider.OIDCScopes,
		Version:          provider.Version,
		CreatedAt:        provider.CreatedAt,
		UpdatedAt:        provider.UpdatedAt,
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

var _ bun.IDB

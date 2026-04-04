package ssoconfigrepository

import (
	"context"

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
	entity := new(tenant.SSOConfig)
	if err := r.db.DB().
		NewSelect().
		Model(entity).
		Where("ssoc.organization_id = ?", organizationID).
		Where("ssoc.provider = ?", provider).
		Where("ssoc.enabled = TRUE").
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSOConfig")
	}

	return entity, nil
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

	return entity, nil
}

var _ bun.IDB

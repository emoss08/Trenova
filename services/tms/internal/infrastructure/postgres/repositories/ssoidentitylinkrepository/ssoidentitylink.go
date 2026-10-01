package ssoidentitylinkrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.SSOIdentityLinkRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sso-identity-link-repository"),
	}
}

func (r *repository) GetBySubject(
	ctx context.Context,
	req repositories.GetSSOIdentityLinkBySubjectRequest,
) (*tenant.SSOIdentityLink, error) {
	cols := buncolgen.SSOIdentityLinkColumns
	entity := new(tenant.SSOIdentityLink)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.SSOConfigID.Eq(), req.SSOConfigID).
		Where(cols.Issuer.Eq(), req.Issuer).
		Where(cols.Subject.Eq(), req.Subject).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSO identity link")
	}

	return entity, nil
}

func (r *repository) GetByUser(
	ctx context.Context,
	req repositories.GetSSOIdentityLinkByUserRequest,
) (*tenant.SSOIdentityLink, error) {
	cols := buncolgen.SSOIdentityLinkColumns
	entity := new(tenant.SSOIdentityLink)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.SSOConfigID.Eq(), req.SSOConfigID).
		Where(cols.Issuer.Eq(), req.Issuer).
		Where(cols.UserID.Eq(), req.UserID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "SSO identity link")
	}

	return entity, nil
}

func (r *repository) Create(ctx context.Context, link *tenant.SSOIdentityLink) error {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(link).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create sso identity link", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) RecordLogin(
	ctx context.Context,
	link *tenant.SSOIdentityLink,
	at int64,
) error {
	cols := buncolgen.SSOIdentityLinkColumns
	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*tenant.SSOIdentityLink)(nil)).
		Set(cols.LastLoginAt.Set(), at).
		Set(cols.UpdatedAt.Set(), at).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.SSOIdentityLinkScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: link.OrganizationID,
				BuID:  link.BusinessUnitID,
			}).Where(cols.ID.Eq(), link.ID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckFound(result, "SSO identity link")
}

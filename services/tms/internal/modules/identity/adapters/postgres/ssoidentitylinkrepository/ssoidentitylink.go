package ssoidentitylinkrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var linkErrors = dberror.Mapping{Entity: "SSO identity link"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.SSOIdentityLinkRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.sso-identity-link-repository"),
	}
}

func links(ctx context.Context, preds ...keel.Pred) *keel.Query[*tenant.SSOIdentityLink] {
	q := tenantdb.SSOIdentityLinks.Query()
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

func (r *repository) GetBySubject(
	ctx context.Context,
	req tenantports.GetSSOIdentityLinkBySubjectRequest,
) (*tenant.SSOIdentityLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SSOIdentityLink, error) {
		l := tenantdb.SSOIdentityLinks
		entity, err := links(
			ctx,
			l.SSOConfigID.Eq(req.SSOConfigID),
			l.Issuer.Eq(req.Issuer),
			l.Subject.Eq(req.Subject),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, linkErrors.Map(err, req.Subject)
		}

		return entity, nil
	})
}

func (r *repository) GetByUser(
	ctx context.Context,
	req tenantports.GetSSOIdentityLinkByUserRequest,
) (*tenant.SSOIdentityLink, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SSOIdentityLink, error) {
		l := tenantdb.SSOIdentityLinks
		entity, err := links(
			ctx,
			l.SSOConfigID.Eq(req.SSOConfigID),
			l.Issuer.Eq(req.Issuer),
			l.UserID.Eq(req.UserID),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, linkErrors.Map(err, req.UserID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(ctx context.Context, link *tenant.SSOIdentityLink) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := tenantdb.SSOIdentityLinks.Create(ctx, r.db.Keel(ctx), link); err != nil {
			r.l.Error("failed to create sso identity link", zap.Error(err))
			return linkErrors.Map(err, link.ID.String())
		}

		return nil
	})
}

func (r *repository) RecordLogin(
	ctx context.Context,
	link *tenant.SSOIdentityLink,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		l := tenantdb.SSOIdentityLinks
		affected, err := links(
			ctx,
			l.ID.Eq(link.ID),
			l.OrganizationID.Eq(link.OrganizationID),
			l.BusinessUnitID.Eq(link.BusinessUnitID),
		).Update(ctx, r.db.Keel(ctx), l.LastLoginAt.To(at), l.UpdatedAt.To(at))
		if err != nil {
			return err
		}

		return dberror.CheckFoundCount(affected, linkErrors.Entity)
	})
}

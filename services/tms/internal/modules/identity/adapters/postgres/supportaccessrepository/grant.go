package supportaccessrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess/supportaccessdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entryLookupReason  = "find the live support grant a verified support assertion names"
	expirySweepReason  = "record the support grants that ended on their own, across organizations"
	entryOrgReason     = "find the business unit of the organization a verified support assertion names"
	grantLockKeyPrefix = "support-access-grants:"
)

var grantErrors = dberror.Mapping{Entity: "Support access grant"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type grantRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewGrantRepository(p Params) supportaccessports.GrantRepository {
	return &grantRepository{db: p.DB, l: p.Logger.Named("postgres.support-access-grant-repository")}
}

func live(now int64) keel.Pred {
	g := supportaccessdb.Grants
	return keel.And(g.RevokedAt.IsNull(), g.ExpiresAt.Gt(now))
}

func withActors(q keel.Query[*supportaccess.Grant]) keel.Query[*supportaccess.Grant] {
	g := supportaccessdb.Grants
	return q.Load(g.GrantedBy(), g.RevokedBy())
}

func (r *grantRepository) Create(
	ctx context.Context,
	req *supportaccessports.CreateGrantRequest,
) (*supportaccess.Grant, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		db := r.db.Keel(ctx)
		if err := keel.LockKey(ctx, db, grantLockKeyPrefix+req.Grant.OrganizationID.String()); err != nil {
			return nil, fmt.Errorf("serialize support grants: %w", err)
		}

		g := supportaccessdb.Grants
		held, err := g.Where(live(req.Now)).
			Order(g.ExpiresAt.Desc()).
			Limit(supportaccess.MaxLiveGrants+1).
			List(ctx, db)
		if err != nil {
			return nil, grantErrors.Map(err, "")
		}
		for _, grant := range held {
			if grant.StaffEmail == req.Grant.StaffEmail {
				return nil, errortypes.NewValidationError(
					"staffEmail",
					errortypes.ErrDuplicate,
					"This staff member already has access. Revoke it to grant different access",
				)
			}
		}
		if len(held) >= supportaccess.MaxLiveGrants {
			return nil, errortypes.NewValidationError(
				"staffEmail",
				errortypes.ErrInvalid,
				"At most {0} support grants may be live at once. Revoke one first",
				supportaccess.MaxLiveGrants,
			)
		}

		if err = tenantdb.Users.Create(ctx, db, req.Principal); err != nil {
			return nil, fmt.Errorf("create the support principal: %w", err)
		}
		if err = g.Create(ctx, db, req.Grant); err != nil {
			return nil, grantErrors.Map(err, "")
		}
		return req.Grant, nil
	})
}

func (r *grantRepository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*supportaccess.Grant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		g := supportaccessdb.Grants
		grant, err := withActors(g.Where(g.ID.Eq(id))).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, grantErrors.Map(err, id.String())
		}
		return grant, nil
	})
}

func (r *grantRepository) List(
	ctx context.Context,
	req *supportaccessports.ListGrantsRequest,
) (*pagination.CursorListResult[*supportaccess.Grant], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*supportaccess.Grant], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				supportaccessdb.Grants.Query(),
				&listquery.Request[supportaccess.Grant]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withActors,
				},
			)
		},
	)
}

func (r *grantRepository) Revoke(
	ctx context.Context,
	grant *supportaccess.Grant,
	now int64,
) (*supportaccess.Grant, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		g := supportaccessdb.Grants
		rows, err := g.Where(g.ID.Eq(grant.ID), live(now)).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			g.RevokedAt.To(now),
			g.RevokedByID.To(grant.RevokedByID),
			g.RevokeReason.To(grant.RevokeReason),
		)
		if err != nil {
			return nil, grantErrors.Map(err, grant.ID.String())
		}
		if len(rows) != 1 {
			return nil, supportaccess.ErrGrantNotFound
		}
		return rows[0], nil
	})
}

func (r *grantRepository) ByPrincipal(
	ctx context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) (*supportaccess.Grant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		g := supportaccessdb.Grants
		grant, err := g.Where(g.PrincipalUserID.Eq(userID)).One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, supportaccess.ErrGrantNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("read the grant of a support principal: %w", err)
		}
		return grant, nil
	})
}

func (r *grantRepository) FindLiveForEntry(
	ctx context.Context,
	req *supportaccessports.FindLiveGrantRequest,
) (*supportaccess.Grant, error) {
	ctx = dbscope.WithSystem(ctx, entryLookupReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.Grant, error) {
		g := supportaccessdb.Grants
		grant, err := g.Where(
			g.OrganizationID.Eq(req.OrganizationID),
			g.StaffEmail.Eq(req.StaffEmail),
			live(req.Now),
		).
			AllTenants().
			Order(g.ExpiresAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, supportaccess.ErrGrantNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("find a live support grant: %w", err)
		}
		return grant, nil
	})
}

func (r *grantRepository) OrganizationBusinessUnit(
	ctx context.Context,
	orgID pulid.ID,
) (pulid.ID, error) {
	ctx = dbscope.WithSystem(ctx, entryOrgReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		o := tenantdb.Organizations
		org, err := o.Where(o.ID.Eq(orgID)).AllTenants().One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return pulid.Nil, supportaccess.ErrGrantNotFound
		}
		if err != nil {
			return pulid.Nil, fmt.Errorf("read the organization of a support assertion: %w", err)
		}
		return org.BusinessUnitID, nil
	})
}

func (r *grantRepository) MarkEntered(
	ctx context.Context,
	req *supportaccessports.MarkEnteredRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		g := supportaccessdb.Grants
		_, err := g.Where(g.ID.Eq(req.Grant.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			g.LastEnteredAt.To(req.At),
			g.LastStaffName.To(req.StaffName),
		)
		if err != nil {
			return fmt.Errorf("record a support entry: %w", err)
		}
		return nil
	})
}

func (r *grantRepository) ListUnrecordedExpiries(
	ctx context.Context,
	now int64,
	limit int,
) ([]*supportaccess.Grant, error) {
	ctx = dbscope.WithSystem(ctx, expirySweepReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*supportaccess.Grant, error) {
		g := supportaccessdb.Grants
		rows, err := g.Where(
			g.RevokedAt.IsNull(),
			g.ExpiryRecordedAt.IsNull(),
			g.ExpiresAt.Lte(now),
		).
			AllTenants().
			Order(g.ExpiresAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list expired support grants: %w", err)
		}
		return rows, nil
	})
}

func (r *grantRepository) MarkExpiryRecorded(
	ctx context.Context,
	grant *supportaccess.Grant,
	at int64,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		g := supportaccessdb.Grants
		n, err := g.Where(
			g.ID.Eq(grant.ID),
			g.RevokedAt.IsNull(),
			g.ExpiryRecordedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), g.ExpiryRecordedAt.To(at))
		if err != nil {
			return false, fmt.Errorf("record a support grant's expiry: %w", err)
		}
		return n == 1, nil
	})
}

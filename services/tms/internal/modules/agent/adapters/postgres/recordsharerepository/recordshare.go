package recordsharerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare/recordsharedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// maxShares bounds one record's share list: a record is shared with a team,
// not with an organization.
const maxShares = 200

var shareErrors = dberror.Mapping{Entity: "Share"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) recordshareports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.record-share-repository"),
	}
}

func (r *repository) Upsert(ctx context.Context, shares []*recordshare.Share) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(shares) == 0 {
			return nil
		}

		s := recordsharedb.Shares
		_, err := s.Upsert(ctx, r.db.Keel(ctx), shares,
			keel.OnConflict(s.OrganizationID, s.BusinessUnitID, s.RecordID, s.SharedWithID).
				Update(s.SharedByID, s.Note, s.Path, s.LastSharedAt, s.RevokedAt, s.RevokedByID).
				Set(s.ShareCount.Increment(1)),
		)
		if err != nil {
			r.l.Error("failed to upsert record shares", zap.Error(err))
			return shareErrors.Map(err, "")
		}

		return nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req *recordshareports.ShareTarget,
) (*recordshare.Share, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recordshare.Share, error) {
		s := recordsharedb.Shares
		share, err := s.Where(s.ID.Eq(req.ShareID), s.RecordID.Eq(req.RecordID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, shareErrors.Map(err, "")
		}

		return share, nil
	})
}

func (r *repository) Revoke(ctx context.Context, share *recordshare.Share) (bool, error) {
	if share.RevokedAt == nil {
		return false, nil
	}
	revokedAt := *share.RevokedAt

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		s := recordsharedb.Shares
		revoked, err := s.Where(s.ID.Eq(share.ID), s.RevokedAt.IsNull()).
			Update(ctx, r.db.Keel(ctx),
				s.RevokedAt.To(revokedAt),
				s.RevokedByID.To(share.RevokedByID),
			)
		if err != nil {
			r.l.Error("failed to revoke a record share", zap.Error(err))
			return false, shareErrors.Map(err, "")
		}

		return revoked > 0, nil
	})
}

func (r *repository) ListByRecord(
	ctx context.Context,
	req *recordshareports.RecordRequest,
) ([]*recordshare.Share, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*recordshare.Share, error) {
		s := recordsharedb.Shares
		shares, err := s.Where(s.RecordID.Eq(req.RecordID), s.RevokedAt.IsNull()).
			Load(userSummary(s.SharedWith()), userSummary(s.SharedBy())).
			Order(s.LastSharedAt.Desc(), s.ID.Asc()).
			Limit(maxShares).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list record shares", zap.Error(err))
			return nil, err
		}
		if shares == nil {
			return []*recordshare.Share{}, nil
		}

		return shares, nil
	})
}

func (r *repository) SharedWith(
	ctx context.Context,
	req *recordshareports.SharedWithRequest,
) ([]pulid.ID, error) {
	if req.UserID.IsNil() || len(req.RecordIDs) == 0 {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		s := recordsharedb.Shares
		shares, err := s.Where(
			s.RecordID.In(req.RecordIDs...),
			s.SharedWithID.Eq(req.UserID),
			s.RevokedAt.IsNull(),
		).Limit(len(req.RecordIDs)).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to read the records shared with a person", zap.Error(err))
			return nil, err
		}

		ids := make([]pulid.ID, 0, len(shares))
		for _, share := range shares {
			ids = append(ids, share.RecordID)
		}

		return ids, nil
	})
}

func userSummary(u *tenantdb.UsersTable) keel.Query[*tenant.User] {
	return u.Pick(
		u.ID,
		u.Name,
		u.Username,
		u.EmailAddress,
		u.Status,
		u.ProfilePicURL,
		u.ThumbnailURL,
	)
}

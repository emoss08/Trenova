package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentprofilecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

type profileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProfiles(p Params) agentprofileports.ProfileRepository {
	return &profileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-profile-repository"),
	}
}

// scopeProfileSubject narrows to one profile. A nil person must match IS NULL,
// so the organization's profile and a person's never read each other's rows.
func scopeProfileSubject(sq *bun.SelectQuery, ref agentprofileports.SubjectRef) *bun.SelectQuery {
	cols := agentprofilecol.ProfileColumns
	sq = agentprofilecol.ProfileScopeTenant(sq, ref.Tenant()).
		Where(cols.Subject.Eq(), ref.Subject)
	if ref.Subject == agentprofile.SubjectUser {
		return sq.Where(cols.UserID.Eq(), ref.PersonID)
	}

	return sq.Where(cols.UserID.IsNull())
}

func scopeProfileSubjectUpdate(
	uq *bun.UpdateQuery,
	ref agentprofileports.SubjectRef,
) *bun.UpdateQuery {
	cols := agentprofilecol.ProfileColumns
	uq = agentprofilecol.ProfileScopeTenantUpdate(uq, ref.Tenant()).
		Where(cols.Subject.Eq(), ref.Subject)
	if ref.Subject == agentprofile.SubjectUser {
		return uq.Where(cols.UserID.Eq(), ref.PersonID)
	}

	return uq.Where(cols.UserID.IsNull())
}

// GetCurrent reads what agents are told, or nil before the first version.
func (r *profileRepository) GetCurrent(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (*agentprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofile.Profile, error) {
		cols := agentprofilecol.ProfileColumns
		entity := new(agentprofile.Profile)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeProfileSubject(sq, ref).
					Where(cols.Status.Eq(), agentprofile.StatusCurrent)
			}).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("get current agent profile: %w", err)
		}

		return entity, nil
	})
}

func (r *profileRepository) GetByID(
	ctx context.Context,
	req agentprofileports.GetProfileRequest,
) (*agentprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofile.Profile, error) {
		cols := agentprofilecol.ProfileColumns
		entity := new(agentprofile.Profile)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentprofilecol.ProfileScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("Profile version not found")
			}

			return nil, fmt.Errorf("get agent profile: %w", err)
		}

		return entity, nil
	})
}

func (r *profileRepository) ListHistory(
	ctx context.Context,
	req agentprofileports.ListProfileHistoryRequest,
) ([]*agentprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentprofile.Profile, error) {
		cols := agentprofilecol.ProfileColumns
		limit := req.Limit
		if limit <= 0 {
			limit = agentprofileports.DefaultHistoryLimit
		}
		limit = min(limit, agentprofileports.MaxHistoryLimit)

		rows := make([]*agentprofile.Profile, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeProfileSubject(sq, req.Ref)
			}).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list agent profile history: %w", err)
		}

		return rows, nil
	})
}

// Publish moves the current version, if any, to Superseded and inserts the
// new one as current, in one transaction: agents are never told two versions,
// and the unique index refuses a second current row should two passes race.
func (r *profileRepository) Publish(
	ctx context.Context,
	entity *agentprofile.Profile,
) (*agentprofile.Profile, error) {
	ref := refOf(entity)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Profile, error) {
		cols := agentprofilecol.ProfileColumns
		dba := r.db.DBForContext(ctx)

		if kept, found, err := r.alreadyKept(ctx, dba, entity); err != nil || found {
			return kept, err
		}

		if _, err := dba.NewUpdate().
			Model((*agentprofile.Profile)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return scopeProfileSubjectUpdate(uq, ref).
					Where(cols.Status.Eq(), agentprofile.StatusCurrent)
			}).
			Set(cols.Status.Set(), agentprofile.StatusSuperseded).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), entity.CheckedAt).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("supersede the current agent profile: %w", err)
		}

		entity.Status = agentprofile.StatusCurrent
		if _, err := dba.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			r.l.Error("failed to publish agent profile",
				zap.String("subject", string(entity.Subject)),
				zap.Error(err))

			return nil, fmt.Errorf("publish agent profile: %w", err)
		}

		return entity, nil
	})
}

func (r *profileRepository) RecordFailure(
	ctx context.Context,
	entity *agentprofile.Profile,
) (*agentprofile.Profile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Profile, error) {
		dba := r.db.DBForContext(ctx)
		if kept, found, err := r.alreadyKept(ctx, dba, entity); err != nil || found {
			return kept, err
		}

		entity.Status = agentprofile.StatusFailed
		if _, err := dba.
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("record failed agent profile: %w", err)
		}

		return entity, nil
	})
}

func (r *profileRepository) MarkChecked(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
	profileID pulid.ID,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := agentprofilecol.ProfileColumns
		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agentprofile.Profile)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return scopeProfileSubjectUpdate(uq, ref).
					Where(cols.ID.Eq(), profileID).
					Where(cols.Status.Eq(), agentprofile.StatusCurrent)
			}).
			Set(cols.CheckedAt.Set(), at).
			Set(cols.UpdatedAt.Set(), at).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("mark agent profile checked: %w", err)
		}

		return dberror.CheckRowsAffected(res, "Profile", profileID.String())
	})
}

// Prune deletes the superseded and failed versions of one profile beyond the
// newest req.Keep. The current version is never pruned.
func (r *profileRepository) Prune(
	ctx context.Context,
	req agentprofileports.PruneProfilesRequest,
) (int, error) {
	keep := max(req.Keep, 1)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := agentprofilecol.ProfileColumns
		dba := r.db.DBForContext(ctx)

		kept := make([]pulid.ID, 0, keep)
		if err := dba.NewSelect().
			Model((*agentprofile.Profile)(nil)).
			Column(cols.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeProfileSubject(sq, req.Ref)
			}).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderDesc()).
			Limit(int64(keep)).
			Scan(ctx, &kept); err != nil {
			return 0, fmt.Errorf("read the agent profile versions to keep: %w", err)
		}
		if len(kept) < keep {
			return 0, nil
		}

		res, err := dba.NewDelete().
			Model((*agentprofile.Profile)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				dq = agentprofilecol.ProfileScopeTenantDelete(dq, req.Ref.Tenant()).
					Where(cols.Subject.Eq(), req.Ref.Subject).
					Where(cols.Status.NotEq(), agentprofile.StatusCurrent).
					Where(cols.ID.NotIn(), bun.List(kept))
				if req.Ref.Subject == agentprofile.SubjectUser {
					return dq.Where(cols.UserID.Eq(), req.Ref.PersonID)
				}

				return dq.Where(cols.UserID.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("prune agent profile history: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("count pruned agent profiles: %w", err)
		}

		return int(affected), nil
	})
}

// alreadyKept finds the version a retried write already committed, by the id
// the plan chose for it, so a Finish that ran twice keeps one version.
func (r *profileRepository) alreadyKept(
	ctx context.Context,
	dba bun.IDB,
	entity *agentprofile.Profile,
) (kept *agentprofile.Profile, found bool, err error) {
	if entity.ID.IsNil() {
		return nil, false, nil
	}
	cols := agentprofilecol.ProfileColumns
	kept = new(agentprofile.Profile)
	err = dba.NewSelect().
		Model(kept).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return agentprofilecol.ProfileScopeTenant(sq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("look for an agent profile version already kept: %w", err)
	}

	return kept, true, nil
}

func refOf(entity *agentprofile.Profile) agentprofileports.SubjectRef {
	ref := agentprofileports.SubjectRef{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Subject:        entity.Subject,
	}
	if entity.UserID != nil {
		ref.PersonID = *entity.UserID
	}

	return ref
}

func entityTenant(entity *agentprofile.Guidance) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}

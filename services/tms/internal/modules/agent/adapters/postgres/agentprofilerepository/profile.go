package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile/agentprofiledb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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

// profileSubject narrows to one profile. A nil person must match IS NULL,
// so the organization's profile and a person's never read each other's rows.
func profileSubject(ref agentprofileports.SubjectRef) keel.Pred {
	p := agentprofiledb.Profiles
	person := p.UserID.IsNull()
	if ref.Subject == agentprofile.SubjectUser {
		person = p.UserID.Eq(ref.PersonID)
	}

	return keel.And(p.Subject.Eq(ref.Subject), person)
}

// GetCurrent reads what agents are told, or nil before the first version.
func (r *profileRepository) GetCurrent(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (*agentprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofile.Profile, error) {
		p := agentprofiledb.Profiles
		entity, err := p.Where(profileSubject(ref), p.Status.Eq(agentprofile.StatusCurrent)).
			Limit(1).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
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
		entity, err := agentprofiledb.Profiles.Get(ctx, r.db.Keel(ctx), req.ID)
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("Profile version not found")
		}
		if err != nil {
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
		limit := req.Limit
		if limit <= 0 {
			limit = agentprofileports.DefaultHistoryLimit
		}
		limit = min(limit, agentprofileports.MaxHistoryLimit)

		p := agentprofiledb.Profiles
		rows, err := p.Where(profileSubject(req.Ref)).
			Order(p.CreatedAt.Desc(), p.ID.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
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
		if kept, found, err := r.alreadyKept(ctx, entity); err != nil || found {
			return kept, err
		}

		p := agentprofiledb.Profiles
		if _, err := p.Where(profileSubject(ref), p.Status.Eq(agentprofile.StatusCurrent)).Update(
			ctx,
			r.db.Keel(ctx),
			p.Status.To(agentprofile.StatusSuperseded),
			p.UpdatedAt.To(entity.CheckedAt),
		); err != nil {
			return nil, fmt.Errorf("supersede the current agent profile: %w", err)
		}

		entity.Status = agentprofile.StatusCurrent
		if err := p.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
		if kept, found, err := r.alreadyKept(ctx, entity); err != nil || found {
			return kept, err
		}

		entity.Status = agentprofile.StatusFailed
		if err := agentprofiledb.Profiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
		p := agentprofiledb.Profiles
		n, err := p.Where(
			profileSubject(ref),
			p.ID.Eq(profileID),
			p.Status.Eq(agentprofile.StatusCurrent),
		).Update(
			ctx,
			r.db.Keel(ctx),
			p.CheckedAt.To(at),
			p.UpdatedAt.To(at),
			p.Version.ToExpr(p.Version),
		)
		if err != nil {
			return fmt.Errorf("mark agent profile checked: %w", err)
		}

		return dberror.CheckAffected(n, "Profile", profileID.String())
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
		p := agentprofiledb.Profiles
		kept, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), p.
			Pick(p.ID).
			Where(profileSubject(req.Ref)).
			Order(p.CreatedAt.Desc(), p.ID.Desc()).
			Limit(keep))
		if err != nil {
			return 0, fmt.Errorf("read the agent profile versions to keep: %w", err)
		}
		if len(kept) < keep {
			return 0, nil
		}

		n, err := p.Where(
			profileSubject(req.Ref),
			p.Status.Ne(agentprofile.StatusCurrent),
			p.ID.NotIn(kept...),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("prune agent profile history: %w", err)
		}

		return int(n), nil
	})
}

// alreadyKept finds the version a retried write already committed, by the id
// the plan chose for it, so a Finish that ran twice keeps one version.
func (r *profileRepository) alreadyKept(
	ctx context.Context,
	entity *agentprofile.Profile,
) (kept *agentprofile.Profile, found bool, err error) {
	if entity.ID.IsNil() {
		return nil, false, nil
	}

	kept, err = agentprofiledb.Profiles.Get(ctx, r.db.Keel(ctx), entity.ID)
	if dberror.IsNotFoundError(err) {
		return nil, false, nil
	}
	if err != nil {
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

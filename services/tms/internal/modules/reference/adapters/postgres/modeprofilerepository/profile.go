package modeprofilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile/modeprofiledb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const profileBusyMessage = "Mode profile is busy. Retry the request."

var profileErrors = dberror.Mapping{Entity: "ModeProfile"}

type ProfileParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type profileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProfileRepository(p ProfileParams) modeprofileports.ModeProfileRepository {
	return &profileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.mode-profile-repository"),
	}
}

func orderedRules() keel.Loader {
	rl := modeprofiledb.Profiles.Rules()
	return rl.Order(rl.RuleKey.Asc())
}

func (r *profileRepository) List(
	ctx context.Context,
	req *modeprofileports.ListModeProfilesRequest,
) (*pagination.ListResult[*modeprofile.Profile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*modeprofile.Profile], error) {
			p := modeprofiledb.Profiles
			q := p.Where(keel.When(
				req.Status != "",
				p.Status.Eq(modeprofile.ProfileStatus(req.Status)),
			))

			result, err := listquery.List(ctx, r.db.Keel(ctx), q,
				&listquery.Request[modeprofile.Profile]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[*modeprofile.Profile]) *keel.Query[*modeprofile.Profile] {
						return q.Load(orderedRules())
					},
					Then: []keel.Order{p.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list mode profiles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *profileRepository) GetByID(
	ctx context.Context,
	req *modeprofileports.GetModeProfileByIDRequest,
) (*modeprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*modeprofile.Profile, error) {
		p := modeprofiledb.Profiles
		q := p.Where(p.ID.Eq(req.ModeProfileID))
		if req.IncludeRules {
			q = q.Load(orderedRules())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.ModeProfileID.String())
		}

		return entity, nil
	})
}

func (r *profileRepository) GetActiveProfiles(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*modeprofile.Profile, error) {
		p := modeprofiledb.Profiles
		entities, err := p.Where(p.Status.Eq(modeprofile.ProfileStatusActive)).
			Cache(resultcache.Settings()...).
			Load(orderedRules()).
			Order(p.Priority.Desc(), p.SpecificityScore.Desc(), p.CreatedAt.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to load active mode profiles", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func stampRules(entity *modeprofile.Profile, resetIDs bool) {
	for _, rule := range entity.Rules {
		if rule == nil {
			continue
		}

		if resetIDs {
			rule.ID = ""
		}
		rule.ModeProfileID = entity.ID
		rule.OrganizationID = entity.OrganizationID
		rule.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *profileRepository) insertRules(ctx context.Context, entity *modeprofile.Profile) error {
	rules := make([]*modeprofile.CapabilityRule, 0, len(entity.Rules))
	for _, rule := range entity.Rules {
		if rule != nil {
			rules = append(rules, rule)
		}
	}
	if len(rules) == 0 {
		return nil
	}

	return modeprofiledb.CapabilityRules.CreateMany(ctx, r.db.Keel(ctx), rules)
}

func (r *profileRepository) Create(
	ctx context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*modeprofile.Profile, error) {
		err := modeprofiledb.Profiles.Create(ctx, r.db.Keel(ctx), entity)
		if err == nil {
			stampRules(entity, false)
			err = r.insertRules(ctx, entity)
		}
		if err != nil {
			r.l.Error("failed to create mode profile", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				profileErrors.Map(err, entity.ID.String()),
				profileBusyMessage,
			)
		}

		return entity, nil
	})
}

func (r *profileRepository) Update(
	ctx context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*modeprofile.Profile, error) {
		if err := r.replace(ctx, entity); err != nil {
			r.l.Error(
				"failed to update mode profile",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(
				profileErrors.Map(err, entity.ID.String()),
				profileBusyMessage,
			)
		}

		return entity, nil
	})
}

func (r *profileRepository) replace(ctx context.Context, entity *modeprofile.Profile) error {
	db := r.db.Keel(ctx)
	if err := modeprofiledb.Profiles.Save(ctx, db, entity); err != nil {
		return err
	}

	rl := modeprofiledb.CapabilityRules
	if _, err := rl.Where(rl.ModeProfileID.Eq(entity.ID)).Delete(ctx, db); err != nil {
		return err
	}

	stampRules(entity, true)

	return r.insertRules(ctx, entity)
}

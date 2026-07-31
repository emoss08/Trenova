package modeprofilerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ProfileParams struct {
	fx.In

	DB     *postgres.Connection
	Cache  repositories.ModeProfileCacheRepository
	Logger *zap.Logger
}

type profileRepository struct {
	db    *postgres.Connection
	cache repositories.ModeProfileCacheRepository
	l     *zap.Logger
}

func NewProfileRepository(p ProfileParams) repositories.ModeProfileRepository {
	return &profileRepository{
		db:    p.DB,
		cache: p.Cache,
		l:     p.Logger.Named("postgres.mode-profile-repository"),
	}
}

func (r *profileRepository) invalidate(ctx context.Context, entity *modeprofile.Profile) {
	if r.cache == nil {
		return
	}

	if err := r.cache.Invalidate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		r.l.Warn("failed to invalidate mode profile cache", zap.Error(err))
	}
}

func orderRules(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(buncolgen.CapabilityRuleColumns.RuleKey.OrderAsc())
}

func (r *profileRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListModeProfilesRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.ProfileTable.Alias,
		req.Filter,
		(*modeprofile.Profile)(nil),
	)

	if req.Status != "" {
		q = q.Where(buncolgen.ProfileColumns.Status.Eq(), req.Status)
	}

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *profileRepository) List(
	ctx context.Context,
	req *repositories.ListModeProfilesRequest,
) (*pagination.ListResult[*modeprofile.Profile], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*modeprofile.Profile, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.ProfileRelations.Rules, orderRules).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count mode profiles", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*modeprofile.Profile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *profileRepository) GetByID(
	ctx context.Context,
	req *repositories.GetModeProfileByIDRequest,
) (*modeprofile.Profile, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ModeProfileID.String()),
	)

	entity := new(modeprofile.Profile)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ProfileScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ProfileColumns.ID.Eq(), req.ModeProfileID)
		})

	if req.IncludeRules {
		q = q.Relation(buncolgen.ProfileRelations.Rules, orderRules)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get mode profile", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *profileRepository) GetActiveProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*modeprofile.Profile, error) {
	log := r.l.With(zap.String("operation", "GetActiveProfiles"))

	cols := buncolgen.ProfileColumns
	entities := make([]*modeprofile.Profile, 0, 8)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.ProfileRelations.Rules, orderRules).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ProfileScopeTenant(sq, tenantInfo).
				Where(cols.Status.Eq(), modeprofile.ProfileStatusActive)
		}).
		Order(cols.Priority.OrderDesc()).
		Order(cols.SpecificityScore.OrderDesc()).
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to load active mode profiles", zap.Error(err))
		return nil, err
	}

	return entities, nil
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
	if len(entity.Rules) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Rules).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *profileRepository) Create(
	ctx context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	log := r.l.With(zap.String("operation", "Create"))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampRules(entity, false)

		return r.insertRules(c, entity)
	})
	if err != nil {
		log.Error("failed to create mode profile", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Mode profile is busy. Retry the request.",
		)
	}

	r.invalidate(ctx, entity)

	return entity, nil
}

func (r *profileRepository) Update(
	ctx context.Context,
	entity *modeprofile.Profile,
) (*modeprofile.Profile, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	ruleCols := buncolgen.CapabilityRuleColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results, "ModeProfile", entity.ID.String(),
		); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*modeprofile.CapabilityRule)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.CapabilityRuleScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(ruleCols.ModeProfileID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampRules(entity, true)

		return r.insertRules(c, entity)
	})
	if err != nil {
		log.Error("failed to update mode profile", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Mode profile is busy. Retry the request.",
		)
	}

	r.invalidate(ctx, entity)

	return entity, nil
}

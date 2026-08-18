package ratematrixrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type DensityScaleParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type densityScaleRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDensityScaleRepository(p DensityScaleParams) repositories.DensityScaleRepository {
	return &densityScaleRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.densityscale-repository"),
	}
}

func orderTiers(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(buncolgen.DensityScaleTierColumns.FromPcf.OrderAsc())
}

func (r *densityScaleRepository) List(
	ctx context.Context,
	req *repositories.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.DensityScale], error) {
	log := r.l.With(zap.String("operation", "List"))

	cols := buncolgen.DensityScaleColumns
	entities := make([]*ratematrix.DensityScale, 0, req.Filter.Pagination.SafeLimit())

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.Rel(buncolgen.DensityScaleRelations.Tiers), orderTiers).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFilters(
				sq,
				buncolgen.DensityScaleTable.Alias,
				req.Filter,
				(*ratematrix.DensityScale)(nil),
			)

			return sq.Apply(buncolgen.DensityScaleApplyTenant(req.Filter.TenantInfo)).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset()).
				Order(cols.EffectiveFrom.OrderDesc())
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count density scales", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*ratematrix.DensityScale]{Items: entities, Total: total}, nil
}

func (r *densityScaleRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDensityScaleRequest,
) (*ratematrix.DensityScale, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.DensityScaleID.String()),
	)

	entity := new(ratematrix.DensityScale)
	cols := buncolgen.DensityScaleColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.Rel(buncolgen.DensityScaleRelations.Tiers), orderTiers).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DensityScaleScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.DensityScaleID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get density scale", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DensityScale")
	}

	return entity, nil
}

// GetOrgDefault returns the scale a rule uses when it does not name one.
//
// A missing default is not an error. An organization that never rates by
// density has no scale, and the caller falls back to the commodity's own class.
func (r *densityScaleRepository) GetOrgDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*ratematrix.DensityScale, error) {
	log := r.l.With(zap.String("operation", "GetOrgDefault"))

	entity := new(ratematrix.DensityScale)
	cols := buncolgen.DensityScaleColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.Rel(buncolgen.DensityScaleRelations.Tiers), orderTiers).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DensityScaleScopeTenant(sq, tenantInfo).
				Where(cols.IsOrgDefault.IsTrue()).
				Where(cols.Status.Eq(), domaintypes.StatusActive)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // an organization may not rate by density at all
		}
		log.Error("failed to get default density scale", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *densityScaleRepository) Create(
	ctx context.Context,
	entity *ratematrix.DensityScale,
) (*ratematrix.DensityScale, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("code", entity.Code))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return iErr
		}

		stampTiers(entity, false)

		return r.insertTiers(c, entity)
	})
	if err != nil {
		log.Error("failed to create density scale", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Density scale is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *densityScaleRepository) Update(
	ctx context.Context,
	entity *ratematrix.DensityScale,
) (*ratematrix.DensityScale, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	cols := buncolgen.DensityScaleColumns
	tierCols := buncolgen.DensityScaleTierColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results,
			"DensityScale",
			entity.ID.String(),
		); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*ratematrix.DensityScaleTier)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.DensityScaleTierScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(tierCols.RateDensityScaleID.Eq(), entity.ID)
			}).
			Exec(c); dErr != nil {
			return dErr
		}

		stampTiers(entity, true)

		return r.insertTiers(c, entity)
	})
	if err != nil {
		log.Error("failed to update density scale", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Density scale is busy. Retry the request.",
		)
	}

	return entity, nil
}

func stampTiers(entity *ratematrix.DensityScale, resetIDs bool) {
	for _, tier := range entity.Tiers {
		if tier == nil {
			continue
		}

		if resetIDs {
			tier.ID = pulid.Nil
		}

		tier.RateDensityScaleID = entity.ID
		tier.OrganizationID = entity.OrganizationID
		tier.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *densityScaleRepository) insertTiers(
	ctx context.Context,
	entity *ratematrix.DensityScale,
) error {
	if len(entity.Tiers) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Tiers).
		Returning("*").
		Exec(ctx)

	return err
}

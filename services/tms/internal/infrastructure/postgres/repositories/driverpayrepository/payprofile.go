package driverpayrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

type payProfileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPayProfile(p Params) repositories.PayProfileRepository {
	return &payProfileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.pay-profile-repository"),
	}
}

func (r *payProfileRepository) List(
	ctx context.Context,
	req *repositories.ListPayProfilesRequest,
) (*pagination.ListResult[*driverpay.PayProfile], error) {
	limit := req.Filter.Pagination.SafeLimit()
	items := make([]*driverpay.PayProfile, 0, limit)

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("dpp.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("dpp.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Relation("Components", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("dppc.sequence ASC")
		}).
		Order("dpp.name ASC").
		Limit(limit).
		Offset(req.Filter.Pagination.SafeOffset())

	if req.Filter.Query != "" {
		query = query.Where(
			"(dpp.name ILIKE ? OR dpp.description ILIKE ?)",
			"%"+req.Filter.Query+"%",
			"%"+req.Filter.Query+"%",
		)
	}
	if req.Classification != "" {
		query = query.Where("dpp.classification = ?", req.Classification)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pay profiles: %w", err)
	}

	return &pagination.ListResult[*driverpay.PayProfile]{Items: items, Total: total}, nil
}

func (r *payProfileRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListPayProfileConnectionRequest,
) (*pagination.CursorListResult[*driverpay.PayProfile], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*driverpay.PayProfile)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"dpp",
				req.Filter,
				(*driverpay.PayProfile)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count pay profiles", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*driverpay.PayProfile]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*driverpay.PayProfile) *bun.SelectQuery {
			return dba.NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.PayProfileTable.All()).
				Relation("Components", func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Order("dppc.sequence ASC")
				})
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				"dpp",
				req.Filter,
				req.Cursor,
				(*driverpay.PayProfile)(nil),
			)
		},
	})
	if err != nil {
		log.Error("failed to scan pay profiles", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *payProfileRepository) GetByID(
	ctx context.Context,
	req repositories.GetPayProfileByIDRequest,
) (*driverpay.PayProfile, error) {
	entity := new(driverpay.PayProfile)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("dpp.id = ?", req.ID).
		Where("dpp.organization_id = ?", req.TenantInfo.OrgID).
		Where("dpp.business_unit_id = ?", req.TenantInfo.BuID)
	if req.IncludeComponents {
		query = query.Relation("Components", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("dppc.sequence ASC")
		})
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "PayProfile")
	}
	return entity, nil
}

func (r *payProfileRepository) Create(
	ctx context.Context,
	entity *driverpay.PayProfile,
) (*driverpay.PayProfile, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("dpp_")
	}
	assignComponentFields(entity)
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
		return nil, fmt.Errorf("create pay profile: %w", err)
	}
	if len(entity.Components) > 0 {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Components).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create pay profile components: %w", err)
		}
	}
	return r.GetByID(ctx, repositories.GetPayProfileByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeComponents: true,
	})
}

func (r *payProfileRepository) Update(
	ctx context.Context,
	entity *driverpay.PayProfile,
) (*driverpay.PayProfile, error) {
	assignComponentFields(entity)
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("status = ?", entity.Status).
		Set("name = ?", entity.Name).
		Set("description = ?", entity.Description).
		Set("classification = ?", entity.Classification).
		Set("currency_code = ?", entity.CurrencyCode).
		Set("guaranteed_period_minimum_minor = ?", entity.GuaranteedPeriodMinimumMinor).
		Set("per_diem_rate_per_mile = ?", entity.PerDiemRatePerMile).
		Set("per_diem_daily_cap_minor = ?", entity.PerDiemDailyCapMinor).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update pay profile: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "PayProfile", entity.ID.String()); err != nil {
		return nil, err
	}
	if entity.Components != nil {
		if _, err = r.db.DBForContext(ctx).
			NewDelete().
			Model((*driverpay.PayProfileComponent)(nil)).
			Where("pay_profile_id = ?", entity.ID).
			Where("organization_id = ?", entity.OrganizationID).
			Where("business_unit_id = ?", entity.BusinessUnitID).
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("replace pay profile components: %w", err)
		}
		if len(entity.Components) > 0 {
			if _, err = r.db.DBForContext(ctx).
				NewInsert().
				Model(&entity.Components).
				Exec(ctx); err != nil {
				return nil, fmt.Errorf("insert pay profile components: %w", err)
			}
		}
	}
	return r.GetByID(ctx, repositories.GetPayProfileByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeComponents: true,
	})
}

func (r *payProfileRepository) CountActiveAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
) (int, error) {
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*driverpay.WorkerPayAssignment)(nil)).
		Where("wpa.organization_id = ?", tenantInfo.OrgID).
		Where("wpa.business_unit_id = ?", tenantInfo.BuID).
		Where("wpa.pay_profile_id = ?", profileID).
		Where("wpa.effective_to IS NULL OR wpa.effective_to > extract(epoch from current_timestamp)::bigint").
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active pay assignments: %w", err)
	}
	return count, nil
}

func assignComponentFields(entity *driverpay.PayProfile) {
	for idx, comp := range entity.Components {
		if comp == nil {
			continue
		}
		comp.OrganizationID = entity.OrganizationID
		comp.BusinessUnitID = entity.BusinessUnitID
		comp.PayProfileID = entity.ID
		comp.Sequence = idx
	}
}

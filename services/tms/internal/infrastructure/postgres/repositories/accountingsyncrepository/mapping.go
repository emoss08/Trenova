package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	mappingEntity      = "Accounting mapping"
	mappingInsertBatch = 500
	mappingLabelField  = "targetLabel"
)

type MappingParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type mappingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMappingRepository(p MappingParams) repositories.AccountingMappingRepository {
	return &mappingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-mapping-repository"),
	}
}

func (r *mappingRepository) ListByConnection(
	ctx context.Context,
	req *repositories.ListAccountingMappingsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	entities := make([]*accountingsync.AccountingMapping, 0)
	cols := buncolgen.AccountingMappingColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingMappingApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Order(cols.TargetType.OrderAsc(), cols.ID.OrderAsc())
	if len(req.TargetTypes) > 0 {
		query = query.Where(cols.TargetType.In(), bun.List(req.TargetTypes))
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *mappingRepository) applyMappingFilters(
	q *bun.SelectQuery,
	req *repositories.ListAccountingMappingsConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.AccountingMappingColumns
	q = q.Where(cols.ConnectionID.Qualified()+" = ?", req.ConnectionID)
	if len(req.TargetTypes) > 0 {
		q = q.Where(cols.TargetType.Qualified()+" IN (?)", bun.List(req.TargetTypes))
	}
	if len(req.States) > 0 {
		q = q.Where(cols.State.Qualified()+" IN (?)", bun.List(req.States))
	}
	if req.RequiredOnly {
		q = q.WhereGroup(" AND ", requiredTargetsGroup)
	}
	if term := stringutils.NormalizeName(req.Search); term != "" {
		q = q.Where(
			cols.SearchLabel.Qualified()+" LIKE ?",
			"%"+stringutils.EscapeLikePattern(term)+"%",
		)
	}
	return q
}

func requiredTargetsGroup(q *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.AccountingMappingColumns
	for _, target := range accountingsync.AllMappingTargetTypes() {
		for _, key := range target.Keys() {
			if !accountingsync.IsRequiredTarget(target, key) {
				continue
			}
			q = q.WhereOr(
				"("+cols.TargetType.Qualified()+" = ? AND "+cols.TrenovaKey.Qualified()+" = ?)",
				target,
				key,
			)
		}
	}
	return q
}

func (r *mappingRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAccountingMappingsConnectionRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingMapping], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	if req.Filter != nil && len(req.Filter.Sort) == 0 {
		req.Filter.Sort = []domaintypes.SortField{
			{Field: mappingLabelField, Direction: dbtype.SortDirectionAsc},
		}
	}

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*accountingsync.AccountingMapping)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AccountingMappingTable.Alias,
					req.Filter,
					(*accountingsync.AccountingMapping)(nil),
				)
				return r.applyMappingFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count accounting mappings", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*accountingsync.AccountingMapping]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*accountingsync.AccountingMapping) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.AccountingMappingTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.AccountingMappingTable.Alias,
					req.Filter,
					req.Cursor,
					(*accountingsync.AccountingMapping)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyMappingFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list accounting mappings", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *mappingRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingMappingRequest,
) (*accountingsync.AccountingMapping, error) {
	entity := new(accountingsync.AccountingMapping)
	cols := buncolgen.AccountingMappingColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.AccountingMappingApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, mappingEntity)
	}

	return entity, nil
}

func (r *mappingRepository) GetByIDs(
	ctx context.Context,
	req repositories.GetAccountingMappingsByIDsRequest,
) ([]*accountingsync.AccountingMapping, error) {
	if len(req.IDs) == 0 {
		return []*accountingsync.AccountingMapping{}, nil
	}

	entities := make([]*accountingsync.AccountingMapping, 0, len(req.IDs))
	cols := buncolgen.AccountingMappingColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.ID.In(), bun.List(req.IDs)).
		Apply(buncolgen.AccountingMappingApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *mappingRepository) GetByTarget(
	ctx context.Context,
	req *repositories.GetAccountingMappingByTargetRequest,
) (*accountingsync.AccountingMapping, error) {
	entity := new(accountingsync.AccountingMapping)
	cols := buncolgen.AccountingMappingColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(buncolgen.AccountingMappingApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.TargetType.Eq(), req.TargetType)
	if req.TargetType.KeyedByObject() {
		query = query.Where(cols.TrenovaObjectID.Eq(), req.TrenovaObjectID)
	} else {
		query = query.Where(cols.TrenovaKey.Eq(), req.TrenovaKey)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, mappingEntity)
	}

	return entity, nil
}

func (r *mappingRepository) CreateMissing(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	var created int64
	for start := 0; start < len(entities); start += mappingInsertBatch {
		batch := entities[start:min(start+mappingInsertBatch, len(entities))]
		results, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&batch).
			On("CONFLICT (organization_id, business_unit_id, connection_id, target_type, " +
				"COALESCE(trenova_object_id, ''), COALESCE(trenova_key, '')) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return created, fmt.Errorf("create accounting mappings: %w", err)
		}
		affected, err := results.RowsAffected()
		if err != nil {
			return created, err
		}
		created += affected
	}

	return created, nil
}

func (r *mappingRepository) ApplyScoring(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) (int64, error) {
	cols := buncolgen.AccountingMappingColumns
	var applied int64

	for _, entity := range entities {
		ov := entity.Version
		entity.Version++
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(
				cols.TargetLabel.String(),
				cols.SearchLabel.String(),
				cols.ExternalID.String(),
				cols.ExternalName.String(),
				cols.State.String(),
				cols.Source.String(),
				cols.Confidence.String(),
				cols.Reason.String(),
				cols.Signals.String(),
				cols.Version.String(),
				cols.UpdatedAt.String(),
			).
			WherePK().
			Where(cols.Version.Eq(), ov).
			Where(cols.State.NotEq(), accountingsync.MappingStateConfirmed).
			Exec(ctx)
		if err != nil {
			entity.Version = ov
			return applied, fmt.Errorf("apply accounting mapping scores: %w", err)
		}
		affected, err := results.RowsAffected()
		if err != nil {
			entity.Version = ov
			return applied, err
		}
		if affected == 0 {
			entity.Version = ov
			continue
		}
		applied += affected
	}

	return applied, nil
}

func (r *mappingRepository) UpdateLabels(
	ctx context.Context,
	entities []*accountingsync.AccountingMapping,
) error {
	cols := buncolgen.AccountingMappingColumns
	for _, entity := range entities {
		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(cols.TargetLabel.String(), cols.SearchLabel.String()).
			WherePK().
			Exec(ctx); err != nil {
			return fmt.Errorf("update accounting mapping labels: %w", err)
		}
	}
	return nil
}

func (r *mappingRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingMapping,
) (*accountingsync.AccountingMapping, error) {
	cols := buncolgen.AccountingMappingColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, mappingEntity, entity.ID.String()); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}

func (r *mappingRepository) CountByState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) ([]repositories.AccountingMappingCount, error) {
	cols := buncolgen.AccountingMappingColumns
	counts := make(
		[]repositories.AccountingMappingCount,
		0,
		len(accountingsync.AllMappingTargetTypes())*3,
	)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*accountingsync.AccountingMapping)(nil)).
		ColumnExpr(cols.TargetType.As("target_type")).
		ColumnExpr(cols.State.As("state")).
		ColumnExpr("COUNT(*) AS count").
		Apply(buncolgen.AccountingMappingApplyTenant(tenantInfo)).
		Where(cols.ConnectionID.Eq(), connectionID).
		GroupExpr(cols.TargetType.Qualified()).
		GroupExpr(cols.State.Qualified()).
		Scan(ctx, &counts); err != nil {
		return nil, err
	}

	return counts, nil
}

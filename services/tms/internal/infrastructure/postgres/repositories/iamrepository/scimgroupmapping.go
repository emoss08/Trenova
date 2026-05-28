package iamrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) filterSCIMGroupRoleMappings(
	q *bun.SelectQuery,
	req *repositories.ListSCIMGroupRoleMappingsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"sgrm",
		req.Filter,
		(*iam.SCIMGroupRoleMapping)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) ListSCIMGroupRoleMappings(
	ctx context.Context,
	req *repositories.ListSCIMGroupRoleMappingsRequest,
) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error) {
	log := r.l.With(
		zap.String("operation", "ListSCIMGroupRoleMappings"),
		zap.Any("request", req),
	)

	entities := make([]*iam.SCIMGroupRoleMapping, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.SCIMGroupRoleMappingColumns
	rel := buncolgen.SCIMGroupRoleMappingRelations

	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Relation(rel.Role).
		Where(cols.DirectoryID.Eq(), req.DirectoryID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterSCIMGroupRoleMappings(sq, req)
		}).
		Order(cols.DisplayName.OrderAsc()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count scim group role mappings", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*iam.SCIMGroupRoleMapping]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) CreateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	log := r.l.With(
		zap.String("operation", "CreateSCIMGroupRoleMapping"),
		zap.Any("entity", entity),
	)

	if _, err := r.db.DB().NewInsert().Model(entity).Exec(ctx); err != nil {
		log.Error("failed to insert scim group role mapping", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	log := r.l.With(
		zap.String("operation", "UpdateSCIMGroupRoleMapping"),
		zap.Any("sgrmID", entity.ID),
	)

	ov := entity.Version
	entity.Version++
	cols := buncolgen.SCIMGroupRoleMappingColumns

	results, err := r.db.DB().NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update scim group role mapping", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "SCIMGroupRoleMapping", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteSCIMGroupRoleMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sgrmID pulid.ID,
) error {
	cols := buncolgen.SCIMGroupRoleMappingColumns
	res, err := r.db.DB().NewDelete().
		Model((*iam.SCIMGroupRoleMapping)(nil)).
		Apply(func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.SCIMGroupRoleMappingScopeTenantDelete(dq, tenantInfo).
				Where(cols.ID.Eq(), sgrmID)
		}).
		Exec(ctx)
	if err != nil {
		return err
	}
	return dberror.CheckRowsAffected(res, "SCIM group role mapping", sgrmID.String())
}

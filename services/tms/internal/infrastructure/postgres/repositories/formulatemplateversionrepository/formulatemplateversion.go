package formulatemplateversionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.FormulaTemplateVersionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-version-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("templateID", version.TemplateID.String()),
		zap.Int64("versionNumber", version.VersionNumber),
	)

	_, err := r.db.DBForContext(ctx).NewInsert().Model(version).Exec(ctx)
	if err != nil {
		log.Error("failed to create formula template version", zap.Error(err))
		return nil, err
	}

	return version, nil
}

func (r *repository) GetByTemplateAndVersion(
	ctx context.Context,
	req *repositories.GetVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetByTemplateAndVersion"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, "FormulaTemplateVersion")
		}
		log.Error("failed to get formula template version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListVersionsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ftv",
		req.Filter,
		(*formulatemplate.FormulaTemplateVersion)(nil),
	)

	q = q.Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
		Relation("CreatedBy")

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("templateID", req.TemplateID.String()),
	)

	entities := make(
		[]*formulatemplate.FormulaTemplateVersion,
		0,
		req.Filter.Pagination.SafeLimit(),
	)
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		Order(buncolgen.FormulaTemplateVersionColumns.VersionNumber.OrderDesc()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list formula template versions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*formulatemplate.FormulaTemplateVersion]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetVersionRange(
	ctx context.Context,
	req *repositories.GetVersionRangeRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetVersionRange"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("fromVersion", req.FromVersion),
		zap.Int64("toVersion", req.ToVersion),
	)

	entities := make([]*formulatemplate.FormulaTemplateVersion, 0, 2)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.VersionNumber.In(), bun.List([]int64{req.FromVersion, req.ToVersion}))
		}).
		Order(buncolgen.FormulaTemplateVersionColumns.VersionNumber.OrderAsc()).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get version range", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetLatestVersion(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetLatestVersion"),
		zap.String("templateID", templateID.String()),
	)

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), templateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), tenantInfo.BuID)
		}).
		Order(buncolgen.FormulaTemplateVersionColumns.VersionNumber.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get latest version", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "FormulaTemplateVersion")
	}

	return entity, nil
}

func (r *repository) GetForkedTemplates(
	ctx context.Context,
	req *repositories.GetForkedTemplatesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	log := r.l.With(
		zap.String("operation", "GetForkedTemplates"),
		zap.String("sourceTemplateID", req.SourceTemplateID.String()),
	)

	entities := make([]*formulatemplate.FormulaTemplate, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FormulaTemplateColumns.SourceTemplateID.Eq(), req.SourceTemplateID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get forked templates", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) UpdateTags(
	ctx context.Context,
	req *repositories.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "UpdateTags"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	tags := make([]formulatemplate.VersionTag, len(req.Tags))
	for i, t := range req.Tags {
		tags[i] = formulatemplate.VersionTag(t)
	}

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("tags = ?", pgdialect.Array(tags)).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
		}).
		Returning("*").
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, "FormulaTemplateVersion")
		}
		log.Error("failed to update version tags", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetEffectiveVersion(
	ctx context.Context,
	req *repositories.GetEffectiveVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetEffectiveVersion"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("asOf", req.AsOf),
	)

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.EffectiveFrom.IsNotNull()).
				Where(buncolgen.FormulaTemplateVersionColumns.EffectiveFrom.Lte(), req.AsOf)
		}).
		OrderExpr("ftv.effective_from DESC").
		OrderExpr("ftv.version_number DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil version means no scheduled version is in effect
		}
		log.Error("failed to get effective formula template version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateEffectiveDate(
	ctx context.Context,
	req *repositories.UpdateEffectiveDateRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "UpdateEffectiveDate"),
		zap.String("templateID", req.TemplateID.String()),
		zap.Int64("versionNumber", req.VersionNumber),
	)

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set("effective_from = ?", req.EffectiveFrom).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
		}).
		Returning("*").
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, "FormulaTemplateVersion")
		}
		log.Error("failed to update version effective date", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListScheduled(
	ctx context.Context,
	req *repositories.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "ListScheduled"),
		zap.String("templateID", req.TemplateID.String()),
	)

	versions := make([]*formulatemplate.FormulaTemplateVersion, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&versions).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(buncolgen.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
				Where(buncolgen.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(buncolgen.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(buncolgen.FormulaTemplateVersionColumns.EffectiveFrom.IsNotNull())
		}).
		OrderExpr("ftv.effective_from ASC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to list scheduled versions", zap.Error(err))
		return nil, err
	}

	return versions, nil
}

func (r *repository) GetLatestByStatus(
	ctx context.Context,
	req *repositories.GetLatestVersionByStatusRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetLatestByStatus"),
		zap.String("templateID", req.TemplateID.String()),
		zap.String("status", req.Status.String()),
	)

	ftv := buncolgen.FormulaTemplateVersionColumns

	entity := new(formulatemplate.FormulaTemplateVersion)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormulaTemplateVersionScopeTenant(sq, req.TenantInfo).
				Where(ftv.TemplateID.Eq(), req.TemplateID).
				Where(ftv.Status.Eq(), req.Status)
		}).
		Order(ftv.VersionNumber.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil version means the template never held that status
		}
		log.Error("failed to get latest formula template version by status", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) ClearScheduled(
	ctx context.Context,
	req *repositories.ListScheduledVersionsRequest,
) (int64, error) {
	log := r.l.With(
		zap.String("operation", "ClearScheduled"),
		zap.String("templateID", req.TemplateID.String()),
	)

	ftv := buncolgen.FormulaTemplateVersionColumns

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*formulatemplate.FormulaTemplateVersion)(nil)).
		Set(ftv.EffectiveFrom.String()+" = NULL").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.FormulaTemplateVersionScopeTenantUpdate(uq, req.TenantInfo).
				Where(ftv.TemplateID.Eq(), req.TemplateID).
				Where(ftv.EffectiveFrom.IsNotNull())
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to clear scheduled versions", zap.Error(err))
		return 0, err
	}

	cleared, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to read cleared version count", zap.Error(err))
		return 0, err
	}

	return cleared, nil
}

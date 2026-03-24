package formulatemplateversionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

	_, err := r.db.DB().NewInsert().Model(version).Exec(ctx)
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
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ftv.template_id = ?", req.TemplateID).
				Where("ftv.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftv.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ftv.version_number = ?", req.VersionNumber)
		}).
		Scan(ctx)
	if err != nil {
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

	q = q.Where("ftv.template_id = ?", req.TemplateID).
		Relation("CreatedBy")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("templateID", req.TemplateID.String()),
	)

	entities := make([]*formulatemplate.FormulaTemplateVersion, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		Order("ftv.version_number DESC").
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
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ftv.template_id = ?", req.TemplateID).
				Where("ftv.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftv.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ftv.version_number IN (?)", bun.In([]int64{req.FromVersion, req.ToVersion}))
		}).
		Order("ftv.version_number ASC").
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
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ftv.template_id = ?", templateID).
				Where("ftv.organization_id = ?", tenantInfo.OrgID).
				Where("ftv.business_unit_id = ?", tenantInfo.BuID)
		}).
		Order("ftv.version_number DESC").
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
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("ft.source_template_id = ?", req.SourceTemplateID).
				Where("ft.organization_id = ?", req.TenantInfo.OrgID).
				Where("ft.business_unit_id = ?", req.TenantInfo.BuID)
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
	err := r.db.DB().
		NewUpdate().
		Model(entity).
		Set("tags = ?", pgdialect.Array(tags)).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.
				Where("ftv.template_id = ?", req.TemplateID).
				Where("ftv.organization_id = ?", req.TenantInfo.OrgID).
				Where("ftv.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ftv.version_number = ?", req.VersionNumber)
		}).
		Returning("*").
		Scan(ctx)
	if err != nil {
		log.Error("failed to update version tags", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

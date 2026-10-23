package formulatemplateversionrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/formulatemplatecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) formulatemplateports.FormulaTemplateVersionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.formula-template-version-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	version *formulatemplate.FormulaTemplateVersion,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
			log := r.l.With(
				zap.String("operation", "Create"),
				zap.String("templateID", version.TemplateID.String()),
				zap.Int64("versionNumber", version.VersionNumber),
			)

			dba := r.db.DBForContext(ctx)
			if _, err := dba.NewInsert().Model(version).Exec(ctx); err != nil {
				log.Error("failed to create formula template version", zap.Error(err))
				return nil, err
			}

			if rows := version.ImportRows(); len(rows) > 0 {
				if _, err := dba.NewInsert().Model(&rows).Exec(ctx); err != nil {
					log.Error("failed to record formula template version imports", zap.Error(err))
					return nil, err
				}
			}

			return version, nil
		},
	)
}

func (r *repository) GetByTemplateAndVersion(
	ctx context.Context,
	req *formulatemplateports.GetVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
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
		},
	)
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *formulatemplateports.ListVersionsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ftv",
		req.Filter,
		(*formulatemplate.FormulaTemplateVersion)(nil),
	)

	q = q.Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
		Relation("CreatedBy")

	return q.Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset()))
}

func (r *repository) List(
	ctx context.Context,
	req *formulatemplateports.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
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
				Order(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.OrderDesc()).
				ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to list formula template versions", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*formulatemplate.FormulaTemplateVersion]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) GetVersionRange(
	ctx context.Context,
	req *formulatemplateports.GetVersionRangeRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.In(), bun.List([]int64{req.FromVersion, req.ToVersion}))
				}).
				Order(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.OrderAsc()).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get version range", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetLatestVersion(
	ctx context.Context,
	templateID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), templateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), tenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), tenantInfo.BuID)
				}).
				Order(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.OrderDesc()).
				Limit(1).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get latest version", zap.Error(err))
				return nil, dberror.HandleNotFoundError(err, "FormulaTemplateVersion")
			}

			return entity, nil
		},
	)
}

func (r *repository) GetForkedTemplates(
	ctx context.Context,
	req *formulatemplateports.GetForkedTemplatesRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplate, error) {
			log := r.l.With(
				zap.String("operation", "GetForkedTemplates"),
				zap.String("sourceTemplateID", req.SourceTemplateID.String()),
			)

			entities := make([]*formulatemplate.FormulaTemplate, 0)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulatemplatecol.FormulaTemplateScopeTenant(sq, req.TenantInfo).
						Where(formulatemplatecol.FormulaTemplateColumns.SourceTemplateID.Eq(), req.SourceTemplateID)
				}).
				Scan(ctx)
			if err != nil {
				log.Error("failed to get forked templates", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) UpdateTags(
	ctx context.Context,
	req *formulatemplateports.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
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
		},
	)
}

func (r *repository) GetEffectiveVersion(
	ctx context.Context,
	req *formulatemplateports.GetEffectiveVersionRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.EffectiveFrom.IsNotNull()).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.EffectiveFrom.Lte(), req.AsOf)
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
		},
	)
}

func (r *repository) UpdateEffectiveDate(
	ctx context.Context,
	req *formulatemplateports.UpdateEffectiveDateRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.VersionNumber.Eq(), req.VersionNumber)
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
		},
	)
}

func (r *repository) ListScheduled(
	ctx context.Context,
	req *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
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
						Where(formulatemplatecol.FormulaTemplateVersionColumns.TemplateID.Eq(), req.TemplateID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.OrganizationID.Eq(), req.TenantInfo.OrgID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.BusinessUnitID.Eq(), req.TenantInfo.BuID).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.Status.Eq(), formulatemplate.StatusActive).
						Where(formulatemplatecol.FormulaTemplateVersionColumns.EffectiveFrom.Gt(), timeutils.NowUnix())
				}).
				Order(formulatemplatecol.FormulaTemplateVersionColumns.EffectiveFrom.OrderAsc()).
				Scan(ctx)
			if err != nil {
				log.Error("failed to list scheduled versions", zap.Error(err))
				return nil, err
			}

			return versions, nil
		},
	)
}

func (r *repository) ListApproved(
	ctx context.Context,
	req *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplate.FormulaTemplateVersion, error) {
			ftv := formulatemplatecol.FormulaTemplateVersionColumns

			versions := make([]*formulatemplate.FormulaTemplateVersion, 0)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&versions).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulatemplatecol.FormulaTemplateVersionScopeTenant(sq, req.TenantInfo).
						Where(ftv.TemplateID.Eq(), req.TemplateID).
						Where(ftv.Status.Eq(), formulatemplate.StatusActive)
				}).
				Order(ftv.VersionNumber.OrderAsc()).
				Scan(ctx)
			if err != nil {
				r.l.With(
					zap.String("operation", "ListApproved"),
					zap.String("templateID", req.TemplateID.String()),
				).Error("failed to list approved versions", zap.Error(err))
				return nil, err
			}

			return versions, nil
		},
	)
}

func (r *repository) ListAuthorIDsSince(
	ctx context.Context,
	req *formulatemplateports.ListVersionAuthorsRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		ftv := formulatemplatecol.FormulaTemplateVersionColumns

		authors := make([]pulid.ID, 0, 2)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*formulatemplate.FormulaTemplateVersion)(nil)).
			Distinct().
			Column(ftv.CreatedByID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return formulatemplatecol.FormulaTemplateVersionScopeTenant(sq, req.TenantInfo).
					Where(ftv.TemplateID.Eq(), req.TemplateID).
					Where(ftv.VersionNumber.Gt(), req.AfterVersion)
			}).
			Scan(ctx, &authors)
		if err != nil {
			r.l.With(
				zap.String("operation", "ListAuthorIDsSince"),
				zap.String("templateID", req.TemplateID.String()),
			).Error("failed to list version authors", zap.Error(err))
			return nil, err
		}

		return authors, nil
	})
}

func (r *repository) GetLatestByStatus(
	ctx context.Context,
	req *formulatemplateports.GetLatestVersionByStatusRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
			log := r.l.With(
				zap.String("operation", "GetLatestByStatus"),
				zap.String("templateID", req.TemplateID.String()),
				zap.String("status", req.Status.String()),
			)

			ftv := formulatemplatecol.FormulaTemplateVersionColumns

			entity := new(formulatemplate.FormulaTemplateVersion)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return formulatemplatecol.FormulaTemplateVersionScopeTenant(sq, req.TenantInfo).
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
		},
	)
}

func (r *repository) ClearScheduled(
	ctx context.Context,
	req *formulatemplateports.ListScheduledVersionsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		log := r.l.With(
			zap.String("operation", "ClearScheduled"),
			zap.String("templateID", req.TemplateID.String()),
		)

		ftv := formulatemplatecol.FormulaTemplateVersionColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*formulatemplate.FormulaTemplateVersion)(nil)).
			Set(ftv.Status.Set(), formulatemplate.StatusInactive).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return formulatemplatecol.FormulaTemplateVersionScopeTenantUpdate(uq, req.TenantInfo).
					Where(ftv.TemplateID.Eq(), req.TemplateID).
					Where(ftv.Status.Eq(), formulatemplate.StatusActive).
					Where(ftv.EffectiveFrom.Gt(), timeutils.NowUnix())
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
	})
}

const defaultLibraryUsageLimit = 200

func (r *repository) ListLibraryUsages(
	ctx context.Context,
	req *formulatemplateports.ListLibraryUsagesRequest,
) ([]*formulatemplateports.LibraryUsage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplateports.LibraryUsage, error) {
			imports := formulatemplatecol.VersionImportColumns
			versions := formulatemplatecol.FormulaTemplateVersionColumns
			templates := formulatemplatecol.FormulaTemplateColumns
			limit := req.Limit
			if limit <= 0 || limit > defaultLibraryUsageLimit {
				limit = defaultLibraryUsageLimit
			}

			type usageRow struct {
				TemplateID           pulid.ID               `bun:"template_id"`
				TemplateName         string                 `bun:"template_name"`
				TemplateStatus       formulatemplate.Status `bun:"template_status"`
				TemplateVersion      int64                  `bun:"template_version"`
				VersionStatus        formulatemplate.Status `bun:"version_status"`
				LibraryVersionNumber int64                  `bun:"library_version_number"`
			}

			rows := make([]usageRow, 0, 16)
			query := r.db.DBForContext(ctx).
				NewSelect().
				TableExpr(
					"? AS ?",
					bun.Ident(formulatemplatecol.VersionImportTable.Name),
					bun.Ident(formulatemplatecol.VersionImportTable.Alias),
				).
				Join(
					"JOIN ? AS ? ON ? AND ? AND ?",
					bun.Ident(formulatemplatecol.FormulaTemplateVersionTable.Name),
					bun.Ident(formulatemplatecol.FormulaTemplateVersionTable.Alias),
					bun.Safe(versions.ID.EqColumn(imports.TemplateVersionID)),
					bun.Safe(versions.OrganizationID.EqColumn(imports.OrganizationID)),
					bun.Safe(versions.BusinessUnitID.EqColumn(imports.BusinessUnitID)),
				).
				Join(
					"JOIN ? AS ? ON ? AND ? AND ?",
					bun.Ident(formulatemplatecol.FormulaTemplateTable.Name),
					bun.Ident(formulatemplatecol.FormulaTemplateTable.Alias),
					bun.Safe(templates.ID.EqColumn(imports.TemplateID)),
					bun.Safe(templates.OrganizationID.EqColumn(imports.OrganizationID)),
					bun.Safe(templates.BusinessUnitID.EqColumn(imports.BusinessUnitID)),
				).
				ColumnExpr(imports.TemplateID.As("template_id")).
				ColumnExpr(templates.Name.As("template_name")).
				ColumnExpr(templates.Status.As("template_status")).
				ColumnExpr(imports.TemplateVersion.As("template_version")).
				ColumnExpr(versions.Status.As("version_status")).
				ColumnExpr(imports.LibraryVersionNumber.As("library_version_number")).
				Where(imports.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(imports.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(imports.LibraryID.Eq(), req.LibraryID)
			if req.VersionNumber != nil {
				query = query.Where(imports.LibraryVersionNumber.Eq(), *req.VersionNumber)
			}
			err := query.
				OrderExpr(imports.LibraryVersionNumber.OrderDesc()).
				OrderExpr(imports.TemplateID.OrderAsc()).
				OrderExpr(imports.TemplateVersion.OrderDesc()).
				Limit(int64(limit)).
				Scan(ctx, &rows)
			if err != nil {
				r.l.Error("failed to list formula library usages", zap.Error(err))
				return nil, err
			}

			usages := make([]*formulatemplateports.LibraryUsage, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				usages = append(usages, &formulatemplateports.LibraryUsage{
					TemplateID:           row.TemplateID,
					TemplateName:         row.TemplateName,
					TemplateStatus:       row.TemplateStatus,
					TemplateVersion:      row.TemplateVersion,
					VersionStatus:        row.VersionStatus,
					LibraryVersionNumber: row.LibraryVersionNumber,
				})
			}
			return usages, nil
		},
	)
}

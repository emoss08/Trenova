package formulatemplateversionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultLibraryUsageLimit = 200

var versionErrors = dberror.Mapping{Entity: "FormulaTemplateVersion"}

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

			db := r.db.Keel(ctx)
			if err := formulatemplatedb.FormulaTemplateVersions.Create(
				ctx,
				db,
				version,
			); err != nil {
				log.Error("failed to create formula template version", zap.Error(err))
				return nil, versionErrors.Map(err, version.ID.String())
			}

			if rows := version.ImportRows(); len(rows) > 0 {
				if err := formulatemplatedb.VersionImports.CreateMany(ctx, db, rows); err != nil {
					log.Error("failed to record formula template version imports", zap.Error(err))
					return nil, err
				}
			}

			if rows := version.TableRows(); len(rows) > 0 {
				if err := formulatemplatedb.VersionTables.CreateMany(ctx, db, rows); err != nil {
					log.Error("failed to record formula template version tables", zap.Error(err))
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
			v := formulatemplatedb.FormulaTemplateVersions
			entity, err := v.Where(
				v.TemplateID.Eq(req.TemplateID),
				v.VersionNumber.Eq(req.VersionNumber),
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				if !dberror.IsNotFoundError(err) {
					r.l.Error(
						"failed to get formula template version",
						zap.String("templateID", req.TemplateID.String()),
						zap.Int64("versionNumber", req.VersionNumber),
						zap.Error(err),
					)
				}
				return nil, versionErrors.Map(err, req.TemplateID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *formulatemplateports.ListVersionsRequest,
) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
			v := formulatemplatedb.FormulaTemplateVersions
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				v.Where(v.TemplateID.Eq(req.TemplateID)),
				&listquery.Request[formulatemplate.FormulaTemplateVersion]{
					Filter: req.Filter,
					Shape: func(
						q *keel.Query[*formulatemplate.FormulaTemplateVersion],
					) *keel.Query[*formulatemplate.FormulaTemplateVersion] {
						return q.Load(v.CreatedBy())
					},
					Then: []keel.Order{v.VersionNumber.Desc()},
				},
			)
			if err != nil {
				r.l.Error(
					"failed to list formula template versions",
					zap.String("templateID", req.TemplateID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return result, nil
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
			v := formulatemplatedb.FormulaTemplateVersions
			entities, err := v.Where(
				v.TemplateID.Eq(req.TemplateID),
				v.VersionNumber.In(req.FromVersion, req.ToVersion),
			).Order(v.VersionNumber.Asc()).Limit(2).List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get version range",
					zap.String("templateID", req.TemplateID.String()),
					zap.Int64("fromVersion", req.FromVersion),
					zap.Int64("toVersion", req.ToVersion),
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetLatestVersion(
	ctx context.Context,
	templateID pulid.ID,
	_ pagination.TenantInfo,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
			v := formulatemplatedb.FormulaTemplateVersions
			entity, err := v.Where(v.TemplateID.Eq(templateID)).
				Order(v.VersionNumber.Desc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get latest version",
					zap.String("templateID", templateID.String()),
					zap.Error(err),
				)
				return nil, versionErrors.Map(err, templateID.String())
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
			ft := formulatemplatedb.FormulaTemplates
			entities, err := ft.Where(ft.SourceTemplateID.Eq(req.SourceTemplateID)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get forked templates",
					zap.String("sourceTemplateID", req.SourceTemplateID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) updateVersion(
	ctx context.Context,
	templateID pulid.ID,
	versionNumber int64,
	set keel.Assignment,
) (*formulatemplate.FormulaTemplateVersion, error) {
	v := formulatemplatedb.FormulaTemplateVersions
	rows, err := v.Where(
		v.TemplateID.Eq(templateID),
		v.VersionNumber.Eq(versionNumber),
	).UpdateReturning(ctx, r.db.Keel(ctx), set)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, versionErrors.Map(keel.ErrNotFound, templateID.String())
	}

	return rows[0], nil
}

func (r *repository) UpdateTags(
	ctx context.Context,
	req *formulatemplateports.UpdateVersionTagsRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*formulatemplate.FormulaTemplateVersion, error) {
			tags := req.Tags
			if tags == nil {
				tags = []string{}
			}

			entity, err := r.updateVersion(
				ctx,
				req.TemplateID,
				req.VersionNumber,
				formulatemplatedb.FormulaTemplateVersions.Tags.To(tags),
			)
			if err != nil && !dberror.IsNotFoundError(err) {
				r.l.Error(
					"failed to update version tags",
					zap.String("templateID", req.TemplateID.String()),
					zap.Int64("versionNumber", req.VersionNumber),
					zap.Error(err),
				)
			}

			return entity, err
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
			v := formulatemplatedb.FormulaTemplateVersions
			entity, err := v.Where(
				v.TemplateID.Eq(req.TemplateID),
				v.EffectiveFrom.IsNotNull(),
				v.EffectiveFrom.Lte(req.AsOf),
			).Order(v.EffectiveFrom.Desc(), v.VersionNumber.Desc()).First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // nil version means no scheduled version is in effect
				}
				r.l.Error(
					"failed to get effective formula template version",
					zap.String("templateID", req.TemplateID.String()),
					zap.Int64("asOf", req.AsOf),
					zap.Error(err),
				)
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
			v := formulatemplatedb.FormulaTemplateVersions
			set := v.EffectiveFrom.ToNull()
			if req.EffectiveFrom != nil {
				set = v.EffectiveFrom.To(*req.EffectiveFrom)
			}

			entity, err := r.updateVersion(ctx, req.TemplateID, req.VersionNumber, set)
			if err != nil && !dberror.IsNotFoundError(err) {
				r.l.Error(
					"failed to update version effective date",
					zap.String("templateID", req.TemplateID.String()),
					zap.Int64("versionNumber", req.VersionNumber),
					zap.Error(err),
				)
			}

			return entity, err
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
			v := formulatemplatedb.FormulaTemplateVersions
			versions, err := v.Where(
				v.TemplateID.Eq(req.TemplateID),
				v.Status.Eq(formulatemplate.StatusActive),
				v.EffectiveFrom.Gt(timeutils.NowUnix()),
			).Order(v.EffectiveFrom.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list scheduled versions",
					zap.String("templateID", req.TemplateID.String()),
					zap.Error(err),
				)
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
			v := formulatemplatedb.FormulaTemplateVersions
			versions, err := v.Where(
				v.TemplateID.Eq(req.TemplateID),
				v.Status.Eq(formulatemplate.StatusActive),
			).Order(v.VersionNumber.Asc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list approved versions",
					zap.String("templateID", req.TemplateID.String()),
					zap.Error(err),
				)
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
		v := formulatemplatedb.FormulaTemplateVersions
		authors, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), v.Group(v.CreatedByID).
			Pick(v.CreatedByID).
			Where(v.TemplateID.Eq(req.TemplateID), v.VersionNumber.Gt(req.AfterVersion)))
		if err != nil {
			r.l.Error(
				"failed to list version authors",
				zap.String("templateID", req.TemplateID.String()),
				zap.Error(err),
			)
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
			v := formulatemplatedb.FormulaTemplateVersions
			entity, err := v.Where(v.TemplateID.Eq(req.TemplateID), v.Status.Eq(req.Status)).
				Order(v.VersionNumber.Desc()).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // nil version means the template never held that status
				}
				r.l.Error(
					"failed to get latest formula template version by status",
					zap.String("templateID", req.TemplateID.String()),
					zap.String("status", req.Status.String()),
					zap.Error(err),
				)
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
		v := formulatemplatedb.FormulaTemplateVersions
		cleared, err := v.Where(
			v.TemplateID.Eq(req.TemplateID),
			v.Status.Eq(formulatemplate.StatusActive),
			v.EffectiveFrom.Gt(timeutils.NowUnix()),
		).Update(ctx, r.db.Keel(ctx), v.Status.To(formulatemplate.StatusInactive))
		if err != nil {
			r.l.Error(
				"failed to clear scheduled versions",
				zap.String("templateID", req.TemplateID.String()),
				zap.Error(err),
			)
			return 0, err
		}

		return cleared, nil
	})
}

func usageLimit(limit int) int {
	if limit <= 0 || limit > defaultLibraryUsageLimit {
		return defaultLibraryUsageLimit
	}

	return limit
}

type matrixUsageRow struct {
	TemplateID          pulid.ID
	TemplateName        string
	TemplateStatus      formulatemplate.Status
	TemplateVersion     int64
	VersionStatus       formulatemplate.Status
	MatrixVersionNumber int64
}

func matrixUsageQuery(
	req *formulatemplateports.ListMatrixUsagesRequest,
) *keel.Query[*formulatemplate.VersionTable] {
	vt := formulatemplatedb.VersionTables
	v := formulatemplatedb.FormulaTemplateVersions
	ft := formulatemplatedb.FormulaTemplates

	q := vt.Join(v, v.ID.EqCol(vt.TemplateVersionID)).
		Join(ft, ft.ID.EqCol(vt.TemplateID)).
		Pick(vt.TemplateID, ft.Name, ft.Status, vt.TemplateVersion, v.Status, vt.MatrixVersionNumber).
		Where(vt.RateMatrixID.Eq(req.RateMatrixID))
	if req.VersionNumber != nil {
		q = q.Where(vt.MatrixVersionNumber.Eq(*req.VersionNumber))
	}

	return q.Order(vt.MatrixVersionNumber.Desc(), vt.TemplateID.Asc(), vt.TemplateVersion.Desc()).
		Limit(usageLimit(req.Limit))
}

func (r *repository) ListMatrixUsages(
	ctx context.Context,
	req *formulatemplateports.ListMatrixUsagesRequest,
) ([]*formulatemplateports.MatrixUsage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplateports.MatrixUsage, error) {
			rows, err := keel.Into[matrixUsageRow](ctx, r.db.Keel(ctx), matrixUsageQuery(req))
			if err != nil {
				r.l.Error("failed to list rate matrix usages", zap.Error(err))
				return nil, err
			}

			usages := make([]*formulatemplateports.MatrixUsage, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				usages = append(usages, &formulatemplateports.MatrixUsage{
					TemplateID:          row.TemplateID,
					TemplateName:        row.TemplateName,
					TemplateStatus:      row.TemplateStatus,
					TemplateVersion:     row.TemplateVersion,
					VersionStatus:       row.VersionStatus,
					MatrixVersionNumber: row.MatrixVersionNumber,
				})
			}

			return usages, nil
		},
	)
}

type libraryUsageRow struct {
	TemplateID           pulid.ID
	TemplateName         string
	TemplateStatus       formulatemplate.Status
	TemplateVersion      int64
	VersionStatus        formulatemplate.Status
	LibraryVersionNumber int64
}

func libraryUsageQuery(
	req *formulatemplateports.ListLibraryUsagesRequest,
) *keel.Query[*formulatemplate.VersionImport] {
	vi := formulatemplatedb.VersionImports
	v := formulatemplatedb.FormulaTemplateVersions
	ft := formulatemplatedb.FormulaTemplates

	q := vi.Join(v, v.ID.EqCol(vi.TemplateVersionID)).
		Join(ft, ft.ID.EqCol(vi.TemplateID)).
		Pick(vi.TemplateID, ft.Name, ft.Status, vi.TemplateVersion, v.Status, vi.LibraryVersionNumber).
		Where(vi.LibraryID.Eq(req.LibraryID))
	if req.VersionNumber != nil {
		q = q.Where(vi.LibraryVersionNumber.Eq(*req.VersionNumber))
	}

	return q.Order(vi.LibraryVersionNumber.Desc(), vi.TemplateID.Asc(), vi.TemplateVersion.Desc()).
		Limit(usageLimit(req.Limit))
}

func (r *repository) ListLibraryUsages(
	ctx context.Context,
	req *formulatemplateports.ListLibraryUsagesRequest,
) ([]*formulatemplateports.LibraryUsage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*formulatemplateports.LibraryUsage, error) {
			rows, err := keel.Into[libraryUsageRow](ctx, r.db.Keel(ctx), libraryUsageQuery(req))
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

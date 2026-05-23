//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editemplaterepository

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.EDITemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-template-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListEDITemplatesRequest,
) *bun.SelectQuery {
	return nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("req", req),
	)

	entities := make([]*edi.EDITemplate, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDITemplateColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.EDITemplateRelations.DocumentType).
		Apply(buncolgen.EDITemplateApplyTenant(req.Filter.TenantInfo))
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	query = applyTemplateSearch(query, req.Filter.Query)
	total, err := query.
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count edi templates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITemplate]{Items: entities, Total: total}, nil
}

func (r *repository) listDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	entities := make([]*edi.EDIDocumentType, 0, 8)
	cols := buncolgen.EDIDocumentTypeColumns
	query := r.db.DBForContext(ctx).NewSelect().Model(&entities).Order(cols.Code.OrderAsc())
	query = filterDocumentTypesQuery(query, req)
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func filterDocumentTypesQuery(
	query *bun.SelectQuery,
	req repositories.ListEDIDocumentTypesRequest,
) *bun.SelectQuery {
	cols := buncolgen.EDIDocumentTypeColumns
	if req.Standard != "" {
		query = query.Where(cols.Standard.Eq(), req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	return query
}

func (r *repository) SelectTemplateOptions(
	ctx context.Context,
	req *repositories.EDITemplateSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	entities := make([]*edi.EDITemplate, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	cols := buncolgen.EDITemplateColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.OrganizationID.Bare(),
			cols.DocumentTypeID.Bare(),
			cols.Name.Bare(),
			cols.Description.Bare(),
			cols.Direction.Bare(),
			cols.Standard.Bare(),
			cols.TransactionSet.Bare(),
			cols.Status.Bare(),
		).
		Relation(buncolgen.EDITemplateRelations.DocumentType).
		Apply(buncolgen.EDITemplateApplyTenant(req.SelectQueryRequest.TenantInfo))
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	query = applyTemplateSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order(cols.Name.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITemplate]{Items: entities, Total: total}, nil
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req repositories.GetEDITemplateByIDRequest,
) (*edi.EDITemplate, error) {
	entity := new(edi.EDITemplate)
	cols := buncolgen.EDITemplateColumns
	rel := buncolgen.EDITemplateRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.DocumentType).
		Relation(rel.Versions, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(buncolgen.EDITemplateVersionColumns.VersionNumber.OrderDesc())
		}).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDITemplateApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITemplate")
	}
	return entity, nil
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	req *repositories.CreateEDITemplateRequest,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(req.Template).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		req.Version.TemplateID = req.Template.ID
		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(req.Version).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		for _, segment := range req.Segments {
			segment.TemplateVersionID = req.Version.ID
			segment.BusinessUnitID = req.Version.BusinessUnitID
			segment.OrganizationID = req.Version.OrganizationID
		}
		if len(req.Segments) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.Segments).Exec(c); err != nil {
				return err
			}
		}
		prepareTemplateScriptLibraries(req.Version, req.ScriptLibraries)
		if len(req.ScriptLibraries) > 0 {
			if _, err := r.db.DBForContext(c).
				NewInsert().
				Model(&req.ScriptLibraries).
				Exec(c); err != nil {
				return err
			}
		}
		req.Version.Segments = req.Segments
		req.Version.ScriptLibraries = req.ScriptLibraries
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return req.Template, req.Version, nil
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *edi.EDITemplate,
) (*edi.EDITemplate, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDITemplateColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Column(
			cols.Name.Bare(),
			cols.Description.Bare(),
			cols.Status.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDITemplate", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListTemplateVersions(
	ctx context.Context,
	req repositories.ListEDITemplateVersionsRequest,
) ([]*edi.EDITemplateVersion, error) {
	entities := make([]*edi.EDITemplateVersion, 0, 8)
	cols := buncolgen.EDITemplateVersionColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.TemplateID.Eq(), req.TemplateID).
		Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo)).
		Order(cols.VersionNumber.OrderDesc()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) GetTemplateVersionByID(
	ctx context.Context,
	req repositories.GetEDITemplateVersionByIDRequest,
) (*edi.EDITemplateVersion, error) {
	entity := new(edi.EDITemplateVersion)
	cols := buncolgen.EDITemplateVersionColumns
	rel := buncolgen.EDITemplateVersionRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Template).
		Relation(rel.Segments, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(buncolgen.EDITemplateSegmentColumns.Sequence.OrderAsc())
		}).
		Relation(rel.ScriptLibraries, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr(buncolgen.EDITemplateScriptLibraryColumns.Name.Expr("lower({}) ASC"))
		}).
		Where(cols.ID.Eq(), req.VersionID).
		Where(cols.TemplateID.Eq(), req.TemplateID).
		Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITemplateVersion")
	}
	return entity, nil
}

func (r *repository) GetActiveTemplateVersion(
	ctx context.Context,
	req repositories.GetActiveEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	entity := new(edi.EDITemplateVersion)
	cols := buncolgen.EDITemplateVersionColumns
	rel := buncolgen.EDITemplateVersionRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Segments, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order(buncolgen.EDITemplateSegmentColumns.Sequence.OrderAsc())
		}).
		Relation(rel.ScriptLibraries, func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr(buncolgen.EDITemplateScriptLibraryColumns.Name.Expr("lower({}) ASC"))
		}).
		Where(cols.TemplateID.Eq(), req.TemplateID).
		Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo))
	if !req.VersionID.IsNil() {
		query = query.Where(cols.ID.Eq(), req.VersionID)
	} else {
		query = query.Where(cols.IsActive.IsTrue())
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITemplateVersion")
	}
	return entity, nil
}

func (r *repository) CreateTemplateVersion(
	ctx context.Context,
	req *repositories.CreateEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	version := req.Version
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(version).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		for _, segment := range req.Segments {
			segment.TemplateVersionID = version.ID
			segment.BusinessUnitID = version.BusinessUnitID
			segment.OrganizationID = version.OrganizationID
		}
		if len(req.Segments) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.Segments).Exec(c); err != nil {
				return err
			}
		}
		prepareTemplateScriptLibraries(version, req.ScriptLibraries)
		if len(req.ScriptLibraries) > 0 {
			if _, err := r.db.DBForContext(c).
				NewInsert().
				Model(&req.ScriptLibraries).
				Exec(c); err != nil {
				return err
			}
		}
		version.Segments = req.Segments
		version.ScriptLibraries = req.ScriptLibraries
		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *repository) UpdateTemplateVersionMetadata(
	ctx context.Context,
	version *edi.EDITemplateVersion,
) (*edi.EDITemplateVersion, error) {
	ov := version.Version
	version.Version++
	cols := buncolgen.EDITemplateVersionColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(version).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Column(
			cols.X12Version.Bare(),
			cols.FunctionalGroupID.Bare(),
			cols.Status.Bare(),
			cols.IsActive.Bare(),
			cols.Notes.Bare(),
			cols.CertificationNotes.Bare(),
			cols.ActivationNotes.Bare(),
			cols.ArchiveNotes.Bare(),
			cols.DeprecatedNotes.Bare(),
			cols.SupersededNotes.Bare(),
			cols.CertifiedByID.Bare(),
			cols.ActivatedByID.Bare(),
			cols.ArchivedByID.Bare(),
			cols.DeprecatedByID.Bare(),
			cols.SupersededByID.Bare(),
			cols.CertifiedAt.Bare(),
			cols.ActivatedAt.Bare(),
			cols.ArchivedAt.Bare(),
			cols.DeprecatedAt.Bare(),
			cols.SupersededAt.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDITemplateVersion",
		version.ID.String(),
	); err != nil {
		return nil, err
	}
	if version.Status != edi.TemplateStatusDraft {
		if err = r.updateTemplateScriptLibraryStatus(ctx, version, version.Status); err != nil {
			return nil, err
		}
	}
	return version, nil
}

func (r *repository) ReplaceTemplateVersionSegments(
	ctx context.Context,
	req repositories.ReplaceEDITemplateVersionSegmentsRequest,
) (*edi.EDITemplateVersion, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.UpdateTemplateVersionMetadata(c, req.Version); err != nil {
			return err
		}
		if _, err := r.db.DBForContext(c).
			NewDelete().
			Model((*edi.EDITemplateSegment)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				cols := buncolgen.EDITemplateSegmentColumns
				return buncolgen.EDITemplateSegmentScopeTenantDelete(dq, pagination.TenantInfo{
					OrgID: req.Version.OrganizationID,
					BuID:  req.Version.BusinessUnitID,
				}).Where(cols.TemplateVersionID.Eq(), req.Version.ID)
			}).
			Exec(c); err != nil {
			return err
		}
		for _, segment := range req.Segments {
			segment.TemplateVersionID = req.Version.ID
			segment.BusinessUnitID = req.Version.BusinessUnitID
			segment.OrganizationID = req.Version.OrganizationID
		}
		if len(req.Segments) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.Segments).Exec(c); err != nil {
				return err
			}
		}
		req.Version.Segments = req.Segments
		return nil
	})
	if err != nil {
		return nil, err
	}
	return req.Version, nil
}

func (r *repository) ListTemplateScriptLibraries(
	ctx context.Context,
	req repositories.ListEDITemplateScriptLibrariesRequest,
) ([]*edi.EDITemplateScriptLibrary, error) {
	entities := make([]*edi.EDITemplateScriptLibrary, 0, 4)
	cols := buncolgen.EDITemplateScriptLibraryColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.TemplateVersionID.Eq(), req.VersionID).
		Apply(buncolgen.EDITemplateScriptLibraryApplyTenant(req.TenantInfo)).
		Where(
			`EXISTS (
				SELECT 1 FROM edi_template_versions etv
				WHERE etv.id = etsl.template_version_id
					AND etv.template_id = ?
					AND etv.organization_id = etsl.organization_id
					AND etv.business_unit_id = etsl.business_unit_id
			)`,
			req.TemplateID,
		).
		OrderExpr(cols.Name.Expr("lower({}) ASC")).
		Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) ReplaceTemplateVersionScriptLibraries(
	ctx context.Context,
	req repositories.ReplaceEDITemplateVersionScriptLibrariesRequest,
) (*edi.EDITemplateVersion, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.UpdateTemplateVersionMetadata(c, req.Version); err != nil {
			return err
		}
		if _, err := r.db.DBForContext(c).
			NewDelete().
			Model((*edi.EDITemplateScriptLibrary)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				cols := buncolgen.EDITemplateScriptLibraryColumns
				return buncolgen.EDITemplateScriptLibraryScopeTenantDelete(
					dq,
					pagination.TenantInfo{
						OrgID: req.Version.OrganizationID,
						BuID:  req.Version.BusinessUnitID,
					},
				).Where(cols.TemplateVersionID.Eq(), req.Version.ID)
			}).
			Exec(c); err != nil {
			return err
		}
		prepareTemplateScriptLibraries(req.Version, req.ScriptLibraries)
		if len(req.ScriptLibraries) > 0 {
			if _, err := r.db.DBForContext(c).
				NewInsert().
				Model(&req.ScriptLibraries).
				Exec(c); err != nil {
				return err
			}
		}
		req.Version.ScriptLibraries = req.ScriptLibraries
		return nil
	})
	if err != nil {
		return nil, err
	}
	return req.Version, nil
}

//nolint:funlen // Template activation updates version state and seed profiles in one transaction.
func (r *repository) ActivateTemplateVersion(
	ctx context.Context,
	req repositories.ActivateEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	version := new(edi.EDITemplateVersion)
	now := timeutils.NowUnix()
	cols := buncolgen.EDITemplateVersionColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(version).
			Relation(buncolgen.EDITemplateVersionRelations.Segments, func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order(buncolgen.EDITemplateSegmentColumns.Sequence.OrderAsc())
			}).
			Where(cols.ID.Eq(), req.VersionID).
			Where(cols.TemplateID.Eq(), req.TemplateID).
			Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo)).
			For("UPDATE").
			Scan(c); err != nil {
			return dberror.HandleNotFoundError(err, "EDITemplateVersion")
		}

		active := make([]*edi.EDITemplateVersion, 0, 1)
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(&active).
			Where(cols.TemplateID.Eq(), req.TemplateID).
			Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo)).
			Where(cols.IsActive.IsTrue()).
			For("UPDATE").
			Scan(c); err != nil {
			return err
		}
		for _, current := range active {
			if current.ID == version.ID {
				continue
			}
			current.IsActive = false
			current.Status = edi.TemplateStatusSuperseded
			current.SupersededByID = req.ActorID
			current.SupersededAt = &now
			current.SupersededNotes = req.Notes
			current.Version++
			if _, err := r.db.DBForContext(c).
				NewUpdate().
				Model(current).
				WherePK().
				Column(
					cols.Status.Bare(),
					cols.IsActive.Bare(),
					cols.SupersededByID.Bare(),
					cols.SupersededAt.Bare(),
					cols.SupersededNotes.Bare(),
					cols.Version.Bare(),
					cols.UpdatedAt.Bare(),
				).
				Exec(c); err != nil {
				return err
			}
			if err := r.updateTemplateScriptLibraryStatus(
				c,
				current,
				edi.TemplateStatusSuperseded,
			); err != nil {
				return err
			}
		}

		version.Status = edi.TemplateStatusActive
		version.IsActive = true
		version.ActivatedByID = req.ActorID
		version.ActivatedAt = &now
		version.ActivationNotes = req.Notes
		version.Version++
		if _, err := r.db.DBForContext(c).
			NewUpdate().
			Model(version).
			WherePK().
			Column(
				cols.Status.Bare(),
				cols.IsActive.Bare(),
				cols.ActivatedByID.Bare(),
				cols.ActivatedAt.Bare(),
				cols.ActivationNotes.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		if err := r.updateTemplateScriptLibraryStatus(
			c,
			version,
			edi.TemplateStatusActive,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func (r *repository) ArchiveTemplateVersion(
	ctx context.Context,
	req repositories.ArchiveEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	version := new(edi.EDITemplateVersion)
	now := timeutils.NowUnix()
	cols := buncolgen.EDITemplateVersionColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(version).
			Relation(buncolgen.EDITemplateVersionRelations.Segments, func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order(buncolgen.EDITemplateSegmentColumns.Sequence.OrderAsc())
			}).
			Where(cols.ID.Eq(), req.VersionID).
			Where(cols.TemplateID.Eq(), req.TemplateID).
			Apply(buncolgen.EDITemplateVersionApplyTenant(req.TenantInfo)).
			For("UPDATE").
			Scan(c); err != nil {
			return dberror.HandleNotFoundError(err, "EDITemplateVersion")
		}
		version.Status = edi.TemplateStatusArchived
		version.IsActive = false
		version.ArchivedByID = req.ActorID
		version.ArchivedAt = &now
		version.ArchiveNotes = req.Notes
		version.Version++
		if _, err := r.db.DBForContext(c).
			NewUpdate().
			Model(version).
			WherePK().
			Column(
				cols.Status.Bare(),
				cols.IsActive.Bare(),
				cols.ArchivedByID.Bare(),
				cols.ArchivedAt.Bare(),
				cols.ArchiveNotes.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		if err := r.updateTemplateScriptLibraryStatus(
			c,
			version,
			edi.TemplateStatusArchived,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return version, nil
}

func prepareTemplateScriptLibraries(
	version *edi.EDITemplateVersion,
	libraries []*edi.EDITemplateScriptLibrary,
) {
	for _, library := range libraries {
		if library == nil {
			continue
		}
		library.ID = pulid.Nil
		library.TemplateVersionID = version.ID
		library.BusinessUnitID = version.BusinessUnitID
		library.OrganizationID = version.OrganizationID
		library.Status = version.Status
		library.Version = 0
	}
}

func (r *repository) updateTemplateScriptLibraryStatus(
	ctx context.Context,
	version *edi.EDITemplateVersion,
	status edi.TemplateStatus,
) error {
	cols := buncolgen.EDITemplateScriptLibraryColumns

	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.EDITemplateScriptLibrary)(nil)).
		Set(cols.Status.Set(), status).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Where(cols.TemplateVersionID.Eq(), version.ID).
		Where(cols.OrganizationID.Eq(), version.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), version.BusinessUnitID).
		Exec(ctx)
	return err
}

//nolint:funlen // Base template seeding is a declarative create-or-update repository operation.
func (r *repository) EnsureBase204Template(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	template := new(edi.EDITemplate)
	version := new(edi.EDITemplateVersion)
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		cols := buncolgen.EDITemplateColumns

		err := r.db.DBForContext(c).
			NewSelect().
			Model(template).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.EDITemplateScopeTenant(sq, tenantInfo).
					Where(cols.Standard.Eq(), edi.EDIStandardX12).
					Where(cols.TransactionSet.Eq(), edi.TransactionSet204).
					Where(cols.Direction.Eq(), edi.DocumentDirectionOutbound).
					Where(cols.Name.Eq(), "Base X12 204 Outbound")
			}).
			Limit(1).
			Scan(c)
		if err != nil && !dberror.IsNotFoundError(err) {
			return err
		}
		if err == nil {
			existing, versionErr := r.GetActiveTemplateVersion(
				c,
				repositories.GetActiveEDITemplateVersionRequest{
					TemplateID: template.ID,
					TenantInfo: tenantInfo,
				},
			)
			if versionErr != nil {
				return versionErr
			}
			*version = *existing
			return nil
		}

		documentTypes, err := r.listDocumentTypes(c, repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		})
		if err != nil {
			return err
		}
		if len(documentTypes) == 0 {
			return errors.New("x12 204 outbound document type is not seeded")
		}

		template = &edi.EDITemplate{
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			DocumentTypeID: documentTypes[0].ID,
			Name:           "Base X12 204 Outbound",
			Description:    "Tenant-scoped base outbound X12 204 template",
			Direction:      edi.DocumentDirectionOutbound,
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Status:         edi.TemplateStatusActive,
		}
		if _, err = r.db.DBForContext(c).
			NewInsert().
			Model(template).
			Returning("*").
			Exec(c); err != nil {
			return err
		}

		activatedAt := timeutils.NowUnix()
		version = &edi.EDITemplateVersion{
			BusinessUnitID:    tenantInfo.BuID,
			OrganizationID:    tenantInfo.OrgID,
			TemplateID:        template.ID,
			VersionNumber:     1,
			X12Version:        edi.DefaultX12204Version,
			FunctionalGroupID: "SM",
			Status:            edi.TemplateStatusActive,
			IsActive:          true,
			Notes:             "Seeded base 004010 Motor Carrier Load Tender profile",
			ActivatedAt:       &activatedAt,
		}
		if _, err = r.db.DBForContext(c).
			NewInsert().
			Model(version).
			Returning("*").
			Exec(c); err != nil {
			return err
		}

		segments := editemplates.Base204Segments(tenantInfo, version.ID)
		if _, err = r.db.DBForContext(c).NewInsert().Model(&segments).Exec(c); err != nil {
			return err
		}
		version.Segments = segments
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return template, version, nil
}

func applyTemplateSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDITemplateColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Name.LowerLike(), term).
			WhereOr(cols.Description.LowerLike(), term)
	})
}

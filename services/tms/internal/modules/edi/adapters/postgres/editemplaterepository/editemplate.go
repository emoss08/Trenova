//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editemplaterepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	editemplates "github.com/emoss08/trenova/internal/modules/edi/domain/edi/templates"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	templateErrors = dberror.Mapping{
		Entity: "EDITemplate",
		Unique: []dberror.Constraint{
			{
				Name:    "idx_edi_templates_name_org_bu",
				Field:   "name",
				Message: "An EDI template with this name already exists in your organization",
			},
		},
	}
	versionErrors = dberror.Mapping{
		Entity: "EDITemplateVersion",
		Unique: []dberror.Constraint{
			{
				Name:    "idx_edi_template_versions_number",
				Field:   "versionNumber",
				Message: "This template already has a version with this number",
			},
		},
	}
	segmentErrors = dberror.Mapping{Entity: "EDITemplateSegment"}
	libraryErrors = dberror.Mapping{
		Entity: "EDITemplateScriptLibrary",
		Unique: []dberror.Constraint{
			{
				Name:    "idx_edi_template_script_libraries_name",
				Field:   "scriptLibraries",
				Message: "Script library names must be unique within a template version",
			},
		},
	}
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

func New(p Params) ediports.EDITemplateRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-template-repository"),
	}
}

type templateFilter struct {
	transactionSet edi.TransactionSet
	direction      edi.DocumentDirection
	status         edi.TemplateStatus
	search         string
}

func (f *templateFilter) where() []keel.Pred {
	t := edidb.EDITemplates
	search := strings.TrimSpace(f.search)
	return []keel.Pred{
		keel.When(f.transactionSet != "", t.TransactionSet.Eq(f.transactionSet)),
		keel.When(f.direction != "", t.Direction.Eq(f.direction)),
		keel.When(f.status != "", t.Status.Eq(f.status)),
		keel.When(search != "", keel.Or(t.Name.ContainsFold(search), t.Description.ContainsFold(search))),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *ediports.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITemplate], error) {
			t := edidb.EDITemplates
			filter := templateFilter{
				transactionSet: req.TransactionSet,
				direction:      req.Direction,
				status:         req.Status,
				search:         req.Filter.Query,
			}
			q := t.Where(filter.where()...).Load(t.DocumentType())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				r.l.Error("failed to list edi templates", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *ediports.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return r.List(ctx, req)
}

func (r *repository) ListTemplatesCursor(
	ctx context.Context,
	req *ediports.ListEDITemplatesRequest,
) (*pagination.CursorListResult[*edi.EDITemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDITemplate], error) {
			t := edidb.EDITemplates
			filter := templateFilter{
				transactionSet: req.TransactionSet,
				direction:      req.Direction,
				status:         req.Status,
			}

			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				t.Where(filter.where()...),
				&listquery.Request[edi.EDITemplate]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withDocumentTypeAndVersions,
				},
			)
		},
	)
}

func (r *repository) SelectTemplateOptions(
	ctx context.Context,
	req *ediports.EDITemplateSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITemplate], error) {
			return listquery.Offset(
				ctx,
				r.db.Keel(ctx),
				templateOptionsQuery(req),
				req.SelectQueryRequest.Pagination,
			)
		},
	)
}

func templateOptionsQuery(
	req *ediports.EDITemplateSelectOptionsRequest,
) *keel.Query[edi.EDITemplate] {
	t := edidb.EDITemplates
	filter := templateFilter{
		transactionSet: req.TransactionSet,
		direction:      req.Direction,
		status:         req.Status,
		search:         req.SelectQueryRequest.Query,
	}

	return t.Pick(
		t.ID,
		t.BusinessUnitID,
		t.OrganizationID,
		t.DocumentTypeID,
		t.Name,
		t.Description,
		t.Direction,
		t.Standard,
		t.TransactionSet,
		t.Status,
	).
		Where(filter.where()...).
		Load(t.DocumentType()).
		Order(t.Name.Asc())
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req ediports.GetEDITemplateByIDRequest,
) (*edi.EDITemplate, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITemplate, error) {
		t := edidb.EDITemplates
		entity, err := withDocumentTypeAndVersions(t.Where(t.ID.Eq(req.ID))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, templateErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTemplate(
	ctx context.Context,
	req *ediports.CreateEDITemplateRequest,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
			db := r.db.Keel(ctx)
			if err := edidb.EDITemplates.Create(ctx, db, req.Template); err != nil {
				return nil, nil, templateErrors.Map(err, req.Template.ID.String())
			}

			req.Version.TemplateID = req.Template.ID
			if err := createVersion(ctx, db, req.Version, req.Segments, req.ScriptLibraries); err != nil {
				return nil, nil, err
			}

			return req.Template, req.Version, nil
		},
	)
}

func (r *repository) UpdateTemplate(
	ctx context.Context,
	entity *edi.EDITemplate,
) (*edi.EDITemplate, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplate, error) {
		t := edidb.EDITemplates
		if err := t.Save(ctx, r.db.Keel(ctx), entity, t.Name, t.Description, t.Status); err != nil {
			return nil, templateErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListTemplateVersions(
	ctx context.Context,
	req ediports.ListEDITemplateVersionsRequest,
) ([]*edi.EDITemplateVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDITemplateVersion, error) {
		tv := edidb.EDITemplateVersions
		return tv.Where(tv.TemplateID.Eq(req.TemplateID)).
			Order(tv.VersionNumber.Desc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetTemplateVersionByID(
	ctx context.Context,
	req ediports.GetEDITemplateVersionByIDRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		tv := edidb.EDITemplateVersions
		entity, err := withContents(tv.Where(tv.ID.Eq(req.VersionID), tv.TemplateID.Eq(req.TemplateID))).
			Load(tv.Template()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, versionErrors.Map(err, req.VersionID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActiveTemplateVersion(
	ctx context.Context,
	req ediports.GetActiveEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		tv := edidb.EDITemplateVersions
		selected := tv.IsActive.IsTrue()
		if req.VersionID.IsNotNil() {
			selected = tv.ID.Eq(req.VersionID)
		}

		entity, err := withContents(tv.Where(tv.TemplateID.Eq(req.TemplateID), selected)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, versionErrors.Map(err, req.TemplateID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTemplateVersion(
	ctx context.Context,
	req *ediports.CreateEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		err := createVersion(ctx, r.db.Keel(ctx), req.Version, req.Segments, req.ScriptLibraries)
		if err != nil {
			return nil, err
		}

		return req.Version, nil
	})
}

func (r *repository) UpdateTemplateVersionMetadata(
	ctx context.Context,
	version *edi.EDITemplateVersion,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		if err := saveVersionMetadata(ctx, r.db.Keel(ctx), version); err != nil {
			return nil, err
		}

		return version, nil
	})
}

func (r *repository) ReplaceTemplateVersionSegments(
	ctx context.Context,
	req ediports.ReplaceEDITemplateVersionSegmentsRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		db := r.db.Keel(ctx)
		if err := saveVersionMetadata(ctx, db, req.Version); err != nil {
			return nil, err
		}

		seg := edidb.EDITemplateSegments
		if _, err := seg.Where(seg.TemplateVersionID.Eq(req.Version.ID)).Delete(ctx, db); err != nil {
			return nil, segmentErrors.Map(err, req.Version.ID.String())
		}
		if err := createSegments(ctx, db, req.Version, req.Segments); err != nil {
			return nil, err
		}
		req.Version.Segments = req.Segments

		return req.Version, nil
	})
}

func (r *repository) ListTemplateScriptLibraries(
	ctx context.Context,
	req ediports.ListEDITemplateScriptLibrariesRequest,
) ([]*edi.EDITemplateScriptLibrary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDITemplateScriptLibrary, error) {
		sl := edidb.EDITemplateScriptLibraries
		tv := edidb.EDITemplateVersions
		return sl.Where(
			sl.TemplateVersionID.Eq(req.VersionID),
			keel.Exists(tv.Where(
				tv.ID.EqCol(sl.TemplateVersionID),
				tv.TemplateID.Eq(req.TemplateID),
			)),
		).
			Order(sl.Name.Lower().Asc()).
			All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ReplaceTemplateVersionScriptLibraries(
	ctx context.Context,
	req ediports.ReplaceEDITemplateVersionScriptLibrariesRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		db := r.db.Keel(ctx)
		if err := saveVersionMetadata(ctx, db, req.Version); err != nil {
			return nil, err
		}

		sl := edidb.EDITemplateScriptLibraries
		if _, err := sl.Where(sl.TemplateVersionID.Eq(req.Version.ID)).Delete(ctx, db); err != nil {
			return nil, libraryErrors.Map(err, req.Version.ID.String())
		}
		if err := createLibraries(ctx, db, req.Version, req.ScriptLibraries); err != nil {
			return nil, err
		}
		req.Version.ScriptLibraries = req.ScriptLibraries

		return req.Version, nil
	})
}

func (r *repository) ActivateTemplateVersion(
	ctx context.Context,
	req ediports.ActivateEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		db := r.db.Keel(ctx)
		tv := edidb.EDITemplateVersions
		now := timeutils.NowUnix()

		version, err := lockVersion(ctx, db, req.TemplateID, req.VersionID)
		if err != nil {
			return nil, err
		}

		active, err := tv.Where(tv.TemplateID.Eq(req.TemplateID), tv.IsActive.IsTrue()).
			Lock(keel.ForUpdate).
			All(ctx, db)
		if err != nil {
			return nil, versionErrors.Map(err, req.TemplateID.String())
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
			if err = tv.Save(
				ctx,
				db,
				current,
				tv.Status,
				tv.IsActive,
				tv.SupersededByID,
				tv.SupersededAt,
				tv.SupersededNotes,
			); err != nil {
				return nil, versionErrors.Map(err, current.ID.String())
			}
			if err = setLibraryStatus(ctx, db, current, edi.TemplateStatusSuperseded); err != nil {
				return nil, err
			}
		}

		version.Status = edi.TemplateStatusActive
		version.IsActive = true
		version.ActivatedByID = req.ActorID
		version.ActivatedAt = &now
		version.ActivationNotes = req.Notes
		if err = tv.Save(
			ctx,
			db,
			version,
			tv.Status,
			tv.IsActive,
			tv.ActivatedByID,
			tv.ActivatedAt,
			tv.ActivationNotes,
		); err != nil {
			return nil, versionErrors.Map(err, version.ID.String())
		}
		if err = setLibraryStatus(ctx, db, version, edi.TemplateStatusActive); err != nil {
			return nil, err
		}

		return version, nil
	})
}

func (r *repository) ArchiveTemplateVersion(
	ctx context.Context,
	req ediports.ArchiveEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITemplateVersion, error) {
		db := r.db.Keel(ctx)
		tv := edidb.EDITemplateVersions
		now := timeutils.NowUnix()

		version, err := lockVersion(ctx, db, req.TemplateID, req.VersionID)
		if err != nil {
			return nil, err
		}

		version.Status = edi.TemplateStatusArchived
		version.IsActive = false
		version.ArchivedByID = req.ActorID
		version.ArchivedAt = &now
		version.ArchiveNotes = req.Notes
		if err = tv.Save(
			ctx,
			db,
			version,
			tv.Status,
			tv.IsActive,
			tv.ArchivedByID,
			tv.ArchivedAt,
			tv.ArchiveNotes,
		); err != nil {
			return nil, versionErrors.Map(err, version.ID.String())
		}
		if err = setLibraryStatus(ctx, db, version, edi.TemplateStatusArchived); err != nil {
			return nil, err
		}

		return version, nil
	})
}

func (r *repository) EnsureBase204Template(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	return r.EnsureBaseTemplate(ctx, tenantInfo, edi.TransactionSet204)
}

func (r *repository) EnsureBaseTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	transactionSet edi.TransactionSet,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	return r.EnsureStarterTemplate(ctx, tenantInfo, transactionSet, edi.DocumentDirectionOutbound)
}

func (r *repository) EnsureStarterTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	transactionSet edi.TransactionSet,
	direction edi.DocumentDirection,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
			db := r.db.Keel(ctx)
			t := edidb.EDITemplates
			templateName := fmt.Sprintf("Base X12 %s %s", transactionSet, direction)

			template, err := t.Where(
				t.Standard.Eq(edi.EDIStandardX12),
				t.TransactionSet.Eq(transactionSet),
				t.Direction.Eq(direction),
				t.Name.Eq(templateName),
			).First(ctx, db)
			switch {
			case err == nil:
				version, versionErr := r.GetActiveTemplateVersion(
					ctx,
					ediports.GetActiveEDITemplateVersionRequest{
						TemplateID: template.ID,
						TenantInfo: tenantInfo,
					},
				)
				if versionErr != nil {
					return nil, nil, versionErr
				}
				return template, version, nil
			case !dberror.IsNotFoundError(err):
				return nil, nil, templateErrors.Map(err, templateName)
			}

			return createStarterTemplate(ctx, db, &starterTemplate{
				tenant:         tenantInfo,
				transactionSet: transactionSet,
				direction:      direction,
				name:           templateName,
			})
		},
	)
}

type starterTemplate struct {
	tenant         pagination.TenantInfo
	transactionSet edi.TransactionSet
	direction      edi.DocumentDirection
	name           string
}

func createStarterTemplate(
	ctx context.Context,
	db *keel.DB,
	starter *starterTemplate,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	dt := edidb.EDIDocumentTypes
	documentType, err := dt.Where(
		dt.Standard.Eq(edi.EDIStandardX12),
		dt.TransactionSet.Eq(starter.transactionSet),
		dt.Direction.Eq(starter.direction),
	).Order(dt.Code.Asc()).First(ctx, db)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil, fmt.Errorf(
				"x12 %s %s document type is not seeded",
				starter.transactionSet,
				strings.ToLower(string(starter.direction)),
			)
		}
		return nil, nil, err
	}

	direction := strings.ToLower(string(starter.direction))
	template := &edi.EDITemplate{
		BusinessUnitID: starter.tenant.BuID,
		OrganizationID: starter.tenant.OrgID,
		DocumentTypeID: documentType.ID,
		Name:           starter.name,
		Description: fmt.Sprintf(
			"Tenant-scoped base %s X12 %s template",
			direction,
			starter.transactionSet,
		),
		Direction:      starter.direction,
		Standard:       edi.EDIStandardX12,
		TransactionSet: starter.transactionSet,
		Status:         edi.TemplateStatusActive,
	}
	if err = edidb.EDITemplates.Create(ctx, db, template); err != nil {
		return nil, nil, templateErrors.Map(err, template.ID.String())
	}

	x12Version := edi.DefaultX12204Version
	if starter.transactionSet == edi.TransactionSet999 {
		x12Version = "005010"
	}
	activatedAt := timeutils.NowUnix()
	version := &edi.EDITemplateVersion{
		BusinessUnitID:    starter.tenant.BuID,
		OrganizationID:    starter.tenant.OrgID,
		TemplateID:        template.ID,
		VersionNumber:     1,
		X12Version:        x12Version,
		FunctionalGroupID: edi.FunctionalGroupDefault(starter.transactionSet),
		Status:            edi.TemplateStatusActive,
		IsActive:          true,
		Notes: fmt.Sprintf(
			"Seeded base %s X12 %s %s profile",
			x12Version,
			starter.transactionSet,
			direction,
		),
		ActivatedAt: &activatedAt,
	}
	if err = edidb.EDITemplateVersions.Create(ctx, db, version); err != nil {
		return nil, nil, versionErrors.Map(err, version.ID.String())
	}

	segments, err := editemplates.StarterSegments(starter.tenant, version.ID, starter.transactionSet)
	if err != nil {
		return nil, nil, err
	}
	if err = createSegments(ctx, db, version, segments); err != nil {
		return nil, nil, err
	}
	version.Segments = segments

	return template, version, nil
}

func createVersion(
	ctx context.Context,
	db *keel.DB,
	version *edi.EDITemplateVersion,
	segments []*edi.EDITemplateSegment,
	libraries []*edi.EDITemplateScriptLibrary,
) error {
	if err := edidb.EDITemplateVersions.Create(ctx, db, version); err != nil {
		return versionErrors.Map(err, version.ID.String())
	}
	if err := createSegments(ctx, db, version, segments); err != nil {
		return err
	}
	if err := createLibraries(ctx, db, version, libraries); err != nil {
		return err
	}
	version.Segments = segments
	version.ScriptLibraries = libraries

	return nil
}

func createSegments(
	ctx context.Context,
	db *keel.DB,
	version *edi.EDITemplateVersion,
	segments []*edi.EDITemplateSegment,
) error {
	if len(segments) == 0 {
		return nil
	}
	for _, segment := range segments {
		segment.TemplateVersionID = version.ID
		segment.BusinessUnitID = version.BusinessUnitID
		segment.OrganizationID = version.OrganizationID
	}
	if err := edidb.EDITemplateSegments.CreateMany(ctx, db, segments); err != nil {
		return segmentErrors.Map(err, version.ID.String())
	}

	return nil
}

func createLibraries(
	ctx context.Context,
	db *keel.DB,
	version *edi.EDITemplateVersion,
	libraries []*edi.EDITemplateScriptLibrary,
) error {
	rows := make([]*edi.EDITemplateScriptLibrary, 0, len(libraries))
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
		rows = append(rows, library)
	}
	if len(rows) == 0 {
		return nil
	}
	if err := edidb.EDITemplateScriptLibraries.CreateMany(ctx, db, rows); err != nil {
		return libraryErrors.Map(err, version.ID.String())
	}

	return nil
}

func saveVersionMetadata(ctx context.Context, db *keel.DB, version *edi.EDITemplateVersion) error {
	tv := edidb.EDITemplateVersions
	if err := tv.Save(
		ctx,
		db,
		version,
		tv.X12Version,
		tv.FunctionalGroupID,
		tv.Status,
		tv.IsActive,
		tv.Notes,
		tv.CertificationNotes,
		tv.ActivationNotes,
		tv.ArchiveNotes,
		tv.DeprecatedNotes,
		tv.SupersededNotes,
		tv.CertifiedByID,
		tv.ActivatedByID,
		tv.ArchivedByID,
		tv.DeprecatedByID,
		tv.SupersededByID,
		tv.CertifiedAt,
		tv.ActivatedAt,
		tv.ArchivedAt,
		tv.DeprecatedAt,
		tv.SupersededAt,
	); err != nil {
		return versionErrors.Map(err, version.ID.String())
	}
	if version.Status == edi.TemplateStatusDraft {
		return nil
	}

	return setLibraryStatus(ctx, db, version, version.Status)
}

func lockVersion(
	ctx context.Context,
	db *keel.DB,
	templateID, versionID pulid.ID,
) (*edi.EDITemplateVersion, error) {
	tv := edidb.EDITemplateVersions
	seg := tv.Segments()
	version, err := tv.Where(tv.ID.Eq(versionID), tv.TemplateID.Eq(templateID)).
		Load(seg.Order(seg.Sequence.Asc())).
		Lock(keel.ForUpdate).
		One(ctx, db)
	if err != nil {
		return nil, versionErrors.Map(err, versionID.String())
	}

	return version, nil
}

func setLibraryStatus(
	ctx context.Context,
	db *keel.DB,
	version *edi.EDITemplateVersion,
	status edi.TemplateStatus,
) error {
	sl := edidb.EDITemplateScriptLibraries
	_, err := sl.Where(sl.TemplateVersionID.Eq(version.ID)).Update(ctx, db, sl.Status.To(status))

	return libraryErrors.Map(err, version.ID.String())
}

func withContents(q *keel.Query[edi.EDITemplateVersion]) *keel.Query[edi.EDITemplateVersion] {
	tv := edidb.EDITemplateVersions
	seg := tv.Segments()
	lib := tv.ScriptLibraries()

	return q.Load(seg.Order(seg.Sequence.Asc()), lib.Order(lib.Name.Lower().Asc()))
}

func withDocumentTypeAndVersions(
	q *keel.Query[edi.EDITemplate],
) *keel.Query[edi.EDITemplate] {
	t := edidb.EDITemplates
	versions := t.Versions()

	return q.Load(t.DocumentType(), versions.Order(versions.VersionNumber.Desc()))
}

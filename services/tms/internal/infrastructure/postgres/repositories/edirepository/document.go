package edirepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

func (r *repository) ListDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	entities := make([]*edi.EDIDocumentType, 0, 8)
	query := r.db.DBForContext(ctx).NewSelect().Model(&entities).Order("edt.code ASC")
	query = filterDocumentTypesQuery(query, req)
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) SelectDocumentTypeOptions(
	ctx context.Context,
	req *repositories.EDIDocumentTypeSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIDocumentType], error) {
	entities := make([]*edi.EDIDocumentType, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column("id", "code", "name", "standard", "transaction_set", "direction", "default_version", "status").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return filterDocumentTypesQuery(sq, repositories.ListEDIDocumentTypesRequest{
				Standard:       req.Standard,
				TransactionSet: req.TransactionSet,
				Direction:      req.Direction,
				Status:         req.Status,
			})
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return applyDocumentTypeSearch(sq, req.SelectQueryRequest.Query)
		})

	total, err := query.
		Order("edt.code ASC").
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIDocumentType]{Items: entities, Total: total}, nil
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	entities := make([]*edi.EDITemplate, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("DocumentType").
		Where("et.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("et.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("et.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("et.direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("et.status = ?", req.Status)
	}
	query = applyTemplateSearch(query, req.Filter.Query)
	total, err := query.
		Order("et.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDITemplate]{Items: entities, Total: total}, nil
}

func (r *repository) SelectTemplateOptions(
	ctx context.Context,
	req *repositories.EDITemplateSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	entities := make([]*edi.EDITemplate, 0, req.SelectQueryRequest.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			"id",
			"business_unit_id",
			"organization_id",
			"document_type_id",
			"name",
			"description",
			"direction",
			"standard",
			"transaction_set",
			"status",
		).
		Relation("DocumentType").
		Where("et.organization_id = ?", req.SelectQueryRequest.TenantInfo.OrgID).
		Where("et.business_unit_id = ?", req.SelectQueryRequest.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("et.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("et.direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("et.status = ?", req.Status)
	}
	query = applyTemplateSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order("et.name ASC").
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("DocumentType").
		Relation("Versions", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("version_number DESC")
		}).
		Where("et.id = ?", req.ID).
		Where("et.organization_id = ?", req.TenantInfo.OrgID).
		Where("et.business_unit_id = ?", req.TenantInfo.BuID).
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
		if _, err := r.db.DBForContext(c).NewInsert().Model(req.Template).Returning("*").Exec(c); err != nil {
			return err
		}
		req.Version.TemplateID = req.Template.ID
		if _, err := r.db.DBForContext(c).NewInsert().Model(req.Version).Returning("*").Exec(c); err != nil {
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
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.ScriptLibraries).Exec(c); err != nil {
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
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Column("name", "description", "status", "version", "updated_at").
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("etv.template_id = ?", req.TemplateID).
		Where("etv.organization_id = ?", req.TenantInfo.OrgID).
		Where("etv.business_unit_id = ?", req.TenantInfo.BuID).
		Order("etv.version_number DESC").
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Template").
		Relation("Segments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("sequence ASC")
		}).
		Relation("ScriptLibraries", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lower(name) ASC")
		}).
		Where("etv.id = ?", req.VersionID).
		Where("etv.template_id = ?", req.TemplateID).
		Where("etv.organization_id = ?", req.TenantInfo.OrgID).
		Where("etv.business_unit_id = ?", req.TenantInfo.BuID).
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
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Segments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("sequence ASC")
		}).
		Relation("ScriptLibraries", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lower(name) ASC")
		}).
		Where("etv.template_id = ?", req.TemplateID).
		Where("etv.organization_id = ?", req.TenantInfo.OrgID).
		Where("etv.business_unit_id = ?", req.TenantInfo.BuID)
	if !req.VersionID.IsNil() {
		query = query.Where("etv.id = ?", req.VersionID)
	} else {
		query = query.Where("etv.is_active = TRUE")
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
		if _, err := r.db.DBForContext(c).NewInsert().Model(version).Returning("*").Exec(c); err != nil {
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
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.ScriptLibraries).Exec(c); err != nil {
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
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(version).
		WherePK().
		Where("version = ?", ov).
		Column(
			"x12_version",
			"functional_group_id",
			"status",
			"is_active",
			"notes",
			"certification_notes",
			"activation_notes",
			"archive_notes",
			"deprecated_notes",
			"superseded_notes",
			"certified_by_id",
			"activated_by_id",
			"archived_by_id",
			"deprecated_by_id",
			"superseded_by_id",
			"certified_at",
			"activated_at",
			"archived_at",
			"deprecated_at",
			"superseded_at",
			"version",
			"updated_at",
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDITemplateVersion", version.ID.String()); err != nil {
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
			Where("template_version_id = ?", req.Version.ID).
			Where("organization_id = ?", req.Version.OrganizationID).
			Where("business_unit_id = ?", req.Version.BusinessUnitID).
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
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("etsl.template_version_id = ?", req.VersionID).
		Where("etsl.organization_id = ?", req.TenantInfo.OrgID).
		Where("etsl.business_unit_id = ?", req.TenantInfo.BuID).
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
		Order("lower(etsl.name) ASC").
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
			Where("template_version_id = ?", req.Version.ID).
			Where("organization_id = ?", req.Version.OrganizationID).
			Where("business_unit_id = ?", req.Version.BusinessUnitID).
			Exec(c); err != nil {
			return err
		}
		prepareTemplateScriptLibraries(req.Version, req.ScriptLibraries)
		if len(req.ScriptLibraries) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.ScriptLibraries).Exec(c); err != nil {
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

func (r *repository) ActivateTemplateVersion(
	ctx context.Context,
	req repositories.ActivateEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	version := new(edi.EDITemplateVersion)
	now := timeutils.NowUnix()
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(version).
			Relation("Segments", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order("sequence ASC")
			}).
			Where("etv.id = ?", req.VersionID).
			Where("etv.template_id = ?", req.TemplateID).
			Where("etv.organization_id = ?", req.TenantInfo.OrgID).
			Where("etv.business_unit_id = ?", req.TenantInfo.BuID).
			For("UPDATE").
			Scan(c); err != nil {
			return dberror.HandleNotFoundError(err, "EDITemplateVersion")
		}

		active := make([]*edi.EDITemplateVersion, 0, 1)
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(&active).
			Where("etv.template_id = ?", req.TemplateID).
			Where("etv.organization_id = ?", req.TenantInfo.OrgID).
			Where("etv.business_unit_id = ?", req.TenantInfo.BuID).
			Where("etv.is_active = TRUE").
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
					"status",
					"is_active",
					"superseded_by_id",
					"superseded_at",
					"superseded_notes",
					"version",
					"updated_at",
				).
				Exec(c); err != nil {
				return err
			}
			if err := r.updateTemplateScriptLibraryStatus(c, current, edi.TemplateStatusSuperseded); err != nil {
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
				"status",
				"is_active",
				"activated_by_id",
				"activated_at",
				"activation_notes",
				"version",
				"updated_at",
			).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		if err := r.updateTemplateScriptLibraryStatus(c, version, edi.TemplateStatusActive); err != nil {
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
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if err := r.db.DBForContext(c).
			NewSelect().
			Model(version).
			Relation("Segments", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Order("sequence ASC")
			}).
			Where("etv.id = ?", req.VersionID).
			Where("etv.template_id = ?", req.TemplateID).
			Where("etv.organization_id = ?", req.TenantInfo.OrgID).
			Where("etv.business_unit_id = ?", req.TenantInfo.BuID).
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
				"status",
				"is_active",
				"archived_by_id",
				"archived_at",
				"archive_notes",
				"version",
				"updated_at",
			).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		if err := r.updateTemplateScriptLibraryStatus(c, version, edi.TemplateStatusArchived); err != nil {
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
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.EDITemplateScriptLibrary)(nil)).
		Set("status = ?", status).
		Set("version = version + 1").
		Set("updated_at = ?", timeutils.NowUnix()).
		Where("template_version_id = ?", version.ID).
		Where("organization_id = ?", version.OrganizationID).
		Where("business_unit_id = ?", version.BusinessUnitID).
		Exec(ctx)
	return err
}

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

		documentTypes, err := r.ListDocumentTypes(c, repositories.ListEDIDocumentTypesRequest{
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
		if _, err = r.db.DBForContext(c).NewInsert().Model(template).Returning("*").Exec(c); err != nil {
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
		if _, err = r.db.DBForContext(c).NewInsert().Model(version).Returning("*").Exec(c); err != nil {
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

func (r *repository) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *repositories.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	entities := make([]*edi.EDIPartnerDocumentProfile, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.Partner).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation("PartnerSettingsSchema").
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(sq, req.Filter.TenantInfo)
		})

	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}

	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = applyPartnerDocumentProfileSearch(query, req.Filter.Query)

	total, err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectPartnerDocumentProfileOptions(
	ctx context.Context,
	req *repositories.EDIPartnerDocumentProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	entities := make(
		[]*edi.EDIPartnerDocumentProfile,
		0,
		req.SelectQueryRequest.Pagination.SafeLimit(),
	)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.Partner).
		Relation(rel.Template).
		Relation(rel.DocumentType).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(
				sq,
				req.SelectQueryRequest.TenantInfo,
			)
		})
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = applyPartnerDocumentProfileSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order(cols.Name.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartnerDocumentProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetPartnerDocumentProfileByID(
	ctx context.Context,
	req repositories.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entity := new(edi.EDIPartnerDocumentProfile)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.Partner).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation(rel.TemplateVersion).
		Relation("PartnerSettingsSchema").
		Where(cols.ID.Eq(), req.ID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileScopeTenant(sq, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}
	return entity, nil
}

func (r *repository) GetActivePartnerDocumentProfile(
	ctx context.Context,
	req repositories.GetActiveEDIPartnerDocumentProfileRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entities := make([]*edi.EDIPartnerDocumentProfile, 0, 2)
	cols := buncolgen.EDIPartnerDocumentProfileColumns
	rel := buncolgen.EDIPartnerDocumentProfileRelations

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.DocumentType).
		Relation(rel.Template).
		Relation("PartnerSettingsSchema").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.EDIPartnerID.Eq(), req.PartnerID).
				Where(cols.Status.Eq(), edi.DocumentStatusActive)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.EDIPartnerDocumentProfileApplyTenant(req.TenantInfo)(sq)
		})
	if req.TransactionSet != "" {
		query = query.Where(cols.TransactionSet.Eq(), req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where(cols.Direction.Eq(), req.Direction)
	}
	err := query.
		Order(cols.CreatedAt.OrderDesc()).
		Limit(2).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}
	if len(entities) == 0 {
		return nil, dberror.HandleNotFoundError(sql.ErrNoRows, "EDIPartnerDocumentProfile")
	}
	if (req.TransactionSet == "" || req.Direction == "") && len(entities) > 1 {
		return nil, errors.New(
			"multiple active EDI document profiles match partner; transaction set and direction are required",
		)
	}
	return entities[0], nil
}

func (r *repository) CreatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Column(
			"template_id",
			"template_version_id",
			"document_type_id",
			"name",
			"status",
			"direction",
			"standard",
			"transaction_set",
			"x12_version_override",
			"functional_group_id",
			"envelope",
			"acknowledgment",
			"validation_mode",
			"partner_settings",
			"partner_settings_schema_id",
			"partner_settings_schema_version",
			"version",
			"updated_at",
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIPartnerDocumentProfile", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func filterDocumentTypesQuery(
	query *bun.SelectQuery,
	req repositories.ListEDIDocumentTypesRequest,
) *bun.SelectQuery {
	if req.Standard != "" {
		query = query.Where("edt.standard = ?", req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where("edt.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("edt.direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("edt.status = ?", req.Status)
	}
	return query
}

func applyDocumentTypeSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("lower(edt.code) LIKE ?", term).
			WhereOr("lower(edt.name) LIKE ?", term).
			WhereOr("lower(edt.default_version) LIKE ?", term)
	})
}

func applyTemplateSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("lower(et.name) LIKE ?", term).
			WhereOr("lower(et.description) LIKE ?", term)
	})
}

func applyPartnerDocumentProfileSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("lower(epdp.name) LIKE ?", term).
			WhereOr("lower(ep.code) LIKE ?", term).
			WhereOr("lower(ep.name) LIKE ?", term).
			WhereOr("lower(et.name) LIKE ?", term)
	})
}

func (r *repository) AllocateControlNumbers(
	ctx context.Context,
	req repositories.AllocateEDIControlNumbersRequest,
) (map[edi.ControlNumberKind]int64, error) {
	allocated := make(map[edi.ControlNumberKind]int64, len(req.Kinds))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		for _, kind := range req.Kinds {
			sequence := &edi.EDIControlNumberSequence{
				BusinessUnitID: req.TenantInfo.BuID,
				OrganizationID: req.TenantInfo.OrgID,
				EDIPartnerID:   req.PartnerID,
				DocumentTypeID: req.DocumentTypeID,
				Kind:           kind,
			}
			_, err := r.db.DBForContext(c).
				NewInsert().
				Model(sequence).
				On(`CONFLICT ("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "kind") DO NOTHING`).
				Exec(c)
			if err != nil {
				return err
			}

			if err = r.db.DBForContext(c).
				NewSelect().
				Model(sequence).
				Where("ecns.edi_partner_id = ?", req.PartnerID).
				Where("ecns.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ecns.organization_id = ?", req.TenantInfo.OrgID).
				Where("ecns.document_type_id = ?", req.DocumentTypeID).
				Where("ecns.kind = ?", kind).
				For("UPDATE").
				Scan(c); err != nil {
				return err
			}

			value := sequence.NextValue
			next := value + 1
			if next > sequence.MaxValue {
				next = sequence.MinValue
			}
			sequence.NextValue = next
			sequence.Version++
			if _, err = r.db.DBForContext(c).
				NewUpdate().
				Model(sequence).
				WherePK().
				Column("next_value", "version", "updated_at").
				Exec(c); err != nil {
				return err
			}
			allocated[kind] = value
		}
		return nil
	})
	return allocated, err
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	entities := make([]*edi.EDIMessage, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ColumnExpr("emsg.*").
		ColumnExpr(`(
			SELECT COUNT(*)
			FROM edi_message_validation_errors AS emve
			WHERE emve.message_id = emsg.id
				AND emve.organization_id = emsg.organization_id
				AND emve.business_unit_id = emsg.business_unit_id
		) AS diagnostic_count`).
		Relation("Partner").
		Relation("PartnerDocumentProfile").
		Relation("Template").
		Where("emsg.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("emsg.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("emsg.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("emsg.direction = ?", req.Direction)
	}
	if !req.PartnerID.IsNil() {
		query = query.Where("emsg.edi_partner_id = ?", req.PartnerID)
	}
	if req.Status != "" {
		query = query.Where("emsg.status = ?", req.Status)
	}
	if req.GeneratedFrom > 0 {
		query = query.Where("emsg.generated_at >= ?", req.GeneratedFrom)
	}
	if req.GeneratedTo > 0 {
		query = query.Where("emsg.generated_at <= ?", req.GeneratedTo)
	}
	query = applyMessageArchiveSearch(query, req.Query)
	total, err := query.
		Order("emsg.generated_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIMessage]{Items: entities, Total: total}, nil
}

func (r *repository) GetMessageByID(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Partner").
		Relation("DocumentType").
		Relation("PartnerDocumentProfile").
		Relation("Template").
		Relation("TemplateVersion").
		Relation("TemplateVersion.ScriptLibraries", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.OrderExpr("lower(name) ASC")
		}).
		Relation("ValidationErrors", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("created_at ASC", "id ASC")
		}).
		Where("emsg.id = ?", req.ID).
		Where("emsg.organization_id = ?", req.TenantInfo.OrgID).
		Where("emsg.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) CreateMessageWithDiagnostics(
	ctx context.Context,
	req repositories.CreateEDIMessageWithDiagnosticsRequest,
) (*edi.EDIMessage, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).NewInsert().Model(req.Message).Returning("*").Exec(c); err != nil {
			return err
		}
		for _, diagnostic := range req.Diagnostics {
			diagnostic.MessageID = req.Message.ID
			diagnostic.BusinessUnitID = req.Message.BusinessUnitID
			diagnostic.OrganizationID = req.Message.OrganizationID
		}
		if len(req.Diagnostics) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.Diagnostics).Exec(c); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	req.Message.ValidationErrors = req.Diagnostics
	return req.Message, nil
}

func applyMessageArchiveSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("lower(emsg.id) LIKE ?", term).
			WhereOr("lower(emsg.shipment_id) LIKE ?", term).
			WhereOr("lower(emsg.transfer_id) LIKE ?", term).
			WhereOr("lower(emsg.interchange_control_number) LIKE ?", term).
			WhereOr("lower(emsg.group_control_number) LIKE ?", term).
			WhereOr("lower(emsg.transaction_control_number) LIKE ?", term)
	})
}

func (r *repository) ListTestCases(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	entities := make([]*edi.EDITestCase, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("etc.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("etc.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if !req.PartnerDocumentProfileID.IsNil() {
		query = query.Where("etc.partner_document_profile_id = ?", req.PartnerDocumentProfileID)
	}
	total, err := query.
		Order("etc.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDITestCase]{Items: entities, Total: total}, nil
}

func (r *repository) GetTestCaseByID(
	ctx context.Context,
	req repositories.GetEDITestCaseByIDRequest,
) (*edi.EDITestCase, error) {
	entity := new(edi.EDITestCase)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("etc.id = ?", req.ID).
		Where("etc.organization_id = ?", req.TenantInfo.OrgID).
		Where("etc.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITestCase")
	}
	return entity, nil
}

func (r *repository) CreateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

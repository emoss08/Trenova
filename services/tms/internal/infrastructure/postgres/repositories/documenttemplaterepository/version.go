package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type VersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type versionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVersionRepository(p VersionParams) repositories.DocumentTemplateVersionRepository {
	return &versionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-template-version-repository"),
	}
}

func (r *versionRepository) GetByID(
	ctx context.Context,
	req *repositories.GetDocumentTemplateVersionByIDRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.VersionID.String()),
	)

	entity := new(documenttemplate.DocumentTemplateVersion)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.DocumentTemplateVersionRelations.Template).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentTemplateVersionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DocumentTemplateVersionColumns.ID.Eq(), req.VersionID)
		}).
		Scan(ctx); err != nil {
		log.Error("failed to get document template version", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "DocumentTemplateVersion")
	}

	return entity, nil
}

func (r *versionRepository) List(
	ctx context.Context,
	req *repositories.ListDocumentTemplateVersionsRequest,
) ([]*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("templateId", req.TemplateID.String()),
	)

	cols := buncolgen.DocumentTemplateVersionColumns
	entities := make([]*documenttemplate.DocumentTemplateVersion, 0, 16)

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentTemplateVersionScopeTenant(sq, req.TenantInfo).
				Where(cols.TemplateID.Eq(), req.TemplateID)
		}).
		Order(cols.VersionNumber.OrderDesc()).
		Scan(ctx); err != nil {
		log.Error("failed to list document template versions", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

// nextVersionNumber reserves the next number in the template's sequence. It runs
// inside the caller's transaction so two concurrent creates collide on the unique
// index rather than both reading the same maximum and one silently overwriting.
func (r *versionRepository) nextVersionNumber(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
) (int64, error) {
	cols := buncolgen.DocumentTemplateVersionColumns

	var current int64
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*documenttemplate.DocumentTemplateVersion)(nil)).
		ColumnExpr(cols.VersionNumber.Expr("COALESCE(MAX({}), 0)")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DocumentTemplateVersionScopeTenant(sq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.TemplateID.Eq(), entity.TemplateID)
		}).
		Scan(ctx, &current)
	if err != nil {
		return 0, err
	}

	return current + 1, nil
}

// copyContentFrom seeds a new draft from an existing version. This is the whole
// mechanism behind both "edit a copy of what is live" and rollback: history is
// never mutated, it is copied forward.
func (r *versionRepository) copyContentFrom(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
	sourceID pulid.ID,
) error {
	source, err := r.GetByID(ctx, &repositories.GetDocumentTemplateVersionByIDRequest{
		VersionID: sourceID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return err
	}

	if source.TemplateID != entity.TemplateID {
		return errortypes.NewBusinessError(
			"The source version belongs to a different template",
		)
	}

	entity.SourceVersionID = &sourceID
	entity.Subject = source.Subject
	entity.BodyHTML = source.BodyHTML
	entity.BodyText = source.BodyText
	entity.CSSContent = source.CSSContent
	entity.HeaderHTML = source.HeaderHTML
	entity.FooterHTML = source.FooterHTML
	entity.PageSize = source.PageSize
	entity.Orientation = source.Orientation
	entity.MarginTop = source.MarginTop
	entity.MarginBottom = source.MarginBottom
	entity.MarginLeft = source.MarginLeft
	entity.MarginRight = source.MarginRight
	entity.ContentHash = source.ContentHash
	entity.StarterHash = source.StarterHash

	return nil
}

func (r *versionRepository) Create(
	ctx context.Context,
	req *repositories.CreateDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(zap.String("operation", "Create"))

	entity := req.Version

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if req.SourceVersionID != nil && !req.SourceVersionID.IsNil() {
			if cErr := r.copyContentFrom(c, entity, *req.SourceVersionID); cErr != nil {
				return cErr
			}
		}

		next, nErr := r.nextVersionNumber(c, entity)
		if nErr != nil {
			return nErr
		}
		entity.VersionNumber = next

		// A new version is always a draft. Reaching Active is what Publish is
		// for, and it has checks a plain insert would skip.
		entity.Status = documenttemplate.VersionStatusDraft
		entity.PublishedAt = nil
		entity.PublishedByID = nil
		entity.ArchivedAt = nil
		entity.ArchivedByID = nil

		_, iErr := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c)

		return iErr
	})
	if err != nil {
		log.Error("failed to create document template version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Update refuses anything but a draft in the WHERE clause as well as in the
// service, so a published version stays byte-identical to what it produced even
// if a new caller forgets the rule.
func (r *versionRepository) Update(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
) (*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	cols := buncolgen.DocumentTemplateVersionColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Where(cols.Status.Eq(), documenttemplate.VersionStatusDraft).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update document template version", zap.Error(err))
		return nil, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return nil, err
	}

	if affected == 0 {
		// Either someone else saved first or the version was published between
		// the editor loading it and saving. Both are the same instruction to the
		// user: reload before editing again.
		return nil, errortypes.NewBusinessError(
			"This version changed since you opened it. Reload it and reapply your edits. " +
				"A published version cannot be edited — create a new draft instead.",
		)
	}

	return entity, nil
}

// Publish makes one version live.
//
// Archive-then-activate-then-repoint runs as one transaction because each step
// alone leaves an observable wrong state: two active versions violate the partial
// unique index, and a template pointing at nothing renders the built-in to every
// customer until the next statement lands.
func (r *versionRepository) Publish(
	ctx context.Context,
	req *repositories.PublishDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "Publish"),
		zap.String("id", req.VersionID.String()),
	)

	vCols := buncolgen.DocumentTemplateVersionColumns
	tCols := buncolgen.DocumentTemplateColumns
	entity := new(documenttemplate.DocumentTemplateVersion)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		incoming, gErr := r.GetByID(c, &repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		})
		if gErr != nil {
			return gErr
		}

		if incoming.Status != documenttemplate.VersionStatusDraft {
			return errortypes.NewBusinessError(
				"Only a draft can be published. This version is " +
					incoming.Status.String() + ".",
			)
		}

		now := timeutils.NowUnix()

		// Archive first: the partial unique index allows exactly one Active row
		// per template and is checked per statement, so activating before
		// archiving would abort the transaction.
		if _, aErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*documenttemplate.DocumentTemplateVersion)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DocumentTemplateVersionScopeTenantUpdate(uq, req.TenantInfo).
					Where(vCols.TemplateID.Eq(), incoming.TemplateID).
					Where(vCols.Status.Eq(), documenttemplate.VersionStatusActive)
			}).
			Set(vCols.Status.Set(), documenttemplate.VersionStatusArchived).
			Set(vCols.ArchivedAt.Set(), now).
			Set(vCols.ArchivedByID.Set(), req.PublishedByID).
			Set(vCols.UpdatedAt.Set(), now).
			Set(vCols.Version.Inc(1)).
			Exec(c); aErr != nil {
			return aErr
		}

		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DocumentTemplateVersionScopeTenantUpdate(uq, req.TenantInfo).
					Where(vCols.ID.Eq(), req.VersionID).
					Where(vCols.Version.Eq(), incoming.Version)
			}).
			Set(vCols.Status.Set(), documenttemplate.VersionStatusActive).
			Set(vCols.PublishedAt.Set(), now).
			Set(vCols.PublishedByID.Set(), req.PublishedByID).
			Set(vCols.PublishNotes.Set(), req.PublishNotes).
			Set(vCols.UpdatedAt.Set(), now).
			Set(vCols.Version.Inc(1)).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		if uErr = dberror.CheckRowsAffected(
			results, "DocumentTemplateVersion", req.VersionID.String(),
		); uErr != nil {
			return uErr
		}

		// Repointing last means no window exists in which the template names a
		// version that is not yet Active.
		_, tErr := r.db.DBForContext(c).
			NewUpdate().
			Model((*documenttemplate.DocumentTemplate)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DocumentTemplateScopeTenantUpdate(uq, req.TenantInfo).
					Where(tCols.ID.Eq(), incoming.TemplateID)
			}).
			Set(tCols.ActiveVersionID.Set(), req.VersionID).
			Set(tCols.UpdatedAt.Set(), now).
			Set(tCols.Version.Inc(1)).
			Exec(c)

		return tErr
	})
	if err != nil {
		log.Error("failed to publish document template version", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"This template is being published by someone else. Retry the request.",
		)
	}

	return entity, nil
}

// Archive retires a version. Archiving the live one also clears the template's
// pointer, because a template naming a version that no longer renders is a
// dangling reference the resolution query would silently skip.
func (r *versionRepository) Archive(
	ctx context.Context,
	req *repositories.ArchiveDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	log := r.l.With(
		zap.String("operation", "Archive"),
		zap.String("id", req.VersionID.String()),
	)

	vCols := buncolgen.DocumentTemplateVersionColumns
	tCols := buncolgen.DocumentTemplateColumns
	entity := new(documenttemplate.DocumentTemplateVersion)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		existing, gErr := r.GetByID(c, &repositories.GetDocumentTemplateVersionByIDRequest{
			VersionID:  req.VersionID,
			TenantInfo: req.TenantInfo,
		})
		if gErr != nil {
			return gErr
		}

		if existing.Status == documenttemplate.VersionStatusArchived {
			return errortypes.NewBusinessError("This version is already archived")
		}

		now := timeutils.NowUnix()
		wasActive := existing.Status == documenttemplate.VersionStatusActive

		if wasActive {
			if _, tErr := r.db.DBForContext(c).
				NewUpdate().
				Model((*documenttemplate.DocumentTemplate)(nil)).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.DocumentTemplateScopeTenantUpdate(uq, req.TenantInfo).
						Where(tCols.ID.Eq(), existing.TemplateID).
						Where(tCols.ActiveVersionID.Eq(), req.VersionID)
				}).
				Set(tCols.ActiveVersionID.SetNull()).
				Set(tCols.UpdatedAt.Set(), now).
				Set(tCols.Version.Inc(1)).
				Exec(c); tErr != nil {
				return tErr
			}
		}

		results, uErr := r.db.DBForContext(c).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.DocumentTemplateVersionScopeTenantUpdate(uq, req.TenantInfo).
					Where(vCols.ID.Eq(), req.VersionID).
					Where(vCols.Version.Eq(), existing.Version)
			}).
			Set(vCols.Status.Set(), documenttemplate.VersionStatusArchived).
			Set(vCols.ArchivedAt.Set(), now).
			Set(vCols.ArchivedByID.Set(), req.ArchivedByID).
			Set(vCols.UpdatedAt.Set(), now).
			Set(vCols.Version.Inc(1)).
			Returning("*").
			Exec(c)
		if uErr != nil {
			return uErr
		}

		return dberror.CheckRowsAffected(
			results, "DocumentTemplateVersion", req.VersionID.String(),
		)
	})
	if err != nil {
		log.Error("failed to archive document template version", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// Delete only ever removes a draft. A published version is the record of what a
// customer received; generated_documents enforces that with RESTRICT, and
// refusing here turns a foreign key error into an explanation.
func (r *versionRepository) Delete(
	ctx context.Context,
	req *repositories.GetDocumentTemplateVersionByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.VersionID.String()),
	)

	cols := buncolgen.DocumentTemplateVersionColumns

	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*documenttemplate.DocumentTemplateVersion)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DocumentTemplateVersionScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.VersionID).
				Where(cols.Status.Eq(), documenttemplate.VersionStatusDraft)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete document template version", zap.Error(err))
		return err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return errortypes.NewBusinessError(
			"Only a draft can be deleted. A published version is the record of what " +
				"was sent — archive it instead.",
		)
	}

	return nil
}

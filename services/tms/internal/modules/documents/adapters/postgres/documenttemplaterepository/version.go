package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var versionErrors = dberror.Mapping{
	Entity: "DocumentTemplateVersion",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_document_template_versions_one_draft",
			Field:   "templateId",
			Message: "This template already has a draft. Edit it or delete it before starting another.",
		},
		{
			Name:    "uq_document_template_versions_one_active",
			Field:   "status",
			Message: "This template already has an active version",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_document_template_versions_template",
			Field:   "templateId",
			Message: "Document template does not exist",
		},
		{
			Name:    "fk_document_template_versions_source",
			Field:   "sourceVersionId",
			Message: "Source version does not exist",
		},
	},
}

type VersionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type versionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewVersionRepository(p VersionParams) documenttemplateports.DocumentTemplateVersionRepository {
	return &versionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.document-template-version-repository"),
	}
}

func (r *versionRepository) GetByID(
	ctx context.Context,
	req *documenttemplateports.GetDocumentTemplateVersionByIDRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateVersion, error) {
			return getVersion(ctx, r.db.Keel(ctx), req.VersionID)
		},
	)
}

func getVersion(
	ctx context.Context,
	db *keel.DB,
	id pulid.ID,
) (*documenttemplate.DocumentTemplateVersion, error) {
	v := documenttemplatedb.DocumentTemplateVersions
	entity, err := v.Where(v.ID.Eq(id)).Load(v.Template()).One(ctx, db)
	if err != nil {
		return nil, versionErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *versionRepository) List(
	ctx context.Context,
	req *documenttemplateports.ListDocumentTemplateVersionsRequest,
) ([]*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*documenttemplate.DocumentTemplateVersion, error) {
			v := documenttemplatedb.DocumentTemplateVersions
			entities, err := v.Where(v.TemplateID.Eq(req.TemplateID)).
				Order(v.VersionNumber.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list document template versions", zap.Error(err))
				return nil, err
			}
			if entities == nil {
				entities = []*documenttemplate.DocumentTemplateVersion{}
			}

			return entities, nil
		},
	)
}

// nextVersionNumber reserves the next number in the template's sequence. It runs
// inside the caller's transaction so two concurrent creates collide on the unique
// index rather than both reading the same maximum and one silently overwriting.
func nextVersionNumber(ctx context.Context, db *keel.DB, templateID pulid.ID) (int64, error) {
	v := documenttemplatedb.DocumentTemplateVersions
	current, err := keel.IntoOne[*int64](
		ctx,
		db,
		v.Pick(v.VersionNumber.Max()).Where(v.TemplateID.Eq(templateID)),
	)
	if err != nil {
		return 0, err
	}
	if current == nil {
		return 1, nil
	}

	return *current + 1, nil
}

// copyContentFrom seeds a new draft from an existing version. This is the whole
// mechanism behind both "edit a copy of what is live" and rollback: history is
// never mutated, it is copied forward.
func copyContentFrom(
	ctx context.Context,
	db *keel.DB,
	entity *documenttemplate.DocumentTemplateVersion,
	sourceID pulid.ID,
) error {
	source, err := getVersion(ctx, db, sourceID)
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
	req *documenttemplateports.CreateDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateVersion, error) {
			db := r.db.Keel(ctx)
			entity := req.Version

			if req.SourceVersionID != nil && !req.SourceVersionID.IsNil() {
				if err := copyContentFrom(ctx, db, entity, *req.SourceVersionID); err != nil {
					return nil, err
				}
			}

			next, err := nextVersionNumber(ctx, db, entity.TemplateID)
			if err != nil {
				return nil, err
			}
			entity.VersionNumber = next

			// A new version is always a draft. Reaching Active is what Publish is
			// for, and it has checks a plain insert would skip.
			entity.Status = documenttemplate.VersionStatusDraft
			entity.PublishedAt = nil
			entity.PublishedByID = nil
			entity.ArchivedAt = nil
			entity.ArchivedByID = nil

			if err = documenttemplatedb.DocumentTemplateVersions.Create(ctx, db, entity); err != nil {
				r.l.Error("failed to create document template version", zap.Error(err))
				return nil, versionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// Update refuses anything but a draft, so a published version stays
// byte-identical to what it produced even if a new caller forgets the rule.
// Publishing and archiving bump the version, so the optimistic lock refuses a
// save of a version that left Draft after the editor loaded it.
func (r *versionRepository) Update(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplateVersion,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateVersion, error) {
			v := documenttemplatedb.DocumentTemplateVersions
			err := staleVersionError()
			if entity.Status == documenttemplate.VersionStatusDraft {
				err = v.Save(ctx, r.db.Keel(ctx), entity)
			}
			if dberror.IsVersionConflict(err) {
				err = staleVersionError()
			}
			if err != nil {
				r.l.Error(
					"failed to update document template version",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, versionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// staleVersionError covers both a concurrent save and a version published between
// the editor loading it and saving. Both are the same instruction to the user:
// reload before editing again.
func staleVersionError() error {
	return errortypes.NewBusinessError(
		"This version changed since you opened it. Reload it and reapply your edits. " +
			"A published version cannot be edited — create a new draft instead.",
	)
}

// Publish makes one version live.
//
// Archive-then-activate-then-repoint runs as one transaction because each step
// alone leaves an observable wrong state: two active versions violate the partial
// unique index, and a template pointing at nothing renders the built-in to every
// customer until the next statement lands.
func (r *versionRepository) Publish(
	ctx context.Context,
	req *documenttemplateports.PublishDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateVersion, error) {
			entity, err := r.publish(ctx, req)
			if err != nil {
				r.l.Error(
					"failed to publish document template version",
					zap.String("id", req.VersionID.String()),
					zap.Error(err),
				)
				return nil, dberror.MapRetryableTransactionError(
					err,
					"This template is being published by someone else. Retry the request.",
				)
			}

			return entity, nil
		},
	)
}

func (r *versionRepository) publish(
	ctx context.Context,
	req *documenttemplateports.PublishDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	db := r.db.Keel(ctx)
	v := documenttemplatedb.DocumentTemplateVersions

	incoming, err := getVersion(ctx, db, req.VersionID)
	if err != nil {
		return nil, err
	}

	if incoming.Status != documenttemplate.VersionStatusDraft {
		return nil, errortypes.NewBusinessError(
			"Only a draft can be published. This version is {0}.",
			incoming.Status.String(),
		)
	}

	now := timeutils.NowUnix()

	// Archive first: the partial unique index allows exactly one Active row per
	// template and is checked per statement, so activating before archiving would
	// abort the transaction.
	if _, err = v.Where(
		v.TemplateID.Eq(incoming.TemplateID),
		v.Status.Eq(documenttemplate.VersionStatusActive),
	).Update(
		ctx,
		db,
		v.Status.To(documenttemplate.VersionStatusArchived),
		v.ArchivedAt.To(now),
		v.ArchivedByID.To(req.PublishedByID),
	); err != nil {
		return nil, versionErrors.Map(err, req.VersionID.String())
	}

	published, err := v.Where(v.ID.Eq(req.VersionID), v.Version.Eq(incoming.Version)).
		UpdateReturning(
			ctx,
			db,
			v.Status.To(documenttemplate.VersionStatusActive),
			v.PublishedAt.To(now),
			v.PublishedByID.To(req.PublishedByID),
			v.PublishNotes.To(req.PublishNotes),
		)
	if err != nil {
		return nil, versionErrors.Map(err, req.VersionID.String())
	}
	if err = dberror.CheckAffected(
		int64(len(published)),
		"DocumentTemplateVersion",
		req.VersionID.String(),
	); err != nil {
		return nil, err
	}

	// Repointing last means no window exists in which the template names a version
	// that is not yet Active.
	t := documenttemplatedb.DocumentTemplates
	if _, err = t.Where(t.ID.Eq(incoming.TemplateID)).
		Update(ctx, db, t.ActiveVersionID.To(req.VersionID)); err != nil {
		return nil, templateErrors.Map(err, incoming.TemplateID.String())
	}

	return published[0], nil
}

// Archive retires a version. Archiving the live one also clears the template's
// pointer, because a template naming a version that no longer renders is a
// dangling reference the resolution query would silently skip.
func (r *versionRepository) Archive(
	ctx context.Context,
	req *documenttemplateports.ArchiveDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplate.DocumentTemplateVersion, error) {
			entity, err := r.archive(ctx, req)
			if err != nil {
				r.l.Error(
					"failed to archive document template version",
					zap.String("id", req.VersionID.String()),
					zap.Error(err),
				)
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *versionRepository) archive(
	ctx context.Context,
	req *documenttemplateports.ArchiveDocumentTemplateVersionRequest,
) (*documenttemplate.DocumentTemplateVersion, error) {
	db := r.db.Keel(ctx)
	v := documenttemplatedb.DocumentTemplateVersions

	existing, err := getVersion(ctx, db, req.VersionID)
	if err != nil {
		return nil, err
	}

	if existing.Status == documenttemplate.VersionStatusArchived {
		return nil, errortypes.NewBusinessError("This version is already archived")
	}

	if existing.Status == documenttemplate.VersionStatusActive {
		t := documenttemplatedb.DocumentTemplates
		if _, err = t.Where(
			t.ID.Eq(existing.TemplateID),
			t.ActiveVersionID.Eq(req.VersionID),
		).Update(ctx, db, t.ActiveVersionID.ToNull()); err != nil {
			return nil, templateErrors.Map(err, existing.TemplateID.String())
		}
	}

	archived, err := v.Where(v.ID.Eq(req.VersionID), v.Version.Eq(existing.Version)).
		UpdateReturning(
			ctx,
			db,
			v.Status.To(documenttemplate.VersionStatusArchived),
			v.ArchivedAt.To(timeutils.NowUnix()),
			v.ArchivedByID.To(req.ArchivedByID),
		)
	if err != nil {
		return nil, versionErrors.Map(err, req.VersionID.String())
	}
	if err = dberror.CheckAffected(
		int64(len(archived)),
		"DocumentTemplateVersion",
		req.VersionID.String(),
	); err != nil {
		return nil, err
	}

	return archived[0], nil
}

// Delete only ever removes a draft. A published version is the record of what a
// customer received; generated_documents enforces that with RESTRICT, and
// refusing here turns a foreign key error into an explanation.
func (r *versionRepository) Delete(
	ctx context.Context,
	req *documenttemplateports.GetDocumentTemplateVersionByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := documenttemplatedb.DocumentTemplateVersions
		n, err := v.Where(
			v.ID.Eq(req.VersionID),
			v.Status.Eq(documenttemplate.VersionStatusDraft),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete document template version",
				zap.String("id", req.VersionID.String()),
				zap.Error(err),
			)
			return versionErrors.Map(err, req.VersionID.String())
		}

		if n == 0 {
			return errortypes.NewBusinessError(
				"Only a draft can be deleted. A published version is the record of what " +
					"was sent — archive it instead.",
			)
		}

		return nil
	})
}

package documenttemplaterepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate/documenttemplatedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// resolvedScan is the projection of the resolution query, filled in the order
// resolveColumns picks. It is a flat row because the statement returns one value
// the version entity does not own: whether an assignment matched. Every version
// column the renderer reads is projected explicitly, so a field added to the
// entity without being added here shows up as a zero value in a preview rather
// than silently in a customer's invoice; the repository test asserts the two stay
// in step.
type resolvedScan struct {
	VersionID       pulid.ID
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	TemplateID      pulid.ID
	SourceVersionID *pulid.ID
	VersionNumber   int64
	Status          documenttemplate.VersionStatus
	Subject         string
	BodyHTML        string
	BodyText        string
	CSSContent      string
	HeaderHTML      string
	FooterHTML      string
	PageSize        documenttemplate.PageSize
	Orientation     documenttemplate.Orientation
	MarginTop       *float64
	MarginBottom    *float64
	MarginLeft      *float64
	MarginRight     *float64
	ContentHash     string
	StarterHash     string
	PublishNotes    string
	PublishedByID   *pulid.ID
	PublishedAt     *int64
	VersionLock     int64
	CreatedAt       int64
	UpdatedAt       int64

	TemplateCode         string
	TemplateName         string
	TemplateDescription  string
	TemplateKind         documenttemplate.Kind
	TemplateIsOrgDefault bool
	TemplateVersionLock  int64
	TemplateCreatedAt    int64
	TemplateUpdatedAt    int64

	AssignmentID *pulid.ID
}

func text(e keel.Expr) keel.Expr {
	return keel.Raw("COALESCE(?::text, '')", e)
}

func resolveColumns(
	v *documenttemplatedb.DocumentTemplateVersionsTable,
	t *documenttemplatedb.DocumentTemplatesTable,
	assignment keel.Expr,
) []keel.Expr {
	return []keel.Expr{
		v.ID,
		v.OrganizationID,
		v.BusinessUnitID,
		v.TemplateID,
		v.SourceVersionID,
		v.VersionNumber,
		v.Status,
		text(v.Subject),
		text(v.BodyHTML),
		text(v.BodyText),
		text(v.CSSContent),
		text(v.HeaderHTML),
		text(v.FooterHTML),
		text(v.PageSize),
		text(v.Orientation),
		v.MarginTop,
		v.MarginBottom,
		v.MarginLeft,
		v.MarginRight,
		v.ContentHash,
		text(v.StarterHash),
		text(v.PublishNotes),
		v.PublishedByID,
		v.PublishedAt,
		v.Version,
		v.CreatedAt,
		v.UpdatedAt,
		t.Code,
		t.Name,
		text(t.Description),
		t.Kind,
		t.IsOrgDefault,
		t.Version,
		t.CreatedAt,
		t.UpdatedAt,
		assignment,
	}
}

func (s *resolvedScan) toRow() *documenttemplateports.ResolvedTemplateRow {
	version := &documenttemplate.DocumentTemplateVersion{
		ID:              s.VersionID,
		OrganizationID:  s.OrganizationID,
		BusinessUnitID:  s.BusinessUnitID,
		TemplateID:      s.TemplateID,
		SourceVersionID: s.SourceVersionID,
		VersionNumber:   s.VersionNumber,
		Status:          s.Status,
		Subject:         s.Subject,
		BodyHTML:        s.BodyHTML,
		BodyText:        s.BodyText,
		CSSContent:      s.CSSContent,
		HeaderHTML:      s.HeaderHTML,
		FooterHTML:      s.FooterHTML,
		PageSize:        s.PageSize,
		Orientation:     s.Orientation,
		MarginTop:       s.MarginTop,
		MarginBottom:    s.MarginBottom,
		MarginLeft:      s.MarginLeft,
		MarginRight:     s.MarginRight,
		ContentHash:     s.ContentHash,
		StarterHash:     s.StarterHash,
		PublishNotes:    s.PublishNotes,
		PublishedByID:   s.PublishedByID,
		PublishedAt:     s.PublishedAt,
		Version:         s.VersionLock,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}

	versionID := s.VersionID
	template := &documenttemplate.DocumentTemplate{
		ID:              s.TemplateID,
		OrganizationID:  s.OrganizationID,
		BusinessUnitID:  s.BusinessUnitID,
		Kind:            s.TemplateKind,
		Code:            s.TemplateCode,
		Name:            s.TemplateName,
		Description:     s.TemplateDescription,
		IsOrgDefault:    s.TemplateIsOrgDefault,
		ActiveVersionID: &versionID,
		Version:         s.TemplateVersionLock,
		CreatedAt:       s.TemplateCreatedAt,
		UpdatedAt:       s.TemplateUpdatedAt,
		ActiveVersion:   version,
	}

	version.Template = template

	source := documenttemplate.SourceOrgDefault
	if s.AssignmentID != nil && !s.AssignmentID.IsNil() {
		source = documenttemplate.SourceAssigned
	}

	return &documenttemplateports.ResolvedTemplateRow{
		Template: template,
		Version:  version,
		Source:   source,
	}
}

// Resolve walks the chain assigned → organization default in one statement and
// returns nil when neither tier matched, which is the caller's signal to render
// the built-in starter. The version table is the base and the template and the
// assignment are reached through explicit joins, because the three tiers have to
// be weighed against each other inside one statement: resolving in steps would
// mean a render could observe a template mid-publish.
func (r *templateRepository) Resolve(
	ctx context.Context,
	req *documenttemplateports.ResolveDocumentTemplateRequest,
) (*documenttemplateports.ResolvedTemplateRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*documenttemplateports.ResolvedTemplateRow, error) {
			rows, err := keel.Into[resolvedScan](ctx, r.db.Keel(ctx), resolveQuery(req))
			if err != nil {
				r.l.Error(
					"failed to resolve document template",
					zap.String("kind", string(req.Kind)),
					zap.Error(err),
				)
				return nil, err
			}
			if len(rows) == 0 {
				return nil, nil //nolint:nilnil // no row means "use the built-in"
			}

			return rows[0].toRow(), nil
		},
	)
}

func resolveQuery(
	req *documenttemplateports.ResolveDocumentTemplateRequest,
) *keel.Query[documenttemplate.DocumentTemplateVersion] {
	v := documenttemplatedb.DocumentTemplateVersions
	t := documenttemplatedb.DocumentTemplates
	a := documenttemplatedb.DocumentTemplateAssignments

	// Only the template's own live version resolves. A draft or an archived
	// version matching on template_id alone would render something nobody
	// published.
	q := v.Join(t, t.ID.EqCol(v.TemplateID), t.ActiveVersionID.EqCol(v.ID)).
		Where(
			v.Status.Eq(documenttemplate.VersionStatusActive),
			t.Kind.Eq(req.Kind),
		)

	if req.CustomerID == nil || req.CustomerID.IsNil() {
		// No customer in scope: the assignment tier cannot apply, so the join is
		// left out rather than made harmless. NULL keeps the projection stable.
		return q.Where(t.IsOrgDefault.IsTrue()).
			Pick(resolveColumns(v, t, keel.Raw("NULL::varchar"))...).
			Limit(1)
	}

	// An assignment beats the organization default. Ordering rather than two
	// queries keeps the tie-break in the same statement that found both.
	return q.LeftJoin(
		a,
		a.TemplateID.EqCol(t.ID),
		a.Kind.EqCol(t.Kind),
		a.CustomerID.Eq(*req.CustomerID),
	).
		Where(keel.Or(a.ID.IsNotNull(), t.IsOrgDefault.IsTrue())).
		Pick(resolveColumns(v, t, a.ID)...).
		Order(keel.Desc(keel.Raw("(? IS NOT NULL)", a.ID))).
		Limit(1)
}

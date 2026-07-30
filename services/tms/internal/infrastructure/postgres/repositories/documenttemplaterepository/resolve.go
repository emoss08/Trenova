package documenttemplaterepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// The resolution query reads the version table as its base and reaches the
// template and the assignment through explicit joins, because the three tiers
// have to be weighed against each other inside one statement. Resolving in steps
// would mean a render could observe a template mid-publish.
var (
	resolveTemplateJoin = "JOIN " + buncolgen.DocumentTemplateTable.As(
		buncolgen.DocumentTemplateTable.Alias,
	) +
		" ON " + buncolgen.DocumentTemplateColumns.ID.EqColumn(
		buncolgen.DocumentTemplateVersionColumns.TemplateID,
	) +
		" AND " + buncolgen.DocumentTemplateColumns.OrganizationID.EqColumn(
		buncolgen.DocumentTemplateVersionColumns.OrganizationID,
	) +
		" AND " + buncolgen.DocumentTemplateColumns.BusinessUnitID.EqColumn(
		buncolgen.DocumentTemplateVersionColumns.BusinessUnitID,
	) +
		// Only the template's own live version resolves. A draft or an archived
		// version matching on template_id alone would render something nobody
		// published.
		" AND " + buncolgen.DocumentTemplateColumns.ActiveVersionID.EqColumn(
		buncolgen.DocumentTemplateVersionColumns.ID,
	)

	resolveAssignmentJoin = "LEFT JOIN " + buncolgen.DocumentTemplateAssignmentTable.As(
		buncolgen.DocumentTemplateAssignmentTable.Alias,
	) +
		" ON " + buncolgen.DocumentTemplateAssignmentColumns.TemplateID.EqColumn(
		buncolgen.DocumentTemplateColumns.ID,
	) +
		" AND " + buncolgen.DocumentTemplateAssignmentColumns.Kind.EqColumn(
		buncolgen.DocumentTemplateColumns.Kind,
	) +
		" AND " + buncolgen.DocumentTemplateAssignmentColumns.OrganizationID.EqColumn(
		buncolgen.DocumentTemplateColumns.OrganizationID,
	) +
		" AND " + buncolgen.DocumentTemplateAssignmentColumns.BusinessUnitID.EqColumn(
		buncolgen.DocumentTemplateColumns.BusinessUnitID,
	) +
		" AND " + buncolgen.DocumentTemplateAssignmentColumns.CustomerID.Eq()

	// An assignment beats the organization default. Ordering rather than two
	// queries keeps the tie-break in the same statement that found both.
	resolveOrder = buncolgen.DocumentTemplateAssignmentColumns.ID.Expr("({} IS NOT NULL) DESC")
)

// resolvedScan is the projection of the resolution query.
//
// It is a flat row rather than a model scan because the statement returns one
// column the version entity does not own — whether an assignment matched — and a
// model scan rejects columns with no field. Every version column the renderer
// reads is projected explicitly, so a field added to the entity without being
// added here shows up as a zero value in a preview rather than silently in a
// customer's invoice; the repository test asserts the two stay in step.
type resolvedScan struct {
	VersionID       pulid.ID                       `bun:"version_id"`
	OrganizationID  pulid.ID                       `bun:"organization_id"`
	BusinessUnitID  pulid.ID                       `bun:"business_unit_id"`
	TemplateID      pulid.ID                       `bun:"template_id"`
	SourceVersionID *pulid.ID                      `bun:"source_version_id"`
	VersionNumber   int64                          `bun:"version_number"`
	Status          documenttemplate.VersionStatus `bun:"status"`
	Subject         string                         `bun:"subject"`
	BodyHTML        string                         `bun:"body_html"`
	BodyText        string                         `bun:"body_text"`
	CSSContent      string                         `bun:"css_content"`
	HeaderHTML      string                         `bun:"header_html"`
	FooterHTML      string                         `bun:"footer_html"`
	PageSize        documenttemplate.PageSize      `bun:"page_size"`
	Orientation     documenttemplate.Orientation   `bun:"orientation"`
	MarginTop       *float64                       `bun:"margin_top"`
	MarginBottom    *float64                       `bun:"margin_bottom"`
	MarginLeft      *float64                       `bun:"margin_left"`
	MarginRight     *float64                       `bun:"margin_right"`
	ContentHash     string                         `bun:"content_hash"`
	StarterHash     string                         `bun:"starter_hash"`
	PublishNotes    string                         `bun:"publish_notes"`
	PublishedByID   *pulid.ID                      `bun:"published_by_id"`
	PublishedAt     *int64                         `bun:"published_at"`
	VersionLock     int64                          `bun:"version_lock"`
	CreatedAt       int64                          `bun:"created_at"`
	UpdatedAt       int64                          `bun:"updated_at"`

	TemplateCode         string                `bun:"template_code"`
	TemplateName         string                `bun:"template_name"`
	TemplateDescription  string                `bun:"template_description"`
	TemplateKind         documenttemplate.Kind `bun:"template_kind"`
	TemplateIsOrgDefault bool                  `bun:"template_is_org_default"`
	TemplateVersionLock  int64                 `bun:"template_version_lock"`
	TemplateCreatedAt    int64                 `bun:"template_created_at"`
	TemplateUpdatedAt    int64                 `bun:"template_updated_at"`

	AssignmentID *pulid.ID `bun:"assignment_id"`
}

func (s *resolvedScan) toRow() *repositories.ResolvedTemplateRow {
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

	return &repositories.ResolvedTemplateRow{
		Template: template,
		Version:  version,
		Source:   source,
	}
}

// Resolve walks the chain assigned → organization default in one statement and
// returns nil when neither tier matched, which is the caller's signal to render
// the built-in starter.
func (r *templateRepository) Resolve(
	ctx context.Context,
	req *repositories.ResolveDocumentTemplateRequest,
) (*repositories.ResolvedTemplateRow, error) {
	log := r.l.With(
		zap.String("operation", "Resolve"),
		zap.String("kind", string(req.Kind)),
	)

	vCols := buncolgen.DocumentTemplateVersionColumns
	tCols := buncolgen.DocumentTemplateColumns
	aCols := buncolgen.DocumentTemplateAssignmentColumns

	scoped := req.CustomerID != nil && !req.CustomerID.IsNil()

	row := new(resolvedScan)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*documenttemplate.DocumentTemplateVersion)(nil)).
		ColumnExpr(vCols.ID.As("version_id")).
		ColumnExpr(vCols.OrganizationID.As("organization_id")).
		ColumnExpr(vCols.BusinessUnitID.As("business_unit_id")).
		ColumnExpr(vCols.TemplateID.As("template_id")).
		ColumnExpr(vCols.SourceVersionID.As("source_version_id")).
		ColumnExpr(vCols.VersionNumber.As("version_number")).
		ColumnExpr(vCols.Status.As("status")).
		ColumnExpr(vCols.Subject.Expr("COALESCE({}, '') AS subject")).
		ColumnExpr(vCols.BodyHTML.Expr("COALESCE({}, '') AS body_html")).
		ColumnExpr(vCols.BodyText.Expr("COALESCE({}, '') AS body_text")).
		ColumnExpr(vCols.CSSContent.Expr("COALESCE({}, '') AS css_content")).
		ColumnExpr(vCols.HeaderHTML.Expr("COALESCE({}, '') AS header_html")).
		ColumnExpr(vCols.FooterHTML.Expr("COALESCE({}, '') AS footer_html")).
		ColumnExpr(vCols.PageSize.As("page_size")).
		ColumnExpr(vCols.Orientation.As("orientation")).
		ColumnExpr(vCols.MarginTop.As("margin_top")).
		ColumnExpr(vCols.MarginBottom.As("margin_bottom")).
		ColumnExpr(vCols.MarginLeft.As("margin_left")).
		ColumnExpr(vCols.MarginRight.As("margin_right")).
		ColumnExpr(vCols.ContentHash.As("content_hash")).
		ColumnExpr(vCols.StarterHash.Expr("COALESCE({}, '') AS starter_hash")).
		ColumnExpr(vCols.PublishNotes.Expr("COALESCE({}, '') AS publish_notes")).
		ColumnExpr(vCols.PublishedByID.As("published_by_id")).
		ColumnExpr(vCols.PublishedAt.As("published_at")).
		ColumnExpr(vCols.Version.As("version_lock")).
		ColumnExpr(vCols.CreatedAt.As("created_at")).
		ColumnExpr(vCols.UpdatedAt.As("updated_at")).
		ColumnExpr(tCols.Code.As("template_code")).
		ColumnExpr(tCols.Name.As("template_name")).
		ColumnExpr(tCols.Description.Expr("COALESCE({}, '') AS template_description")).
		ColumnExpr(tCols.Kind.As("template_kind")).
		ColumnExpr(tCols.IsOrgDefault.As("template_is_org_default")).
		ColumnExpr(tCols.Version.As("template_version_lock")).
		ColumnExpr(tCols.CreatedAt.As("template_created_at")).
		ColumnExpr(tCols.UpdatedAt.As("template_updated_at")).
		Join(resolveTemplateJoin)

	if scoped {
		q = q.
			ColumnExpr(aCols.ID.As("assignment_id")).
			Join(resolveAssignmentJoin, *req.CustomerID)
	} else {
		// No customer in scope: the assignment tier cannot apply, so the join is
		// left out rather than made harmless. NULL keeps the projection stable.
		q = q.ColumnExpr("NULL AS assignment_id")
	}

	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		sq = buncolgen.DocumentTemplateVersionScopeTenant(sq, req.TenantInfo).
			Where(vCols.Status.Eq(), documenttemplate.VersionStatusActive).
			Where(tCols.Kind.Eq(), req.Kind)

		if scoped {
			return sq.WhereGroup(" AND ", func(iq *bun.SelectQuery) *bun.SelectQuery {
				return iq.
					WhereOr(aCols.ID.IsNotNull()).
					WhereOr(tCols.IsOrgDefault.IsTrue())
			})
		}

		return sq.Where(tCols.IsOrgDefault.IsTrue())
	})

	if scoped {
		q = q.OrderExpr(resolveOrder)
	}

	if err := q.Limit(1).Scan(ctx, row); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Nothing customized: the built-in is the third tier, so no row is a
			// result, not a failure. The service renders the embedded starter.
			return nil, nil //nolint:nilnil // no row means "use the built-in"
		}
		log.Error("failed to resolve document template", zap.Error(err))
		return nil, err
	}

	return row.toRow(), nil
}

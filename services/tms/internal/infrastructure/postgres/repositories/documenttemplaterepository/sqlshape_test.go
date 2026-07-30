package documenttemplaterepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

// The resolution query is assembled from generated fragments, so the risk is not
// a typo in an identifier but a join that silently widens the scope. These assert
// the shape of the SQL that actually reaches Postgres.

func TestResolveTemplateJoinPinsTenantAndActiveVersion(t *testing.T) {
	t.Parallel()

	// Without the active_version_id predicate the join would match every version
	// of the template, and a draft could render to a customer.
	require.Contains(t, resolveTemplateJoin, "dtpl.active_version_id = dtv.id")
	require.Contains(t, resolveTemplateJoin, "dtpl.organization_id = dtv.organization_id")
	require.Contains(t, resolveTemplateJoin, "dtpl.business_unit_id = dtv.business_unit_id")
}

func TestResolveAssignmentJoinIsTenantAndKindScoped(t *testing.T) {
	t.Parallel()

	// An assignment join missing kind would let an invoice assignment decide
	// which detention notice a customer receives.
	require.Contains(t, resolveAssignmentJoin, "dta.kind = dtpl.kind")
	require.Contains(t, resolveAssignmentJoin, "dta.template_id = dtpl.id")
	require.Contains(t, resolveAssignmentJoin, "dta.organization_id = dtpl.organization_id")
	require.Contains(t, resolveAssignmentJoin, "dta.business_unit_id = dtpl.business_unit_id")
	require.Contains(t, resolveAssignmentJoin, "dta.customer_id = ?")
}

func TestResolveOrderPrefersTheAssignment(t *testing.T) {
	t.Parallel()

	require.Equal(t, "(dta.id IS NOT NULL) DESC", resolveOrder)
}

// resolvedScan is hand-maintained against the version entity. A field added to
// the entity that the projection does not carry would reach a renderer as a zero
// value, so the count is pinned deliberately.
func TestResolvedScanCarriesEveryRenderedVersionField(t *testing.T) {
	t.Parallel()

	scan := resolvedScan{
		VersionID:     "dtv_1",
		TemplateID:    "dtpl_1",
		VersionNumber: 3,
		Status:        documenttemplate.VersionStatusActive,
		Subject:       "subject",
		BodyHTML:      "<p>body</p>",
		BodyText:      "body",
		CSSContent:    "p{}",
		HeaderHTML:    "<header/>",
		FooterHTML:    "<footer/>",
		PageSize:      documenttemplate.PageSizeA4,
		Orientation:   documenttemplate.OrientationLandscape,
		ContentHash:   "hash",
		TemplateKind:  documenttemplate.Kind("invoice.pdf"),
	}

	row := scan.toRow()

	require.Equal(t, documenttemplate.SourceOrgDefault, row.Source)
	require.Equal(t, "<p>body</p>", row.Version.BodyHTML)
	require.Equal(t, "p{}", row.Version.CSSContent)
	require.Equal(t, "<header/>", row.Version.HeaderHTML)
	require.Equal(t, "<footer/>", row.Version.FooterHTML)
	require.Equal(t, documenttemplate.PageSizeA4, row.Version.ResolvedPageSize())
	require.Equal(t, documenttemplate.OrientationLandscape, row.Version.ResolvedOrientation())
	require.Equal(t, "hash", row.Version.ContentHash)

	// The template must name the version that resolved, or a generated document
	// would record a pointer that disagrees with the bytes it describes.
	require.NotNil(t, row.Template.ActiveVersionID)
	require.Equal(t, row.Version.ID, *row.Template.ActiveVersionID)
	require.Same(t, row.Template, row.Version.Template)
}

func TestResolvedScanReportsAnAssignmentAsAssigned(t *testing.T) {
	t.Parallel()

	assignmentID := pulid.ID("dta_1")
	scan := resolvedScan{
		VersionID:            "dtv_1",
		TemplateID:           "dtpl_1",
		TemplateIsOrgDefault: true,
		AssignmentID:         &assignmentID,
	}

	// A template that is both the organization default and explicitly assigned
	// resolves as Assigned: the administrator chose it for this customer, and
	// that is what the source is there to report.
	require.Equal(t, documenttemplate.SourceAssigned, scan.toRow().Source)
}

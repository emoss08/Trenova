package documenttemplaterepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const resolveTemplateJoin = `JOIN "document_templates" dt ON dt."business_unit_id" = $1 ` +
	`AND dt."organization_id" = $2 AND dt."id" = dtv."template_id" AND dt."active_version_id" = dtv."id"`

func TestResolveWithoutACustomerReadsOnlyTheLiveOrgDefault(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, resolveQuery(&documenttemplateports.ResolveDocumentTemplateRequest{
		Kind: documenttemplate.KindInvoicePDF,
	}))

	assert.Contains(t, sqlText, resolveTemplateJoin, sqlText)
	assert.Contains(
		t,
		sqlText,
		`WHERE dtv."business_unit_id" = $1 AND dtv."organization_id" = $2 AND dtv."status" = 'Active' `+
			`AND dt."kind" = 'invoice.pdf' AND dt."is_org_default" IS TRUE LIMIT 1`,
		sqlText,
	)
	assert.Contains(t, sqlText, `coalesce(CAST(dtv."subject" AS text), $3)`, sqlText)
	assert.Contains(t, sqlText, `coalesce(CAST(dt."description" AS text), $13)`, sqlText)
	assert.Contains(t, sqlText, `CAST(NULL AS text) FROM`, sqlText)
	assert.NotContains(t, sqlText, "document_template_assignments", sqlText)
	require.Len(t, args, 13)
	assert.Equal(t, []any{"bu_1", "org_1"}, args[:2])
	for _, blank := range args[2:] {
		assert.Equal(t, "", blank)
	}
}

func TestResolveForACustomerPrefersItsAssignmentOfTheSameKind(t *testing.T) {
	t.Parallel()

	customerID := pulid.ID("cus_1")
	sqlText, args := keeltest.Render(t, resolveQuery(&documenttemplateports.ResolveDocumentTemplateRequest{
		Kind:       documenttemplate.KindInvoicePDF,
		CustomerID: &customerID,
	}))

	assert.Contains(t, sqlText, resolveTemplateJoin, sqlText)
	assert.Contains(
		t,
		sqlText,
		`LEFT JOIN "document_template_assignments" dta ON dta."business_unit_id" = $1 `+
			`AND dta."organization_id" = $2 AND dta."template_id" = dt."id" `+
			`AND dta."kind" = dt."kind" AND dta."customer_id" = $14`,
		sqlText,
	)
	assert.Contains(
		t,
		sqlText,
		`AND (dta."id" IS NOT NULL OR dt."is_org_default" IS TRUE) `+
			`ORDER BY dta."id" IS NOT NULL DESC LIMIT 1`,
		sqlText,
	)
	require.Len(t, args, 14)
	assert.Equal(t, "cus_1", args[13])
}

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
	require.Equal(t, documenttemplate.SourceAssigned, scan.toRow().Source)
}

package documenttemplateservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// generated_documents exists to answer "what exactly did we send that customer
// in March" after a template has been edited. These pin the parts of the record
// that question depends on.
func TestRecordingAGeneratedDocumentNamesTheVersionThatProducedIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	templateID := pulid.MustNew("dtpl_")
	versionID := pulid.MustNew("dtv_")
	referenceID := pulid.MustNew("inv_")

	h.generated.EXPECT().
		Create(mock.Anything, mock.MatchedBy(
			func(entity *documenttemplate.GeneratedDocument) bool {
				return entity.TemplateVersionID != nil &&
					*entity.TemplateVersionID == versionID &&
					entity.ContentHash == "hash-abc" &&
					entity.Status == documenttemplate.GenerationStatusCompleted &&
					entity.GeneratedAt != nil
			},
		)).
		Return(&documenttemplate.GeneratedDocument{ID: pulid.MustNew("gd_")}, nil).
		Once()

	_, err := h.service.RecordGeneratedDocument(
		t.Context(),
		&services.RecordGeneratedDocumentRequest{
			TenantInfo: h.tenantInfo,
			Kind:       documenttemplate.KindInvoicePDF,
			Rendered: &services.RenderedDocument{
				Source:      documenttemplate.SourceOrgDefault,
				TemplateID:  &templateID,
				VersionID:   &versionID,
				ContentHash: "hash-abc",
			},
			ReferenceType: "invoice",
			ReferenceID:   referenceID,
			FileName:      "invoice-INV-1001.pdf",
			FilePath:      "invoice/x/invoice-INV-1001.pdf",
			FileSize:      2048,
		},
	)
	require.NoError(t, err)
}

// A built-in render has no row to point at, so the version link is nil — but the
// content hash still identifies the exact bytes, which is what keeps the record
// useful.
func TestRecordingABuiltInRenderKeepsTheContentHash(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	h.generated.EXPECT().
		Create(mock.Anything, mock.MatchedBy(
			func(entity *documenttemplate.GeneratedDocument) bool {
				return entity.TemplateVersionID == nil &&
					entity.FromBuiltIn() &&
					entity.ContentHash == "starter-hash"
			},
		)).
		Return(&documenttemplate.GeneratedDocument{ID: pulid.MustNew("gd_")}, nil).
		Once()

	_, err := h.service.RecordGeneratedDocument(
		t.Context(),
		&services.RecordGeneratedDocumentRequest{
			TenantInfo: h.tenantInfo,
			Kind:       documenttemplate.KindInvoicePDF,
			Rendered: &services.RenderedDocument{
				Source:      documenttemplate.SourceBuiltIn,
				ContentHash: "starter-hash",
			},
			ReferenceType: "invoice",
			ReferenceID:   pulid.MustNew("inv_"),
			FileName:      "invoice.pdf",
			FilePath:      "invoice/x/invoice.pdf",
			FileSize:      1024,
		},
	)
	require.NoError(t, err)
}

// A completed row with no stored file would promise bytes nobody can fetch. The
// database refuses it; the service refuses it first so the caller gets a field
// error rather than a constraint violation.
func TestRecordingRefusesADocumentWithNowhereToFetchIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.service.RecordGeneratedDocument(
		t.Context(),
		&services.RecordGeneratedDocumentRequest{
			TenantInfo: h.tenantInfo,
			Kind:       documenttemplate.KindInvoicePDF,
			Rendered: &services.RenderedDocument{
				Source: documenttemplate.SourceBuiltIn,
			},
			ReferenceType: "invoice",
			ReferenceID:   pulid.MustNew("inv_"),
			FileName:      "invoice.pdf",
			FileSize:      1024,
		},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "where it was stored")
}

func TestRecordingRefusesARenderItCannotAttribute(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.service.RecordGeneratedDocument(
		t.Context(),
		&services.RecordGeneratedDocumentRequest{
			TenantInfo:    h.tenantInfo,
			Kind:          documenttemplate.KindInvoicePDF,
			ReferenceType: "invoice",
			ReferenceID:   pulid.MustNew("inv_"),
			FileName:      "invoice.pdf",
		},
	)
	require.Error(t, err)
}

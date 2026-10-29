package detentionnoticeservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/detentiontemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/generateddocument"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var _ detentionports.DetentionNoticeRepository = (*stubNoticeRepo)(nil)

// The PDF bytes must not survive into the provenance that travels through a
// Temporal payload. A notice PDF is small, but nothing bounds it, and a workflow
// argument over the payload limit fails the filing with an error that says
// nothing about why.
func TestRenderNoticePDFKeepsTheBytesOutOfTheProvenance(t *testing.T) {
	t.Parallel()

	occurrence := pdfOccurrence()
	versionID := pulid.MustNew("dtv_")
	resolver := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderDocument(mock.Anything, mock.Anything).
		Return(&documenttemplateports.RenderedDocument{
			PDF:         []byte("%PDF-1.7 notice"),
			HTML:        "<html>notice</html>",
			Source:      documenttemplate.SourceOrgDefault,
			VersionID:   &versionID,
			ContentHash: "hash-1",
		}, nil).
		Once()

	svc := &Service{l: zap.NewNop(), templates: resolver}

	pdf, err := svc.renderNoticePDF(
		t.Context(),
		pagination.TenantInfo{OrgID: occurrence.OrganizationID, BuID: occurrence.BusinessUnitID},
		occurrence,
		nil,
		&detentiontemplate.DetentionNoticeContext{},
	)
	require.NoError(t, err)

	assert.Equal(t, []byte("%PDF-1.7 notice"), pdf.Content)
	assert.Contains(t, pdf.FileName, "SHP-1001")
	require.NotNil(t, pdf.Provenance)
	assert.Nil(t, pdf.Provenance.PDF, "the payload must not carry the document")
	assert.Empty(t, pdf.Provenance.HTML)
	assert.Equal(t, &versionID, pdf.Provenance.VersionID,
		"the version behind the bytes is the whole point of carrying provenance")
	assert.Equal(t, "hash-1", pdf.Provenance.ContentHash)
}

// A notice is the evidence a detention charge rests on. A template or renderer
// problem must cost the attachment, never the notice.
func TestBuildNoticeSendsWithoutAPDFWhenTheDocumentRenderFails(t *testing.T) {
	t.Parallel()

	occurrence := pdfOccurrence()
	occurrence.PolicySnapshot = nil

	resolver := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&documenttemplateports.RenderedMessage{
			Subject: "Detention notice",
			Text:    "plain",
			HTML:    "<p>html</p>",
		}, nil).
		Once()
	resolver.EXPECT().
		RenderDocument(mock.Anything, mock.Anything).
		Return(nil, errors.New("the renderer is unavailable")).
		Once()

	svc := &Service{
		l:              zap.NewNop(),
		templates:      resolver,
		contextBuilder: &detentionnoticerender.ContextBuilder{},
		now:            func() int64 { return 1_767_225_600 },
	}

	content, err := svc.BuildNotice(t.Context(), &detentionnoticerender.BuildNoticeParams{
		Occurrence:   occurrence,
		Kind:         detentionnotice.NoticeKindFinal,
		FacilityName: "Memphis DC",
		ShipmentRef:  "SHP-1001",
		Location:     time.UTC,
		AttachPDF:    true,
	})
	require.NoError(t, err, "a failed attachment must not fail the notice")

	assert.Equal(t, "Detention notice", content.Subject)
	assert.Equal(t, "<p>html</p>", content.HTML)
	assert.Nil(t, content.PDF)
}

func TestNoticeAttachmentsDescribeThePDFForTheProvider(t *testing.T) {
	t.Parallel()

	assert.Nil(t, noticeAttachments(nil))
	assert.Nil(t, noticeAttachments(&detentionnoticerender.NoticePDF{FileName: "empty.pdf"}),
		"a zero-byte render is not an attachment")

	attachments := noticeAttachments(&detentionnoticerender.NoticePDF{
		Content:  []byte("%PDF-1.7"),
		FileName: "detention-notice.pdf",
	})
	require.Len(t, attachments, 1)
	assert.Equal(t, "detention-notice.pdf", attachments[0].FileName)
	assert.Equal(t, "application/pdf", attachments[0].ContentType)
	assert.Equal(t, []byte("%PDF-1.7"), attachments[0].Content)
	assert.Equal(t, int64(8), attachments[0].SizeBytes)
}

// Filing points the notice at the document and logs the render against the
// template version behind it. Both halves matter: the first is what a dispute
// follows, the second is what survives the template being edited afterwards.
func TestAttachNoticePDFDocumentFilesTheDocumentAndRecordsTheRender(t *testing.T) {
	t.Parallel()

	noticeID := pulid.MustNew("dtn_")
	occurrenceID := pulid.MustNew("dto_")
	documentID := pulid.MustNew("doc_")
	versionID := pulid.MustNew("dtv_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	repo := &stubNoticeRepo{}
	resolver := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RecordGeneratedDocument(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *documenttemplateports.RecordGeneratedDocumentRequest,
		) (*generateddocument.GeneratedDocument, error) {
			assert.Equal(t, documenttemplate.KindDetentionNoticePDF, req.Kind)
			assert.Equal(t, noticePDFReferenceType, req.ReferenceType)
			assert.Equal(t, occurrenceID, req.ReferenceID)
			require.NotNil(t, req.DocumentID)
			assert.Equal(t, documentID, *req.DocumentID)
			require.NotNil(t, req.Rendered)
			assert.Equal(t, &versionID, req.Rendered.VersionID)
			return &generateddocument.GeneratedDocument{}, nil
		}).
		Once()

	svc := &Service{l: zap.NewNop(), noticeRepo: repo, templates: resolver}

	require.NoError(t, svc.AttachNoticePDFDocument(
		t.Context(),
		&detentionports.AttachNoticePDFDocumentParams{
			TenantInfo:   tenantInfo,
			NoticeID:     noticeID,
			OccurrenceID: occurrenceID,
			DocumentID:   documentID,
			FileName:     "detention-notice.pdf",
			FileSize:     16,
			Provenance:   &documenttemplateports.RenderedDocument{VersionID: &versionID},
		},
	))

	require.Len(t, repo.attached, 1)
	assert.Equal(t, noticeID, repo.attached[0].NoticeID)
	assert.Equal(t, documentID, repo.attached[0].DocumentID)
	assert.Equal(t, tenantInfo.OrgID, repo.attached[0].TenantInfo.OrgID)
}

// The document is filed and the notice already points at it. Failing here would
// retry the whole filing and duplicate the document over a missing audit row.
func TestAttachNoticePDFDocumentSucceedsWhenTheAuditRowFails(t *testing.T) {
	t.Parallel()

	repo := &stubNoticeRepo{}
	resolver := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RecordGeneratedDocument(mock.Anything, mock.Anything).
		Return(nil, errors.New("generated_documents is unavailable")).
		Once()

	svc := &Service{l: zap.NewNop(), noticeRepo: repo, templates: resolver}

	require.NoError(t, svc.AttachNoticePDFDocument(
		t.Context(),
		&detentionports.AttachNoticePDFDocumentParams{
			NoticeID:   pulid.MustNew("dtn_"),
			DocumentID: pulid.MustNew("doc_"),
			Provenance: &documenttemplateports.RenderedDocument{},
		},
	))
	require.Len(t, repo.attached, 1)
}

func TestAttachNoticePDFDocumentRefusesAnIncompleteFiling(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop(), noticeRepo: &stubNoticeRepo{}}

	require.Error(t, svc.AttachNoticePDFDocument(t.Context(), nil))
	require.Error(t, svc.AttachNoticePDFDocument(
		t.Context(),
		&detentionports.AttachNoticePDFDocumentParams{NoticeID: pulid.MustNew("dtn_")},
	))
}

// The sweep reads the policy to decide whether to send at all. Passing it back in
// is what stops the send path reading the same row again for every occurrence
// under that policy — with a nil repository, a re-read would panic.
func TestAttachesNoticePDFUsesThePolicyTheCallerAlreadyLoaded(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}
	occurrence := pdfOccurrence()

	tenantInfo := pagination.TenantInfo{OrgID: occurrence.OrganizationID}

	assert.True(t, svc.attachesNoticePDF(t.Context(), occurrence, tenantInfo,
		&detentionpolicy.DetentionPolicy{AttachNoticePDF: true}))
	assert.False(t, svc.attachesNoticePDF(t.Context(), occurrence, tenantInfo,
		&detentionpolicy.DetentionPolicy{AttachNoticePDF: false}))

	// No policy on the occurrence and none passed: nothing to read, nothing to
	// attach, and no repository call to make.
	assert.False(t, svc.attachesNoticePDF(t.Context(), occurrence, tenantInfo, nil))
}

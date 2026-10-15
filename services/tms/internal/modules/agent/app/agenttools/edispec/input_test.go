package edispec

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDocuments struct{ doc *document.Document }

func (f *fakeDocuments) GetByID(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	return f.doc, nil
}

type fakeContents struct{ content *documentcontent.Content }

func (f *fakeContents) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return f.content, nil
}

type readFixture struct {
	thread   pulid.ID
	uploader pulid.ID
	doc      *document.Document
	content  *documentcontent.Content
}

func newReadFixture() *readFixture {
	thread := pulid.MustNew("ath_")
	uploader := pulid.MustNew("usr_")
	return &readFixture{
		thread:   thread,
		uploader: uploader,
		doc: &document.Document{
			ID:           pulid.MustNew("doc_"),
			ResourceType: document.ConversationResourceType,
			ResourceID:   thread.String(),
			UploadedByID: uploader,
			OriginalName: "acme-guide.pdf",
		},
		content: &documentcontent.Content{
			Status: documentcontent.StatusIndexed,
			Pages: []*documentcontent.Page{
				{PageNumber: 1, ExtractedText: "Acme Retail Group"},
				{PageNumber: 2, ExtractedText: "AS2"},
			},
		},
	}
}

func (f *readFixture) read(ctx context.Context, req *ReadRequest) (*ediports.SpecDocument, error) {
	reader := NewDocumentReader(&fakeDocuments{doc: f.doc}, &fakeContents{content: f.content})
	if req.DocumentID.IsNil() {
		req.DocumentID = f.doc.ID
	}
	return reader.Read(ctx, req)
}

func TestRead_GivesThePagesOfAGuideAttachedToTheConversation(t *testing.T) {
	t.Parallel()

	f := newReadFixture()
	spec, err := f.read(t.Context(), &ReadRequest{ThreadID: f.thread, AttachedBy: f.uploader})
	require.NoError(t, err)
	assert.Equal(t, "acme-guide.pdf", spec.FileName)
	assert.Equal(t, []ediports.SpecPage{{Number: 1, Text: "Acme Retail Group"}, {Number: 2, Text: "AS2"}}, spec.Pages)

	_, err = f.read(t.Context(), &ReadRequest{ThreadID: f.thread})
	require.NoError(t, err, "an approver reads the conversation's own file")
}

func TestRead_RefusesAGuideFromElsewhere(t *testing.T) {
	t.Parallel()

	f := newReadFixture()
	_, err := f.read(t.Context(), &ReadRequest{ThreadID: pulid.MustNew("ath_")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was not attached to this conversation")

	_, err = f.read(t.Context(), &ReadRequest{ThreadID: f.thread, AttachedBy: pulid.MustNew("usr_")})
	require.Error(t, err, "another person's upload")

	_, err = f.read(t.Context(), &ReadRequest{})
	require.ErrorIs(t, err, ErrNoConversation)
}

func TestRead_WaitsForTheReadingToFinish(t *testing.T) {
	t.Parallel()

	f := newReadFixture()
	f.content.Status = documentcontent.StatusExtracting
	_, err := f.read(t.Context(), &ReadRequest{ThreadID: f.thread})
	require.ErrorIs(t, err, ErrStillReading)

	f.content.Status = documentcontent.StatusFailed
	f.content.FailureMessage = "encrypted PDF"
	_, err = f.read(t.Context(), &ReadRequest{ThreadID: f.thread})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted PDF")
}

func TestParse_KeepsEveryCitationAsTheModelGaveIt(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	documentID := pulid.MustNew("doc_")
	input, err := Parse(map[string]any{
		ParamDocumentID:     documentID.String(),
		ParamDocumentDigest: " sha256:ab ",
		ParamGaps: []any{map[string]any{
			"kind": "Ambiguous", "summary": "Appendix B is missing",
			"citation": map[string]any{"page": 6, "quote": "Appendix B"},
		}},
		ParamResolutions: []any{map[string]any{
			"field": "partner.timezone", "value": "America/Chicago", "source": "Person",
		}},
	}, tenant)
	require.NoError(t, err)
	assert.Equal(t, documentID, input.DocumentID)
	assert.Equal(t, "sha256:ab", input.Digest)
	assert.Equal(t, tenant, input.Draft.TenantInfo)
	require.Len(t, input.Draft.Gaps, 1)
	assert.Equal(t, 6, input.Draft.Gaps[0].Citation.Page)
	require.Len(t, input.Draft.Resolutions, 1)
	assert.Equal(t, ediports.ResolutionSourcePerson, input.Draft.Resolutions[0].Source)

	_, err = Parse(map[string]any{ParamDocumentID: "not-an-id"}, tenant)
	require.Error(t, err)
}

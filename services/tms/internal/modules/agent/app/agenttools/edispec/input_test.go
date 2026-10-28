package edispec

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
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

func (f *readFixture) read(
	ctx context.Context,
	req *ReadRequest,
) (*ediconnectiondraftports.SpecDocument, error) {
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
	assert.Equal(
		t,
		[]ediconnectiondraftports.SpecPage{
			{Number: 1, Text: "Acme Retail Group"},
			{Number: 2, Text: "AS2"},
		},
		spec.Pages,
	)

	_, err = f.read(t.Context(), &ReadRequest{ThreadID: f.thread})
	require.NoError(t, err, "an approver reads the conversation's own file")
}

func TestRead_RefusesAGuideFromElsewhere(t *testing.T) {
	t.Parallel()

	f := newReadFixture()
	_, err := f.read(t.Context(), &ReadRequest{ThreadID: pulid.MustNew("ath_")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was not attached to this conversation")

	_, err = f.read(
		t.Context(),
		&ReadRequest{ThreadID: f.thread, AttachedBy: pulid.MustNew("usr_")},
	)
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
	assert.Equal(
		t,
		ediconnectiondraftports.ResolutionSourcePerson,
		input.Draft.Resolutions[0].Source,
	)

	_, err = Parse(map[string]any{ParamDocumentID: "not-an-id"}, tenant)
	require.Error(t, err)
}

func TestRedactQuotes_KeepsSecretsOutOfTheStoredArguments(t *testing.T) {
	t.Parallel()

	secretQuote := "Username: acme Password: Acme#Test2026"
	params := map[string]any{
		ParamDocumentID: pulid.MustNew("doc_").String(),
		ParamValues: []any{map[string]any{
			"field": "transport.username", "value": "acme", "confidence": 0.95,
			"citation": map[string]any{"page": 2, "quote": secretQuote, "section": "2.1"},
		}},
		ParamRequirements: []any{map[string]any{
			"segment": "ISA",
			"citation": map[string]any{
				"page": 3, "quote": "ISA*01*LOGIN     *06*PASSWORD12*ZZ*ACME",
			},
		}},
		ParamMappings: []any{map[string]any{
			"sourceValue": "CL",
			"citation":    map[string]any{"page": 4, "quote": "API key = k-123 for code CL"},
		}},
		ParamGaps: []any{map[string]any{
			"summary":  "The token: t-999 is sent by email",
			"citation": map[string]any{"page": 6, "quote": "token: t-999"},
		}},
		ParamResolutions: []any{
			map[string]any{"field": "partner.timezone", "value": "America/Chicago"},
		},
	}

	redacted := RedactQuotes(params)

	input, err := Parse(redacted, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, "Username: acme Password: [redacted]", input.Draft.Values[0].Citation.Quote)
	assert.Equal(t, "acme", input.Draft.Values[0].Value)
	assert.Equal(t, 2, input.Draft.Values[0].Citation.Page)
	assert.Equal(t, "2.1", input.Draft.Values[0].Citation.Section)
	assert.Equal(
		t,
		"ISA*01*XXXXXXXXXX*06*XXXXXXXXXX*ZZ*ACME",
		input.Draft.Requirements[0].Citation.Quote,
	)
	assert.Equal(t, "API key = [redacted] for code CL", input.Draft.Mappings[0].Citation.Quote)
	assert.Equal(t, "The token: [redacted] is sent by email", input.Draft.Gaps[0].Summary)
	assert.Equal(t, "token: [redacted]", input.Draft.Gaps[0].Citation.Quote)
	assert.Equal(t, params[ParamResolutions], redacted[ParamResolutions])

	original, err := Parse(params, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.Equal(t, secretQuote, original.Draft.Values[0].Citation.Quote,
		"the arguments the model sent are left as it sent them")
}

func TestRedactQuotes_LeavesWhatItCannotReadForValidateToRefuse(t *testing.T) {
	t.Parallel()

	params := map[string]any{
		ParamValues: "not a list",
		ParamGaps:   []any{"not an object", map[string]any{"citation": "not an object"}},
	}

	assert.Equal(t, params, RedactQuotes(params))
	assert.Nil(t, RedactQuotes(nil))
}

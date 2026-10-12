package inboundreplyservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendCarriesAttachmentsByObjectKeyWithTheThreadingHeaders(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.email.result.Status = email.MessageStatusSent
	eld := f.documents.add("ELD_SEED-PAY-002_Oct3.pdf", 48_000)
	bol := f.documents.add("BOL.pdf", 12_000)
	req := sendRequest(f, "key-attach")
	req.DocumentIDs = []pulid.ID{eld.ID, bol.ID}

	reply, err := f.service.Send(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, f.email.requests, 1)
	sent := f.email.requests[0]
	require.Len(t, sent.Attachments, 2)
	assert.Equal(t, eld.StoragePath, sent.Attachments[0].ObjectKey)
	assert.Equal(t, email.AttachmentSealDocument, sent.Attachments[0].Seal)
	assert.Empty(t, sent.Attachments[0].Content)
	assert.Equal(t, "<abc123@partner.example>", sent.Headers["In-Reply-To"])
	assert.Contains(t, sent.Headers["References"], "<abc123@partner.example>")

	assert.Equal(t, []triage.ReplyAttachment{
		{
			DocumentID:  eld.ID,
			FileName:    "ELD_SEED-PAY-002_Oct3.pdf",
			ContentType: "application/pdf",
			ByteSize:    48_000,
		},
		{DocumentID: bol.ID, FileName: "BOL.pdf", ContentType: "application/pdf", ByteSize: 12_000},
	}, reply.Attachments)

	asked := f.documents.last()
	require.NotNil(t, asked)
	assert.False(t, asked.System)
	assert.Equal(t, testUserID, asked.TenantInfo.UserID)
	assert.Equal(t, []string{"edi@partner.example"}, asked.Recipients)
	assert.Equal(t, "documentIds", asked.Field)

	var replied *triage.Activity
	for _, activity := range f.recorder.activities {
		if activity.Kind == triage.ActivityReplied {
			replied = activity
		}
	}
	require.NotNil(t, replied)
	assert.Equal(t, "ELD_SEED-PAY-002_Oct3.pdf, BOL.pdf", replied.Detail)
}

func TestSendKeepsTheSnapshotWhenTheDocumentIsRenamedLater(t *testing.T) {
	t.Parallel()

	f := newFixture()
	eld := f.documents.add("ELD.pdf", 1_000)
	req := sendRequest(f, "key-rename")
	req.DocumentIDs = []pulid.ID{eld.ID}

	reply, err := f.service.Send(t.Context(), req)
	require.NoError(t, err)
	eld.OriginalName = "renamed.pdf"

	stored, err := f.replies.GetByID(t.Context(), inboundmessageports.GetInboundReplyRequest{
		ID:         reply.ID,
		TenantInfo: testTenant,
	})
	require.NoError(t, err)
	require.Len(t, stored.Attachments, 1)
	assert.Equal(t, "ELD.pdf", stored.Attachments[0].FileName)
}

func TestSendRefusesAnUnknownDocumentBeforeAnythingIsStored(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := sendRequest(f, "key-missing")
	req.DocumentIDs = []pulid.ID{pulid.MustNew("doc_")}

	_, err := f.service.Send(t.Context(), req)

	require.Error(t, err)
	assert.True(t, errortypes.IsMultiError(err))
	assert.Empty(t, f.email.requests)
	assert.Empty(t, f.replies.byStatus(triage.ReplySending))
	assert.Empty(t, f.replies.byStatus(triage.ReplyFailed))
}

func TestSendRetryToAnAlternateAddressCarriesTheSameAttachments(t *testing.T) {
	t.Parallel()

	f := newFixture()
	eld := f.documents.add("ELD.pdf", 1_000)
	f.email.err = errortypes.NewBusinessError("Recipient edi@partner.example was rejected by the provider")
	first := sendRequest(f, "key-first")
	first.DocumentIDs = []pulid.ID{eld.ID}

	failed, err := f.service.Send(t.Context(), first)
	require.NoError(t, err)
	assert.Equal(t, triage.ReplyFailed, failed.Status)
	assert.Len(t, failed.Attachments, 1)

	f.email.err = nil
	retry := sendRequest(f, "key-retry")
	retry.ToAddress = "ap@partner.example"
	retry.DocumentIDs = []pulid.ID{eld.ID}

	resent, err := f.service.Send(t.Context(), retry)
	require.NoError(t, err)

	require.Len(t, f.email.requests, 2)
	assert.Equal(t, []string{"ap@partner.example"}, f.email.requests[1].To)
	require.Len(t, f.email.requests[1].Attachments, 1)
	assert.Equal(t, eld.StoragePath, f.email.requests[1].Attachments[0].ObjectKey)
	assert.Equal(t, failed.Attachments, resent.Attachments)
	assert.Equal(t, []string{"ap@partner.example"}, f.documents.last().Recipients)
}

func TestSendWithoutAttachmentsKeepsTheAddressOutOfTheDetail(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.email.result.Status = email.MessageStatusSent

	reply, err := f.service.Send(t.Context(), sendRequest(f, "key-plain"))
	require.NoError(t, err)
	assert.Empty(t, reply.Attachments)
	assert.Empty(t, f.email.requests[0].Attachments)
	for _, activity := range f.recorder.activities {
		if activity.Kind == triage.ActivityReplied {
			assert.Equal(t, "dispatch@trenova.example", activity.Subject)
			assert.Empty(t, activity.Detail)
		}
	}
}

func TestSaveDraftStoresAttachmentsAndKeepsThemWhenTheyAreNotSent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	eld := f.documents.add("ELD.pdf", 1_000)

	saved, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo:         testTenant,
		UserID:             testUserID,
		MessageID:          f.message.ID,
		Body:               "Attached is the record.",
		DocumentIDs:        []pulid.ID{eld.ID},
		ReplaceAttachments: true,
	})
	require.NoError(t, err)
	require.Len(t, saved.Attachments, 1)
	assert.Equal(t, eld.ID, saved.Attachments[0].DocumentID)
	assert.False(t, f.documents.last().System)
	assert.Empty(t, f.documents.last().Recipients)

	kept, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Body:       "Attached is the record, as promised.",
	})
	require.NoError(t, err)
	require.Len(t, kept.Attachments, 1)

	cleared, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo:         testTenant,
		UserID:             testUserID,
		MessageID:          f.message.ID,
		Body:               "No file after all.",
		ReplaceAttachments: true,
	})
	require.NoError(t, err)
	assert.Empty(t, cleared.Attachments)
}

func TestSaveDraftRefusesADocumentThatCannotBeAttached(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo:         testTenant,
		UserID:             testUserID,
		MessageID:          f.message.ID,
		Body:               "Attached.",
		DocumentIDs:        []pulid.ID{pulid.MustNew("doc_")},
		ReplaceAttachments: true,
	})

	require.Error(t, err)
	assert.Empty(t, f.replies.byStatus(triage.ReplyDraft))
}

func standByFixture(t *testing.T) *fixture {
	t.Helper()

	f := newFixture()
	f.message.Classification = inboundmessage.ClassificationDetentionDispute
	f.message.Subject = "Detention on SEED-PAY-002"
	f.message.ActionState.StopsDecision = inboundmessage.StopsDecisionStandBy
	f.nova.text = `{"body":"Hi Dana,\n\nThe detention stands; the attached record shows the times.\n\nTrenova Freight Dispatch"}`

	return f
}

func sectionNamed(f *fixture, title string) (string, bool) {
	for _, section := range f.nova.requests[len(f.nova.requests)-1].Context.Sections {
		if section.Title == title {
			return section.Content, section.Trusted
		}
	}

	return "", false
}

func TestStandByDraftAttachesTheDetentionEvidence(t *testing.T) {
	t.Parallel()

	f := standByFixture(t)
	eld := f.documents.add("ELD_SEED-PAY-002_Oct3.pdf", 48_000)
	f.evidence.evidence = &inboundmessageports.TriageDetentionEvidence{
		DocumentID: eld.ID,
		FileName:   eld.OriginalName,
	}

	draft, err := f.service.DraftReplyFor(t.Context(), f.message, triagenova.Interactive)
	require.NoError(t, err)

	require.Len(t, draft.Attachments, 1)
	assert.Equal(t, eld.ID, draft.Attachments[0].DocumentID)
	require.Len(t, f.evidence.asked, 1)
	assert.Equal(t, f.message.MatchedShipmentID, f.evidence.asked[0].ShipmentID)
	assert.True(t, f.documents.last().System)

	attached, trusted := sectionNamed(f, "Attachments")
	assert.True(t, trusted)
	assert.Contains(t, attached, "ELD_SEED-PAY-002_Oct3.pdf")
	task, _ := sectionNamed(f, "What to write")
	assert.Contains(t, task, "attached")
}

func TestStandByDraftWithoutEvidenceClaimsNoAttachment(t *testing.T) {
	t.Parallel()

	f := standByFixture(t)

	draft, err := f.service.DraftReplyFor(t.Context(), f.message, triagenova.Interactive)
	require.NoError(t, err)

	assert.Empty(t, draft.Attachments)
	attached, _ := sectionNamed(f, "Attachments")
	assert.Contains(t, attached, "Nothing is attached")
	task, _ := sectionNamed(f, "What to write")
	assert.NotContains(t, strings.ToLower(task), "attached record")
	assert.Contains(t, f.nova.requests[0].System, "Attachments")
}

func TestStandByDraftWhenEvidenceFailsStillDrafts(t *testing.T) {
	t.Parallel()

	f := standByFixture(t)
	f.evidence.err = errors.New("renderer down")

	draft, err := f.service.DraftReplyFor(t.Context(), f.message, triagenova.Interactive)
	require.NoError(t, err)
	assert.Empty(t, draft.Attachments)
}

func TestOtherNovaDraftsAttachNothing(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.nova.text = `{"body":"Hi Dana,\n\nS-104251 is running behind.\n\nTrenova Freight Dispatch"}`
	f.evidence.evidence = &inboundmessageports.TriageDetentionEvidence{DocumentID: pulid.MustNew("doc_")}

	draft, err := f.service.DraftReplyFor(t.Context(), f.message, triagenova.Background)
	require.NoError(t, err)

	assert.Empty(t, draft.Attachments)
	assert.Empty(t, f.evidence.asked)
}

func TestNovaRedraftKeepsWhatAPersonAttached(t *testing.T) {
	t.Parallel()

	f := standByFixture(t)
	bol := f.documents.add("BOL.pdf", 1_000)
	eld := f.documents.add("ELD.pdf", 2_000)
	f.evidence.evidence = &inboundmessageports.TriageDetentionEvidence{DocumentID: eld.ID}
	_, err := f.service.SaveDraft(t.Context(), &SaveDraftRequest{
		TenantInfo:         testTenant,
		UserID:             testUserID,
		MessageID:          f.message.ID,
		Body:               "Draft",
		DocumentIDs:        []pulid.ID{bol.ID},
		ReplaceAttachments: true,
	})
	require.NoError(t, err)

	draft, err := f.service.DraftReplyFor(t.Context(), f.message, triagenova.Interactive)
	require.NoError(t, err)

	require.Len(t, draft.Attachments, 2)
	assert.Equal(t, bol.ID, draft.Attachments[0].DocumentID)
	assert.Equal(t, eld.ID, draft.Attachments[1].DocumentID)
}

func TestAttachablesListsTheMessageAndItsRecordsFiles(t *testing.T) {
	t.Parallel()

	f := newFixture()
	customerID := pulid.MustNew("cus_")
	f.messages.messages[f.message.ID].MatchedCustomerID = customerID
	own := f.documents.add("rate-con.pdf", 10)
	own.ResourceType = "inbound_message"
	own.ResourceID = f.message.ID.String()
	bol := f.documents.add("BOL.pdf", 20)
	bol.ResourceType = "shipment"
	bol.ResourceID = f.message.MatchedShipmentID.String()
	bol.DocumentType = &documenttype.DocumentType{Name: "Bill of lading"}
	stray := f.documents.add("other.pdf", 30)
	stray.ResourceType = "shipment"
	stray.ResourceID = pulid.MustNew("shp_").String()
	f.documents.mailable = []*document.Document{own, bol, stray}

	attachables, err := f.service.Attachables(t.Context(), &AttachablesRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
	})
	require.NoError(t, err)

	require.Len(t, attachables, 2)
	assert.Equal(t, triage.AttachableFromMessage, attachables[0].Source)
	assert.Equal(t, "rate-con.pdf", attachables[0].FileName)
	assert.Equal(t, triage.AttachableFromShipment, attachables[1].Source)
	assert.Equal(t, "Bill of lading", attachables[1].DocumentType)

	require.NotNil(t, f.documents.listed)
	assert.Equal(t, testUserID, f.documents.listed.TenantInfo.UserID)
	assert.Equal(t, []documentports.MailDocumentSource{
		{ResourceType: "inbound_message", ResourceID: f.message.ID.String()},
		{ResourceType: "shipment", ResourceID: f.message.MatchedShipmentID.String()},
		{ResourceType: "customer", ResourceID: customerID.String()},
	}, f.documents.listed.Sources)
}

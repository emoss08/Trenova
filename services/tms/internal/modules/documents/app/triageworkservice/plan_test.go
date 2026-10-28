package triageworkservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Each plan runs the checks its write runs and changes a copy the way the write
changes the record, so what an agent's preview shows is what the write does.
None of them writes: the fakes record every write, and each test asserts there
were none before making the write and comparing.
*/

type stubNova struct {
	available bool
}

func (s stubNova) Available(triagenova.Mode) bool { return s.available }

func (s stubNova) Complete(
	context.Context,
	triagenova.Mode,
	*llmports.StructuredCompletionRequest,
) (*llmports.StructuredCompletionResult, error) {
	return nil, errors.New("a plan never asks Nova")
}

func TestPlanLogException_IsTheHoldTheWritePlaces(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &LogExceptionRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Kind:       inboundmessage.ExceptionDamage,
		Quantity:   2,
		Unit:       " cases ",
		Note:       " crushed corners ",
	}

	plan, err := f.service.PlanLogException(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, f.exceptions.requests, "a plan places no hold")
	assert.Equal(t, "2 cases damaged", plan.Summary)
	assert.Equal(t, inboundmessage.ExceptionDamage, plan.After.ActionState.ExceptionKind)
	assert.Empty(t, plan.Message.ActionState.ExceptionKind)

	_, err = f.service.LogException(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, f.exceptions.requests, 1)
	assert.Equal(t, plan.Hold, f.exceptions.requests[0], "the hold placed is the one planned")
	assert.Equal(t, plan.After.ActionState, f.storedMessage().ActionState)

	f.messages.messages[f.message.ID].MatchedShipmentID = pulid.Nil
	_, err = f.service.PlanLogException(t.Context(), req)
	assert.True(t, errortypes.IsBusinessError(err), "an unlinked message has no shipment to hold")
}

func TestPlanBlockSender_IsWhatTheWriteBlocksAndSettles(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &BlockSenderRequest{TenantInfo: testTenant, UserID: testUserID, MessageID: f.message.ID}

	plan, err := f.service.PlanBlockSender(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.Changed)
	assert.Equal(t, "billing@carrier.example", plan.Sender)
	assert.Equal(t, []string{"billing@carrier.example"}, plan.Blocked)
	assert.Equal(t, inboundmessage.StatusIgnored, plan.After.Status)
	mailbox := f.mailboxes.mailboxes[f.message.MailboxID]
	assert.Empty(t, mailbox.BlockedSenders, "a plan blocks no one")

	_, err = f.service.BlockSender(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, plan.Blocked, f.mailboxes.mailboxes[f.message.MailboxID].BlockedSenders)
	assert.Equal(t, plan.After.Status, f.storedMessage().Status)
	assert.Equal(t, plan.After.ReviewNote, f.storedMessage().ReviewNote)

	again, err := f.service.PlanBlockSender(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, again.Changed, "a sender already blocked is not blocked twice")
}

func TestPlanUnblockSender_SaysWhenTheSenderIsNotBlocked(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &UnblockSenderRequest{
		TenantInfo: testTenant,
		MailboxID:  f.message.MailboxID,
		Sender:     "billing@carrier.example",
	}

	plan, err := f.service.PlanUnblockSender(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, plan.Changed)

	f.mailboxes.mailboxes[f.message.MailboxID].BlockedSenders = []string{"billing@carrier.example"}
	plan, err = f.service.PlanUnblockSender(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.Changed)
	assert.Empty(t, plan.Blocked)
	mailbox := f.mailboxes.mailboxes[f.message.MailboxID]
	assert.Len(t, mailbox.BlockedSenders, 1, "a plan unblocks no one")

	sender, err := SenderOf(f.message)
	require.NoError(t, err)
	assert.Equal(t, req.Sender, sender, "the sender a block records is the one an unblock names")
}

func TestPlanDecideStops_ChecksWhatTheDecisionChecks(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &DecideStopsRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Decision:   inboundmessage.StopsDecisionWaive,
	}

	_, err := f.service.PlanDecideStops(t.Context(), req)
	require.Error(t, err, "stop times are checked only for detention disputes")

	f.messages.messages[f.message.ID].Classification = inboundmessage.ClassificationDetentionDispute
	plan, err := f.service.PlanDecideStops(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, plan.Unchanged)
	assert.Equal(t, inboundmessage.StopsDecisionWaive, plan.After.ActionState.StopsDecision)
	assert.Empty(t, f.detention.requests, "a plan settles no detention")
	assert.Empty(t, f.drafter.drafted, "a plan drafts no reply")

	_, err = f.service.DecideStops(t.Context(), req)
	require.NoError(t, err)
	again, err := f.service.PlanDecideStops(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, again.Unchanged, "the same decision twice changes nothing")
}

func TestPlanCreateLoad_IsTheLoadTheWriteCreates(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].MatchedCustomerID = pulid.MustNew("cus_")
	req := &CreateLoadRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageID:  f.message.ID,
		Fields: []LoadFieldInput{
			{Key: inboundmessage.LoadFieldOrigin, Value: " Columbus, OH "},
		},
		IdempotencyKey: "load-key",
	}

	plan, err := f.service.PlanCreateLoad(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, plan.AlreadyCreated)
	assert.Empty(t, f.loads.requests, "a plan creates no load")
	assert.Equal(t, f.messages.messages[f.message.ID].MatchedCustomerID, plan.Load.CustomerID)

	_, err = f.service.CreateLoad(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, f.loads.requests, 1)
	assert.Equal(t, plan.Load, f.loads.requests[0], "the load created is the one planned")

	again, err := f.service.PlanCreateLoad(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, again.AlreadyCreated)
}

func TestPlanDiscardLoad_RefusesWhatTheWriteRefuses(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].LoadDraft = &inboundmessage.LoadDraft{
		Fields: []inboundmessage.LoadDraftField{
			{Key: inboundmessage.LoadFieldOrigin, Value: "Dayton, OH"},
		},
	}
	req := &LoadRequest{TenantInfo: testTenant, UserID: testUserID, MessageID: f.message.ID}

	plan, err := f.service.PlanDiscardLoad(t.Context(), req)
	require.NoError(t, err)
	assert.NotNil(t, plan.Message.LoadDraft)
	assert.Nil(t, plan.After.LoadDraft)
	assert.NotNil(t, f.storedMessage().LoadDraft, "a plan discards nothing")

	f.messages.messages[f.message.ID].ActionState.CreatedShipmentID = pulid.MustNew("shp_")
	_, err = f.service.PlanDiscardLoad(t.Context(), req)
	require.Error(t, err, "a load already created from the message keeps its draft")
}

func TestPlanDraftLoad_SaysWhenNovaMustReadTheBody(t *testing.T) {
	t.Parallel()

	f := newFixture()
	delete(f.messages.attachments, f.attachment.ID)
	req := &LoadRequest{TenantInfo: testTenant, UserID: testUserID, MessageID: f.message.ID}

	f.service.nova = stubNova{available: true}
	plan, err := f.service.PlanDraftLoad(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.NeedsNova)
	assert.Nil(t, plan.Draft)
	assert.Nil(t, f.storedMessage().LoadDraft, "a plan drafts nothing")

	f.service.nova = stubNova{}
	_, err = f.service.PlanDraftLoad(t.Context(), req)
	require.ErrorIs(t, err, triagenova.ErrUnavailable)
}

func TestPlanFile_IsWhatTheWholeFilingRecords(t *testing.T) {
	t.Parallel()

	f := newFixture()
	existingID := pulid.MustNew("doc_")
	f.docRepo.docs[existingID] = &document.Document{ID: existingID, Status: document.StatusActive}
	f.messages.messages[f.message.ID].Conflicts = []inboundmessage.Conflict{{
		ID:                 "dup:" + f.attachment.ID.String(),
		Kind:               inboundmessage.ConflictDuplicateDocument,
		AttachmentID:       f.attachment.ID,
		ExistingDocumentID: existingID,
		Resolution:         inboundmessage.ConflictReplaceExisting,
	}}
	req := &FileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
	}

	plan, err := f.service.PlanFile(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, f.pod.ID, plan.DocumentType.ID,
		"with no type named it files as what it was read as")
	assert.Equal(t, existingID, plan.Replaces)
	assert.Empty(t, f.documents.lineage, "a plan files nothing")
	assert.False(t, f.storedAttachment().Filed())

	_, err = f.service.File(t.Context(), req)
	require.NoError(t, err)
	stored := f.storedAttachment()
	assert.Equal(t, plan.After.FiledShipmentID, stored.FiledShipmentID)
	assert.Equal(t, plan.After.FiledAs, stored.FiledAs)
	assert.Equal(t, plan.After.FiledBy, stored.FiledBy)
	assert.Equal(t, plan.After.FiledAt, stored.FiledAt)

	_, err = f.service.PlanFile(t.Context(), req)
	require.Error(t, err, "a file already filed is refused")
}

func TestPlanFile_PlansEachPageRangeAsItsOwnDocument(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.attachments[f.attachment.ID].PageCount = 4
	req := &FileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
		Pages: []PageRange{
			{FromPage: 1, ToPage: 2, DocumentTypeID: f.bol.ID},
			{FromPage: 3, ToPage: 3, DocumentTypeID: f.pod.ID},
		},
	}

	plan, err := f.service.PlanFile(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, plan.PagesUnchecked)
	require.Len(t, plan.Ranges, 2)
	assert.Equal(t, "Bill of Lading", plan.Ranges[0].DocumentType)
	assert.Equal(t, "paperwork (pages 1-2).pdf", plan.Ranges[0].FileName)
	assert.Equal(t, "paperwork (page 3).pdf", plan.Ranges[1].FileName)
	assert.Equal(t, "2 documents (Bill of Lading, Proof of Delivery)", plan.After.FiledAs)
	assert.Empty(t, f.uploads.sessions, "a plan stages nothing")

	req.Pages = []PageRange{{FromPage: 3, ToPage: 5, DocumentTypeID: f.pod.ID}}
	_, err = f.service.PlanFile(t.Context(), req)
	require.Error(t, err, "a range past the last page is refused")

	f.messages.attachments[f.attachment.ID].PageCount = 0
	plan, err = f.service.PlanFile(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.PagesUnchecked, "an unknown page count is checked when the write runs")
}

func TestPlanUnfile_IsWhatTheWriteTakesBack(t *testing.T) {
	t.Parallel()

	f := newFixture()
	req := &UnfileRequest{TenantInfo: testTenant, UserID: testUserID, AttachmentID: f.attachment.ID}

	_, err := f.service.PlanUnfile(t.Context(), req)
	require.Error(t, err, "a file that is not filed cannot be unfiled")

	_, err = f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		UserID:         testUserID,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.NoError(t, err)
	filed := len(f.documents.lineage)

	plan, err := f.service.PlanUnfile(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, plan.After.FiledShipmentID.IsNil())
	assert.Empty(t, plan.Removed)
	assert.Len(t, f.documents.lineage, filed, "a plan moves no document")

	_, err = f.service.Unfile(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, plan.After.FiledShipmentID, f.storedAttachment().FiledShipmentID)
	assert.Equal(t, plan.After.FiledAs, f.storedAttachment().FiledAs)
}

package triageworkservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWholeAttachesToShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()

	message, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		UserID:         testUserID,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, message)

	require.Len(t, f.documents.lineage, 1)
	assert.Equal(t, inboundmessageports.DocumentResourceType, f.documents.lineage[0].resourceType)
	assert.Equal(t, f.shipmentID.String(), f.documents.lineage[0].resourceID)

	doc := f.docRepo.docs[f.attachment.DocumentID]
	require.NotNil(t, doc.DocumentTypeID)
	assert.Equal(t, f.pod.ID, *doc.DocumentTypeID)

	stored := f.storedAttachment()
	assert.Equal(t, f.shipmentID, stored.FiledShipmentID)
	assert.Equal(t, "Proof of Delivery", stored.FiledAs)
	assert.Equal(t, testUserID, stored.FiledBy)
	assert.Equal(t, int64(1_791_000_000), stored.FiledAt)

	filed := f.recorder.find(triage.ActivityFiled)
	require.NotNil(t, filed)
	assert.Equal(t, f.shipmentID, filed.RecordID)
	assert.Equal(t, []pulid.ID{f.pod.ID}, f.events.attached)
	assert.Contains(t, f.notifier.changed, f.message.ID)
}

func TestFileWithoutATypeFilesAsWhatItWasReadAs(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
		ShipmentID:   f.shipmentID,
	})
	require.NoError(t, err)

	doc := f.docRepo.docs[f.attachment.DocumentID]
	require.NotNil(t, doc.DocumentTypeID)
	assert.Equal(t, f.pod.ID, *doc.DocumentTypeID)
	assert.Equal(t, "Proof of Delivery", f.storedAttachment().FiledAs)
}

func TestFileRequiresTypeAndShipment(t *testing.T) {
	t.Parallel()

	f := newFixture()
	unknown := &inboundmessage.InboundAttachment{
		ID:         pulid.MustNew("iatt_"),
		MessageID:  f.message.ID,
		DocumentID: pulid.MustNew("doc_"),
		FileName:   "scan.pdf",
		Kind:       inboundmessage.AttachmentUnknown,
	}
	f.messages.attachments[unknown.ID] = unknown

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:   testTenant,
		AttachmentID: unknown.ID,
		ShipmentID:   f.shipmentID,
	})
	require.Error(t, err)
	assert.Equal(t, []string{"documentTypeId"}, errorFields(err))

	_, err = f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     pulid.MustNew("shp_"),
		DocumentTypeID: f.pod.ID,
	})
	require.Error(t, err)
	assert.Equal(t, []string{"shipmentId"}, errorFields(err))

	_, err = f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: pulid.MustNew("dt_"),
	})
	require.Error(t, err)
	assert.Equal(t, []string{"documentTypeId"}, errorFields(err))
	assert.Empty(t, f.documents.lineage)
}

func TestFileRefusesUnreadableOrFiled(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.attachments[f.attachment.ID].FailureText = "Could not read"

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	f.messages.attachments[f.attachment.ID].FailureText = ""
	f.messages.attachments[f.attachment.ID].FiledShipmentID = f.shipmentID

	_, err = f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestFileReplacesExistingThenUnfileRestores(t *testing.T) {
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

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		UserID:         testUserID,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, document.StatusArchived, f.docRepo.docs[existingID].Status)
	assert.Contains(t, f.recorder.find(triage.ActivityFiled).Detail, "replacing")

	_, err = f.service.Unfile(t.Context(), &UnfileRequest{
		TenantInfo:   testTenant,
		UserID:       testUserID,
		AttachmentID: f.attachment.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, document.StatusActive, f.docRepo.docs[existingID].Status)
	last := f.documents.lineage[len(f.documents.lineage)-1]
	assert.Equal(t, inboundResourceType, last.resourceType)
	assert.Equal(t, f.message.ID.String(), last.resourceID)
	assert.False(t, f.storedAttachment().Filed())
	assert.NotNil(t, f.recorder.find(triage.ActivityUnfiled))
}

func TestReplaceExistingWithoutKnownDocumentArchivesCurrent(t *testing.T) {
	t.Parallel()

	f := newFixture()
	currentID := pulid.MustNew("doc_")
	f.docRepo.docs[currentID] = &document.Document{ID: currentID, Status: document.StatusActive}
	f.work.current = []pulid.ID{currentID, f.attachment.DocumentID}
	f.messages.messages[f.message.ID].Conflicts = []inboundmessage.Conflict{{
		ID:           "dup:" + f.attachment.ID.String(),
		Kind:         inboundmessage.ConflictDuplicateDocument,
		AttachmentID: f.attachment.ID,
		Resolution:   inboundmessage.ConflictReplaceExisting,
	}}

	_, err := f.service.File(t.Context(), &FileRequest{
		TenantInfo:     testTenant,
		AttachmentID:   f.attachment.ID,
		ShipmentID:     f.shipmentID,
		DocumentTypeID: f.pod.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, document.StatusArchived, f.docRepo.docs[currentID].Status)
	assert.Equal(t, document.StatusActive, f.docRepo.docs[f.attachment.DocumentID].Status)
}

func TestUnfileNotFiled(t *testing.T) {
	t.Parallel()

	f := newFixture()

	_, err := f.service.Unfile(t.Context(), &UnfileRequest{TenantInfo: testTenant, AttachmentID: f.attachment.ID})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestFileManyFilesMatchedByKind(t *testing.T) {
	t.Parallel()

	f := newFixture()
	other := &inboundmessage.InboundAttachment{
		ID:         pulid.MustNew("iatt_"),
		MessageID:  f.message.ID,
		DocumentID: pulid.MustNew("doc_"),
		FileName:   "unknown.pdf",
		Kind:       inboundmessage.AttachmentUnknown,
	}
	f.messages.attachments[other.ID] = other

	result, err := f.service.FileMany(t.Context(), &FileManyRequest{
		TenantInfo: testTenant,
		UserID:     testUserID,
		MessageIDs: []pulid.ID{f.message.ID, f.message.ID, pulid.MustNew("imsg_")},
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Affected)
	require.Len(t, result.Messages, 1)
	assert.Equal(t, f.shipmentID, f.storedAttachment().FiledShipmentID)
	assert.False(t, f.messages.attachments[other.ID].Filed())
}

func TestFileManySkipsStatusRequests(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.messages.messages[f.message.ID].Classification = inboundmessage.ClassificationStatusRequest

	result, err := f.service.FileMany(t.Context(), &FileManyRequest{
		TenantInfo: testTenant,
		MessageIDs: []pulid.ID{f.message.ID},
	})
	require.NoError(t, err)
	assert.Zero(t, result.Affected)
	assert.Empty(t, f.documents.lineage)
}

func TestFileOnMatchedShipmentByNova(t *testing.T) {
	t.Parallel()

	f := newFixture()
	message := *f.message

	filed, err := f.service.FileOnMatchedShipment(t.Context(), inboundmessageports.FileOnMatchedRequest{
		Message: &message,
		ActorID: testUserID,
		ByNova:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, filed)
	assert.True(t, f.storedAttachment().FiledBy.IsNil())
}

func TestFileKeepsExistingUnlessReplaceChosen(t *testing.T) {
	t.Parallel()

	for _, resolution := range []inboundmessage.ConflictResolution{"", inboundmessage.ConflictKeepBoth} {
		f := newFixture()
		existingID := pulid.MustNew("doc_")
		f.docRepo.docs[existingID] = &document.Document{ID: existingID, Status: document.StatusActive}
		f.messages.messages[f.message.ID].Conflicts = []inboundmessage.Conflict{{
			ID:                 "dup:" + f.attachment.ID.String(),
			Kind:               inboundmessage.ConflictDuplicateDocument,
			AttachmentID:       f.attachment.ID,
			ExistingDocumentID: existingID,
			Resolution:         resolution,
		}}

		_, err := f.service.File(t.Context(), &FileRequest{
			TenantInfo:     testTenant,
			AttachmentID:   f.attachment.ID,
			ShipmentID:     f.shipmentID,
			DocumentTypeID: f.pod.ID,
		})
		require.NoError(t, err)
		assert.Equal(t, document.StatusActive, f.docRepo.docs[existingID].Status, string(resolution))
		assert.NotContains(t, f.recorder.find(triage.ActivityFiled).Detail, "replacing")
	}
}

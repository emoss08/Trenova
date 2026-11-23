package workercredentialservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreate_NamesWhatARenewalArchives(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).Maybe()
	h.workerRepo.EXPECT().UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).Maybe()

	planned, err := h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, workercredential.CredentialStatusActive, planned.Credential.Status)
	assert.Equal(t, "FORKLIFT", planned.Credential.CredentialType.Code)
	assert.Nil(t, planned.Superseded)
	assert.Empty(t, h.repo.createReqs)

	first, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.Error(t, err, "a second active credential needs renew")

	renewal, err := h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		Renew:  true,
		UserID: h.userID,
	})
	require.NoError(t, err)
	require.NotNil(t, renewal.Superseded)
	assert.Equal(t, first.ID, renewal.Superseded.ID)
	assert.Len(t, h.repo.createReqs, 1)

	edit := *first
	edit.Number = "N-2"
	change, err := h.svc.PlanUpdate(t.Context(), &edit, h.actor())
	require.NoError(t, err)
	assert.Equal(t, "N-2", change.After.Number)
	assert.Equal(t, "N-1", h.repo.credentials[first.ID].Number)

	archive, err := h.svc.PlanArchive(t.Context(), &workercredentialservice.StatusRequest{
		ID: first.ID, TenantInfo: h.tenant, Reason: "Left the company", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, workercredential.CredentialStatusArchived, archive.After.Status)
	assert.Equal(t, workercredential.CredentialStatusActive, h.repo.credentials[first.ID].Status)
}

func TestUpdateKeepsTheStoredNumberForAnEditorWithoutPersonalDetails(t *testing.T) {
	h := newHarnessSeeing(t, fixedFieldAccess(0))
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	edit := *created
	edit.Number = ""
	edit.Notes = "renewed at the depot"
	updated, err := h.svc.Update(t.Context(), &edit, h.actor())
	require.NoError(t, err)

	assert.Equal(t, "N-1", updated.Number)
	assert.Equal(t, "renewed at the depot", updated.Notes)
	assert.Equal(t, "N-1", h.repo.credentials[created.ID].Number)
}

func TestUpdateTakesTheNumberFromAnEditorWithPersonalDetails(t *testing.T) {
	h := newHarnessSeeing(t, fixedFieldAccess(protectedfield.AllGroups()))
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	edit := *created
	edit.Number = "N-2"
	updated, err := h.svc.Update(t.Context(), &edit, h.actor())
	require.NoError(t, err)

	assert.Equal(t, "N-2", updated.Number)
}

func (h *harness) fileDocumentsOnWorker() {
	h.docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req documentports.GetDocumentByIDRequest,
		) (*document.Document, error) {
			return &document.Document{
				ID:           req.ID,
				ResourceType: "worker",
				ResourceID:   h.wrk.ID.String(),
			}, nil
		}).
		Maybe()
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
}

func (h *harness) verifiedWithDocument(
	t *testing.T,
	typ *workercredential.WorkerCredentialType,
) (*workercredential.WorkerCredential, pulid.ID) {
	t.Helper()
	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(typ, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)
	docID := pulid.MustNew("doc_")
	_, err = h.svc.AttachDocument(t.Context(), &workercredentialservice.AttachDocumentRequest{
		ID:         created.ID,
		DocumentID: docID,
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	verified, err := h.svc.Verify(t.Context(), &workercredentialservice.StatusRequest{
		ID: created.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.NoError(t, err)
	require.True(t, verified.IsVerified())
	require.Equal(t, docID, verified.DocumentID)

	edit := *verified
	edit.CredentialType = nil
	edit.Document = nil
	return &edit, docID
}

func TestUpdate_KeepsTheDocumentACallerCarriesOver(t *testing.T) {
	h := newHarness(t)
	h.fileDocumentsOnWorker()
	edit, docID := h.verifiedWithDocument(t, h.forklift)

	edit.Notes = "renewed at the depot"
	updated, err := h.svc.Update(t.Context(), edit, h.actor())
	require.NoError(t, err)

	assert.Equal(t, docID, updated.DocumentID)
	assert.Equal(t, docID, h.repo.credentials[edit.ID].DocumentID)
	assert.True(t, updated.IsVerified(), "carrying the same document over is no change")
}

func TestUpdate_SetsADocumentOnACredentialWithout(t *testing.T) {
	h := newHarness(t)
	h.fileDocumentsOnWorker()
	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	docID := pulid.MustNew("doc_")
	edit := *created
	edit.DocumentID = docID
	updated, err := h.svc.Update(t.Context(), &edit, h.actor())
	require.NoError(t, err)

	assert.Equal(t, docID, updated.DocumentID)
	assert.Equal(t, docID, h.repo.credentials[created.ID].DocumentID)
}

func TestUpdate_RefusesADocumentFiledOnAnotherWorker(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	foreign := pulid.MustNew("doc_")
	h.docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&document.Document{
			ID:           foreign,
			ResourceType: "worker",
			ResourceID:   pulid.MustNew("wrk_").String(),
		}, nil).
		Once()
	edit := *created
	edit.DocumentID = foreign
	_, err = h.svc.Update(t.Context(), &edit, h.actor())

	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "documentId", verr.Field())
	assert.True(t, h.repo.credentials[created.ID].DocumentID.IsNil())
}

func TestUpdate_ReplacingTheDocumentClearsVerification(t *testing.T) {
	h := newHarness(t)
	h.fileDocumentsOnWorker()
	edit, docID := h.verifiedWithDocument(t, h.forklift)

	replacement := pulid.MustNew("doc_")
	edit.DocumentID = replacement
	updated, err := h.svc.Update(t.Context(), edit, h.actor())
	require.NoError(t, err)

	assert.Equal(t, replacement, updated.DocumentID)
	assert.NotEqual(t, docID, updated.DocumentID)
	assert.False(t, updated.IsVerified(), "a new document needs a fresh verification")
	assert.True(t, updated.VerifiedByID.IsNil())
}

func TestUpdate_RemovingTheDocumentClearsItAndTheVerification(t *testing.T) {
	h := newHarness(t)
	h.fileDocumentsOnWorker()
	edit, docID := h.verifiedWithDocument(t, h.medCard)
	require.True(t, h.medCard.RequiresDocument)

	planned, err := h.svc.PlanUpdate(t.Context(), &workercredential.WorkerCredential{
		ID:               edit.ID,
		OrganizationID:   edit.OrganizationID,
		BusinessUnitID:   edit.BusinessUnitID,
		Number:           edit.Number,
		IssuingAuthority: edit.IssuingAuthority,
		IssuedAt:         edit.IssuedAt,
		ExpiresAt:        edit.ExpiresAt,
		Notes:            edit.Notes,
		Version:          edit.Version,
	}, h.actor())
	require.NoError(t, err)
	assert.True(t, planned.After.DocumentID.IsNil(), "the plan shows the document removed")
	assert.False(t, planned.After.IsVerified())
	assert.Equal(t, docID, h.repo.credentials[edit.ID].DocumentID, "planning writes nothing")

	before := len(h.audits.entries)
	edit.DocumentID = pulid.Nil
	updated, err := h.svc.Update(t.Context(), edit, h.actor())
	require.NoError(t, err, "a type that requires a document still lets a wrong scan come off")

	assert.True(t, updated.DocumentID.IsNil())
	assert.True(t, h.repo.credentials[edit.ID].DocumentID.IsNil())
	assert.False(t, updated.IsVerified(), "removing the document is a change of document")
	assert.True(t, updated.VerifiedByID.IsNil())
	assert.Equal(t, workercredential.CredentialStatusActive, updated.Status)

	require.Len(t, h.audits.entries, before+1)
	entry := h.audits.entries[before]
	assert.Equal(t, permission.OpUpdate, entry.Operation)
	assert.Equal(t, docID.String(), entry.PreviousState["documentId"])
	assert.NotEqual(t, docID.String(), entry.CurrentState["documentId"])

	_, err = h.svc.Verify(t.Context(), &workercredentialservice.StatusRequest{
		ID: edit.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "it cannot be verified again until a document is attached")
	assert.Equal(t, "documentId", verr.Field())
}

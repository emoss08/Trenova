package documentmailservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	svc         *Service
	repo        *documentportsmock.MockDocumentRepository
	permissions *permissionportsmock.MockPermissionEngine
	audit       *auditportsmock.MockAuditService
	tenant      pagination.TenantInfo
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	repo := documentportsmock.NewMockDocumentRepository(t)
	permissions := permissionportsmock.NewMockPermissionEngine(t)
	audit := auditportsmock.NewMockAuditService(t)

	return &harness{
		svc: New(Params{
			Logger:      zap.NewNop(),
			Repo:        repo,
			Permissions: permissions,
			Audit:       audit,
		}),
		repo:        repo,
		permissions: permissions,
		audit:       audit,
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
}

func (h *harness) doc(name string, size int64, resourceType string) *document.Document {
	return &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		FileName:       "stored-" + name,
		OriginalName:   name,
		FileSize:       size,
		FileType:       "application/pdf",
		StoragePath:    h.tenant.OrgID.String() + "/" + resourceType + "/" + name,
		Status:         document.StatusActive,
		ResourceType:   resourceType,
		ResourceID:     pulid.MustNew("shp_").String(),
	}
}

func (h *harness) stored(docs ...*document.Document) {
	h.repo.EXPECT().GetByIDs(mock.Anything, mock.MatchedBy(
		func(req documentports.BulkDeleteDocumentRequest) bool {
			return req.TenantInfo.OrgID == h.tenant.OrgID && req.TenantInfo.BuID == h.tenant.BuID
		},
	)).Return(docs, nil).Once()
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)

	return fieldErr.Field
}

func (h *harness) allow(allowed map[string]bool) {
	h.permissions.EXPECT().CheckBatch(mock.Anything, mock.Anything).RunAndReturn(
		func(
			_ context.Context,
			req *permissionports.BatchPermissionCheckRequest,
		) (*permissionports.BatchPermissionCheckResult, error) {
			results := make([]permissionports.PermissionCheckResult, 0, len(req.Checks))
			for _, check := range req.Checks {
				results = append(results, permissionports.PermissionCheckResult{
					Allowed: allowed[check.Resource] && check.Operation == permission.OpRead,
				})
			}

			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		},
	).Once()
}

func ids(docs ...*document.Document) []pulid.ID {
	out := make([]pulid.ID, 0, len(docs))
	for _, doc := range docs {
		out = append(out, doc.ID)
	}

	return out
}

func TestResolveTurnsDocumentsIntoSealedObjectKeyAttachments(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	eld := h.doc("ELD_SEED-PAY-002_Oct3.pdf", 2048, "shipment")
	bol := h.doc("BOL.pdf", 4096, "shipment")
	h.stored(bol, eld)
	h.allow(map[string]bool{"document": true, "shipment": true})

	attachments, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{eld.ID, bol.ID, eld.ID},
	})

	require.NoError(t, err)
	require.Len(t, attachments, 2)
	assert.Equal(t, eld.ID, attachments[0].Document.ID)
	assert.Equal(t, emailports.EmailAttachment{
		FileName:    "ELD_SEED-PAY-002_Oct3.pdf",
		ContentType: "application/pdf",
		ObjectKey:   eld.StoragePath,
		SizeBytes:   2048,
		Seal:        email.AttachmentSealDocument,
	}, attachments[0].Attachment)
	assert.Empty(t, attachments[0].Attachment.Content)
	assert.Equal(t, bol.ID, attachments[1].Document.ID)
}

func TestResolveRefusesADocumentOutsideTheTenantOrMissing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	known := h.doc("BOL.pdf", 10, "shipment")
	missing := pulid.MustNew("doc_")
	h.stored(known)

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{known.ID, missing},
		Field:       "input.documentIds",
	})

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "input.documentIds[1]", multiErr.Errors[0].Field)
}

func TestResolveRefusesAnArchivedDocument(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	archived := h.doc("old.pdf", 10, "shipment")
	archived.Status = document.StatusArchived
	h.stored(archived)

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{archived.ID},
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Equal(t, "documentIds[0]", multiErr.Errors[0].Field)
}

func TestResolveRefusesADocumentThePersonCannotRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	statement := h.doc("statement.pdf", 10, "customer")
	h.stored(statement)
	h.allow(map[string]bool{"document": true, "shipment": true})

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{statement.ID},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestResolveRefusesAPersonWithoutDocumentRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	bol := h.doc("BOL.pdf", 10, "shipment")
	h.stored(bol)
	h.allow(map[string]bool{"shipment": true})

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{bol.ID},
	})

	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestResolveForTheSystemSkipsThePersonsAccess(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	eld := h.doc("eld.pdf", 10, "shipment")
	h.stored(eld)
	tenant := h.tenant
	tenant.UserID = pulid.Nil

	attachments, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  tenant,
		DocumentIDs: []pulid.ID{eld.ID},
		System:      true,
	})

	require.NoError(t, err)
	assert.Len(t, attachments, 1)
}

func TestResolveRefusesAPersonlessRequestThatIsNotTheSystem(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	eld := h.doc("eld.pdf", 10, "shipment")
	h.stored(eld)
	tenant := h.tenant
	tenant.UserID = pulid.Nil

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  tenant,
		DocumentIDs: []pulid.ID{eld.ID},
	})

	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestResolveRefusesFilesOverTheEmailBudget(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.doc("scan-1.pdf", emailports.MaxAttachmentBytes-100, "shipment")
	second := h.doc("scan-2.pdf", 101, "shipment")
	h.stored(first, second)
	h.allow(map[string]bool{"document": true, "shipment": true})

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: ids(first, second),
	})

	assert.Equal(t, "documentIds", fieldOf(t, err))
	assert.Contains(t, err.Error(), "7.5 MB")
}

func TestResolveAcceptsFilesExactlyAtTheBudget(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.doc("scan-1.pdf", emailports.MaxAttachmentBytes-100, "shipment")
	second := h.doc("scan-2.pdf", 100, "shipment")
	h.stored(first, second)
	h.allow(map[string]bool{"document": true, "shipment": true})

	attachments, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: ids(first, second),
	})

	require.NoError(t, err)
	assert.Len(t, attachments, 2)
}

func TestResolveRefusesTooManyFiles(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	requested := make([]pulid.ID, 0, documentports.MaxMailAttachments+1)
	for range documentports.MaxMailAttachments + 1 {
		requested = append(requested, pulid.MustNew("doc_"))
	}

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: requested,
	})

	assert.Equal(t, "documentIds", fieldOf(t, err))
}

func TestResolveWithRecipientsRecordsEachDisclosure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	eld := h.doc("eld.pdf", 10, "shipment")
	h.stored(eld)
	h.allow(map[string]bool{"document": true, "shipment": true})
	h.audit.EXPECT().LogAction(mock.MatchedBy(func(params *auditports.LogActionParams) bool {
		return params.ResourceID == eld.ID.String() &&
			params.Resource == permission.ResourceDocument &&
			params.Operation == permission.OpRead &&
			params.UserID == h.tenant.UserID
	}), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

	_, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  h.tenant,
		DocumentIDs: []pulid.ID{eld.ID},
		Recipients:  []string{"ap@freshhaul.com"},
	})

	require.NoError(t, err)
}

func TestResolveWithNoDocumentsReadsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	attachments, err := h.svc.Resolve(t.Context(), &documentports.ResolveMailAttachmentsRequest{
		TenantInfo: h.tenant,
	})

	require.NoError(t, err)
	assert.Empty(t, attachments)
}

func TestMailableListsSendableDocumentsOfReadableSources(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	shipmentID := pulid.MustNew("shp_").String()
	customerID := pulid.MustNew("cus_").String()
	bol := h.doc("BOL.pdf", 10, "shipment")
	archived := h.doc("old.pdf", 10, "shipment")
	archived.Status = document.StatusArchived
	h.allow(map[string]bool{"document": true, "shipment": true})
	h.repo.EXPECT().GetByResourceID(mock.Anything, mock.MatchedBy(
		func(req *documentports.GetDocumentsByResourceRequest) bool {
			return req.ResourceType == "shipment" && req.ResourceID == shipmentID &&
				req.IncludeDocumentType
		},
	)).Return([]*document.Document{bol, archived}, nil).Once()

	docs, err := h.svc.Mailable(t.Context(), &documentports.MailableDocumentsRequest{
		TenantInfo: h.tenant,
		Sources: []documentports.MailDocumentSource{
			{ResourceType: "shipment", ResourceID: shipmentID},
			{ResourceType: "customer", ResourceID: customerID},
		},
	})

	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, bol.ID, docs[0].ID)
}

func TestMailableWithoutDocumentReadListsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.allow(map[string]bool{"shipment": true})

	docs, err := h.svc.Mailable(t.Context(), &documentports.MailableDocumentsRequest{
		TenantInfo: h.tenant,
		Sources: []documentports.MailDocumentSource{
			{ResourceType: "shipment", ResourceID: "shp_1"},
		},
	})

	require.NoError(t, err)
	assert.Empty(t, docs)
}

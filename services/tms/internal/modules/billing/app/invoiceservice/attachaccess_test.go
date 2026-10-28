package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An invoice email goes to whatever address the draft names, so an
// attachment is a way to send a document out. A biller who cannot read
// drivers may not attach a driver's drug test result to one.
func TestRequireAttachable_HoldsAttachmentsToWhatTheSenderReads(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	actor := testutil.NewSessionActor(tenantInfo.UserID, tenantInfo.OrgID, tenantInfo.BuID)
	drugTest := &document.Document{
		ID:           pulid.MustNew("doc_"),
		ResourceType: "worker",
		OriginalName: "dot-drug-test.pdf",
	}
	pod := &document.Document{
		ID:           pulid.MustNew("doc_"),
		ResourceType: "shipment",
		OriginalName: "pod.pdf",
	}

	biller := &Service{
		permissions: &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceWorker.String()},
		},
		documentService: &fakeInvoiceDocumentService{currentDocument: drugTest},
	}
	err := biller.requireAttachable(t.Context(), tenantInfo, actor, []pulid.ID{drugTest.ID})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	biller.documentService = &fakeInvoiceDocumentService{currentDocument: pod}
	require.NoError(t, biller.requireAttachable(t.Context(), tenantInfo, actor, []pulid.ID{pod.ID}))

	noDocuments := &Service{
		permissions: &mocks.DenyResourcesPermissionEngine{
			Denied: []string{permission.ResourceDocument.String()},
		},
		documentService: &fakeInvoiceDocumentService{currentDocument: pod},
	}
	err = noDocuments.requireAttachable(t.Context(), tenantInfo, actor, []pulid.ID{pod.ID})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	require.NoError(t, noDocuments.requireAttachable(t.Context(), tenantInfo, actor, nil),
		"clearing the attachments sends nothing")
}

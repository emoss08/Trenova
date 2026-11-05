package invoicesendservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func linkedPart() *invoiceports.InvoiceSendPlanPart {
	return &invoiceports.InvoiceSendPlanPart{
		PartNumber: 1,
		Links: []*invoiceports.InvoiceSendPlanDocumentLink{
			{DocumentID: pulid.MustNew("doc_"), FileName: "pod.pdf", SizeBytes: 1024},
		},
	}
}

func shareRequest() (*invoiceports.InvoiceSendRequest, pulid.ID, pulid.ID, pulid.ID) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	return &invoiceports.InvoiceSendRequest{
		InvoiceID:  pulid.MustNew("inv_"),
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}, orgID, buID, userID
}

func TestMaterializePartLinks_UsesTheConfiguredAddress(t *testing.T) {
	t.Parallel()

	req, orgID, buID, userID := shareRequest()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		CreateDocumentShareToken(mock.Anything, mock.Anything).
		Return(&invoice.DocumentShareToken{ID: pulid.MustNew("idst_")}, nil).
		Once()
	svc := &Service{repo: repo, webBaseURL: "https://app.example.com"}
	part := linkedPart()

	body, linked, err := svc.materializePartLinks(
		t.Context(),
		req,
		testutil.NewSessionActor(userID, orgID, buID),
		part,
	)

	require.NoError(t, err)
	require.Len(t, linked, 1)
	assert.True(t, strings.HasPrefix(
		part.Links[0].URL,
		"https://app.example.com/api/v1/billing/invoices/shared-documents/",
	))
	assert.Contains(t, body, part.Links[0].URL)
}

func TestMaterializePartLinks_MintsNothingWithoutAConfiguredAddress(t *testing.T) {
	t.Parallel()

	req, orgID, buID, userID := shareRequest()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	svc := &Service{repo: repo}
	part := linkedPart()

	_, linked, err := svc.materializePartLinks(
		t.Context(),
		req,
		testutil.NewSessionActor(userID, orgID, buID),
		part,
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "app.webBaseUrl")
	assert.Empty(t, linked)
	assert.Empty(t, part.Links[0].URL)
	repo.AssertNotCalled(t, "CreateDocumentShareToken", mock.Anything, mock.Anything)
}

func TestMaterializePartLinks_NeedsNoAddressWhenThereAreNoLinks(t *testing.T) {
	t.Parallel()

	req, orgID, buID, userID := shareRequest()
	svc := &Service{}

	body, linked, err := svc.materializePartLinks(
		t.Context(),
		req,
		testutil.NewSessionActor(userID, orgID, buID),
		&invoiceports.InvoiceSendPlanPart{PartNumber: 1},
	)

	require.NoError(t, err)
	assert.Empty(t, body)
	assert.Empty(t, linked)
}

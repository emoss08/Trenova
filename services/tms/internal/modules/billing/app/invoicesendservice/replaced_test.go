package invoicesendservice

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type sharedDocumentContent struct {
	documentports.InvoiceDocumentService
	reads int
}

func (f *sharedDocumentContent) GetDownloadContent(
	context.Context,
	documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	f.reads++
	return &documentports.DocumentContent{
		Document:    &document.Document{OriginalName: "INV-1.pdf"},
		Body:        io.NopCloser(strings.NewReader("%PDF")),
		ContentType: "application/pdf",
	}, nil
}

func adjustmentControls(
	t *testing.T,
	policy invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicy,
) *tenantportsmock.MockInvoiceAdjustmentControlRepository {
	t.Helper()
	repo := tenantportsmock.NewMockInvoiceAdjustmentControlRepository(t)
	repo.EXPECT().GetByOrgID(mock.Anything, mock.Anything).
		Return(&invoiceadjustmentcontrol.InvoiceAdjustmentControl{
			SupersededInvoiceVisibilityPolicy: policy,
		}, nil).
		Maybe()
	return repo
}

func TestHiddenAsReplaced_FollowsTheReplacedInvoicePolicy(t *testing.T) {
	t.Parallel()

	replaced := &invoice.Invoice{
		OrganizationID:        pulid.MustNew("org_"),
		SupersededByInvoiceID: pulid.MustNew("inv_"),
	}
	current := &invoice.Invoice{OrganizationID: pulid.MustNew("org_")}
	tests := []struct {
		name    string
		entity  *invoice.Invoice
		policy  invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicy
		reads   int
		refused bool
	}{
		{
			name:    "a replaced invoice when customers see current invoices only",
			entity:  replaced,
			policy:  invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally,
			reads:   1,
			refused: true,
		},
		{
			name:   "a replaced invoice when customers see replaced invoices too",
			entity: replaced,
			policy: invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally,
			reads:  1,
		},
		{
			name:   "a current invoice, without reading the policy",
			entity: current,
			policy: invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			controls := adjustmentControls(t, tt.policy)
			svc := &Service{adjustmentRepo: controls}

			hidden, err := svc.hiddenAsReplaced(t.Context(), tt.entity)

			require.NoError(t, err)
			assert.Equal(t, tt.refused, hidden)
			controls.AssertNumberOfCalls(t, "GetByOrgID", tt.reads)
		})
	}
}

func sharedReplacedInvoice(t *testing.T) *invoiceportsmock.MockInvoiceRepository {
	t.Helper()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().GetDocumentShareToken(mock.Anything, mock.Anything).
		Return(&invoice.DocumentShareToken{
			ID:             pulid.MustNew("invdst_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			DocumentID:     pulid.MustNew("doc_"),
			ExpiresAt:      time.Now().Add(time.Hour).Unix(),
			Invoice: &invoice.Invoice{
				OrganizationID:        orgID,
				BusinessUnitID:        buID,
				SupersededByInvoiceID: pulid.MustNew("inv_"),
			},
		}, nil).
		Once()
	return repo
}

func TestDownloadSharedDocument_RefusesAReplacedInvoiceCustomersNoLongerSee(t *testing.T) {
	t.Parallel()

	documents := &sharedDocumentContent{}
	svc := &Service{
		repo: sharedReplacedInvoice(t),
		adjustmentRepo: adjustmentControls(
			t,
			invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally,
		),
		documentService: documents,
	}

	result, err := svc.DownloadSharedDocument(
		t.Context(),
		&invoiceports.DownloadInvoiceDocumentRequest{Token: "token"},
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsNotFoundError(err))
	assert.EqualError(
		t,
		err,
		"This invoice has been replaced. Ask the sender for the current invoice",
	)
	assert.Zero(t, documents.reads)
}

func TestDownloadSharedDocument_ServesAReplacedInvoiceCustomersStillSee(t *testing.T) {
	t.Parallel()

	repo := sharedReplacedInvoice(t)
	repo.EXPECT().UpdateDocumentShareToken(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			share *invoice.DocumentShareToken,
		) (*invoice.DocumentShareToken, error) {
			return share, nil
		}).
		Once()
	documents := &sharedDocumentContent{}
	svc := &Service{
		repo: repo,
		adjustmentRepo: adjustmentControls(
			t,
			invoiceadjustmentcontrol.SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally,
		),
		documentService: documents,
	}

	result, err := svc.DownloadSharedDocument(
		t.Context(),
		&invoiceports.DownloadInvoiceDocumentRequest{Token: "token"},
	)

	require.NoError(t, err)
	assert.Equal(t, "INV-1.pdf", result.FileName)
	assert.Equal(t, []byte("%PDF"), result.Body)
	assert.Equal(t, 1, documents.reads)
}

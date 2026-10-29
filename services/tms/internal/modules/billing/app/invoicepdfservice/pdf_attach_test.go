package invoicepdfservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAttachGeneratedPDFRecordsTheDocumentAndAnnouncesIt(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	draft := f.Invoice(invoice.StatusPosted)
	documentID := pulid.MustNew("doc_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.InvoiceID, TenantInfo: f.TenantInfo}).
		Return(draft, nil).
		Once()
	repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *invoice.Invoice) bool {
			return entity.PDFDocumentID == documentID
		})).
		RunAndReturn(func(_ context.Context, entity *invoice.Invoice) (*invoice.Invoice, error) {
			return entity, nil
		}).
		Once()

	svc, published := newAttachService(t, repo)
	updated, err := svc.AttachGeneratedPDF(t.Context(), &invoiceports.AttachGeneratedPDFRequest{
		InvoiceID:  f.InvoiceID,
		DocumentID: documentID,
		TenantInfo: f.TenantInfo,
		Actor:      f.AuditActor(),
	})

	require.NoError(t, err)
	assert.Equal(t, documentID, updated.PDFDocumentID)
	require.Len(t, *published, 1)
	assert.Equal(t, invoicerealtime.RealtimeActionPDFGenerated, (*published)[0].Action)
}

func newAttachService(
	t *testing.T,
	repo *invoiceportsmock.MockInvoiceRepository,
) (*Service, *[]*realtimeports.PublishResourceInvalidationRequest) {
	t.Helper()

	published := make([]*realtimeports.PublishResourceInvalidationRequest, 0, 1)
	realtime := realtimeportsmock.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *realtimeports.PublishResourceInvalidationRequest) error {
			published = append(published, req)
			return nil
		}).
		Maybe()

	return &Service{
		l:    zap.NewNop(),
		repo: repo,
		rec: invoiceaction.NewRecorder(invoiceaction.RecorderParams{
			Logger:       zap.NewNop(),
			AuditService: &mocks.NoopAuditService{},
			Realtime:     realtime,
		}),
	}, &published
}

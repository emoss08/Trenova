package invoiceservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newBackgroundService(
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

	svc := newVoidService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.realtime = realtime
	return svc, &published
}

func (f voidFixture) auditActor() kernelports.AuditActor {
	return kernelports.AuditActor{
		PrincipalType: f.actor.PrincipalType,
		PrincipalID:   f.userID,
		UserID:        f.userID,
	}
}

func TestAttachGeneratedPDFRecordsTheDocumentAndAnnouncesIt(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	draft := f.invoice(invoice.StatusPosted)
	documentID := pulid.MustNew("doc_")

	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.invoiceID, TenantInfo: f.tenantInfo}).
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

	svc, published := newBackgroundService(t, repo)
	updated, err := svc.AttachGeneratedPDF(t.Context(), &invoiceports.AttachGeneratedPDFRequest{
		InvoiceID:  f.invoiceID,
		DocumentID: documentID,
		TenantInfo: f.tenantInfo,
		Actor:      f.auditActor(),
	})

	require.NoError(t, err)
	assert.Equal(t, documentID, updated.PDFDocumentID)
	require.Len(t, *published, 1)
	assert.Equal(t, invoice.RealtimeActionPDFGenerated, (*published)[0].Action)
}

func TestReportAutoPostFailureAnnouncesTheReasonOnADraft(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.invoice(invoice.StatusDraft), nil).
		Once()

	svc, published := newBackgroundService(t, repo)
	err := svc.ReportAutoPostFailure(t.Context(), &invoiceports.ReportAutoPostFailureRequest{
		InvoiceID:  f.invoiceID,
		TenantInfo: f.tenantInfo,
		Reason:     "  The fiscal period is closed ",
		Actor:      f.auditActor(),
	})

	require.NoError(t, err)
	require.Len(t, *published, 1)
	event := (*published)[0]
	assert.Equal(t, invoice.RealtimeActionPostFailed, event.Action)
	snapshot, ok := event.Entity.(*invoice.RealtimeSnapshot)
	require.True(t, ok)
	assert.Equal(t, "The fiscal period is closed", snapshot.PostError)
	assert.True(
		t,
		snapshot.NotifiedUserID.IsNil(),
		"no notification service, so nobody was notified",
	)
}

func TestReportAutoPostFailureStaysQuietOnceTheInvoiceMovedOn(t *testing.T) {
	t.Parallel()

	for _, status := range []invoice.Status{invoice.StatusPosted, invoice.StatusVoided} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			f := newVoidFixture()
			repo := invoiceportsmock.NewMockInvoiceRepository(t)
			repo.EXPECT().
				GetByID(mock.Anything, mock.Anything).
				Return(f.invoice(status), nil).
				Once()

			svc, published := newBackgroundService(t, repo)
			err := svc.ReportAutoPostFailure(
				t.Context(),
				&invoiceports.ReportAutoPostFailureRequest{
					InvoiceID:  f.invoiceID,
					TenantInfo: f.tenantInfo,
					Reason:     "closed",
					Actor:      f.auditActor(),
				},
			)

			require.NoError(t, err)
			assert.Empty(t, *published)
		})
	}
}

func TestReportAutoPostFailureNeverFailsWhenTheInvoiceCannotBeRead(t *testing.T) {
	t.Parallel()

	f := newVoidFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Invoice not found")).
		Once()

	svc, published := newBackgroundService(t, repo)
	err := svc.ReportAutoPostFailure(t.Context(), &invoiceports.ReportAutoPostFailureRequest{
		InvoiceID:  f.invoiceID,
		TenantInfo: f.tenantInfo,
		Actor:      f.auditActor(),
	})

	require.NoError(t, err)
	assert.Empty(t, *published)
}

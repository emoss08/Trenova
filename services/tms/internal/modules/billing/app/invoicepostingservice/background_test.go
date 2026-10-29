package invoicepostingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/billingqueueportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/realtimeportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

	svc := newPostingTestService(repo, billingqueueportsmock.NewMockBillingQueueRepository(t))
	svc.realtime = realtime
	svc.rec = invoiceaction.NewRecorder(invoiceaction.RecorderParams{
		Logger:       zap.NewNop(),
		AuditService: &mocks.NoopAuditService{},
		Realtime:     realtime,
	})
	return svc, &published
}

// newPostingTestService is a posting service over mocked repositories that
// never touches a connection.
func newPostingTestService(
	repo *invoiceportsmock.MockInvoiceRepository,
	queueRepo *billingqueueportsmock.MockBillingQueueRepository,
) *Service {
	return &Service{
		l:                zap.NewNop(),
		db:               invoiceservicetest.FakeTx{},
		repo:             repo,
		billingQueueRepo: queueRepo,
		validator: invoicevalidation.NewValidator(
			invoicevalidation.ValidatorParams{Logger: zap.NewNop()},
		),
		sequenceGenerator: testutil.TestSequenceGenerator{SingleValue: "INV-2002"},
		auditService:      &mocks.NoopAuditService{},
		realtime:          &mocks.NoopRealtimeService{},
		rec: invoiceaction.NewRecorder(invoiceaction.RecorderParams{
			Logger:       zap.NewNop(),
			AuditService: &mocks.NoopAuditService{},
			Realtime:     &mocks.NoopRealtimeService{},
		}),
	}
}

func TestReportAutoPostFailureAnnouncesTheReasonOnADraft(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(f.Invoice(invoice.StatusDraft), nil).
		Once()

	svc, published := newBackgroundService(t, repo)
	err := svc.ReportAutoPostFailure(t.Context(), &invoiceports.ReportAutoPostFailureRequest{
		InvoiceID:  f.InvoiceID,
		TenantInfo: f.TenantInfo,
		Reason:     "  The fiscal period is closed ",
		Actor:      f.AuditActor(),
	})

	require.NoError(t, err)
	require.Len(t, *published, 1)
	event := (*published)[0]
	assert.Equal(t, invoicerealtime.RealtimeActionPostFailed, event.Action)
	snapshot, ok := event.Entity.(*invoicerealtime.RealtimeSnapshot)
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

			f := invoiceservicetest.NewFixture()
			repo := invoiceportsmock.NewMockInvoiceRepository(t)
			repo.EXPECT().
				GetByID(mock.Anything, mock.Anything).
				Return(f.Invoice(status), nil).
				Once()

			svc, published := newBackgroundService(t, repo)
			err := svc.ReportAutoPostFailure(
				t.Context(),
				&invoiceports.ReportAutoPostFailureRequest{
					InvoiceID:  f.InvoiceID,
					TenantInfo: f.TenantInfo,
					Reason:     "closed",
					Actor:      f.AuditActor(),
				},
			)

			require.NoError(t, err)
			assert.Empty(t, *published)
		})
	}
}

func TestReportAutoPostFailureNeverFailsWhenTheInvoiceCannotBeRead(t *testing.T) {
	t.Parallel()

	f := invoiceservicetest.NewFixture()
	repo := invoiceportsmock.NewMockInvoiceRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Invoice not found")).
		Once()

	svc, published := newBackgroundService(t, repo)
	err := svc.ReportAutoPostFailure(t.Context(), &invoiceports.ReportAutoPostFailureRequest{
		InvoiceID:  f.InvoiceID,
		TenantInfo: f.TenantInfo,
		Actor:      f.AuditActor(),
	})

	require.NoError(t, err)
	assert.Empty(t, *published)
}

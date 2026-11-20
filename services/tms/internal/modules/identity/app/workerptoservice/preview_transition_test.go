package workerptoservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
	"github.com/emoss08/trenova/internal/testutil/notificationtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func quietDriverNotify(
	t *testing.T,
	workers workerports.WorkerRepository,
) *drivernotificationservice.Service {
	t.Helper()

	templates := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&documenttemplateports.RenderedMessage{Subject: "PTO", Text: "Your PTO changed."}, nil).
		Maybe()

	return drivernotificationservice.New(drivernotificationservice.Params{
		Logger:        zap.NewNop(),
		WorkerRepo:    workers,
		Notifications: notificationtest.Discarding(),
		Templates:     templates,
	})
}

// UpdateStatus fails until the preview has run, and nothing is sent while
// it runs, which is what holds the preview to reading.
func TestPreviewReject_IsWhatRejectSaves(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusRequested)
	var saved *worker.WorkerPTO
	writable := false
	workers := workerportsmock.NewMockWorkerRepository(t)
	workers.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{ID: current.WorkerID}, nil)

	svc := &Service{
		db:            dbtest.NopConnection{},
		realtime:      &mocks.NoopRealtimeService{},
		ledger:        untrackedLedger(),
		organizations: utcOrganizations{},
		l:             zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(context.Context, *workerptoports.GetPTOByIDRequest) (*worker.WorkerPTO, error) {
				copied := *current
				return &copied, nil
			},
			updateStatus: func(
				_ context.Context,
				req *workerptoports.UpdatePTOStatusRequest,
			) (*worker.WorkerPTO, error) {
				if !writable {
					return nil, errors.New("a preview wrote")
				}
				saved = applyStatus(req, current)
				return saved, nil
			},
		},
		auditService: &fakeAuditService{},
		driverNotify: quietDriverNotify(t, workers),
	}
	request := func() *workerptoports.UpdatePTOStatusRequest {
		return &workerptoports.UpdatePTOStatusRequest{
			ID:         current.ID,
			TenantInfo: tenantFor(current),
			UserID:     pulid.MustNew("usr_"),
			Reason:     "  Coverage gap on that route  ",
		}
	}
	req := request()

	preview, err := svc.PreviewReject(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "  Coverage gap on that route  ", req.Reason,
		"a preview must not rewrite the caller's request")

	writable = true
	final := request()
	final.UserID = req.UserID
	_, err = svc.Reject(t.Context(), final)
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, worker.PTOStatusRequested, preview.Before.Status)
	assert.Equal(t, saved.Status, preview.After.Status)
	assert.Equal(t, saved.RejectorID, preview.After.RejectorID)
	assert.Equal(t, saved.RejectionReason, preview.After.RejectionReason)
	assert.False(t, preview.ReturnsLedger)
}

func TestPreviewCancel_SaysApprovedDaysGoBack(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusApproved)
	workers := workerportsmock.NewMockWorkerRepository(t)
	workers.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{ID: current.WorkerID}, nil)
	svc := &Service{
		db:            dbtest.NopConnection{},
		realtime:      &mocks.NoopRealtimeService{},
		ledger:        untrackedLedger(),
		organizations: utcOrganizations{},
		l:             zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(context.Context, *workerptoports.GetPTOByIDRequest) (*worker.WorkerPTO, error) {
				return current, nil
			},
		},
		driverNotify: quietDriverNotify(t, workers),
	}

	preview, err := svc.PreviewCancel(t.Context(), &workerptoports.UpdatePTOStatusRequest{
		ID:         current.ID,
		TenantInfo: tenantFor(current),
		UserID:     pulid.MustNew("usr_"),
		Reason:     "Route covered",
	})
	require.NoError(t, err)
	assert.True(t, preview.ReturnsLedger)
	assert.Equal(t, worker.PTOStatusCancelled, preview.After.Status)
	assert.Equal(t, "Route covered", preview.After.CancellationReason)
}

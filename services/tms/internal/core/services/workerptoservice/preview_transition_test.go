package workerptoservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/smsjobs"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type recordingStarter struct {
	serviceports.WorkflowStarter

	texts []*smsjobs.SendSMSPayload
}

func (r *recordingStarter) Enabled() bool { return true }

func (r *recordingStarter) StartWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	payload, ok := args[0].(*smsjobs.SendSMSPayload)
	if !ok {
		return nil, errors.New("not an SMS")
	}
	r.texts = append(r.texts, payload)

	return nil, nil
}

// UpdateStatus fails until the preview has run, and nothing is texted while
// it runs, which is what holds the preview to reading.
func TestPreviewReject_IsWhatRejectSavesAndTexts(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusRequested)
	var saved *worker.WorkerPTO
	writable := false
	starter := &recordingStarter{}
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.User{Name: "Dana Ortiz"}, nil)
	workers := mocks.NewMockWorkerRepository(t)
	workers.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{ID: current.WorkerID, PhoneNumber: "+15555550100"}, nil)

	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(context.Context, *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error) {
				copied := *current
				return &copied, nil
			},
			updateStatus: func(
				_ context.Context,
				req *repositories.UpdatePTOStatusRequest,
			) (*worker.WorkerPTO, error) {
				if !writable {
					return nil, errors.New("a preview wrote")
				}
				saved = applyStatus(req, current)
				return saved, nil
			},
		},
		userRepo:        users,
		workerRepo:      workers,
		workflowStarter: starter,
		auditService:    &fakeAuditService{},
	}
	request := func() *repositories.UpdatePTOStatusRequest {
		return &repositories.UpdatePTOStatusRequest{
			ID:         current.ID,
			TenantInfo: tenantFor(current),
			UserID:     pulid.MustNew("usr_"),
			Reason:     "  Coverage gap on that route  ",
		}
	}
	req := request()

	preview, err := svc.PreviewReject(t.Context(), req)
	require.NoError(t, err)
	require.Empty(t, starter.texts, "a preview must not text the driver")
	assert.Equal(t, "  Coverage gap on that route  ", req.Reason,
		"a preview must not rewrite the caller's request")

	writable = true
	final := request()
	final.UserID = req.UserID
	_, err = svc.Reject(t.Context(), final)
	require.NoError(t, err)
	require.NotNil(t, saved)
	require.Len(t, starter.texts, 1)

	assert.Equal(t, worker.PTOStatusRequested, preview.Before.Status)
	assert.Equal(t, saved.Status, preview.After.Status)
	assert.Equal(t, saved.RejectorID, preview.After.RejectorID)
	assert.Equal(t, saved.RejectionReason, preview.After.RejectionReason)
	assert.False(t, preview.ReturnsLedger)
	require.NotNil(t, preview.SMS)
	assert.Equal(t, starter.texts[0].PhoneNumber, preview.SMS.PhoneNumber)
	assert.Equal(t, starter.texts[0].Message, preview.SMS.Message)
}

func TestPreviewCancel_SaysApprovedDaysGoBack(t *testing.T) {
	t.Parallel()

	current := storedPTO(worker.PTOStatusApproved)
	svc := &Service{
		l: zap.NewNop(),
		repo: &fakePTORepo{
			getByID: func(context.Context, *repositories.GetPTOByIDRequest) (*worker.WorkerPTO, error) {
				return current, nil
			},
		},
	}

	preview, err := svc.PreviewCancel(t.Context(), &repositories.UpdatePTOStatusRequest{
		ID:         current.ID,
		TenantInfo: tenantFor(current),
		UserID:     pulid.MustNew("usr_"),
		Reason:     "Route covered",
	})
	require.NoError(t, err)
	assert.True(t, preview.ReturnsLedger)
	assert.Equal(t, worker.PTOStatusCancelled, preview.After.Status)
	assert.Equal(t, "Route covered", preview.After.CancellationReason)
	assert.Nil(t, preview.SMS)
}

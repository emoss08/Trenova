package emailservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type sendHarness struct {
	svc       *Service
	repo      *mocks.MockEmailRepository
	workflows *mocks.MockWorkflowStarter
	tenant    pagination.TenantInfo
	profile   *email.Profile
}

func newSendHarness(t *testing.T) *sendHarness {
	t.Helper()

	tenant := testTenantInfo()
	profile := &email.Profile{
		ID:             pulid.MustNew("emlprof_"),
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
		Provider:       email.ProviderResend,
		SenderEmail:    "dispatch@example.com",
		SenderName:     "Dispatch",
		Status:         email.ProfileStatusActive,
	}

	repo := mocks.NewMockEmailRepository(t)
	repo.EXPECT().GetAssignedProfile(mock.Anything, tenant, email.PurposeGeneral).
		Return(profile, nil).Maybe()
	repo.EXPECT().HasSuppression(mock.Anything, tenant, mock.Anything).Return(false, nil).Maybe()

	workflows := mocks.NewMockWorkflowStarter(t)
	workflows.EXPECT().Enabled().Return(true).Maybe()

	return &sendHarness{
		svc: &Service{
			l:               zap.NewNop(),
			repo:            repo,
			validator:       NewValidator(),
			workflowStarter: workflows,
		},
		repo:      repo,
		workflows: workflows,
		tenant:    tenant,
		profile:   profile,
	}
}

func (h *sendHarness) request(to ...string) *services.SendEmailRequest {
	return &services.SendEmailRequest{
		TenantInfo:     h.tenant,
		Purpose:        email.PurposeGeneral,
		To:             to,
		Subject:        "Load offer",
		HTML:           "<p>offer</p>",
		Text:           "offer",
		IdempotencyKey: "tender-offer-tof_1",
	}
}

func (h *sendHarness) stored(status email.MessageStatus, to ...string) *email.Message {
	return &email.Message{
		ID:             pulid.MustNew("emlmsg_"),
		BusinessUnitID: h.tenant.BuID,
		OrganizationID: h.tenant.OrgID,
		ProfileID:      h.profile.ID,
		Purpose:        email.PurposeGeneral,
		Provider:       h.profile.Provider,
		IdempotencyKey: "tender-offer-tof_1",
		Status:         status,
		ToRecipients:   to,
	}
}

func TestSend_ReplayOfASentMessageReturnsItWithoutSendingAgain(t *testing.T) {
	t.Parallel()

	h := newSendHarness(t)
	stored := h.stored(email.MessageStatusSent, "carrier@example.com")
	h.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		Return(stored, false, nil).Once()

	msg, err := h.svc.Send(t.Context(), h.request("Carrier@Example.com"))

	require.NoError(t, err)
	assert.Equal(t, stored.ID, msg.ID)
	assert.True(t, msg.Replayed)
	h.workflows.AssertNotCalled(t, "StartWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSend_ReplayOfAQueuedMessageRestartsItsSendWorkflow(t *testing.T) {
	t.Parallel()

	h := newSendHarness(t)
	stored := h.stored(email.MessageStatusQueued, "carrier@example.com")
	h.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		Return(stored, false, nil).Once()
	h.workflows.EXPECT().StartWorkflow(mock.Anything, mock.MatchedBy(
		func(options client.StartWorkflowOptions) bool {
			return options.ID == "email-send-"+h.tenant.OrgID.String()+"-"+
				h.tenant.BuID.String()+"-"+stored.ID.String()
		},
	), mock.Anything, mock.Anything).Return(nil, nil).Once()

	msg, err := h.svc.Send(t.Context(), h.request("carrier@example.com"))

	require.NoError(t, err)
	assert.True(t, msg.Replayed)
}

func TestSend_ReplayOfAFailedMessageRequeuesAndSendsIt(t *testing.T) {
	t.Parallel()

	h := newSendHarness(t)
	stored := h.stored(email.MessageStatusFailed, "carrier@example.com")
	stored.LastError = "workflow start failed"
	stored.FailedAt = 100
	h.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		Return(stored, false, nil).Once()
	h.repo.EXPECT().UpdateMessage(mock.Anything, mock.MatchedBy(func(msg *email.Message) bool {
		return msg.Status == email.MessageStatusQueued && msg.LastError == "" && msg.FailedAt == 0
	})).RunAndReturn(func(_ context.Context, msg *email.Message) (*email.Message, error) {
		return msg, nil
	}).Once()
	h.workflows.EXPECT().StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).Once()

	msg, err := h.svc.Send(t.Context(), h.request("carrier@example.com"))

	require.NoError(t, err)
	assert.Equal(t, email.MessageStatusQueued, msg.Status)
	assert.True(t, msg.Replayed)
}

func TestSend_ReplayToDifferentRecipientsIsRefused(t *testing.T) {
	t.Parallel()

	h := newSendHarness(t)
	h.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		Return(h.stored(email.MessageStatusSent, "carrier@example.com"), false, nil).Once()

	_, err := h.svc.Send(t.Context(), h.request("someone-else@example.com"))

	var business *errortypes.BusinessError
	require.True(t, errors.As(err, &business))
}

func TestSend_FirstSendStartsTheWorkflow(t *testing.T) {
	t.Parallel()

	h := newSendHarness(t)
	h.repo.EXPECT().CreateMessageOnce(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, msg *email.Message) (*email.Message, bool, error) {
			msg.ID = pulid.MustNew("emlmsg_")
			return msg, true, nil
		}).Once()
	h.workflows.EXPECT().StartWorkflow(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil).Once()

	msg, err := h.svc.Send(t.Context(), h.request("carrier@example.com"))

	require.NoError(t, err)
	assert.False(t, msg.Replayed)
	assert.Equal(t, "tender-offer-tof_1", msg.IdempotencyKey)
}

var _ repositories.EmailRepository = (*mocks.MockEmailRepository)(nil)

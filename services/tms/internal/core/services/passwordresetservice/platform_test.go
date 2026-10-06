package passwordresetservice

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type platformHarness struct {
	svc       *Service
	plans     *mocks.MockPlanService
	platform  *mocks.MockPlatformEmailService
	tenantMsg *mocks.MockEmailService
	user      *tenant.User
	rendered  []*services.RenderMessageRequest
}

func newPlatformHarness(t *testing.T) *platformHarness {
	t.Helper()

	user := activeUser()
	svc := newService(t, &stubUserRepo{user: user}, &stubTokenRepo{}, &stubSessionRepo{})
	h := &platformHarness{
		svc:       svc,
		plans:     mocks.NewMockPlanService(t),
		platform:  mocks.NewMockPlatformEmailService(t),
		tenantMsg: mocks.NewMockEmailService(t),
		user:      user,
	}

	templates := mocks.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().RenderMessage(mock.Anything, mock.MatchedBy(func(req *services.RenderMessageRequest) bool {
		h.rendered = append(h.rendered, req)
		return true
	})).Return(&services.RenderedMessage{
		Subject: "Reset your password",
		HTML:    "<p>reset</p>",
		Text:    "reset",
	}, nil).Maybe()
	orgs := mocks.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.Organization{Name: "Acme"}, nil).Maybe()

	svc.templates = templates
	svc.orgs = orgs
	svc.emailService = h.tenantMsg
	svc.plans = h.plans
	svc.platform = h.platform

	return h
}

func TestResetEmailForAManagedOrganizationUsesThePlatformSender(t *testing.T) {
	t.Parallel()

	h := newPlatformHarness(t)
	now := time.Now().Unix()
	h.plans.EXPECT().IsCloud().Return(true)
	h.plans.EXPECT().Resolve(mock.Anything, h.user.CurrentOrganizationID, h.user.BusinessUnitID).Return(
		platformplan.NewManaged(platformplan.Unlimited(), &subscription.Subscription{
			OrganizationID: h.user.CurrentOrganizationID,
			BusinessUnitID: h.user.BusinessUnitID,
			Status:         subscription.StatusTrialing,
			TrialEndsAt:    now + 3_600,
			ReadOnlyUntil:  now + 7_200,
		}, now), nil,
	)
	h.platform.EXPECT().SendPasswordReset(mock.Anything, mock.MatchedBy(func(msg *services.PasswordResetEmail) bool {
		return msg.To == h.user.EmailAddress &&
			msg.Name == h.user.Name &&
			msg.CompanyName == "Acme" &&
			strings.Contains(msg.ResetURL, "/auth/reset?token=") &&
			msg.ExpiresInMinutes > 0 &&
			msg.ExpiresAt > now &&
			strings.HasPrefix(msg.IdempotencyKey, "password-reset-")
	})).Return(nil)

	require.NoError(t, h.svc.RequestReset(t.Context(), h.user.EmailAddress))
	assert.Empty(t, h.rendered, "a platform-sent email must never render tenant-authored content")
}

func TestResetEmailForAnUnmanagedOrganizationUsesTheTenantSender(t *testing.T) {
	t.Parallel()

	h := newPlatformHarness(t)
	h.plans.EXPECT().IsCloud().Return(true)
	h.plans.EXPECT().Resolve(mock.Anything, mock.Anything, mock.Anything).Return(
		platformplan.NewUnmanaged(
			platformplan.Unlimited(),
			platformplan.OriginInternal,
			h.user.CurrentOrganizationID,
			h.user.BusinessUnitID,
			time.Now().Unix(),
		), nil,
	)
	h.tenantMsg.EXPECT().Send(mock.Anything, mock.Anything).Return(nil, nil)

	require.NoError(t, h.svc.RequestReset(t.Context(), h.user.EmailAddress))
	assert.Empty(t, h.platform.Calls)
	require.Len(t, h.rendered, 1)
}

func TestResetEmailOutsideCloudUsesTheTenantSender(t *testing.T) {
	t.Parallel()

	h := newPlatformHarness(t)
	h.plans.EXPECT().IsCloud().Return(false)
	h.tenantMsg.EXPECT().Send(mock.Anything, mock.Anything).Return(nil, nil)

	require.NoError(t, h.svc.RequestReset(t.Context(), h.user.EmailAddress))
}

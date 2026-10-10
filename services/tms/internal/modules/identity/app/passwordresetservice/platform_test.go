package passwordresetservice

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/mocks/documenttemplateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/subscription"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type platformHarness struct {
	svc       *Service
	plans     *platformplanportsmock.MockPlanService
	platform  *kernelportsmock.MockPlatformEmailService
	tenantMsg *emailportsmock.MockEmailService
	user      *tenant.User
	rendered  []*documenttemplateports.RenderMessageRequest
}

func newPlatformHarness(t *testing.T) *platformHarness {
	t.Helper()

	user := activeUser()
	svc := newService(t, &stubUserRepo{user: user}, &stubTokenRepo{}, &stubSessionRepo{})
	h := &platformHarness{
		svc:       svc,
		plans:     platformplanportsmock.NewMockPlanService(t),
		platform:  kernelportsmock.NewMockPlatformEmailService(t),
		tenantMsg: emailportsmock.NewMockEmailService(t),
		user:      user,
	}

	templates := documenttemplateportsmock.NewMockDocumentTemplateResolver(t)
	templates.EXPECT().RenderMessage(mock.Anything, mock.MatchedBy(func(req *documenttemplateports.RenderMessageRequest) bool {
		h.rendered = append(h.rendered, req)
		return true
	})).Return(&documenttemplateports.RenderedMessage{
		Subject: "Reset your password",
		HTML:    "<p>reset</p>",
		Text:    "reset",
	}, nil).Maybe()
	orgs := tenantportsmock.NewMockOrganizationRepository(t)
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
	h.plans.EXPECT().EnforcesPlans().Return(true)
	h.plans.EXPECT().Resolve(mock.Anything, h.user.CurrentOrganizationID, h.user.BusinessUnitID).Return(
		platformplan.NewManaged(platformplan.Unlimited(), &subscription.Subscription{
			OrganizationID: h.user.CurrentOrganizationID,
			BusinessUnitID: h.user.BusinessUnitID,
			Status:         subscription.StatusTrialing,
			TrialEndsAt:    now + 3_600,
			ReadOnlyUntil:  now + 7_200,
		}, now), nil,
	)
	h.platform.EXPECT().SendPasswordReset(mock.Anything, mock.MatchedBy(func(msg *kernelports.PasswordResetEmail) bool {
		return msg.To == h.user.EmailAddress &&
			msg.Locale == i18n.Locale(h.user.Locale) &&
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
	h.plans.EXPECT().EnforcesPlans().Return(true)
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
	h.plans.EXPECT().EnforcesPlans().Return(false)
	h.tenantMsg.EXPECT().Send(mock.Anything, mock.Anything).Return(nil, nil)

	require.NoError(t, h.svc.RequestReset(t.Context(), h.user.EmailAddress))
}

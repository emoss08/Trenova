package emailservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/testutil/mocks/emailportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/webhooksig"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type webhookSenderStub struct {
	ProviderSender
}

func (webhookSenderStub) IntegrationType() integration.Type {
	return integration.TypePostmark
}

func TestResolveTenantByWebhookToken_AnUnconfiguredSecretAdmitsNobody(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	repo := emailportsmock.NewMockEmailRepository(t)
	repo.EXPECT().
		GetEmailWebhookConfig(mock.Anything, emailports.GetEmailWebhookConfigRequest{
			IntegrationType: integration.TypePostmark,
			Token:           "a-token",
		}).
		Return(&emailports.EmailWebhookConfig{TenantInfo: tenantInfo}, nil).
		Once()
	svc := &Service{
		repo: repo,
		providerSenders: map[email.Provider]ProviderSender{
			email.ProviderPostmark: webhookSenderStub{},
		},
	}

	got, secret, err := svc.ResolveTenantByWebhookToken(
		t.Context(),
		email.ProviderPostmark,
		"a-token",
	)

	require.NoError(t, err)
	require.Equal(t, tenantInfo, got)
	require.Empty(t, secret)
	require.ErrorIs(
		t,
		webhooksig.VerifyBasicAuth(secret, "Basic cG9zdG1hcms6cGFzcw=="),
		webhooksig.ErrMissingHeaders,
	)
}

func TestResolveTenantByWebhookToken_RefusesAProviderWithNoSender(t *testing.T) {
	t.Parallel()

	svc := &Service{repo: emailportsmock.NewMockEmailRepository(t)}

	_, _, err := svc.ResolveTenantByWebhookToken(t.Context(), email.ProviderPostmark, "a-token")

	require.Error(t, err)
}

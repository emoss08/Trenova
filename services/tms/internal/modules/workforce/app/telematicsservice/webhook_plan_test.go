package telematicsservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type webhookConfigs struct {
	telematicsports.TelematicsRepository

	config *telematicsports.TelematicsWebhookConfig
}

func (w webhookConfigs) GetWebhookConfigByToken(
	context.Context,
	integration.Type,
	string,
) (*telematicsports.TelematicsWebhookConfig, error) {
	return w.config, nil
}

func TestASamsaraWebhookIsTakenOnlyForAnOrganizationWithSamsara(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := &Service{repo: webhookConfigs{config: &telematicsports.TelematicsWebhookConfig{
		TenantInfo: tenant,
	}}}
	req := &ProcessWebhookRequest{ProviderType: integration.TypeSamsara, Token: "hook"}

	svc.features = plantest.SharedServer().
		Without(tenant.OrgID, platformcatalog.FeatureSamsaraIntegration)
	err := svc.ProcessWebhook(t.Context(), req)
	require.True(
		t,
		errortypes.IsPlanRestrictionError(err),
		"refused before anything is read: %v",
		err,
	)

	svc.features = plantest.SharedServer().
		Planned(tenant.OrgID).
		Without(pulid.MustNew("org_"), platformcatalog.FeatureSamsaraIntegration)
	err = svc.ProcessWebhook(t.Context(), req)
	require.Error(t, err)
	assert.False(t, errortypes.IsPlanRestrictionError(err),
		"another organization's plan does not count; it goes on to check the secret")
	assert.Contains(t, err.Error(), "secret is not configured")
}

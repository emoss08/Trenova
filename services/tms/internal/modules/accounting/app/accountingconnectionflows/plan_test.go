package accountingconnectionflows

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type switchedPlan struct {
	mu   sync.Mutex
	gate platformcatalogports.FeatureGate
}

func (p *switchedPlan) use(gate platformcatalogports.FeatureGate) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gate = gate
}

func (p *switchedPlan) current() platformcatalogports.FeatureGate {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gate
}

func (p *switchedPlan) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return p.current().Enabled(ctx, feature)
}

func (p *switchedPlan) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return p.current().EnabledAnywhere(ctx, feature)
}

func TestAWebhookIsRecordedOnlyForAnOrganizationWithAccounting(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conn := h.connect(t)
	h.connector.webhookRealms = []string{testRealm}
	webhook := &accountingconnectionports.ReceiveAccountingWebhookRequest{
		IntegrationType: integration.TypeQuickBooksOnline,
		Signature:       instanceVerifier,
		Body:            []byte("[]"),
	}

	h.plan.use(
		plantest.SharedServer().Without(conn.OrganizationID, platformcatalog.FeatureAccounting),
	)
	require.NoError(t, h.svc.webhooks.ReceiveWebhook(t.Context(), webhook),
		"answered like a webhook for no connection, so the provider does not retry")
	assert.Nil(t, h.connections.rows[conn.ID].LastWebhookAt, "nothing is recorded")

	h.plan.use(
		plantest.SharedServer().
			Planned(conn.OrganizationID).
			Without(pulid.MustNew("org_"), platformcatalog.FeatureAccounting),
	)
	require.NoError(t, h.svc.webhooks.ReceiveWebhook(t.Context(), webhook))
	assert.NotNil(t, h.connections.rows[conn.ID].LastWebhookAt,
		"another organization's plan does not count")
}

func TestConnectingNeedsOnlyTheBaseAccountingIntegration(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.plan.use(plantest.Without(platformcatalog.FeatureAccounting))
	_ = h.start(t)

	h.plan.use(plantest.Without(platformcatalog.FeatureAccountingIntegration))
	_, err := h.svc.auth.StartAuthorization(
		t.Context(),
		&accountingconnectionports.StartAccountingAuthorizationRequest{
			TenantInfo:      h.tenant,
			UserID:          h.userID,
			IntegrationType: integration.TypeQuickBooksOnline,
		},
	)
	require.Error(t, err)
}

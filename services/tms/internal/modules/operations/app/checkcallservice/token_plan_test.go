package checkcallservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestACarrierLinkAnswersOnlyForAnOrganizationWithCustomerTracking(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seedLink(h)

	h.svc.features = plantest.SharedServer().
		Without(h.tenant.OrgID, platformcatalog.FeatureCustomerTracking)
	_, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "This tracking link is no longer valid",
		"the same answer as an unknown link")

	h.svc.features = plantest.SharedServer().Planned(h.tenant.OrgID).Without(
		pulid.MustNew("org_"),
		platformcatalog.FeatureCustomerTracking,
	)
	view, err := h.svc.PreviewByToken(t.Context(), carrierBoundToken)
	require.NoError(t, err, "another organization's plan does not count")
	assert.NotNil(t, view)
}

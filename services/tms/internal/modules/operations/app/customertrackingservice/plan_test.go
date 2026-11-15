package customertrackingservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestTrackingLinksAndPreviewsAreRefusedWithoutCustomerTracking(t *testing.T) {
	t.Parallel()

	h := newHarness()
	h.service.features = plantest.Without(platformcatalog.FeatureCustomerTracking)
	tenant := plantest.Tenant()
	shipmentID := pulid.MustNew("shp_")

	_, err := h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: tenant,
		ShipmentID: shipmentID,
		IssuedByID: tenant.UserID,
	})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "issue a link")

	_, err = h.service.PreviewIssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: tenant,
		ShipmentID: shipmentID,
	})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "preview a link")

	_, err = h.service.RevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
		TenantInfo: tenant,
		ShipmentID: shipmentID,
	})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "revoke links")

	_, err = h.service.PreviewRevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
		TenantInfo: tenant,
		ShipmentID: shipmentID,
	})
	assert.True(t, errortypes.IsPlanRestrictionError(err), "preview a revoke")

	_, err = h.service.ListLinks(t.Context(), tenant, shipmentID)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "list links")

	_, err = h.service.ListNotifications(t.Context(), tenant, shipmentID)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "list status updates")

	_, err = h.service.PreviewForShipment(t.Context(), tenant, shipmentID)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "preview what the customer sees")

	assert.Empty(t, h.links.rows, "no link is minted off the plan")
}

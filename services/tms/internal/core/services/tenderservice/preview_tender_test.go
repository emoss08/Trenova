package tenderservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPreviewWaterfall_IsTheTenderCreateWaterfallStarts(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	blocked := blockedCarrier("lapsed")
	warned := warnedCarrier("expiring")
	healthy := eligibleCarrier("acme")
	deps.guideRepo.matched = &tender.RoutingGuide{
		ID:     pulid.MustNew("rgd_"),
		Name:   "Dallas outbound",
		Status: "Active",
		Entries: []*tender.RoutingGuideEntry{
			guideEntry(blocked.ID, 1),
			guideEntry(warned.ID, 2),
			guideEntry(healthy.ID, 3),
		},
	}
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{blocked, warned, healthy}, nil)
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Return(&notification.Notification{}, nil).Maybe()
	request := &CreateWaterfallTenderRequest{TenantInfo: tenantInfo, ShipmentMoveID: move.ID}

	preview, err := deps.svc.PreviewWaterfall(t.Context(), request)
	require.NoError(t, err)
	assert.Nil(t, deps.tenderRepo.created, "a preview must not save a tender")
	assert.Empty(t, deps.workflows.started, "a preview must not start a workflow")
	assert.Empty(t, deps.events.events, "a preview must not record events")

	created, err := deps.svc.CreateWaterfall(t.Context(), request)
	require.NoError(t, err)

	require.Len(t, preview.Tender.Offers, len(created.Offers))
	for i := range created.Offers {
		assert.Equal(t, created.Offers[i].CarrierID, preview.Tender.Offers[i].CarrierID)
		assert.True(t, created.Offers[i].Rate.Equal(preview.Tender.Offers[i].Rate))
		assert.Equal(t, created.Offers[i].Channel, preview.Tender.Offers[i].Channel)
		assert.Equal(t, created.Offers[i].RecipientEmail, preview.Tender.Offers[i].RecipientEmail)
	}
	assert.Equal(t, created.Screening, preview.Screening)
	assert.Equal(t, "Dallas outbound", preview.Guide.Name)
	assert.Equal(t, "acme", preview.Carriers[healthy.ID])
}

func TestPreviewSpot_ReturnsTheRefusalCreateSpotMakes(t *testing.T) {
	deps := setupCreateTest(t)
	tenantInfo := createTestTenant()
	move := tenderableMove()
	expectTenderableMove(deps, tenantInfo, move)

	warned := warnedCarrier("expiring")
	deps.carrierRepo.On("GetByIDs", mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{warned}, nil)
	request := &CreateSpotTenderRequest{
		TenantInfo:     tenantInfo,
		ShipmentMoveID: move.ID,
		Mode:           tender.ModeSpotBroadcast,
		Lines:          []SpotTenderLine{spotLine(warned.ID)},
	}

	preview, err := deps.svc.PreviewSpot(t.Context(), request)
	require.NoError(t, err)
	require.Len(t, preview.Warnings, 1)
	require.Error(t, preview.RefusalError)

	_, createErr := deps.svc.CreateSpot(t.Context(), request)
	require.Error(t, createErr)
	assert.Equal(t, createErr.Error(), preview.RefusalError.Error())
	assert.Nil(t, deps.tenderRepo.created)
}

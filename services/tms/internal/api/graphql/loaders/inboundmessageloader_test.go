package loaders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubInboundAttachmentCounter struct {
	asked repositories.CountInboundAttachmentsRequest
	count map[pulid.ID]int
}

func (s *stubInboundAttachmentCounter) CountAttachmentsByMessageIDs(
	_ context.Context,
	req repositories.CountInboundAttachmentsRequest,
) (map[pulid.ID]int, error) {
	s.asked = req

	return s.count, nil
}

type stubShipmentSummaryLister struct {
	asked     *repositories.ListShipmentSummariesRequest
	summaries []*repositories.ShipmentSummary
}

func (s *stubShipmentSummaryLister) ListSummariesByIDs(
	_ context.Context,
	req *repositories.ListShipmentSummariesRequest,
) ([]*repositories.ShipmentSummary, error) {
	s.asked = req

	return s.summaries, nil
}

func TestInboundAttachmentCountBatchesAPageAndCountsAMessageWithNoneAsZero(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	withFiles := pulid.MustNew("imsg_")
	without := pulid.MustNew("imsg_")
	repo := &stubInboundAttachmentCounter{count: map[pulid.ID]int{withFiles: 3}}
	factory := &InboundAttachmentCountLoaderFactory{messages: repo}

	values, errs := factory.batchFunc(tenant)(t.Context(), []string{
		withFiles.String(), without.String(), withFiles.String(),
	})

	assert.Equal(t, tenant, repo.asked.TenantInfo)
	assert.Equal(t, []pulid.ID{withFiles, without}, repo.asked.MessageIDs,
		"one query for the page, each message asked about once")
	for _, err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, []int{3, 0, 3}, values)
}

func TestShipmentSummaryLoaderReturnsTheSummaryAndNotFoundForAShipmentOutOfReach(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	known := pulid.MustNew("shp_")
	gone := pulid.MustNew("shp_")
	repo := &stubShipmentSummaryLister{summaries: []*repositories.ShipmentSummary{
		{ShipmentID: known, ProNumber: "PRO-4471"},
	}}
	factory := &ShipmentSummaryByIDLoaderFactory{shipments: repo}

	values, errs := factory.batchFunc(tenant)(t.Context(), []string{known.String(), gone.String()})

	assert.Equal(t, tenant, repo.asked.TenantInfo)
	require.NoError(t, errs[0])
	assert.Equal(t, "PRO-4471", values[0].ProNumber)
	require.Error(t, errs[1])
	assert.True(t, errortypes.IsNotFoundError(errs[1]))
}

package ediservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/mocks/ediportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetEDIAckHealth_GroupsOverdueMessagesByPartnerOldestFirst(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	acme, bolt, quiet := pulid.MustNew("edip_"), pulid.MustNew("edip_"), pulid.MustNew("edip_")
	oldest := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: acme, GeneratedAt: 100}
	boltOld := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: bolt, GeneratedAt: 200}
	acmeOld := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: acme, GeneratedAt: 300}
	dead := &edi.EDIMessage{ID: pulid.MustNew("edimsg_"), EDIPartnerID: bolt}

	repo := ediportsmock.NewMockEDIMessageRepository(t)
	var countsReq ediports.GetEDIMessageStatusCountsRequest
	repo.EXPECT().GetAckStatusCounts(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIMessageStatusCountsRequest) { countsReq = req }).
		Return(map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusPending:  4,
			edi.MessageAcknowledgmentStatusAccepted: 9,
		}, nil)
	var countReq ediports.GetEDIOverdueAckCountRequest
	repo.EXPECT().GetOverdueAckCount(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req ediports.GetEDIOverdueAckCountRequest) { countReq = req }).
		Return(4, nil)
	var listReq *ediports.ListOverdueEDIAcksRequest
	repo.EXPECT().ListOverdueAcks(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.ListOverdueEDIAcksRequest) { listReq = req }).
		Return(&ediports.OverdueEDIAcks{
			Messages: []*edi.EDIMessage{oldest, boltOld, acmeOld},
			Partners: []*ediports.OverdueEDIAckPartner{
				{PartnerID: acme, PartnerName: "Acme", Count: 2, OldestGeneratedAt: 100},
				{PartnerID: bolt, PartnerName: "Bolt", Count: 1, OldestGeneratedAt: 200},
				{PartnerID: quiet, PartnerName: "Quiet", Count: 1, OldestGeneratedAt: 250},
			},
		}, nil)
	var deadReq *ediports.ListRecentEDIMessageFailuresRequest
	repo.EXPECT().ListRecentDeadLettered(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req *ediports.ListRecentEDIMessageFailuresRequest) { deadReq = req }).
		Return([]*edi.EDIMessage{dead}, nil)

	service := &Service{l: zap.NewNop(), messageRepo: repo}
	before := timeutils.NowUnix()
	health, err := service.GetEDIAckHealth(t.Context(), &GetEDIAckHealthRequest{
		TenantInfo:   tenant,
		OverdueLimit: 3,
	})
	require.NoError(t, err)

	assert.Equal(t, DefaultOverdueAckAgeSeconds, health.OverdueAckAge,
		"the age the EDI overview counts overdue by")
	assert.Equal(t, countReq.PendingSince, listReq.PendingSince,
		"the count and the list read one cutoff")
	assert.GreaterOrEqual(t, countReq.PendingSince, before-DefaultOverdueAckAgeSeconds)
	assert.Equal(t, 3, listReq.Limit)
	assert.GreaterOrEqual(t, countsReq.Since, before-defaultAckHealthWindowSeconds)
	assert.Equal(t, defaultSummaryFeedLimit, deadReq.Limit)

	assert.Equal(t, 4, health.OverdueAckCount)
	assert.Equal(t, 9, health.AckStatusCounts[edi.MessageAcknowledgmentStatusAccepted])
	require.Len(t, health.OverdueByPartner, 3)
	assert.Equal(t, acme, health.OverdueByPartner[0].PartnerID)
	assert.Equal(t, []*edi.EDIMessage{oldest, acmeOld}, health.OverdueByPartner[0].Messages)
	assert.Equal(t, bolt, health.OverdueByPartner[1].PartnerID)
	assert.Equal(t, []*edi.EDIMessage{boltOld}, health.OverdueByPartner[1].Messages)
	assert.Equal(t, quiet, health.OverdueByPartner[2].PartnerID)
	assert.Empty(t, health.OverdueByPartner[2].Messages,
		"a partner the row limit left out keeps its count")
	assert.Equal(t, 1, health.OverdueByPartner[2].OverdueCount)
	assert.Equal(t, 1, health.OverdueOmitted)
	assert.Equal(t, []*edi.EDIMessage{dead}, health.RecentDeadLettered)
}

func TestGetEDIAckHealth_KeepsToOnePartnerAndTheAgeAsked(t *testing.T) {
	t.Parallel()

	partner := pulid.MustNew("edip_")
	repo := ediportsmock.NewMockEDIMessageRepository(t)
	repo.EXPECT().GetAckStatusCounts(mock.Anything, mock.MatchedBy(
		func(req ediports.GetEDIMessageStatusCountsRequest) bool { return req.PartnerID == partner },
	)).Return(map[edi.MessageAcknowledgmentStatus]int{}, nil)
	repo.EXPECT().GetOverdueAckCount(mock.Anything, mock.MatchedBy(
		func(req ediports.GetEDIOverdueAckCountRequest) bool { return req.PartnerID == partner },
	)).Return(0, nil)
	repo.EXPECT().ListOverdueAcks(mock.Anything, mock.MatchedBy(
		func(req *ediports.ListOverdueEDIAcksRequest) bool { return req.PartnerID == partner },
	)).Return(&ediports.OverdueEDIAcks{}, nil)
	repo.EXPECT().ListRecentDeadLettered(mock.Anything, mock.MatchedBy(
		func(req *ediports.ListRecentEDIMessageFailuresRequest) bool {
			return req.PartnerID == partner
		},
	)).Return(nil, nil)

	service := &Service{l: zap.NewNop(), messageRepo: repo}
	health, err := service.GetEDIAckHealth(t.Context(), &GetEDIAckHealthRequest{
		PartnerID:     partner,
		OverdueAckAge: 3600,
		Since:         42,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 3600, health.OverdueAckAge)
	assert.EqualValues(t, 42, health.Since)
	assert.Empty(t, health.OverdueByPartner)
	assert.Zero(t, health.OverdueOmitted)
}

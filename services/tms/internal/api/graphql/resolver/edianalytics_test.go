package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEDIPartnerScorecardsToModel(t *testing.T) {
	t.Parallel()

	avgAck := 120.0
	oldest := timeutils.NowUnix() - 7200
	rows := []*repositories.EDIPartnerScorecardRow{
		{
			PartnerID:         pulid.MustNew("edip_"),
			PartnerName:       "ACME Freight",
			PartnerCode:       "ACME",
			OutboundTotal:     12,
			SentCount:         9,
			FailedCount:       2,
			DeadLetteredCount: 1,
			ReceivedCount:     5,
			AvgAckSeconds:     &avgAck,
			OverdueAckCount:   3,
			OldestPendingAt:   &oldest,
		},
		{
			PartnerID:   pulid.MustNew("edip_"),
			PartnerName: "No Traffic",
			PartnerCode: "NONE",
		},
	}

	cards := ediPartnerScorecardsToModel(rows)
	require.Len(t, cards, 2)
	require.NotNil(t, cards[0].DeliverySuccessRate)
	assert.InDelta(t, 0.75, *cards[0].DeliverySuccessRate, 0.001)
	require.NotNil(t, cards[0].OldestPendingAgeSeconds)
	assert.GreaterOrEqual(t, *cards[0].OldestPendingAgeSeconds, 7200)
	assert.Equal(t, &avgAck, cards[0].AvgAckSeconds)
	assert.Nil(t, cards[1].DeliverySuccessRate)
	assert.Nil(t, cards[1].OldestPendingAgeSeconds)
}

func TestEDIVolumeSeriesToModel(t *testing.T) {
	t.Parallel()

	series := &ediservice.EDIVolumeSeries{
		BucketSeconds: 3600,
		Points: []*repositories.EDIVolumePoint{
			{BucketStart: 1780000000, OutboundCount: 4, SentCount: 3, FailedCount: 1, ReceivedCount: 2},
		},
	}
	points := ediVolumeSeriesToModel(series)
	require.Len(t, points, 1)
	assert.Equal(t, 1780000000, points[0].BucketStart)
	assert.Equal(t, 3600, points[0].BucketSeconds)
	assert.Equal(t, 3, points[0].SentCount)
}

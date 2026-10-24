package edimessagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestLifecycleMessageQuery_MatchesTheLifecycleIdentity(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, lifecycleMessageQuery(
		&ediports.GetServiceFailure214LifecycleMessageRequest{
			ServiceFailureID: pulid.ID("sf_1"),
			Trigger:          "Reviewed",
		},
	).Limit(1))

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}

func TestScorecardQuery_GroupsEveryCountByPartner(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, scorecardQuery(&ediports.GetEDIPartnerScorecardsRequest{
		Since:                  100,
		OverdueAckPendingSince: 200,
		PendingOver4hBefore:    300,
		PendingOver24hBefore:   400,
	}))

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}

func TestVolumeQuery_GroupsByTheBucketItSelects(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, volumeQuery(&ediports.GetEDIVolumeSeriesRequest{Since: 100}))

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}

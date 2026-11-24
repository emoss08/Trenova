package detentionintelligence

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFacilities_RankEveryFacilityOverThePeriodsWholeDays(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{rollup: &detentionports.IntelligenceRollup{
		Facilities: []detentionports.FacilityIntelligence{{
			LocationID:      pulid.MustNew("loc_"),
			PastFreeMinutes: 3680,
			DriverPayAmount: nullMoney("900"),
		}},
	}}
	facilities, err := newService(repo).Facilities(context.Background(), &FacilitiesRequest{
		Period: boardcatalog.PeriodThisQuarter,
	})
	require.NoError(t, err)

	require.Len(t, facilities, 1)
	assert.False(t, facilities[0].DriverPayAmount.Valid, "withheld without margins")
	assert.Empty(t, repo.got.LocationIDs, "every facility")
	assert.Equal(t, defaultFacilities, repo.got.Limit)
	assert.Equal(t, zone, repo.got.Timezone)
	local := now.In(chicago)
	quarter := time.Date(local.Year(), local.Month()-(local.Month()-1)%3, 1, 0, 0, 0, 0, chicago)
	assert.Equal(t, quarter.Unix(), repo.got.Current.From)

	_, err = newService(&fakeRepo{rollup: &detentionports.IntelligenceRollup{}}).Facilities(
		context.Background(),
		&FacilitiesRequest{Period: boardcatalog.PeriodThisQuarter, Limit: maxFacilities + 1},
	)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "limit", validation.Path.String())

	visible, err := newService(&fakeRepo{rollup: &detentionports.IntelligenceRollup{
		Facilities: []detentionports.FacilityIntelligence{{DriverPayAmount: nullMoney("900")}},
	}}).Facilities(context.Background(), &FacilitiesRequest{
		Period:  boardcatalog.PeriodLastWeek,
		Visible: protectedfield.GroupSet(0).With(protectedfield.Margins),
	})
	require.NoError(t, err)
	assert.True(t, visible[0].DriverPayAmount.Valid)
}

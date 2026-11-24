package detentionreview

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStayOf_AStoppedStayReadsWhatItRecordedAndWhyToHoldIt(t *testing.T) {
	t.Parallel()

	occurrence := &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		Status:             detention.OccurrenceStatusPending,
		NotificationStatus: detention.NotificationStatusLate,
		RawDwellMinutes:    255,
		FreeMinutesGranted: 120,
		BillableAmount:     decimal.RequireFromString("168.75"),
	}
	stay := stayOf(&detentionports.DetentionStay{Occurrence: occurrence}, nil, 1_000_000)

	assert.Equal(t, int32(255), stay.MinutesOnSite)
	assert.Equal(t, int32(135), stay.MinutesPastFree)
	assert.Equal(t, "168.75", stay.BillableAmount.StringFixed(2))
	assert.Zero(t, stay.PricedAt)
	assert.Equal(t, detention.HoldBackNoticeLate, stay.HoldBack)
}

func TestStayOf_ARunningStayIsCountedAndPricedAsOfNow(t *testing.T) {
	t.Parallel()

	const now = int64(2_000_000)
	occurrence := &detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		Status:             detention.OccurrenceStatusAccruing,
		IsOpen:             true,
		ClockStartAt:       now - 3*60*60,
		FreeMinutesGranted: 120,
		BillableAmount:     decimal.RequireFromString("0"),
	}
	priced := map[pulid.ID]detentionengine.ComputeResult{
		occurrence.ID: {BillableAmount: decimal.RequireFromString("75")},
	}
	stay := stayOf(&detentionports.DetentionStay{Occurrence: occurrence}, priced, now)

	assert.Equal(t, int32(180), stay.MinutesOnSite)
	assert.Equal(t, int32(60), stay.MinutesPastFree)
	assert.Equal(t, "75", stay.BillableAmount.String(), "a running clock is priced now")
	assert.Equal(t, now, stay.PricedAt)
	assert.Equal(t, detention.HoldBackNone, stay.HoldBack)
}

func TestListStays_RefusesALimitPastAHundredBeforeReading(t *testing.T) {
	t.Parallel()

	svc := &Service{now: func() int64 { return 1 }}
	for _, limit := range []int{-1, detentionports.MaxStays + 1} {
		_, err := svc.ListStays(context.Background(), &StaysRequest{
			Period: boardcatalog.PeriodLastWeek,
			Limit:  limit,
		})
		var validation *errortypes.Error
		require.ErrorAs(t, err, &validation, "limit %d", limit)
		assert.Equal(t, "limit", validation.Path.String())
	}
}

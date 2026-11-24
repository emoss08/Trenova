package operationsboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const boardNow = int64(1_795_000_000)

func settledStay(id pulid.ID) stayRow {
	return stayRow{
		ID:                 id,
		ProNumber:          "S26110041283015",
		Place:              "Romeo, MI",
		Status:             string(detention.OccurrenceStatusPending),
		NotificationStatus: string(detention.NotificationStatusSent),
		FreeMinutes:        120,
		DwellMinutes:       255,
		PastFreeMinutes:    135,
		BillableAmount:     decimal.RequireFromString("168.75"),
		DriverPayAmount:    decimal.NewNullDecimal(decimal.RequireFromString("54.00")),
		Currency:           "USD",
	}
}

func TestDriverPayIsWithheldFromAReaderWithoutMargins(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("dto_")
	rows := []stayRow{settledStay(id)}

	withheld := driverPay(rows, protectedfield.GroupSet(0))
	assert.Empty(t, withheld, "no reading, not a zero, for a reader without margins")

	visible := driverPay(rows, protectedfield.GroupSet(0).With(protectedfield.Margins))
	require.Contains(t, visible, id)
	assert.Equal(t, "54", visible[id].Value.Number.String())
	assert.Equal(t, "USD", visible[id].Value.Currency)
}

func TestDriverPayReadsNothingForAReaderWithoutMargins(t *testing.T) {
	t.Parallel()

	reader := &DriverPay{}
	readings, err := reader.Readings(context.Background(), &boardreaderports.RecordRequest{
		IDs: []pulid.ID{pulid.MustNew("dto_")},
	})
	require.NoError(t, err)
	assert.Empty(t, readings)
}

func TestABillableStayReadsItsAmountAgainstItsFreeTime(t *testing.T) {
	t.Parallel()

	row := settledStay(pulid.MustNew("dto_"))
	reading := billable(context.Background(), &row, boardNow)

	assert.Equal(t, boardcatalog.ValueMoney, reading.Value.Kind)
	assert.Equal(t, "168.75", reading.Value.Number.StringFixed(2))
	require.NotNil(t, reading.Bounded)
	assert.Equal(t, int64(255), reading.Bounded.Value.IntPart())
	assert.Equal(t, int64(120), reading.Bounded.Bound.IntPart())
	require.NotNil(t, reading.Fact)
	assert.Equal(t, "{0} on site", reading.Fact.Text)
	assert.Equal(t, int64(255*60), reading.Fact.Value.Number.IntPart())
	assert.Empty(t, reading.Tone)
}

func TestAStayNotReadyToBillSaysWhy(t *testing.T) {
	t.Parallel()

	past := boardNow - 3600
	future := boardNow + 3600
	cases := []struct {
		name string
		edit func(*stayRow)
		want string
	}{
		{"notice missed", func(r *stayRow) {
			r.NotificationStatus = string(detention.NotificationStatusMissed)
		}, "Notice missed"},
		{"notice window closed unsent", func(r *stayRow) {
			r.NotificationStatus = string(detention.NotificationStatusPending)
			r.NoticeDeadlineAt = &past
		}, "Notice missed"},
		{"notice failed", func(r *stayRow) {
			r.NotificationStatus = string(detention.NotificationStatusFailed)
		}, "Notice never reached the customer"},
		{"notice late", func(r *stayRow) {
			r.NotificationStatus = string(detention.NotificationStatusLate)
		}, "Notice sent late"},
		{"departure by hand", func(r *stayRow) { r.DepartureByHand = true }, "Departure entered by hand"},
		{"arrival by hand", func(r *stayRow) { r.ArrivalByHand = true }, "Arrival entered by hand"},
		{"notice still open", func(r *stayRow) {
			r.NotificationStatus = string(detention.NotificationStatusPending)
			r.NoticeDeadlineAt = &future
		}, ""},
		{"billed stays are not held", func(r *stayRow) {
			r.Status = string(detention.OccurrenceStatusBilled)
			r.NotificationStatus = string(detention.NotificationStatusMissed)
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			row := settledStay(pulid.MustNew("dto_"))
			tc.edit(&row)
			reading := billable(context.Background(), &row, boardNow)
			if tc.want == "" {
				assert.Equal(t, "{0} on site", reading.Fact.Text)
				assert.Empty(t, reading.Tone)
				return
			}
			assert.Equal(t, tc.want, reading.Fact.Text)
			assert.Nil(t, reading.Fact.Value)
			assert.Equal(t, boardcatalog.ToneWarn, reading.Tone)
		})
	}
}

func TestAStayStillOnSiteBillsNothingYetAndCountsItsTimeLive(t *testing.T) {
	t.Parallel()

	row := settledStay(pulid.MustNew("dto_"))
	row.IsOpen = true
	row.Status = string(detention.OccurrenceStatusAccruing)
	row.ClockStartAt = boardNow - 3*3600

	reading := billable(context.Background(), &row, boardNow)
	assert.True(t, reading.Value.Number.IsZero())
	assert.Equal(t, "Still on site", reading.Standing.Text)
	assert.Equal(t, int64(180), reading.Bounded.Value.IntPart())

	dwell, past := onSite(&row, boardNow)
	assert.Equal(t, int64(180), dwell)
	assert.Equal(t, int64(60), past)
}

type fakeIntelligence struct {
	detentionports.DetentionIntelligenceRepository
	calls      int
	got        *detentionports.IntelligenceRequest
	facilities []detentionports.FacilityIntelligence
	totals     detentionports.IntelligenceSummary
	arrivals   []detentionports.ArrivalCell
}

func (f *fakeIntelligence) FacilityFigures(
	_ context.Context,
	req *detentionports.IntelligenceRequest,
) ([]detentionports.FacilityIntelligence, error) {
	f.calls++
	f.got = req
	return f.facilities, nil
}

func (f *fakeIntelligence) TotalFigures(
	_ context.Context,
	req *detentionports.IntelligenceRequest,
) (*detentionports.IntelligenceSummary, error) {
	f.calls++
	f.got = req
	return &f.totals, nil
}

func (f *fakeIntelligence) ArrivalFigures(
	_ context.Context,
	req *detentionports.IntelligenceRequest,
) ([]detentionports.ArrivalCell, error) {
	f.calls++
	f.got = req
	return f.arrivals, nil
}

func TestDwellReadsEveryFacilityOfABlockInOneCallThisQuarter(t *testing.T) {
	t.Parallel()

	romeo, monroe, quiet := pulid.MustNew("loc_"), pulid.MustNew("loc_"), pulid.MustNew("loc_")
	fake := &fakeIntelligence{facilities: []detentionports.FacilityIntelligence{
		{
			LocationID: romeo, StopCount: 46, PastFreeMinutes: 3680,
			MedianFreeMinutes: 120, MedianDwellMinutes: 220, P90DwellMinutes: 410,
			DriverPayAmount: decimal.NewNullDecimal(decimal.NewFromInt(900)),
		},
		{
			LocationID: monroe, StopCount: 1, PastFreeMinutes: 30,
			MedianFreeMinutes: 120, MedianDwellMinutes: 150, P90DwellMinutes: 150,
		},
	}}
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 23, 9, 41, 0, 0, chicago).Unix()

	readings, err := NewDwell(DetentionParams{Intelligence: fake}).Readings(
		context.Background(),
		&boardreaderports.RecordRequest{
			Tenant:   pagination.TenantInfo{OrgID: pulid.MustNew("org_")},
			IDs:      []pulid.ID{romeo, monroe, quiet},
			Now:      now,
			Location: chicago,
		},
	)
	require.NoError(t, err)

	assert.Equal(t, 1, fake.calls)
	assert.Equal(t, []pulid.ID{romeo, monroe, quiet}, fake.got.LocationIDs)
	assert.Equal(t, "America/Chicago", fake.got.Timezone)
	assert.Equal(t,
		time.Date(2026, time.October, 1, 0, 0, 0, 0, chicago).Unix(), fake.got.Current.From)
	assert.Equal(t,
		time.Date(2026, time.November, 24, 0, 0, 0, 0, chicago).Unix()-1, fake.got.Current.To)

	require.Len(t, readings, 3)
	assert.Equal(t, int64(3680*60), readings[romeo].Value.Number.IntPart())
	assert.Equal(t, "46 stops, median 3h 40m, one in ten over 6h 50m", readings[romeo].Fact.Text)
	require.NotNil(t, readings[romeo].Spread)
	assert.Equal(t, "120", readings[romeo].Spread.Allowance.String())
	assert.Equal(t, "220", readings[romeo].Spread.Median.String())
	assert.Equal(t, "410", readings[romeo].Spread.Tail.String())
	assert.Equal(t, "1 stop, median 2h 30m, one in ten over 2h 30m", readings[monroe].Fact.Text)
	assert.True(t, readings[quiet].Value.Number.IsZero())
	assert.Equal(t, "No stops this quarter", readings[quiet].Fact.Text)
	assert.False(t, fake.facilities[0].DriverPayAmount.Valid,
		"driver pay is cleared for a reader without margins")
}

func TestDetentionTotalsReadTheOrganizationsFiguresForThePeriod(t *testing.T) {
	t.Parallel()

	fake := &fakeIntelligence{totals: detentionports.IntelligenceSummary{
		Totals: detentionports.IntelligenceTotals{
			DetainedCount:     214,
			PastFreeMinutes:   12_880,
			BilledAmount:      decimal.RequireFromString("9840.00"),
			WaivedAmount:      decimal.RequireFromString("2310.00"),
			NoticeDueCount:    50,
			NoticeOnTimeCount: 44,
		},
		Recovery: []detentionports.RecoveryBucket{
			{
				Bucket:          detentionports.RecoveryPending,
				OccurrenceCount: 165,
				BillableAmount:  decimal.RequireFromString("21356.25"),
			},
			{
				Bucket:          detentionports.RecoveryBilled,
				OccurrenceCount: 15,
				BillableAmount:  decimal.RequireFromString("4237.50"),
			},
		},
	}}
	read, err := NewDetentionTotals(DetentionParams{Intelligence: fake}).Read(
		context.Background(),
		&boardreaderports.MetricRequest{Period: boardcatalog.PeriodThisQuarter, Now: boardNow},
	)
	require.NoError(t, err)

	assert.Empty(t, fake.got.LocationIDs, "every facility together")
	assert.Equal(t, "21356.25", read.Figures["readyToBill"].Number.String())
	assert.Equal(t, int64(165), read.Figures["readyToBillStays"].Number.IntPart())
	assert.Equal(t, int64(12_880*60), read.Figures["pastFreeTime"].Number.IntPart())
	assert.Equal(t, "9840", read.Figures["billed"].Number.String())
	assert.Equal(t, "2310", read.Figures["waived"].Number.String())
	assert.Equal(t, "0.88", read.Figures["noticeOnTime"].Number.String())
	assert.Equal(t, "214 detained stops at every facility, this quarter", read.Source)

	none := detentionTotals(context.Background(), boardcatalog.PeriodLastWeek,
		&detentionports.IntelligenceSummary{})
	assert.Contains(t, none.Figures, "noticeOnTime")
	assert.Nil(t, none.Figures["noticeOnTime"], "no notice was due, so no share is read")
	assert.True(t, none.Figures["readyToBill"].Number.IsZero(),
		"no stopped charge waits, so nothing is ready: a true zero, read from the rollup")
}

func TestArrivalSharesAreTheOrganizationsByTwoHourWindow(t *testing.T) {
	t.Parallel()

	cells := []detentionports.ArrivalCell{
		{Weekday: 1, Hour: 8, StopCount: 20, DetainedCount: 12},
		{Weekday: 2, Hour: 9, StopCount: 20, DetainedCount: 10},
		{Weekday: 1, Hour: 4, StopCount: 30, DetainedCount: 6},
		{Weekday: 3, Hour: 23, StopCount: 2, DetainedCount: 2},
	}
	read := arrivalShares(context.Background(), boardcatalog.PeriodThisQuarter, cells)

	require.NotNil(t, read.Chart)
	assert.Len(t, read.Chart.X, 12)
	assert.Equal(t, "12 AM", read.Chart.X[0].Label)
	assert.Equal(t, "8 AM", read.Chart.X[4].Label)
	assert.Equal(t, []int{4}, read.Chart.Marked, "the 10:00 PM window has too few arrivals to lead")
	assert.Equal(t, "8 AM to 10 AM", read.Figures["worstWindow"].Text)
	assert.Equal(t, "0.55", read.Figures["worstWindowShare"].Number.String())
	assert.Equal(t, "0.4166666666666667", read.Figures["detainedShare"].Number.String())
	assert.Nil(t, read.Chart.Series[0].Points[0], "no truck arrived after midnight")
	assert.Equal(t, "72 stops at every facility together, this quarter", read.Source)

	empty := arrivalShares(context.Background(), boardcatalog.PeriodThisQuarter, nil)
	assert.Nil(t, empty.Chart)
	assert.Contains(t, empty.Figures, "worstWindow")
	assert.Nil(t, empty.Figures["worstWindow"])
}

func TestAStayIsReadOnlyThroughItsShipment(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, stayQuery([]pulid.ID{pulid.MustNew("dto_")}))

	assert.Contains(t, sql, `JOIN "shipments"`,
		"a stay whose shipment is outside the reader's row scope reads as nothing")
	assert.NotContains(t, sql, `LEFT JOIN "shipments"`)
}

func TestAStaysStatusIsTheDetentionCardsWordAndTone(t *testing.T) {
	t.Parallel()

	cases := map[detention.OccurrenceStatus]struct {
		words string
		tone  boardcatalog.Tone
	}{
		detention.OccurrenceStatusAccruing:    {"At the dock", boardcatalog.ToneInfo},
		detention.OccurrenceStatusPending:     {"Waiting on approval", boardcatalog.ToneWarn},
		detention.OccurrenceStatusApproved:    {"Approved", boardcatalog.ToneInfo},
		detention.OccurrenceStatusBilled:      {"Billed", boardcatalog.ToneOK},
		detention.OccurrenceStatusDisputed:    {"Disputed", boardcatalog.ToneWarn},
		detention.OccurrenceStatusWaived:      {"Waived", ""},
		detention.OccurrenceStatusNotBillable: {"Not billable", ""},
	}
	for status, want := range cases {
		words, tone := stayStatus(context.Background(), string(status))
		assert.Equal(t, want.words, words, status)
		assert.Equal(t, want.tone, tone, status)
	}
}

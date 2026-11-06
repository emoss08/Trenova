package demandforecastservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	mondayMidnight = int64(1_793_577_600)
	testNow        = mondayMidnight + 2*3600
	hour           = int64(3600)
	org            = pulid.ID("org_a")
	bu             = pulid.ID("bu_a")
	acme           = pulid.ID("cus_acme")
	globex         = pulid.ID("cus_globex")
)

var (
	tenantInfo    = pagination.TenantInfo{OrgID: org, BuID: bu}
	dallasPoint   = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	garlandPoint  = dispatchmarket.Point{Latitude: 32.9126, Longitude: -96.6389}
	mesquitePoint = dispatchmarket.Point{Latitude: 32.7668, Longitude: -96.5992}
	houstonPoint  = dispatchmarket.Point{Latitude: 29.7604, Longitude: -95.3698}
	errUnexpected = errors.New("unexpected call")
)

type fakeRepository struct {
	rows       *demandforecastports.HistoryRows
	historyErr error
	replaceErr error
	model      *demandforecastports.Model
	booked     []demandforecastports.BookedSlot
	starts     []demandforecastports.DriverStart

	histories []*demandforecastports.HistoryRequest
	replaced  []*demandforecastports.ReplaceRequest
	startsFor []*demandforecastports.DriverStartsRequest
}

func (f *fakeRepository) History(
	_ context.Context,
	req *demandforecastports.HistoryRequest,
) (*demandforecastports.HistoryRows, error) {
	f.histories = append(f.histories, req)
	if f.historyErr != nil {
		return nil, f.historyErr
	}

	return f.rows, nil
}

func (f *fakeRepository) Replace(_ context.Context, req *demandforecastports.ReplaceRequest) error {
	f.replaced = append(f.replaced, req)

	return f.replaceErr
}

func (f *fakeRepository) Quality(
	context.Context,
	pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) ModelForCells(
	context.Context,
	*demandforecastports.CellsRequest,
) (*demandforecastports.Model, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) Model(
	context.Context,
	pagination.TenantInfo,
) (*demandforecastports.Model, error) {
	return f.model, nil
}

func (f *fakeRepository) ListBooked(
	context.Context,
	*demandforecastports.BookedRequest,
) ([]demandforecastports.BookedSlot, error) {
	return f.booked, nil
}

func (f *fakeRepository) ListDriverStarts(
	_ context.Context,
	req *demandforecastports.DriverStartsRequest,
) ([]demandforecastports.DriverStart, error) {
	f.startsFor = append(f.startsFor, req)

	return f.starts, nil
}

func (f *fakeRepository) CustomerNames(
	context.Context,
	*demandforecastports.CustomerNamesRequest,
) (map[pulid.ID]string, error) {
	return map[pulid.ID]string{}, nil
}

func (f *fakeRepository) ListTenants(
	context.Context,
	*demandforecastports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	return nil, errUnexpected
}

func newService(
	repo demandforecastports.Repository,
	controls dispatchcontrolports.DispatchControlRepository,
	organizations tenantports.OrganizationRepository,
) *Service {
	svc := New(Params{
		Logger:        zap.NewNop(),
		Repo:          repo,
		Controls:      controls,
		Organizations: organizations,
	})
	svc.clock = func() int64 { return testNow }

	return svc
}

func cellOf(t *testing.T, point dispatchmarket.Point) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(point)
	require.True(t, ok)

	return cell
}

func location(
	id pulid.ID,
	point dispatchmarket.Point,
	city string,
	loads int32,
	medianCents int64,
) demandforecastports.LocationSummary {
	return demandforecastports.LocationSummary{
		LocationID:         id,
		Latitude:           point.Latitude,
		Longitude:          point.Longitude,
		City:               city,
		State:              "TX",
		Loads:              loads,
		MedianRevenueCents: medianCents,
	}
}

func TestHistoryOf_GroupsLocationsIntoAreasNamedForTheirBusiestPlace(t *testing.T) {
	t.Parallel()

	dallas, houston := cellOf(t, dallasPoint), cellOf(t, houstonPoint)
	require.Equal(t, dallas, cellOf(t, garlandPoint))
	require.Equal(t, dallas, cellOf(t, mesquitePoint))
	week := dispatchmarket.WeekOf(testNow) - 1
	rows := &demandforecastports.HistoryRows{
		Locations: []demandforecastports.LocationSummary{
			location("loc_garland", garlandPoint, "Garland", 10, 20_000),
			location("loc_dallas", dallasPoint, "Dallas", 30, 40_000),
			location("loc_mesquite", mesquitePoint, "Mesquite", 20, 0),
			location("loc_houston", houstonPoint, "Houston", 5, 0),
			location("loc_nowhere", dispatchmarket.Point{}, "Nowhere", 9, 90_000),
		},
		Counts: []demandforecastports.LocationCount{
			{CustomerID: acme, LocationID: "loc_dallas", Week: week, Slot: 7, Loads: 3},
			{CustomerID: globex, LocationID: "loc_garland", Week: week, Slot: 7, Loads: 2},
			{CustomerID: acme, LocationID: "loc_houston", Week: week, Slot: 9, Loads: 1},
			{CustomerID: acme, LocationID: "loc_nowhere", Week: week, Slot: 7, Loads: 4},
		},
		Leads: []demandforecast.LeadCount{{CustomerID: acme, Bucket: 5, Loads: 6}},
	}

	history := HistoryOf(testNow, rows)

	areas := []demandforecast.AreaSummary{
		{Cell: dallas, Label: "Dallas, TX", MedianRevenueCents: 35_000, Loads: 60},
		{Cell: houston, Label: "Houston, TX", MedianRevenueCents: 0, Loads: 5},
	}
	slices.SortFunc(areas, func(a, b demandforecast.AreaSummary) int {
		return cmp.Compare(a.Cell, b.Cell)
	})
	assert.Equal(t, &demandforecast.History{
		Now: testNow,
		Counts: []demandforecast.LoadCount{
			{CustomerID: acme, Cell: dallas, Week: week, Slot: 7, Loads: 3},
			{CustomerID: globex, Cell: dallas, Week: week, Slot: 7, Loads: 2},
			{CustomerID: acme, Cell: houston, Week: week, Slot: 9, Loads: 1},
		},
		Leads: rows.Leads,
		Areas: areas,
	}, history)
}

func fitRows(currentWeek int64) *demandforecastports.HistoryRows {
	rows := &demandforecastports.HistoryRows{
		Locations: []demandforecastports.LocationSummary{
			location("loc_dallas", dallasPoint, "Dallas", 240, 40_000),
		},
		Leads: []demandforecast.LeadCount{{CustomerID: acme, Bucket: 5, Loads: 200}},
	}
	for back := int64(1); back <= 12; back++ {
		loads := int32(18)
		if back%2 == 0 {
			loads = 22
		}
		rows.Counts = append(rows.Counts, demandforecastports.LocationCount{
			CustomerID: acme,
			LocationID: "loc_dallas",
			Week:       currentWeek - back,
			Slot:       7,
			Loads:      loads,
		})
	}

	return rows
}

func TestFit_ReadsTheBacktestSpanAndSavesTheModelForTheTenant(t *testing.T) {
	t.Parallel()

	rows := fitRows(dispatchmarket.WeekOf(testNow))
	repo := &fakeRepository{rows: rows}
	svc := newService(repo, nil, nil)

	quality, err := svc.Fit(t.Context(), tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, []*demandforecastports.HistoryRequest{{
		TenantInfo: tenantInfo,
		Since:      testNow - (26+8+1)*dispatchmarket.WeekSeconds,
		Until:      testNow,
	}}, repo.histories)
	assert.Equal(t, demandforecast.Backtest(HistoryOf(testNow, rows)), quality)

	require.Len(t, repo.replaced, 1)
	replaced := repo.replaced[0]
	assert.Equal(t, tenantInfo, replaced.TenantInfo)
	assert.Same(t, quality, replaced.Quality)
	require.NotEmpty(t, replaced.Fitted.Rates)
	require.NotEmpty(t, replaced.Fitted.Areas)
	require.Len(t, replaced.Fitted.Curves, 2)
	for _, rate := range replaced.Fitted.Rates {
		assert.True(t, rate.OrganizationID.IsNil(), "keel fills tenant columns on insert")
	}
}

func TestFit_ReturnsTheRepositoryError(t *testing.T) {
	t.Parallel()

	historyDown := errors.New("history down")
	replaceDown := errors.New("replace down")
	tests := []struct {
		name         string
		repo         *fakeRepository
		want         error
		wantReplaced int
	}{
		{
			name: "reading history",
			repo: &fakeRepository{historyErr: historyDown},
			want: historyDown,
		},
		{
			name: "saving the model",
			repo: &fakeRepository{
				rows:       fitRows(dispatchmarket.WeekOf(testNow)),
				replaceErr: replaceDown,
			},
			want:         replaceDown,
			wantReplaced: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			quality, err := newService(tt.repo, nil, nil).Fit(t.Context(), tenantInfo)

			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, quality)
			assert.Len(t, tt.repo.replaced, tt.wantReplaced)
		})
	}
}

type area struct {
	point dispatchmarket.Point
	label string
}

var (
	tenantCurve = []float64{1, 0.9, 0.8, 0.7, 0.6, 0.45, 0.35, 0.25, 0.15, 0.1, 0.05, 0.02, 0}
	dallasArea  = area{point: dallasPoint, label: "Dallas, TX"}
	sixAreas    = []area{
		dallasArea,
		{point: houstonPoint, label: "Houston, TX"},
		{point: point(29.4241, -98.4936), label: "San Antonio, TX"},
		{point: point(35.4676, -97.5164), label: "Oklahoma City, OK"},
		{point: point(39.7392, -104.9903), label: "Denver, CO"},
		{point: point(41.8781, -87.6298), label: "Chicago, IL"},
	}
)

func point(latitude, longitude float64) dispatchmarket.Point {
	return dispatchmarket.Point{Latitude: latitude, Longitude: longitude}
}

func forecastModel(
	t *testing.T,
	quality *demandforecast.Quality,
	areas ...area,
) *demandforecastports.Model {
	t.Helper()
	model := &demandforecastports.Model{
		Quality: quality,
		Curves:  []*demandforecast.BookingCurve{{Shares: tenantCurve}},
	}
	cells := map[int64]struct{}{}
	for _, a := range areas {
		cell := cellOf(t, a.point)
		cells[cell] = struct{}{}
		model.Areas = append(model.Areas, &demandforecast.Area{
			Cell: cell, Label: a.label, MedianRevenueCents: 50_000, Loads: 400,
		})
		for slot := range dispatchmarket.SlotsPerWeek {
			model.Rates = append(model.Rates, &demandforecast.Rate{
				Cell:  cell,
				Slot:  slot,
				Alpha: 10 + float64(slot%5)*10,
				Beta:  10,
			})
		}
	}
	require.Len(t, cells, len(areas), "every area is its own map cell")

	return model
}

func readyOutlookService(t *testing.T, areas ...area) (*Service, *fakeRepository) {
	t.Helper()
	repo := &fakeRepository{
		model: forecastModel(
			t,
			&demandforecast.Quality{Status: demandforecast.StatusReady},
			areas...,
		),
		starts: []demandforecastports.DriverStart{{
			WorkerID:  pulid.ID("wrk_ana"),
			Latitude:  dallasPoint.Latitude,
			Longitude: dallasPoint.Longitude,
			FreeAt:    testNow,
		}},
		booked: []demandforecastports.BookedSlot{{
			Latitude:  dallasPoint.Latitude,
			Longitude: dallasPoint.Longitude,
			SlotStart: mondayMidnight + 12*hour,
			Total:     2,
			Open:      1,
		}},
	}
	controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	controls.EXPECT().GetOrCreate(mock.Anything, org, bu).
		Return(dispatchcontrol.NewDefaultDispatchControl(org, bu), nil)
	organizations := tenantportsmock.NewMockOrganizationRepository(t)
	organizations.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{Timezone: "America/Chicago"}, nil)

	return newService(repo, controls, organizations), repo
}

func TestOutlook_ClampsTheHorizonAndTheLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hours   int
		limit   int
		windows int
		latest  int64
	}{
		{
			name:    "unset reads 48 hours and keeps 20 windows",
			windows: 20,
			latest:  mondayMidnight + 48*hour,
		},
		{
			name:    "too much reads 96 hours and keeps 50 windows",
			hours:   500,
			limit:   500,
			windows: 50,
			latest:  mondayMidnight + 96*hour,
		},
		{
			name:    "a short horizon keeps every window it has",
			hours:   24,
			windows: 18,
			latest:  mondayMidnight + 24*hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := readyOutlookService(t, sixAreas...)

			outlook, err := svc.Outlook(t.Context(), &demandforecastports.OutlookRequest{
				TenantInfo: tenantInfo,
				Hours:      tt.hours,
				Limit:      tt.limit,
			})
			require.NoError(t, err)

			assert.Equal(t, []*demandforecastports.DriverStartsRequest{{
				TenantInfo: tenantInfo,
				Now:        testNow,
			}}, repo.startsFor)
			assert.Equal(t, testNow, outlook.GeneratedAt)
			require.Len(t, outlook.Windows, tt.windows)
			latest := int64(0)
			for _, window := range outlook.Windows {
				latest = max(latest, window.WindowStart)
			}
			assert.Equal(t, tt.latest, latest)
		})
	}
}

func TestOutlook_PutsTheBiggestShortfallFirst(t *testing.T) {
	t.Parallel()

	svc, _ := readyOutlookService(t, dallasArea)
	req := &demandforecastports.OutlookRequest{TenantInfo: tenantInfo, Hours: 96, Limit: 50}

	full, err := svc.Outlook(t.Context(), req)
	require.NoError(t, err)
	req.Limit = 3
	limited, err := svc.Outlook(t.Context(), req)
	require.NoError(t, err)

	require.Len(t, full.Windows, 9)
	for i := 1; i < len(full.Windows); i++ {
		before, after := full.Windows[i-1], full.Windows[i]
		assert.GreaterOrEqual(t, before.Shortfall, after.Shortfall, "window %d", i)
		if before.Shortfall == after.Shortfall {
			assert.GreaterOrEqual(t, before.ValueCents, after.ValueCents, "window %d", i)
		}
	}
	assert.Greater(t, full.Windows[0].Shortfall, full.Windows[8].Shortfall)
	assert.NotEqual(t, mondayMidnight, full.Windows[0].WindowStart,
		"the soonest window is mostly booked already, so it is not the biggest shortfall")
	assert.Equal(t, full.Windows[:3], limited.Windows)
}

func TestOutlook_DescribesEachWindowInTheOrganizationsTimeZone(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	svc, _ := readyOutlookService(t, dallasArea)

	outlook, err := svc.Outlook(t.Context(), &demandforecastports.OutlookRequest{
		TenantInfo: tenantInfo,
	})
	require.NoError(t, err)

	require.Len(t, outlook.Windows, 5)
	byStart := map[int64]*demandforecastports.OutlookWindow{}
	for _, window := range outlook.Windows {
		byStart[window.WindowStart] = window
		assert.Equal(t, cellOf(t, dallasPoint), window.Cell)
		assert.Equal(t, "Dallas, TX", window.AreaLabel)
		assert.Equal(t, window.WindowStart+12*hour, window.WindowEnd)
		assert.Equal(t, 1, window.DriversExpected)
		assert.Equal(t, demandforecast.Explain(&demandforecast.Reason{
			AreaLabel:       window.AreaLabel,
			WindowStart:     window.WindowStart,
			WindowEnd:       window.WindowEnd,
			TypicalLow:      window.TypicalLow,
			TypicalHigh:     window.TypicalHigh,
			Booked:          window.Booked,
			DriversExpected: window.DriversExpected,
			Customers:       window.Customers,
			ValueCents:      window.ValueCents,
		}, chicago), window.Explanation)
	}

	require.Contains(t, byStart, mondayMidnight)
	first := byStart[mondayMidnight]
	assert.Contains(t, first.Explanation, "Ending near Dallas, TX is worth ")
	assert.Contains(t, first.Explanation, " usually appear there Sun 18:00–Mon 06:00",
		"Monday 00:00 UTC is Sunday evening in Chicago")
	assert.Zero(t, first.Booked)
	require.Contains(t, byStart, mondayMidnight+12*hour)
	assert.Equal(t, 2, byStart[mondayMidnight+12*hour].Booked)
}

func TestOutlook_ShowsNoWindowsUntilTheForecastIsReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		quality *demandforecast.Quality
		areas   []area
	}{
		{name: "never fitted", areas: []area{dallasArea}},
		{
			name:    "not enough history",
			quality: &demandforecast.Quality{Status: demandforecast.StatusInsufficientHistory},
			areas:   []area{dallasArea},
		},
		{
			name:    "no better than last week",
			quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill},
			areas:   []area{dallasArea},
		},
		{
			name:    "ready with no areas",
			quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeRepository{model: forecastModel(t, tt.quality, tt.areas...)}
			svc := newService(
				repo,
				dispatchcontrolportsmock.NewMockDispatchControlRepository(t),
				tenantportsmock.NewMockOrganizationRepository(t),
			)

			outlook, err := svc.Outlook(t.Context(), &demandforecastports.OutlookRequest{
				TenantInfo: tenantInfo,
			})
			require.NoError(t, err)

			assert.Same(t, tt.quality, outlook.Quality)
			assert.Equal(t, testNow, outlook.GeneratedAt)
			assert.NotNil(t, outlook.Windows)
			assert.Empty(t, outlook.Windows)
			assert.Empty(t, repo.startsFor, "driver starts are not read for a forecast nobody sees")
		})
	}
}

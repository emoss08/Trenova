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
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgholidayports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
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

	quality    *demandforecast.Quality
	qualityErr error
	modelReads int

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
	if f.quality == nil && f.qualityErr == nil {
		return nil, errUnexpected
	}

	return f.quality, f.qualityErr
}

type fakeFits struct {
	err     error
	started []pagination.TenantInfo
}

func (f *fakeFits) StartFitting(_ context.Context, tenantInfo pagination.TenantInfo) error {
	f.started = append(f.started, tenantInfo)

	return f.err
}

func (f *fakeRepository) Model(
	context.Context,
	pagination.TenantInfo,
) (*demandforecastports.Model, error) {
	f.modelReads++

	return f.model, nil
}

type fakeHolidays struct {
	orgholidayports.OrgHolidayRepository

	rows []*pto.OrgHoliday
	err  error
}

func (f *fakeHolidays) List(
	context.Context,
	*orgholidayports.ListOrgHolidaysRequest,
) ([]*pto.OrgHoliday, error) {
	return f.rows, f.err
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

func (f *fakeRepository) Days(
	context.Context,
	pagination.TenantInfo,
) ([]*demandforecast.Day, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) ListOpenLanes(
	context.Context,
	*demandforecastports.WindowRequest,
) ([]demandforecastports.OpenLane, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) ListDepartures(
	context.Context,
	*demandforecastports.WindowRequest,
) ([]demandforecastports.Departure, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) ListAreaLocations(
	context.Context,
	*demandforecastports.AreaLocationsRequest,
) ([]pulid.ID, error) {
	return nil, errUnexpected
}

func (f *fakeRepository) ListTenants(
	context.Context,
	*demandforecastports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	return nil, errUnexpected
}

type fakeOrganizations struct {
	tenantports.OrganizationRepository

	timezone string
}

func (f fakeOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: f.timezone}, nil
}

func newService(
	repo demandforecastports.Repository,
	controls dispatchcontrolports.DispatchControlRepository,
	organizations tenantports.OrganizationRepository,
) *Service {
	return newServiceWithFits(repo, &fakeFits{}, controls, organizations)
}

func newServiceWithFits(
	repo demandforecastports.Repository,
	fits demandforecastports.FitStarter,
	controls dispatchcontrolports.DispatchControlRepository,
	organizations tenantports.OrganizationRepository,
) *Service {
	svc := New(Params{
		Logger:        zap.NewNop(),
		Repo:          repo,
		Fits:          fits,
		Holidays:      &fakeHolidays{},
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
		Leads: []demandfit.LeadCount{{CustomerID: acme, Bucket: 5, Loads: 6}},
	}

	history := HistoryOf(testNow, rows)

	areas := []demandfit.AreaSummary{
		{Cell: dallas, Label: "Dallas, TX", MedianRevenueCents: 35_000, Loads: 60},
		{Cell: houston, Label: "Houston, TX", MedianRevenueCents: 0, Loads: 5},
	}
	slices.SortFunc(areas, func(a, b demandfit.AreaSummary) int {
		return cmp.Compare(a.Cell, b.Cell)
	})
	assert.Equal(t, &demandfit.History{
		Now: testNow,
		Counts: []demandfit.LoadCount{
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
		Leads: []demandfit.LeadCount{{CustomerID: acme, Bucket: 5, Loads: 200}},
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
	svc := newService(repo, nil, fakeOrganizations{timezone: "America/Chicago"})
	closed := time.Date(2026, time.October, 28, 0, 0, 0, 0, time.UTC).Unix()
	ahead := time.Date(2026, time.November, 3, 0, 0, 0, 0, time.UTC).Unix()
	svc.holidays = &fakeHolidays{rows: []*pto.OrgHoliday{
		{Name: "Founders day", HolidayDate: closed, Kind: pto.HolidayKindHoliday},
		{Name: "Election day", HolidayDate: ahead, Kind: pto.HolidayKindHoliday},
		{Name: "Peak freeze", HolidayDate: ahead + 86_400, Kind: pto.HolidayKindBlackout},
	}}

	quality, err := svc.Fit(t.Context(), tenantInfo)
	require.NoError(t, err)

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	history := HistoryOf(testNow, rows)
	history.Grid = dispatchmarket.GridIn(chicago)
	history.Holidays = demandforecast.Holidays{closed / 86_400: {}, ahead / 86_400: {}}
	want := demandfit.Train(history)
	wantQuality, wantDays := want.Quality, want.Days

	assert.Equal(t, []*demandforecastports.HistoryRequest{{
		TenantInfo: tenantInfo,
		Since:      testNow - (26+8+1)*dispatchmarket.WeekSeconds,
		Until:      testNow,
		Timezone:   "America/Chicago",
	}}, repo.histories)
	assert.Equal(t, wantQuality, quality)
	assert.Equal(t, "America/Chicago", quality.Timezone,
		"the fit records the clock its slots were counted on")
	assert.Equal(t, []int64{ahead / 86_400}, quality.Holidays,
		"a holiday in the days ahead is kept for the forecast; a blackout is not one")

	require.Len(t, repo.replaced, 1)
	replaced := repo.replaced[0]
	assert.Equal(t, tenantInfo, replaced.TenantInfo)
	assert.Same(t, quality, replaced.Quality)
	require.NotEmpty(t, replaced.Fitted.Rates)
	require.NotEmpty(t, replaced.Fitted.Areas)
	require.Len(t, replaced.Fitted.Curves, 2)
	require.NotEmpty(t, wantDays)
	assert.Equal(t, wantDays, replaced.Days,
		"every fit stores the out-of-sample track record, in the organization's own days")
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

			quality, err := newService(tt.repo, nil, fakeOrganizations{}).Fit(
				t.Context(),
				tenantInfo,
			)

			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, quality)
			assert.Len(t, tt.repo.replaced, tt.wantReplaced)
		})
	}
}

func TestStanding_RefitsAForecastThatIsNotReadyAndWasFittedOverAnHourAgo(t *testing.T) {
	t.Parallel()

	fresh := testNow - demandforecast.RefitAfterSeconds + 1
	old := testNow - demandforecast.RefitAfterSeconds
	tests := []struct {
		name    string
		quality *demandforecast.Quality
		refits  bool
	}{
		{
			name:    "never fitted",
			quality: &demandforecast.Quality{Status: demandforecast.StatusInsufficientHistory},
			refits:  true,
		},
		{
			name: "short of history an hour ago",
			quality: &demandforecast.Quality{
				Status:   demandforecast.StatusInsufficientHistory,
				FittedAt: old,
			},
			refits: true,
		},
		{
			name:    "no skill an hour ago",
			quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill, FittedAt: old},
			refits:  true,
		},
		{
			name: "short of history within the hour",
			quality: &demandforecast.Quality{
				Status:   demandforecast.StatusInsufficientHistory,
				FittedAt: fresh,
			},
		},
		{
			name:    "ready, however old",
			quality: &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fits := &fakeFits{}
			svc := newServiceWithFits(&fakeRepository{quality: tt.quality}, fits, nil, nil)

			standing, err := svc.Standing(t.Context(), tenantInfo)

			require.NoError(t, err)
			assert.Same(t, tt.quality, standing.Quality)
			assert.Equal(t, tt.refits, standing.Refitting)
			if tt.refits {
				assert.Equal(t, []pagination.TenantInfo{tenantInfo}, fits.started)
			} else {
				assert.Empty(t, fits.started)
			}
		})
	}
}

func TestStanding_StillAnswersWhenTheRefitCannotStart(t *testing.T) {
	t.Parallel()

	quality := &demandforecast.Quality{Status: demandforecast.StatusInsufficientHistory}
	fits := &fakeFits{err: errors.New("temporal down")}
	svc := newServiceWithFits(&fakeRepository{quality: quality}, fits, nil, nil)

	standing, err := svc.Standing(t.Context(), tenantInfo)

	require.NoError(t, err)
	assert.Same(t, quality, standing.Quality)
	assert.False(t, standing.Refitting)
	assert.Len(t, fits.started, 1)
}

func TestStanding_ReturnsTheRepositoryErrorWithoutRefitting(t *testing.T) {
	t.Parallel()

	down := errors.New("quality down")
	fits := &fakeFits{}
	svc := newServiceWithFits(&fakeRepository{qualityErr: down}, fits, nil, nil)

	standing, err := svc.Standing(t.Context(), tenantInfo)

	require.ErrorIs(t, err, down)
	assert.Nil(t, standing)
	assert.Empty(t, fits.started)
}

func TestFit_FailsWhenTheHolidayCalendarCannotBeRead(t *testing.T) {
	t.Parallel()

	down := errors.New("holidays down")
	repo := &fakeRepository{rows: fitRows(dispatchmarket.WeekOf(testNow))}
	svc := newService(repo, nil, fakeOrganizations{})
	svc.holidays = &fakeHolidays{err: down}

	quality, err := svc.Fit(t.Context(), tenantInfo)

	require.ErrorIs(t, err, down)
	assert.Nil(t, quality)
	assert.Empty(
		t,
		repo.replaced,
		"a fit that cannot tell holidays from working days saves nothing",
	)
}

func TestForecaster_ReadsTheModelOncePerFit(t *testing.T) {
	t.Parallel()

	cell := cellOf(t, dallasPoint)
	fitted := func(at int64) *demandforecast.Quality {
		return &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: at}
	}
	repo := &fakeRepository{quality: fitted(100)}
	repo.model = &demandforecastports.Model{
		Quality: repo.quality,
		Rates:   []*demandforecast.Rate{{Cell: cell, Slot: 3, Alpha: 4, Beta: 2}},
		Areas: []*demandforecast.Area{
			{Cell: cell, Label: "Dallas, TX", MedianRevenueCents: 45_000},
		},
	}
	svc := newService(repo, nil, nil)
	req := &demandforecastports.ForecasterRequest{TenantInfo: tenantInfo, Now: testNow}

	first, err := svc.Forecaster(t.Context(), req)
	require.NoError(t, err)
	second, err := svc.Forecaster(t.Context(), req)
	require.NoError(t, err)

	assert.True(t, first.Ready())
	assert.Equal(t, 1, repo.modelReads, "the second read reuses the fitted model")
	assert.Same(t, first.Index, second.Index)

	repo.quality = fitted(200)
	repo.model.Quality = repo.quality
	third, err := svc.Forecaster(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, 2, repo.modelReads, "a new fit is read once")
	assert.NotSame(t, first.Index, third.Index)

	other, err := svc.Forecaster(t.Context(), &demandforecastports.ForecasterRequest{
		TenantInfo: pagination.TenantInfo{OrgID: "org_b", BuID: bu},
		Now:        testNow,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.modelReads, "another organization never shares a model")
	assert.NotSame(t, third.Index, other.Index)
}

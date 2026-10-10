package weatheralertservice

import (
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type repoStub struct {
	tenants           []pagination.TenantInfo
	activeAlerts      []*weatheralert.WeatherAlert
	alert             *weatheralert.WeatherAlert
	activities        []*weatheralert.Activity
	upserted          []*weatheralert.WeatherAlert
	stored            map[string]*weatheralert.WeatherAlert
	failUpsertFor     pulid.ID
	listByNWSIDsCalls int
	expireCalls       int
}

func storedKey(orgID pulid.ID, nwsID string) string {
	return orgID.String() + "|" + nwsID
}

func (s *repoStub) ListTenants(context.Context) ([]pagination.TenantInfo, error) {
	return s.tenants, nil
}

func (s *repoStub) GetActiveAlerts(
	context.Context,
	pagination.TenantInfo,
) ([]*weatheralert.WeatherAlert, error) {
	return s.activeAlerts, nil
}

func (s *repoStub) GetByID(
	context.Context,
	repositories.GetWeatherAlertByIDRequest,
) (*weatheralert.WeatherAlert, error) {
	return s.alert, nil
}

func (s *repoStub) GetActivities(
	context.Context,
	repositories.GetWeatherAlertByIDRequest,
) ([]*weatheralert.Activity, error) {
	return s.activities, nil
}

func (s *repoStub) ListByNWSIDs(
	_ context.Context,
	req repositories.ListWeatherAlertsByNWSIDsRequest,
) ([]*weatheralert.WeatherAlert, error) {
	s.listByNWSIDsCalls++
	alerts := make([]*weatheralert.WeatherAlert, 0, len(req.NWSIDs))
	for _, nwsID := range req.NWSIDs {
		if alert, ok := s.stored[storedKey(req.TenantInfo.OrgID, nwsID)]; ok {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

func (s *repoStub) UpsertAlert(
	_ context.Context,
	alert *weatheralert.WeatherAlert,
) (*repositories.UpsertWeatherAlertResult, error) {
	if !s.failUpsertFor.IsNil() && alert.OrganizationID == s.failUpsertFor {
		return nil, errors.New("database unavailable")
	}

	s.upserted = append(s.upserted, alert)
	if s.stored == nil {
		s.stored = make(map[string]*weatheralert.WeatherAlert)
	}
	s.stored[storedKey(alert.OrganizationID, alert.NWSID)] = alert

	return &repositories.UpsertWeatherAlertResult{Alert: alert}, nil
}

func (s *repoStub) ExpireStaleAlerts(
	context.Context,
) (*repositories.ExpireWeatherAlertsResult, error) {
	s.expireCalls++
	return &repositories.ExpireWeatherAlertsResult{}, nil
}

type feedStateStub struct {
	state *repositories.WeatherAlertFeedState
	saves int
}

func (s *feedStateStub) Get(context.Context) (*repositories.WeatherAlertFeedState, error) {
	if s.state == nil {
		return nil, nil //nolint:nilnil // nil means no state was saved
	}

	state := *s.state
	return &state, nil
}

func (s *feedStateStub) Save(
	_ context.Context,
	state *repositories.WeatherAlertFeedState,
	_ time.Duration,
) error {
	saved := *state
	s.state = &saved
	s.saves++
	return nil
}

type nwsRequest struct {
	method         string
	userAgent      string
	acceptEncoding string
	ifNoneMatch    string
}

type nwsServer struct {
	*httptest.Server

	mu                sync.Mutex
	body              string
	etag              string
	honorsIfNoneMatch bool
	requests          []nwsRequest
}

func newNWSServer(t *testing.T, body, etag string) *nwsServer {
	t.Helper()

	srv := &nwsServer{body: body, etag: etag}
	srv.Server = httptest.NewServer(http.HandlerFunc(srv.serve))
	t.Cleanup(srv.Close)

	return srv
}

func (s *nwsServer) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, nwsRequest{
		method:         r.Method,
		userAgent:      r.Header.Get("User-Agent"),
		acceptEncoding: r.Header.Get("Accept-Encoding"),
		ifNoneMatch:    r.Header.Get("If-None-Match"),
	})

	if s.etag != "" {
		w.Header().Set("ETag", s.etag)
	}
	if s.honorsIfNoneMatch && s.etag != "" && r.Header.Get("If-None-Match") == s.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/geo+json")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		_, _ = w.Write([]byte(s.body))
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	_, _ = gz.Write([]byte(s.body))
	_ = gz.Close()
}

func (s *nwsServer) setFeed(body, etag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.body = body
	s.etag = etag
}

func (s *nwsServer) recorded() []nwsRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]nwsRequest(nil), s.requests...)
}

func nwsFeature(id, event, expires string) string {
	return `{"geometry":{"type":"Polygon","coordinates":[[[-97,32],[-96,32],[-96,33],[-97,33],[-97,32]]]},` +
		`"properties":{"id":"` + id + `","event":"` + event + `","severity":"Severe",` +
		`"messageType":"Alert","effective":"2026-04-15T12:00:00Z","expires":"` + expires + `"}}`
}

func nwsFeedBody(features ...string) string {
	return `{"type":"FeatureCollection","features":[` + strings.Join(features, ",") + `]}`
}

func newTenants(n int) []pagination.TenantInfo {
	tenants := make([]pagination.TenantInfo, 0, n)
	for range n {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		})
	}

	return tenants
}

func newPollingService(
	repo *repoStub,
	feedState *feedStateStub,
	srv *nwsServer,
) *Service {
	service := New(Params{Logger: zap.NewNop(), Repo: repo, FeedState: feedState})
	service.feedURL = srv.URL

	return service
}

func TestMapAlertCategory(t *testing.T) {
	t.Parallel()

	service := New(Params{Logger: zap.NewNop(), Repo: &repoStub{}})

	assert.Equal(
		t,
		weatheralert.AlertCategoryWinterWeather,
		service.mapAlertCategory("Winter Storm Warning"),
	)
	assert.Equal(
		t,
		weatheralert.AlertCategoryWindStorm,
		service.mapAlertCategory("High Wind Watch"),
	)
	assert.Equal(
		t,
		weatheralert.AlertCategoryFloodWater,
		service.mapAlertCategory("Flash Flood Warning"),
	)
	assert.Equal(t, weatheralert.AlertCategoryFire, service.mapAlertCategory("Red Flag Warning"))
	assert.Equal(
		t,
		weatheralert.AlertCategoryHeat,
		service.mapAlertCategory("Excessive Heat Warning"),
	)
	assert.Equal(
		t,
		weatheralert.AlertCategoryTornadoSevereStorm,
		service.mapAlertCategory("Severe Thunderstorm Warning"),
	)
	assert.Equal(
		t,
		weatheralert.AlertCategoryTropicalStormHurricane,
		service.mapAlertCategory("Tropical Storm Warning"),
	)
	assert.Equal(
		t,
		weatheralert.AlertCategoryWindStorm,
		service.mapAlertCategory("Marine Weather Statement"),
	)
	assert.Equal(t, weatheralert.AlertCategoryOther, service.mapAlertCategory("Dense Fog Advisory"))
}

func TestPollNWSAlertsFetchesTheFeedOnceForEveryTenant(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
		`{"geometry":null,"properties":{"id":"urn:oid:2","event":"Flood Warning","messageType":"Alert"}}`,
	), `W/"feed-1"`)
	repo := &repoStub{tenants: newTenants(5)}
	feedState := &feedStateStub{}
	service := newPollingService(repo, feedState, srv)

	var beats int
	result, err := service.PollNWSAlerts(t.Context(), func(...any) { beats++ })
	require.NoError(t, err)

	requests := srv.recorded()
	require.Len(t, requests, 1, "one NWS request per poll, whatever the tenant count")
	assert.Equal(t, http.MethodGet, requests[0].method)
	assert.Equal(t, nwsUserAgent, requests[0].userAgent)
	assert.Contains(t, requests[0].acceptEncoding, "gzip")
	assert.Empty(t, requests[0].ifNoneMatch, "nothing is known about the feed yet")

	assert.False(t, result.FeedUnchanged)
	assert.Equal(t, 1, result.AlertsInFeed, "an alert without geometry is skipped")
	assert.Equal(t, 5, result.TenantsScanned)
	require.Len(t, result.Tenants, 5)
	for i, synced := range result.Tenants {
		require.NoError(t, synced.Err)
		assert.Equal(t, repo.tenants[i], synced.TenantInfo)
		assert.Equal(t, 1, synced.Written)
		assert.Zero(t, synced.Unchanged)
	}
	require.Len(t, repo.upserted, 5)
	assert.Equal(t, weatheralert.AlertCategoryWinterWeather, repo.upserted[0].AlertCategory)
	assert.Equal(t, repo.tenants[0].OrgID, repo.upserted[0].OrganizationID)
	assert.GreaterOrEqual(t, beats, 5)

	require.Equal(t, 1, feedState.saves)
	assert.Equal(t, `W/"feed-1"`, feedState.state.ETag)
	assert.Equal(t, digestTenants(repo.tenants), feedState.state.TenantsDigest)
}

func TestPollNWSAlertsSkipsAnUnchangedFeed(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
	), `W/"feed-1"`)
	repo := &repoStub{tenants: newTenants(3)}
	feedState := &feedStateStub{}
	service := newPollingService(repo, feedState, srv)

	_, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, repo.upserted, 3)
	listCalls := repo.listByNWSIDsCalls

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	requests := srv.recorded()
	require.Len(t, requests, 2)
	assert.Equal(t, `W/"feed-1"`, requests[1].ifNoneMatch)
	assert.True(t, result.FeedUnchanged)
	assert.Equal(t, 3, result.TenantsScanned)
	assert.Empty(t, result.Tenants)
	assert.Len(t, repo.upserted, 3, "an unchanged feed writes nothing")
	assert.Equal(t, listCalls, repo.listByNWSIDsCalls, "an unchanged feed reads nothing")
	assert.Equal(t, 1, feedState.saves)
}

func TestPollNWSAlertsTreatsNotModifiedAsUnchanged(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
	), `"feed-1"`)
	srv.honorsIfNoneMatch = true
	repo := &repoStub{tenants: newTenants(2)}
	feedState := &feedStateStub{state: &repositories.WeatherAlertFeedState{
		ETag:          `"feed-1"`,
		TenantsDigest: digestTenants(repo.tenants),
	}}
	service := newPollingService(repo, feedState, srv)

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	requests := srv.recorded()
	require.Len(t, requests, 1)
	assert.Equal(t, `"feed-1"`, requests[0].ifNoneMatch)
	assert.True(t, result.FeedUnchanged)
	assert.Empty(t, repo.upserted)
	assert.Zero(t, repo.listByNWSIDsCalls)
	assert.Zero(t, feedState.saves)
}

func TestPollNWSAlertsSyncsEveryTenantWhenTheTenantsChange(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
	), `W/"feed-1"`)
	srv.honorsIfNoneMatch = true
	repo := &repoStub{tenants: newTenants(2)}
	feedState := &feedStateStub{state: &repositories.WeatherAlertFeedState{
		ETag:          `W/"feed-1"`,
		TenantsDigest: digestTenants(repo.tenants[:1]),
	}}
	service := newPollingService(repo, feedState, srv)

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	requests := srv.recorded()
	require.Len(t, requests, 1)
	assert.Empty(t, requests[0].ifNoneMatch, "a new organization has not seen this feed")
	assert.False(t, result.FeedUnchanged)
	assert.Len(t, repo.upserted, 2)
	assert.Equal(t, digestTenants(repo.tenants), feedState.state.TenantsDigest)
}

func TestPollNWSAlertsSkipsAlertsThatAreAlreadyStored(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
		nwsFeature("urn:oid:2", "Flood Warning", "2026-04-15T18:00:00Z"),
	), `W/"feed-1"`)
	repo := &repoStub{tenants: newTenants(2)}
	feedState := &feedStateStub{}
	service := newPollingService(repo, feedState, srv)

	_, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, repo.upserted, 4)

	srv.setFeed(nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
		nwsFeature("urn:oid:2", "Flood Warning", "2026-04-15T21:00:00Z"),
		nwsFeature("urn:oid:3", "Heat Advisory", "2026-04-15T21:00:00Z"),
	), `W/"feed-2"`)

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	assert.False(t, result.FeedUnchanged)
	require.Len(t, result.Tenants, 2)
	for _, synced := range result.Tenants {
		require.NoError(t, synced.Err)
		assert.Equal(t, 2, synced.Written, "the extended alert and the new one are written")
		assert.Equal(t, 1, synced.Unchanged, "the alert whose message did not change is not")
	}

	rewritten := make([]string, 0, 4)
	for _, alert := range repo.upserted[4:] {
		rewritten = append(rewritten, alert.NWSID)
	}
	assert.ElementsMatch(
		t,
		[]string{"urn:oid:2", "urn:oid:3", "urn:oid:2", "urn:oid:3"},
		rewritten,
	)
	assert.Equal(t, `W/"feed-2"`, feedState.state.ETag)
}

func TestPollNWSAlertsIsolatesATenantFailure(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
	), `W/"feed-1"`)
	tenants := newTenants(3)
	repo := &repoStub{tenants: tenants, failUpsertFor: tenants[1].OrgID}
	feedState := &feedStateStub{}
	service := newPollingService(repo, feedState, srv)

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	require.Len(t, result.Tenants, 3)
	require.NoError(t, result.Tenants[0].Err)
	require.Error(t, result.Tenants[1].Err)
	require.NoError(t, result.Tenants[2].Err)
	assert.Len(t, repo.upserted, 2)
	assert.Zero(t, feedState.saves, "a feed one tenant missed is fetched in full next time")

	repo.failUpsertFor = ""
	result, err = service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	assert.False(t, result.FeedUnchanged)
	require.Len(t, result.Tenants, 3)
	assert.Equal(t, 1, result.Tenants[1].Written, "the tenant that failed catches up")
	assert.Equal(t, 1, result.Tenants[0].Unchanged)
	assert.Equal(t, 1, feedState.saves)
}

func TestPollNWSAlertsWithoutTenantsDoesNotFetch(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(), `W/"feed-1"`)
	service := newPollingService(&repoStub{}, &feedStateStub{}, srv)

	result, err := service.PollNWSAlerts(t.Context(), nil)
	require.NoError(t, err)

	assert.Zero(t, result.TenantsScanned)
	assert.Empty(t, srv.recorded())
}

func TestPollNWSAlertsFailsOnAnUnexpectedStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	repo := &repoStub{tenants: newTenants(1)}
	feedState := &feedStateStub{}
	service := New(Params{Logger: zap.NewNop(), Repo: repo, FeedState: feedState})
	service.feedURL = srv.URL

	_, err := service.PollNWSAlerts(t.Context(), nil)
	require.Error(t, err)
	assert.Empty(t, repo.upserted)
	assert.Zero(t, feedState.saves)
}

func TestPollNWSAlertsForTenantSyncsOneTenant(t *testing.T) {
	t.Parallel()

	srv := newNWSServer(t, nwsFeedBody(
		nwsFeature("urn:oid:1", "Winter Storm Warning", "2026-04-15T18:00:00Z"),
	), `W/"feed-1"`)
	tenants := newTenants(2)
	repo := &repoStub{tenants: tenants}
	service := newPollingService(repo, &feedStateStub{}, srv)

	require.NoError(t, service.PollNWSAlertsForTenant(t.Context(), tenants[1]))
	require.NoError(t, service.PollNWSAlertsForTenant(t.Context(), tenants[1]))

	require.Len(t, srv.recorded(), 2)
	require.Len(t, repo.upserted, 1, "the second pass finds the alert already stored")
	assert.Equal(t, tenants[1].OrgID, repo.upserted[0].OrganizationID)
}

func TestGetActiveAlertsReturnsFeatureCollection(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Flood Warning",
				AlertCategory: weatheralert.AlertCategoryFloodWater,
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, result.Features, 1)
	assert.Equal(t, "FeatureCollection", result.Type)
	assert.Equal(t, "Flood Warning", result.Features[0].Properties.Event)
}

func TestGetActiveAlertsSkipsExpiredAlerts(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	expiredAt := time.Now().UTC().Add(-1 * time.Minute).Unix()
	expires := time.Now().UTC().Add(-1 * time.Minute).Unix()
	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Flood Warning",
				AlertCategory: weatheralert.AlertCategoryFloodWater,
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
				ExpiredAt:     &expiredAt,
				Expires:       &expires,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	require.Empty(t, result.Features)
}

func TestGetAlertDetailReturnsActivities(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	alertID := pulid.MustNew("walt_")
	repo := &repoStub{
		alert: &weatheralert.WeatherAlert{
			ID:            alertID,
			NWSID:         "urn:oid:1",
			Event:         "Flood Warning",
			AlertCategory: weatheralert.AlertCategoryFloodWater,
			Geometry:      geometry,
			FirstSeenAt:   1,
			LastUpdatedAt: 2,
		},
		activities: []*weatheralert.Activity{
			{WeatherAlertID: alertID, ActivityType: weatheralert.ActivityTypeIssued, Timestamp: 1},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	result, err := service.GetAlertDetail(
		t.Context(),
		&serviceports.GetWeatherAlertDetailRequest{ID: alertID},
	)
	require.NoError(t, err)
	require.NotNil(t, result.Feature)
	require.Len(t, result.Activities, 1)
}

func TestGetActiveAlertsRejectsLegacyAlertCategory(t *testing.T) {
	t.Parallel()

	geometry, err := parseGeometry(map[string]any{
		"type": "Polygon",
		"coordinates": []any{
			[]any{
				[]any{-97.0, 32.0},
				[]any{-96.0, 32.0},
				[]any{-96.0, 33.0},
				[]any{-97.0, 33.0},
				[]any{-97.0, 32.0},
			},
		},
	})
	require.NoError(t, err)

	repo := &repoStub{
		activeAlerts: []*weatheralert.WeatherAlert{
			{
				ID:            pulid.MustNew("walt_"),
				NWSID:         "urn:oid:1",
				Event:         "Coastal Flood Warning",
				AlertCategory: weatheralert.AlertCategory("coastal_marine_tsunami"),
				Geometry:      geometry,
				FirstSeenAt:   1,
				LastUpdatedAt: 2,
			},
		},
	}
	service := New(Params{Logger: zap.NewNop(), Repo: repo})

	_, err = service.GetActiveAlerts(t.Context(), pagination.TenantInfo{})
	require.Error(t, err)
}

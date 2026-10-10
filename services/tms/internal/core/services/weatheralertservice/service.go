//nolint:gocritic // existing legacy workflow/API shape is intentionally kept stable
package weatheralertservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/paulmach/orb/geojson"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	nwsAlertsURL            = "https://api.weather.gov/alerts/active"
	nwsUserAgent            = "Trenova TMS (trenova.app)"
	nwsFeedStateTTL         = 24 * time.Hour
	upsertHeartbeatInterval = 50
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Repo      repositories.WeatherAlertRepository
	FeedState repositories.WeatherAlertFeedStateStore
}

type Service struct {
	logger     *zap.Logger
	repo       repositories.WeatherAlertRepository
	feedState  repositories.WeatherAlertFeedStateStore
	httpClient *http.Client
	feedURL    string
}

type nwsFeed struct {
	state       repositories.WeatherAlertFeedState
	notModified bool
	alerts      []*weatheralert.WeatherAlert
	nwsIDs      []string
}

type nwsActiveAlertsResponse struct {
	Features []nwsAlertFeature `json:"features"`
}

type nwsAlertFeature struct {
	Geometry   map[string]any     `json:"geometry"`
	Properties nwsAlertProperties `json:"properties"`
}

type nwsAlertProperties struct {
	ID          string `json:"id"`
	Event       string `json:"event"`
	Severity    string `json:"severity"`
	Urgency     string `json:"urgency"`
	Certainty   string `json:"certainty"`
	Headline    string `json:"headline"`
	Description string `json:"description"`
	Instruction string `json:"instruction"`
	AreaDesc    string `json:"areaDesc"`
	Effective   string `json:"effective"`
	Expires     string `json:"expires"`
	Onset       string `json:"onset"`
	Ends        string `json:"ends"`
	Status      string `json:"status"`
	MessageType string `json:"messageType"`
	SenderName  string `json:"senderName"`
	Response    string `json:"response"`
	Category    string `json:"category"`
}

func New(p Params) *Service {
	return &Service{
		logger:    p.Logger.Named("service.weather-alert"),
		repo:      p.Repo,
		feedState: p.FeedState,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		feedURL: nwsAlertsURL,
	}
}

func (s *Service) PollNWSAlerts(
	ctx context.Context,
	heartbeat serviceports.WeatherAlertPollHeartbeat,
) (*serviceports.PollNWSAlertsResult, error) {
	tenants, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list tenants for weather alerts").
			WithInternal(err)
	}

	result := &serviceports.PollNWSAlertsResult{TenantsScanned: len(tenants)}
	if len(tenants) == 0 {
		return result, nil
	}

	tenantsDigest := digestTenants(tenants)
	feed, err := s.fetchActiveAlerts(ctx, s.knownFeedState(ctx, tenantsDigest))
	if err != nil {
		return nil, err
	}
	if feed.notModified {
		result.FeedUnchanged = true
		return result, nil
	}

	result.AlertsInFeed = len(feed.alerts)
	result.Tenants = make([]serviceports.WeatherAlertTenantSync, 0, len(tenants))
	allSynced := true
	for _, tenantInfo := range tenants {
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		beat(heartbeat, "syncing-nws-alerts", tenantInfo.OrgID.String())
		synced := s.syncTenant(ctx, tenantInfo, feed, heartbeat)
		if synced.Err != nil {
			allSynced = false
		}
		result.Tenants = append(result.Tenants, synced)
	}

	if allSynced {
		feed.state.TenantsDigest = tenantsDigest
		s.saveFeedState(ctx, &feed.state)
	}

	return result, nil
}

func (s *Service) ListWeatherAlertTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	tenants, err := s.repo.ListTenants(ctx)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list tenants for weather alerts").
			WithInternal(err)
	}
	if limit > 0 && len(tenants) > limit {
		tenants = tenants[:limit]
	}

	return tenants, nil
}

func (s *Service) PollNWSAlertsForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	feed, err := s.fetchActiveAlerts(ctx, nil)
	if err != nil {
		return err
	}

	return s.syncTenant(ctx, tenantInfo, feed, nil).Err
}

func (s *Service) syncTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	feed *nwsFeed,
	heartbeat serviceports.WeatherAlertPollHeartbeat,
) serviceports.WeatherAlertTenantSync {
	synced := serviceports.WeatherAlertTenantSync{TenantInfo: tenantInfo}
	tenantCtx := dbscope.WithTenant(ctx, tenantInfo.DBTenant())

	stored, err := s.repo.ListByNWSIDs(tenantCtx, repositories.ListWeatherAlertsByNWSIDsRequest{
		TenantInfo: tenantInfo,
		NWSIDs:     feed.nwsIDs,
	})
	if err != nil {
		synced.Err = errortypes.NewBusinessError("failed to read stored weather alerts").
			WithInternal(err)
		return synced
	}

	storedByNWSID := make(map[string]*weatheralert.WeatherAlert, len(stored))
	for _, alert := range stored {
		storedByNWSID[alert.NWSID] = alert
	}

	for _, alert := range feed.alerts {
		if alert.IsSameMessage(storedByNWSID[alert.NWSID]) {
			synced.Unchanged++
			continue
		}

		_, err = s.repo.UpsertAlert(tenantCtx, cloneAlertForTenant(alert, tenantInfo))
		if err != nil {
			synced.Err = errortypes.NewBusinessError("failed to upsert weather alert").
				WithInternal(err)
			return synced
		}

		synced.Written++
		if synced.Written%upsertHeartbeatInterval == 0 {
			beat(heartbeat, "syncing-nws-alerts", tenantInfo.OrgID.String(), synced.Written)
		}
	}

	return synced
}

func (s *Service) knownFeedState(
	ctx context.Context,
	tenantsDigest string,
) *repositories.WeatherAlertFeedState {
	state, err := s.feedState.Get(ctx)
	if err != nil {
		s.logger.Warn("failed to read NWS feed state; fetching the full feed", zap.Error(err))
		return nil
	}
	if state == nil || state.TenantsDigest != tenantsDigest {
		return nil
	}

	return state
}

func (s *Service) saveFeedState(ctx context.Context, state *repositories.WeatherAlertFeedState) {
	if state.ETag == "" && state.LastModified == "" {
		return
	}

	if err := s.feedState.Save(ctx, state, nwsFeedStateTTL); err != nil {
		s.logger.Warn("failed to save NWS feed state", zap.Error(err))
	}
}

func digestTenants(tenants []pagination.TenantInfo) string {
	var b strings.Builder
	b.Grow(len(tenants) * 64)
	for _, tenantInfo := range tenants {
		b.WriteString(tenantInfo.OrgID.String())
		b.WriteByte(':')
		b.WriteString(tenantInfo.BuID.String())
		b.WriteByte('\n')
	}

	return hashutils.SHA256Hex(b.String())
}

func isSameFeed(
	known *repositories.WeatherAlertFeedState,
	received *repositories.WeatherAlertFeedState,
) bool {
	if known == nil {
		return false
	}
	if received.ETag != "" {
		return received.ETag == known.ETag
	}

	return received.LastModified != "" && received.LastModified == known.LastModified
}

func beat(heartbeat serviceports.WeatherAlertPollHeartbeat, details ...any) {
	if heartbeat != nil {
		heartbeat(details...)
	}
}

func (s *Service) ExpireStaleWeatherAlerts(ctx context.Context) error {
	if _, err := s.repo.ExpireStaleAlerts(ctx); err != nil {
		return errortypes.NewBusinessError("failed to expire stale weather alerts").
			WithInternal(err)
	}

	return nil
}

func (s *Service) GetActiveAlerts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*serviceports.WeatherAlertFeatureCollection, error) {
	alerts, err := s.repo.GetActiveAlerts(ctx, tenantInfo)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to retrieve active weather alerts").
			WithInternal(err)
	}

	features := make([]*serviceports.WeatherAlertFeature, 0, len(alerts))
	now := time.Now().UTC().Unix()
	for _, alert := range alerts {
		if !isAlertActive(alert, now) {
			continue
		}

		feature, convErr := toFeature(alert)
		if convErr != nil {
			return nil, convErr
		}
		features = append(features, feature)
	}

	return &serviceports.WeatherAlertFeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}, nil
}

func isAlertActive(alert *weatheralert.WeatherAlert, now int64) bool {
	if alert == nil {
		return false
	}

	if alert.ExpiredAt != nil && *alert.ExpiredAt <= now {
		return false
	}

	if alert.Expires != nil && *alert.Expires <= now {
		return false
	}

	return true
}

func (s *Service) GetAlertDetail(
	ctx context.Context,
	req *serviceports.GetWeatherAlertDetailRequest,
) (*serviceports.WeatherAlertDetail, error) {
	alert, err := s.repo.GetByID(ctx, repositories.GetWeatherAlertByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	activities, err := s.repo.GetActivities(ctx, repositories.GetWeatherAlertByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to retrieve weather alert activities").
			WithInternal(err)
	}

	feature, err := toFeature(alert)
	if err != nil {
		return nil, err
	}

	return &serviceports.WeatherAlertDetail{
		Feature:    feature,
		Activities: activities,
	}, nil
}

func (s *Service) fetchActiveAlerts(
	ctx context.Context,
	known *repositories.WeatherAlertFeedState,
) (*nwsFeed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.feedURL, http.NoBody)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to build NWS weather alerts request").
			WithInternal(err)
	}

	req.Header.Set("User-Agent", nwsUserAgent)
	req.Header.Set("Accept", "application/geo+json")
	if known != nil {
		if known.ETag != "" {
			req.Header.Set("If-None-Match", known.ETag)
		}
		if known.LastModified != "" {
			req.Header.Set("If-Modified-Since", known.LastModified)
		}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to fetch NWS weather alerts").
			WithInternal(err)
	}
	defer resp.Body.Close()

	feed := &nwsFeed{
		state: repositories.WeatherAlertFeedState{
			ETag:         resp.Header.Get("ETag"),
			LastModified: resp.Header.Get("Last-Modified"),
		},
	}

	switch {
	case resp.StatusCode == http.StatusNotModified && known != nil:
		feed.notModified = true
		return feed, nil
	case resp.StatusCode != http.StatusOK:
		return nil, errortypes.NewBusinessError(
			"NWS weather alerts request failed with status {0}", resp.StatusCode,
		)
	case isSameFeed(known, &feed.state):
		feed.notModified = true
		return feed, nil
	}

	payload := new(nwsActiveAlertsResponse)
	if err = sonic.ConfigDefault.NewDecoder(resp.Body).Decode(payload); err != nil {
		return nil, errortypes.NewBusinessError("failed to decode NWS weather alerts response").
			WithInternal(err)
	}

	feed.alerts, err = s.parseAlerts(payload)
	if err != nil {
		return nil, err
	}

	feed.nwsIDs = make([]string, 0, len(feed.alerts))
	for _, alert := range feed.alerts {
		feed.nwsIDs = append(feed.nwsIDs, alert.NWSID)
	}

	return feed, nil
}

func (s *Service) parseAlerts(
	payload *nwsActiveAlertsResponse,
) ([]*weatheralert.WeatherAlert, error) {
	alerts := make([]*weatheralert.WeatherAlert, 0, len(payload.Features))
	now := time.Now().UTC().Unix()
	for _, feature := range payload.Features {
		if feature.Geometry == nil {
			continue
		}

		geometry, err := parseGeometry(feature.Geometry)
		if err != nil {
			return nil, errortypes.NewBusinessError("failed to parse NWS weather alert geometry").
				WithInternal(err)
		}

		effective, err := parseNWSTime(feature.Properties.Effective)
		if err != nil {
			return nil, err
		}
		expires, err := parseNWSTime(feature.Properties.Expires)
		if err != nil {
			return nil, err
		}
		onset, err := parseNWSTime(feature.Properties.Onset)
		if err != nil {
			return nil, err
		}
		ends, err := parseNWSTime(feature.Properties.Ends)
		if err != nil {
			return nil, err
		}

		alert := &weatheralert.WeatherAlert{
			NWSID:         strings.TrimSpace(feature.Properties.ID),
			Event:         strings.TrimSpace(feature.Properties.Event),
			Severity:      strings.TrimSpace(feature.Properties.Severity),
			Urgency:       strings.TrimSpace(feature.Properties.Urgency),
			Certainty:     strings.TrimSpace(feature.Properties.Certainty),
			Headline:      strings.TrimSpace(feature.Properties.Headline),
			Description:   strings.TrimSpace(feature.Properties.Description),
			Instruction:   strings.TrimSpace(feature.Properties.Instruction),
			AreaDesc:      strings.TrimSpace(feature.Properties.AreaDesc),
			Effective:     effective,
			Expires:       expires,
			Onset:         onset,
			Ends:          ends,
			Status:        strings.TrimSpace(feature.Properties.Status),
			MessageType:   strings.TrimSpace(feature.Properties.MessageType),
			SenderName:    strings.TrimSpace(feature.Properties.SenderName),
			Response:      strings.TrimSpace(feature.Properties.Response),
			Category:      strings.TrimSpace(feature.Properties.Category),
			AlertCategory: s.mapAlertCategory(feature.Properties.Event),
			Geometry:      geometry,
			FirstSeenAt:   now,
			LastUpdatedAt: now,
		}

		if alert.NWSID == "" || alert.Event == "" {
			continue
		}

		if strings.EqualFold(alert.MessageType, "Cancel") {
			alert.ExpiredAt = &now
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (s *Service) mapAlertCategory(event string) weatheralert.AlertCategory {
	normalized := strings.ToLower(strings.TrimSpace(event))

	switch {
	case containsAny(normalized, "winter storm", "ice storm", "blizzard", "frost", "freeze", "wind chill", "snow"):
		return weatheralert.AlertCategoryWinterWeather
	case containsAny(normalized, "tornado", "severe thunderstorm", "severe weather"):
		return weatheralert.AlertCategoryTornadoSevereStorm
	case containsAny(normalized, "hurricane", "tropical storm", "tropical cyclone", "typhoon"):
		return weatheralert.AlertCategoryTropicalStormHurricane
	case containsAny(normalized, "high wind", "wind advisory", "extreme wind", "storm", "marine", "beach hazard", "rip current", "lakeshore"):
		return weatheralert.AlertCategoryWindStorm
	case containsAny(normalized, "flood", "flash flood", "hydrologic", "river", "coastal flood", "storm surge", "tsunami"):
		return weatheralert.AlertCategoryFloodWater
	case containsAny(normalized, "red flag", "fire weather", "fire warning"):
		return weatheralert.AlertCategoryFire
	case containsAny(normalized, "heat", "excessive heat"):
		return weatheralert.AlertCategoryHeat
	default:
		return weatheralert.AlertCategoryOther
	}
}

func containsAny(value string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}

	return false
}

func cloneAlertForTenant(
	alert *weatheralert.WeatherAlert,
	tenantInfo pagination.TenantInfo,
) *weatheralert.WeatherAlert {
	return &weatheralert.WeatherAlert{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		NWSID:          alert.NWSID,
		Event:          alert.Event,
		Severity:       alert.Severity,
		Urgency:        alert.Urgency,
		Certainty:      alert.Certainty,
		Headline:       alert.Headline,
		Description:    alert.Description,
		Instruction:    alert.Instruction,
		AreaDesc:       alert.AreaDesc,
		Effective:      cloneInt64Ptr(alert.Effective),
		Expires:        cloneInt64Ptr(alert.Expires),
		Onset:          cloneInt64Ptr(alert.Onset),
		Ends:           cloneInt64Ptr(alert.Ends),
		Status:         alert.Status,
		MessageType:    alert.MessageType,
		SenderName:     alert.SenderName,
		Response:       alert.Response,
		Category:       alert.Category,
		AlertCategory:  alert.AlertCategory,
		Geometry:       alert.Geometry,
		FirstSeenAt:    alert.FirstSeenAt,
		LastUpdatedAt:  alert.LastUpdatedAt,
		ExpiredAt:      cloneInt64Ptr(alert.ExpiredAt),
	}
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func parseGeometry(raw map[string]any) (*postgis.Geometry, error) {
	payload, err := sonic.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal geometry: %w", err)
	}

	geo, err := geojson.UnmarshalGeometry(payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal geometry: %w", err)
	}

	return &postgis.Geometry{Geometry: geo.Geometry()}, nil
}

func parseNWSTime(value string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to parse NWS alert time").WithInternal(err)
	}

	unix := parsed.Unix()
	return &unix, nil
}

func toFeature(alert *weatheralert.WeatherAlert) (*serviceports.WeatherAlertFeature, error) {
	if !alert.AlertCategory.IsValid() {
		return nil, errortypes.NewBusinessError("failed to convert weather alert category")
	}

	geometry, err := alert.Geometry.GeoJSON()
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to convert weather alert geometry").
			WithInternal(err)
	}

	return &serviceports.WeatherAlertFeature{
		Type:     "Feature",
		ID:       alert.ID,
		Geometry: geometry,
		Properties: serviceports.WeatherAlertFeatureProperties{
			ID:            alert.ID,
			NWSID:         alert.NWSID,
			Event:         alert.Event,
			Severity:      alert.Severity,
			Urgency:       alert.Urgency,
			Certainty:     alert.Certainty,
			Headline:      alert.Headline,
			Description:   alert.Description,
			Instruction:   alert.Instruction,
			AreaDesc:      alert.AreaDesc,
			Effective:     alert.Effective,
			Expires:       alert.Expires,
			Onset:         alert.Onset,
			Ends:          alert.Ends,
			Status:        alert.Status,
			MessageType:   alert.MessageType,
			SenderName:    alert.SenderName,
			Response:      alert.Response,
			Category:      alert.Category,
			AlertCategory: alert.AlertCategory,
			FirstSeenAt:   alert.FirstSeenAt,
			LastUpdatedAt: alert.LastUpdatedAt,
			ExpiredAt:     alert.ExpiredAt,
		},
	}, nil
}

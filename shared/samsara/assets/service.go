package assets

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	Create(ctx context.Context, req CreateRequest) (Asset, error)
	List(ctx context.Context, params ListParams) (ListResponse, error)
	Update(ctx context.Context, id string, req UpdateRequest) (Asset, error)
	Delete(ctx context.Context, ids []string) error
	StreamLocationAndSpeed(
		ctx context.Context,
		params LocationStreamParams,
	) (LocationStreamResponse, error)
	StreamLocationPages(
		ctx context.Context,
		params LocationStreamParams,
		fn func(*LocationStreamResponse) error,
	) error
	CurrentLocations(
		ctx context.Context,
		params CurrentLocationsParams,
	) (CurrentLocationsResponse, error)
	HistoricalLocations(
		ctx context.Context,
		params HistoricalLocationsParams,
	) (HistoricalLocationsResponse, error)
}

type service struct {
	client httpx.Requester
	now    func() time.Time
}

func NewService(client httpx.Requester) Service {
	return newService(client, time.Now)
}

func newService(client httpx.Requester, nowFn func() time.Time) *service {
	return &service{
		client: client,
		now:    nowFn,
	}
}

//nolint:gocritic // request is copied intentionally to avoid caller mutation during validation.
func (s *service) Create(
	ctx context.Context,
	req CreateRequest,
) (Asset, error) {
	out := createResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/assets",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Asset{}, err
	}
	return out.Data, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) List(
	ctx context.Context,
	params ListParams,
) (ListResponse, error) {
	if err := params.Validate(); err != nil {
		return ListResponse{}, err
	}

	out := ListResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/assets",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListResponse{}, err
	}
	return out, nil
}

func (s *service) Delete(ctx context.Context, ids []string) error {
	cleanIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			cleanIDs = append(cleanIDs, trimmed)
		}
	}
	if len(cleanIDs) == 0 {
		return ErrAssetIDsRequired
	}

	for _, id := range cleanIDs {
		query := url.Values{}
		query.Set("id", id)
		if err := s.client.Do(ctx, httpx.Request{
			Method:         http.MethodDelete,
			Path:           "/assets",
			Query:          query,
			ExpectedStatus: []int{http.StatusNoContent},
		}); err != nil {
			return err
		}
	}
	return nil
}

//nolint:gocritic // request is copied intentionally to keep update validation side-effect free.
func (s *service) Update(
	ctx context.Context,
	id string,
	req UpdateRequest,
) (Asset, error) {
	assetID := strings.TrimSpace(id)
	if assetID == "" {
		return Asset{}, ErrAssetIDRequired
	}

	query := url.Values{}
	query.Set("id", assetID)

	out := updateResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   "/assets",
		Query:  query,
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Asset{}, err
	}
	return out.Data, nil
}

func (s *service) StreamLocationAndSpeed(
	ctx context.Context,
	params LocationStreamParams,
) (LocationStreamResponse, error) {
	if err := params.Validate(); err != nil {
		return LocationStreamResponse{}, err
	}

	out := LocationStreamResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/assets/location-and-speed/stream",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return LocationStreamResponse{}, err
	}
	return out, nil
}

func (s *service) StreamLocationPages(
	ctx context.Context,
	params LocationStreamParams,
	fn func(*LocationStreamResponse) error,
) error {
	if fn == nil {
		return ErrCallbackNil
	}
	if err := params.Validate(); err != nil {
		return err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	for {
		page, err := s.StreamLocationAndSpeed(ctx, params)
		if err != nil {
			return err
		}
		if err = fn(&page); err != nil {
			return err
		}
		if !page.Pagination.HasNextPage || page.Pagination.EndCursor == nil ||
			strings.TrimSpace(*page.Pagination.EndCursor) == "" {
			break
		}
		params.After = *page.Pagination.EndCursor
	}
	return nil
}

func (s *service) CurrentLocations(
	ctx context.Context,
	params CurrentLocationsParams,
) (CurrentLocationsResponse, error) {
	if params.LookbackWindow == 0 {
		params.LookbackWindow = 15 * time.Minute
	}
	if params.LookbackWindow < 0 {
		return CurrentLocationsResponse{}, ErrCurrentLocationsLookbackWindow
	}

	end := s.now().UTC()
	start := end.Add(-params.LookbackWindow)
	streamParams := LocationStreamParams{
		StartTime:                     &start,
		EndTime:                       &end,
		IDs:                           params.IDs,
		IncludeSpeed:                  params.IncludeSpeed,
		IncludeReverseGeo:             params.IncludeReverseGeo,
		IncludeGeofenceLookup:         params.IncludeGeofenceLookup,
		IncludeHighFrequencyLocations: params.IncludeHighFrequencyLocations,
		IncludeExternalIDs:            params.IncludeExternalIDs,
		Limit:                         params.Limit,
	}

	latestByAsset := make(map[string]StreamRecord)
	latestTimes := make(map[string]time.Time)
	if err := s.StreamLocationPages(ctx, streamParams, func(page *LocationStreamResponse) error {
		for _, record := range page.Data {
			recordTime, err := parseRFC3339Time(record.HappenedAtTime)
			if err != nil {
				return err
			}
			assetID := record.Asset.Id
			existingTime, ok := latestTimes[assetID]
			if !ok || recordTime.After(existingTime) {
				latestTimes[assetID] = recordTime
				latestByAsset[assetID] = record
			}
		}
		return nil
	}); err != nil {
		return CurrentLocationsResponse{}, err
	}

	records := make([]StreamRecord, 0, len(latestByAsset))
	for _, record := range latestByAsset {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Asset.Id == records[j].Asset.Id {
			ti, errI := parseRFC3339Time(records[i].HappenedAtTime)
			tj, errJ := parseRFC3339Time(records[j].HappenedAtTime)
			if errI != nil || errJ != nil {
				return records[i].HappenedAtTime < records[j].HappenedAtTime
			}
			return ti.Before(tj)
		}
		return records[i].Asset.Id < records[j].Asset.Id
	})
	return CurrentLocationsResponse{Data: records}, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HistoricalLocations(
	ctx context.Context,
	params HistoricalLocationsParams,
) (HistoricalLocationsResponse, error) {
	if params.StartTime.IsZero() {
		return HistoricalLocationsResponse{}, ErrLocationStartTimeRequired
	}
	if params.EndTime.IsZero() {
		return HistoricalLocationsResponse{}, ErrLocationWindowInvalid
	}
	if params.EndTime.Before(params.StartTime) {
		return HistoricalLocationsResponse{}, ErrLocationWindowInvalid
	}

	start := params.StartTime.UTC()
	end := params.EndTime.UTC()
	streamParams := LocationStreamParams{
		StartTime:                     &start,
		EndTime:                       &end,
		IDs:                           params.IDs,
		IncludeSpeed:                  params.IncludeSpeed,
		IncludeReverseGeo:             params.IncludeReverseGeo,
		IncludeGeofenceLookup:         params.IncludeGeofenceLookup,
		IncludeHighFrequencyLocations: params.IncludeHighFrequencyLocations,
		IncludeExternalIDs:            params.IncludeExternalIDs,
		Limit:                         params.Limit,
	}

	history := make([]StreamRecord, 0)
	if err := s.StreamLocationPages(ctx, streamParams, func(page *LocationStreamResponse) error {
		history = append(history, page.Data...)
		return nil
	}); err != nil {
		return HistoricalLocationsResponse{}, err
	}

	sort.Slice(history, func(i, j int) bool {
		ti, errI := parseRFC3339Time(history[i].HappenedAtTime)
		tj, errJ := parseRFC3339Time(history[j].HappenedAtTime)
		if errI != nil || errJ != nil {
			if history[i].Asset.Id == history[j].Asset.Id {
				return history[i].HappenedAtTime < history[j].HappenedAtTime
			}
			return history[i].Asset.Id < history[j].Asset.Id
		}
		if history[i].Asset.Id == history[j].Asset.Id {
			return ti.Before(tj)
		}
		if ti.Equal(tj) {
			return history[i].Asset.Id < history[j].Asset.Id
		}
		return ti.Before(tj)
	})

	return HistoricalLocationsResponse{Data: history}, nil
}

func parseRFC3339Time(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

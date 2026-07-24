package compliance

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	HOSClocks(ctx context.Context, params HOSClocksParams) (HOSClocksResponse, error)
	HOSClocksAll(ctx context.Context, params HOSClocksParams) ([]HOSClock, error)
	HOSDailyLogs(ctx context.Context, params HOSDailyLogsParams) (HOSDailyLogsResponse, error)
	HOSViolations(ctx context.Context, params HOSViolationsParams) (HOSViolationsResponse, error)
	HOSLogs(ctx context.Context, params HOSLogsParams) (HOSLogsResponse, error)
	DriverTachographHistory(
		ctx context.Context,
		params DriverTachographParams,
	) (DriverTachographResponse, error)
	VehicleTachographHistory(
		ctx context.Context,
		params VehicleTachographParams,
	) (VehicleTachographResponse, error)
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HOSClocks(
	ctx context.Context,
	params HOSClocksParams,
) (HOSClocksResponse, error) {
	if err := params.Validate(); err != nil {
		return HOSClocksResponse{}, err
	}

	out := HOSClocksResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/hos/clocks",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return HOSClocksResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HOSClocksAll(
	ctx context.Context,
	params HOSClocksParams,
) ([]HOSClock, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	items := make([]HOSClock, 0)
	for {
		page, err := s.HOSClocks(ctx, params)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Data...)
		if !page.Pagination.HasNextPage || strings.TrimSpace(page.Pagination.EndCursor) == "" {
			break
		}
		params.After = page.Pagination.EndCursor
	}
	return items, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HOSDailyLogs(
	ctx context.Context,
	params HOSDailyLogsParams,
) (HOSDailyLogsResponse, error) {
	if err := params.Validate(); err != nil {
		return HOSDailyLogsResponse{}, err
	}

	out := HOSDailyLogsResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/hos/daily-logs",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return HOSDailyLogsResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HOSViolations(
	ctx context.Context,
	params HOSViolationsParams,
) (HOSViolationsResponse, error) {
	out := HOSViolationsResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/hos/violations",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return HOSViolationsResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HOSLogs(ctx context.Context, params HOSLogsParams) (HOSLogsResponse, error) {
	out := HOSLogsResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/hos/logs",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return HOSLogsResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) DriverTachographHistory(
	ctx context.Context,
	params DriverTachographParams,
) (DriverTachographResponse, error) {
	out := DriverTachographResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/drivers/tachograph-files/history",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return DriverTachographResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) VehicleTachographHistory(
	ctx context.Context,
	params VehicleTachographParams,
) (VehicleTachographResponse, error) {
	out := VehicleTachographResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/vehicles/tachograph-files/history",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return VehicleTachographResponse{}, err
	}
	return out, nil
}

// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package pcmiler

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/google/go-querystring/query"
	"github.com/imroc/req/v3"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type Client interface {
	SingleSearch(ctx context.Context, req *SingleSearchParams) (*LocationResponse, error)
}

type ClientParams struct {
	fx.In

	Logger *logger.Logger
}

type client struct {
	l  *zerolog.Logger
	rc *req.Client
}

func NewClient(p ClientParams) Client {
	log := p.Logger.With().Str("client", "pcmiler").Logger()

	reqClient := req.C().
		SetTimeout(10 * time.Second).
		EnableDumpEachRequest().
		EnableCompression()

	c := &client{
		l:  &log,
		rc: reqClient,
	}

	return c
}

func (c *client) SingleSearch(
	ctx context.Context,
	params *SingleSearchParams,
) (*LocationResponse, error) {
	v, err := query.Values(params)
	if err != nil {
		c.l.Error().Err(err).Msg("failed to parse single search params")
		return nil, err
	}

	url := fmt.Sprintf("%s?%s", SingleSearchURL, v.Encode())
	c.l.Debug().Msgf("Making single search request to %s", url)

	var locationResp LocationResponse
	resp, err := c.rc.R().
		SetContext(ctx).
		SetSuccessResult(&locationResp).
		Get(url)
	if err != nil {
		c.l.Error().Err(err).Msg("failed to make single search request")
		return nil, err
	}

	if resp.IsErrorState() {
		c.l.Error().Interface("error", resp.Err).Msg("single search request failed")
		return nil, resp.Err
	}

	return &locationResp, nil
}

type RouteReportParams struct {
	AuthToken string `url:"authToken"`

	// Comma separated list of lat/long pairs separated by semi colons
	// Example: -76.123456,42.123456;-76.123126,42.123126
	Stops string `url:"stops"`

	// The asset ID to use for the route report
	AssetID string `url:"assetId"`

	// The place ID to use for the route report
	PlaceID string `url:"placeId"`

	// The region to use for the route report
	// Default is 4 (North America)
	Region Region `url:"region"`

	// Comma separated list of reports to generate
	// Example: "Mileage,Detail,CalcMiles,Directions,Geotunnel,LeastCost,Road,State,WeatherAlerts"
	Reports string `url:"reports"`
}

// func (c *client) RouteReport(ctx context.Context, params *RouteReportParams) (any, error) {

// }

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

func (c *client) SingleSearch(ctx context.Context, params *SingleSearchParams) (*LocationResponse, error) {
	v, err := query.Values(params)
	if err != nil {
		c.l.Error().Err(err).Msg("failed to parse single search params")
		return nil, err
	}

	url := fmt.Sprintf("%s?%s", SingleSearchURL, v.Encode())
	c.l.Trace().Msgf("Making single search request to %s", url)

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

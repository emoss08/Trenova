// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package routing

import (
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/pcmiler"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/net/context"
)

type ServiceParams struct {
	fx.In

	Logger *logger.Logger
	Repo   repositories.PCMilerConfigurationRepository
	Client pcmiler.Client
}

type Service struct {
	l      *zerolog.Logger
	repo   repositories.PCMilerConfigurationRepository
	client pcmiler.Client
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().Str("service", "routing").Logger()

	return &Service{
		l:      &log,
		repo:   p.Repo,
		client: p.Client,
	}
}

type SingleSearchParams struct {
	// The query to search for
	Query string `json:"query" query:"query"`

	// The options for the PCMiler configuration
	ConfigOpts repositories.GetPCMilerConfigurationOptions
}

func (s *Service) SingleSearch(
	ctx context.Context,
	opts SingleSearchParams,
) (*ports.ListResult[*pcmiler.Location], error) {
	config, err := s.repo.GetPCMilerConfiguration(ctx, opts.ConfigOpts)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get PCMiler configuration")
		return nil, err
	}

	params := &pcmiler.SingleSearchParams{
		AuthToken: config.APIKey,
		Query:     opts.Query,
		Countries: "US",
	}

	resp, err := s.client.SingleSearch(ctx, params)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to make single search request")
		return nil, err
	}

	return &ports.ListResult[*pcmiler.Location]{
		Items: resp.Locations,
		Total: len(resp.Locations),
	}, nil
}

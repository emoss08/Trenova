package usstate

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/trenova-app/transport/internal/core/ports/repositories"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/pkg/types"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger *logger.Logger
	Repo   repositories.UsStateRepository
}

type Service struct {
	repo repositories.UsStateRepository
	l    *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().Str("service", "state").Logger()

	return &Service{
		repo: p.Repo,
		l:    &log,
	}
}

// SelectOptions returns a list of select options for us states.
func (s *Service) SelectOptions(ctx context.Context) ([]types.SelectOption, error) {
	result, err := s.repo.List(ctx)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to list us states")
		return nil, eris.Wrap(err, "failed to list us states")
	}

	options := make([]types.SelectOption, len(result.States))
	for i, state := range result.States {
		options[i] = types.SelectOption{
			Label: state.Name,
			Value: state.ID.String(),
		}
	}

	return options, nil
}

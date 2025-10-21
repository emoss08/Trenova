package usstate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.UsStateRepository
}

type Service struct {
	repo repositories.UsStateRepository
	l    *zap.Logger
}

func NewService(p ServiceParams) *Service {
	return &Service{
		repo: p.Repo,
		l:    p.Logger.Named("service.usstate"),
	}
}

// SelectOptions returns a list of select options for us states.
func (s *Service) SelectOptions(ctx context.Context) ([]*domaintypes.SelectOption, error) {
	result, err := s.repo.List(ctx)
	if err != nil {
		s.l.Error("failed to list us states", zap.Error(err))
		return nil, err
	}

	options := make([]*domaintypes.SelectOption, len(result.Items))
	for i, state := range result.Items {
		options[i] = &domaintypes.SelectOption{
			Label: state.Name,
			Value: state.ID.String(),
		}
	}

	return options, nil
}

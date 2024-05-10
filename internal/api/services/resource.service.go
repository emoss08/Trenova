package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/rs/zerolog"
)

type ResourceService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewResourceService(s *api.Server) *ResourceService {
	return &ResourceService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetResources gets the system assigned resources.
func (s *ResourceService) GetResources(ctx context.Context, limit, offset int) ([]*ent.Resource, int, error) {
	entityCount, countErr := s.Client.Resource.Query().Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := s.Client.Resource.Query().
		Limit(limit).
		Offset(offset).
		WithPermissions().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

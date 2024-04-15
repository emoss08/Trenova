package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
)

type USStateService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewUSStateService creates a new shipment type service.
func NewUSStateService(s *api.Server) *USStateService {
	return &USStateService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetUSStates gets the shipment types for an organization.
func (r *USStateService) GetUSStates(ctx context.Context) ([]*ent.UsState, error) {
	entities, err := r.Client.UsState.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

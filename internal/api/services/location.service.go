package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/models"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

type LocationService struct {
	Client       *ent.Client
	Logger       *zerolog.Logger
	QueryService *models.QueryService
}

// NewLocationService creates a new location service.
func NewLocationService(s *api.Server) *LocationService {
	return &LocationService{
		Client: s.Client,
		Logger: s.Logger,
		QueryService: &models.QueryService{
			Client: s.Client,
			Logger: s.Logger,
		},
	}
}

func (r *LocationService) GetLocations(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Location, int, error) {
	return r.QueryService.GetLocations(ctx, limit, offset, orgID, buID)
}

func (r *LocationService) CreateLocation(ctx context.Context, newEntity *types.LocationRequest) (*ent.Location, error) {
	var createdEntity *ent.Location

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		createdEntity, err = r.QueryService.CreateLocationEntity(ctx, tx, newEntity)
		if err != nil {
			return err
		}

		// If comments are provided, create them and associate them with the location
		if len(newEntity.Comments) > 0 {
			if err = r.QueryService.CreateLocationComments(ctx, tx, createdEntity.ID, newEntity); err != nil {
				return err
			}
		}

		// If locations are provided, create them and associate them with the location
		if len(newEntity.Contacts) > 0 {
			if err = r.QueryService.CreateLocationContacts(ctx, tx, createdEntity.ID, newEntity); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

func (r *LocationService) UpdateLocation(ctx context.Context, entity *types.LocationUpdateRequest) (*ent.Location, error) {
	var updatedEntity *ent.Location

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.QueryService.UpdateLocationEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		if err = r.QueryService.SyncLocationComments(ctx, tx, entity, updatedEntity); err != nil {
			return err
		}

		return r.QueryService.SyncLocationComments(ctx, tx, entity, updatedEntity)
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

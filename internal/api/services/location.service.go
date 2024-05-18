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

// GetLocations retrieves a list of locations for a given organization and business unit.
// It returns a slice of Location entities, the total number of location records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - []*ent.Location: A slice of Location entities.
//   - int: The total number of location records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *LocationService) GetLocations(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Location, int, error) {
	return r.QueryService.GetLocations(ctx, limit, offset, orgID, buID)
}

// CreateLocation creates a new location entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - newEntity *LocationRequest: The location request containing the details of the location to be created.
//
// Returns:
//   - *ent.Location: A pointer to the created Location entity.
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
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

// UpdateLocation updates a location entity.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - entity *LocationUpdateRequest: The location update request containing the details of the location to be updated.
//
// Returns:
//   - *ent.Location: A pointer to the updated Location entity.
//   - error: An error object that indicates why the update failed, nil if no error occurred.
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

		return r.QueryService.SyncLocationContacts(ctx, tx, entity, updatedEntity)
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/googleapi"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// GoogleAPIService is the service for google api.
type GoogleAPIService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewGoogleAPIService creates a new google api service.
func NewGoogleAPIService(s *api.Server) *GoogleAPIService {
	return &GoogleAPIService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetGoogleAPI gets the google api settings for an organization.
func (r *GoogleAPIService) GetGoogleAPI(ctx context.Context, orgID, buID uuid.UUID) (*ent.GoogleApi, error) {
	googleAPI, err := r.Client.GoogleApi.Query().Where(
		googleapi.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return googleAPI, nil
}

// UpdateGoogleAPI updates the google api settings for an organization.
func (r *GoogleAPIService) UpdateGoogleAPI(
	ctx context.Context, entity *ent.GoogleApi,
) (*ent.GoogleApi, error) {
	updatedEntity := new(ent.GoogleApi)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateGoogleAPIEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *GoogleAPIService) updateGoogleAPIEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.GoogleApi,
) (*ent.GoogleApi, error) {
	updateOp := tx.GoogleApi.UpdateOneID(entity.ID).
		SetAPIKey(entity.APIKey).
		SetMileageUnit(entity.MileageUnit).
		SetAddCustomerLocation(entity.AddCustomerLocation).
		SetAutoGeocode(entity.AutoGeocode).
		SetAddLocation(entity.AddLocation).
		SetTrafficModel(entity.TrafficModel)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

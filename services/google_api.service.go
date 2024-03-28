package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/googleapi"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// GoogleAPIOps is the service for google api settings.
type GoogleAPIOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewGoogleAPIOps creates a new google api service.
func NewGoogleAPIOps(ctx context.Context) *GoogleAPIOps {
	return &GoogleAPIOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetGoogleAPI gets the google api settings for an organization.
func (r *GoogleAPIOps) GetGoogleAPI(orgID, buID uuid.UUID) (*ent.GoogleApi, error) {
	googleAPI, err := r.client.GoogleApi.Query().Where(
		googleapi.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return googleAPI, nil
}

// UpdateGoogleAPI updates the google api settings for an organization.
func (r *GoogleAPIOps) UpdateGoogleAPI(googleAPI ent.GoogleApi) (*ent.GoogleApi, error) {
	updatedGoogleAPI, err := r.client.GoogleApi.
		UpdateOneID(googleAPI.ID).
		SetAPIKey(googleAPI.APIKey).
		SetMileageUnit(googleAPI.MileageUnit).
		SetAddCustomerLocation(googleAPI.AddCustomerLocation).
		SetAutoGeocode(googleAPI.AutoGeocode).
		SetAddLocation(googleAPI.AddLocation).
		SetTrafficModel(googleAPI.TrafficModel).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedGoogleAPI, nil
}

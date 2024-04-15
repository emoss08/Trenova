package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/organizationfeatureflag"
	"github.com/google/uuid"
)

type FeatureFlagService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewFeatureFlagService creates a new feature flag service.
func NewFeatureFlagService(s *api.Server) *FeatureFlagService {
	return &FeatureFlagService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetFeatureFlags gets the feature flags for an organization.
func (r *FeatureFlagService) GetFeatureFlags(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.OrganizationFeatureFlag, int, error) {
	entityCount, countErr := r.Client.OrganizationFeatureFlag.Query().Where(
		organizationfeatureflag.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.OrganizationFeatureFlag.Query().
		Limit(limit).
		Offset(offset).
		WithOrganization().
		WithFeatureFlag().
		Where(
			organizationfeatureflag.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/organizationfeatureflag"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type OrganizationFeatureFlagOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewOrganizationFeatureFlagOps creates a new organization feature flag service.
func NewOrganizationFeatureFlagOps() *OrganizationFeatureFlagOps {
	return &OrganizationFeatureFlagOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetOrganizationFeatureFlags gets the feature flags for assigned to an organization.
func (r *OrganizationFeatureFlagOps) GetOrganizationFeatureFlags(
	ctx context.Context, limit, offset int, orgID uuid.UUID,
) ([]*ent.OrganizationFeatureFlag, int, error) {
	entityCount, countErr := r.client.Debug().OrganizationFeatureFlag.Query().Where(
		organizationfeatureflag.HasOrganizationWith(
			organization.IDEQ(orgID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.OrganizationFeatureFlag.Query().
		Limit(limit).
		WithOrganization().
		WithFeatureFlag().
		Offset(offset).
		Where(
			organizationfeatureflag.HasOrganizationWith(
				organization.IDEQ(orgID),
			),
		).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

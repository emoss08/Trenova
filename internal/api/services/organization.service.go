package services

import (
	"context"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/businessunit"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/api"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// OrganizationOps is the service for organization.
type OrganizationService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewOrganizationOps creates a new organization service.
func NewOrganizationOps(s *api.Server) *OrganizationService {
	return &OrganizationService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetUserOrganization returns the organization of the user.
func (r *OrganizationService) GetUserOrganization(ctx context.Context, buID, orgID uuid.UUID) (*ent.Organization, error) {
	org, err := r.Client.Organization.
		Query().
		Where(
			organization.And(
				organization.ID(orgID),
				organization.HasBusinessUnitWith(
					businessunit.ID(buID),
				),
			),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

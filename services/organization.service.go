package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/businessunit"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// OrganizationOps is the service for organization.
type OrganizatinOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewOrganizationOps creates a new organization service.
func NewOrganizationOps(ctx context.Context) *OrganizatinOps {
	return &OrganizatinOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetUserOrganization returns the organization of the user.
func (r *OrganizatinOps) GetUserOrganization(buID, orgID uuid.UUID) (*ent.Organization, error) {
	org, err := r.client.Organization.
		Query().
		Where(
			organization.And(
				organization.ID(orgID),
				organization.HasBusinessUnitWith(
					businessunit.ID(buID),
				),
			),
		).
		Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

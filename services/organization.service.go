package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/emoss08/trenova/ent/user"
	"github.com/google/uuid"
)

// UserOps is the service for user
type OrganizatinOps struct {
	ctx    context.Context
	client *ent.Client
}

func NewOrganizationOps(ctx context.Context) *OrganizatinOps {
	return &OrganizatinOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetUserOrganization returns the organization of the user
func (r OrganizatinOps) GetUserOrganization(userID uuid.UUID) (*ent.Organization, error) {
	org, err := r.client.Organization.
		Query().
		Where(organization.HasUsersWith(user.ID(userID))).
		Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return org, nil
}

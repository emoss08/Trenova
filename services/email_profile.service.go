package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/emailprofile"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// EmailProfileOps is the service for email profiles.
type EmailProfileOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewEmailProfileOps creates a new email profiles service.
func NewEmailProfileOps(ctx context.Context) *EmailProfileOps {
	return &EmailProfileOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetEmailProfiles gets the email profiles for an organization.
func (r *EmailProfileOps) GetEmailProfiles(limit, offset int, orgID, buID uuid.UUID) ([]*ent.EmailProfile, int, error) {
	emailProfileCount, countErr := r.client.EmailProfile.Query().Where(
		emailprofile.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	emailProfiles, err := r.client.EmailProfile.Query().
		Limit(limit).
		Offset(offset).
		Where(
			emailprofile.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return emailProfiles, emailProfileCount, nil
}

// CreateEmailProfile creates a new email profile for an organization.
func (r *EmailProfileOps) CreateEmailProfile(newEmailProfile ent.EmailProfile) (*ent.EmailProfile, error) {
	emailProfile, err := r.client.EmailProfile.Create().
		SetOrganizationID(newEmailProfile.OrganizationID).
		SetBusinessUnitID(newEmailProfile.BusinessUnitID).
		SetName(newEmailProfile.Name).
		SetEmail(newEmailProfile.Email).
		SetProtocol(newEmailProfile.Protocol).
		SetHost(newEmailProfile.Host).
		SetPort(newEmailProfile.Port).
		SetUsername(newEmailProfile.Username).
		SetPassword(newEmailProfile.Password).
		SetIsDefault(newEmailProfile.IsDefault).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return emailProfile, nil
}

// UpdateEmailProfile updates an email profile for an organization.
func (r *EmailProfileOps) UpdateEmailProfile(emailProfile ent.EmailProfile) (*ent.EmailProfile, error) {
	updatedEmailProfile, err := r.client.EmailProfile.UpdateOneID(emailProfile.ID).
		SetName(emailProfile.Name).
		SetEmail(emailProfile.Email).
		SetProtocol(emailProfile.Protocol).
		SetHost(emailProfile.Host).
		SetPort(emailProfile.Port).
		SetUsername(emailProfile.Username).
		SetPassword(emailProfile.Password).
		SetIsDefault(emailProfile.IsDefault).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEmailProfile, nil
}

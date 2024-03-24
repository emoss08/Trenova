package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/emailcontrol"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

// EmailControlOps is the service for email control settings.
type EmailControlOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewEmailControlOps creates a new email control service.
func NewEmailControlOps(ctx context.Context) *EmailControlOps {
	return &EmailControlOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetEmailControl gets the email control settings for an organization.
func (r *EmailControlOps) GetEmailControl(orgID, buID uuid.UUID) (*ent.EmailControl, error) {
	emailControl, err := r.client.EmailControl.Query().Where(
		emailcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return emailControl, nil
}

// UpdateEmailControl updates the email control settings for an organization.
func (r *EmailControlOps) UpdateEmailControl(emailControl ent.EmailControl) (*ent.EmailControl, error) {
	updateEmailControl, err := r.client.EmailControl.
		UpdateOneID(emailControl.ID).
		SetNillableBillingEmailProfileID(emailControl.BillingEmailProfileID).
		SetNillableRateExpirtationEmailProfileID(emailControl.RateExpirtationEmailProfileID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateEmailControl, nil
}

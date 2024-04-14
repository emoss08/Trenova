package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/emailcontrol"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
)

// EmailControlService is the service for email control settings.
type EmailControlService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewEmailControlService creates a new email control service.
func NewEmailControlService(s *api.Server) *EmailControlService {
	return &EmailControlService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetEmailControl gets the email control settings for an organization.
func (r *EmailControlService) GetEmailControl(ctx context.Context, orgID, buID uuid.UUID) (*ent.EmailControl, error) {
	emailControl, err := r.Client.EmailControl.Query().Where(
		emailcontrol.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Only(ctx)
	if err != nil {
		return nil, err
	}

	return emailControl, nil
}

// UpdateEmailControl updates the email control settings for an organization.
func (r *EmailControlService) UpdateEmailControl(
	ctx context.Context, ec *ent.EmailControl,
) (*ent.EmailControl, error) {
	updatedEntity := new(ent.EmailControl)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateEmailControlEntity(ctx, tx, ec)
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

func (r *EmailControlService) updateEmailControlEntity(
	ctx context.Context, tx *ent.Tx, ec *ent.EmailControl,
) (*ent.EmailControl, error) {
	updateOp := tx.EmailControl.UpdateOneID(ec.ID).
		SetNillableBillingEmailProfileID(ec.BillingEmailProfileID).
		SetNillableRateExpirtationEmailProfileID(ec.RateExpirtationEmailProfileID)

	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}

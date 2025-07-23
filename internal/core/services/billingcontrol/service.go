// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package billingcontrol

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/billingcontrolvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.BillingControlRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *billingcontrolvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.BillingControlRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *billingcontrolvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "billingcontrol").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

// Get returns a billing control
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetBillingControlRequest containing orgID, buID, and userID.
//
// Returns:
//   - *billing.BillingControl: The billing control entity.
//   - error: If any database operation fails.
func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetBillingControlRequest,
) (*billing.BillingControl, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	// * Check if the user has permission to read the billing control
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceBillingControl,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	// * If the user does not have permission to read the billing control, return an error
	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read billing control",
		)
	}

	// * Get the billing control by organization ID
	entity, err := s.repo.GetByOrgID(ctx, req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get billing control")
		return nil, err
	}

	return entity, nil
}

// Update updates a billing control
//
// Parameters:
//   - ctx: The context for the operation.
//   - bc: The billing control entity to update.
//   - userID: The user ID of the user updating the billing control.
//
// Returns:
//   - *billing.BillingControl: The updated billing control entity.
//   - error: If any database operation fails.
func (s *Service) Update(
	ctx context.Context,
	bc *billing.BillingControl,
	userID pulid.ID,
) (*billing.BillingControl, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("orgID", bc.OrganizationID.String()).
		Str("buID", bc.BusinessUnitID.String()).
		Logger()

	// * Check if the user has permission to update the billing control
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceBillingControl,
				Action:         permission.ActionUpdate,
				BusinessUnitID: bc.BusinessUnitID,
				OrganizationID: bc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	// * If the user does not have permission to update the billing control, return an error
	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update billing control",
		)
	}

	// * Create a validation context for the billing control
	// * IsUpdate is true because we are updating an existing entity
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	// * Validate the billing control
	if err := s.v.Validate(ctx, valCtx, bc); err != nil {
		return nil, err
	}

	// * Get the original billing control for comparison when logging the action
	original, err := s.repo.GetByOrgID(ctx, bc.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get billing control")
		return nil, err
	}

	// * Update the billing control
	updatedEntity, err := s.repo.Update(ctx, bc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update billing control")
		return nil, err
	}

	// * Log the action
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceBillingControl,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Billing Control updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}

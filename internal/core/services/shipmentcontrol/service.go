package shipmentcontrol

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentcontrolvalidator"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams defines the dependencies required for initializing the Service.
// This includes a logger, shipment control repository, permission service, audit service, and validator.
type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.ShipmentControlRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *shipmentcontrolvalidator.Validator
}

// Service is a service that manages shipment control entities.
// It provides methods to get and update shipment control entities.
type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentControlRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *shipmentcontrolvalidator.Validator
}

// NewService initializes a new instance of service with its dependencies.
//
// Parameters:
//   - p: ServiceParams containing logger, shipment control repository, permission service, audit service, and validator.
//
// Returns:
//   - A new instance of service.
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmentcontrol").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

// Get returns a shipment control
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetShipmentControlRequest containing orgID, buID, and userID.
//
// Returns:
//   - *shipment.ShipmentControl: The shipment control entity.
//   - error: If any database operation fails.
func (s *Service) Get(ctx context.Context, req *repositories.GetShipmentControlRequest) (*shipment.ShipmentControl, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	// * Check if the user has permission to read the shipment control
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceShipmentControl,
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

	// * If the user does not have permission to read the shipment control, return an error
	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read shipment control")
	}

	// * Get the shipment control by organization ID
	entity, err := s.repo.GetByOrgID(ctx, req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment control")
		return nil, err
	}

	return entity, nil
}

// Update updates a shipment control
//
// Parameters:
//   - ctx: The context for the operation.
//   - sc: The shipment control entity to update.
//   - userID: The user ID of the user updating the shipment control.
//
// Returns:
//   - *shipment.ShipmentControl: The updated shipment control entity.
//   - error: If any database operation fails.
func (s *Service) Update(ctx context.Context, sc *shipment.ShipmentControl, userID pulid.ID) (*shipment.ShipmentControl, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("orgID", sc.OrganizationID.String()).
		Str("buID", sc.BusinessUnitID.String()).
		Logger()

	// * Check if the user has permission to update the shipment control
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipmentControl,
				Action:         permission.ActionUpdate,
				BusinessUnitID: sc.BusinessUnitID,
				OrganizationID: sc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	// * Check if the user has permission to update the shipment control
	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update shipment control")
	}

	// * Create a validation context for the shipment control
	// * IsUpdate is true because we are updating an existing entity
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	// * Validate the shipment control
	if err := s.v.Validate(ctx, valCtx, sc); err != nil {
		return nil, err
	}

	// * Get the original shipment control for comparison when logging the action
	original, err := s.repo.GetByOrgID(ctx, sc.OrganizationID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment control")
		return nil, err
	}

	// * Update the shipment control
	updatedEntity, err := s.repo.Update(ctx, sc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment control")
		return nil, err
	}

	// * Log the action
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentControl,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment Control updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}

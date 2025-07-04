package consolidation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator/consolidationvalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

// ServiceParams defines dependencies required for initializing the ConsolidationService.
// This includes repositories, permission service, audit service, and logger.
type ServiceParams struct {
	fx.In

	Logger            *logger.Logger
	ConsolidationRepo repositories.ConsolidationRepository
	ShipmentRepo      repositories.ShipmentRepository
	PermService       services.PermissionService
	AuditService      services.AuditService
	Validator         *consolidationvalidator.Validator
}

// Service implements business logic for consolidation management.
// It handles consolidation group operations, shipment assignments, and status management.
type Service struct {
	l                 *zerolog.Logger
	consolidationRepo repositories.ConsolidationRepository
	shipmentRepo      repositories.ShipmentRepository
	ps                services.PermissionService
	as                services.AuditService
	v                 *consolidationvalidator.Validator
}

// NewService creates a new consolidation service instance.
//
// Parameters:
//   - p: ServiceParams containing all required dependencies.
//
// Returns:
//   - *Service: A ready-to-use consolidation service instance.
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "consolidation").
		Logger()

	return &Service{
		l:                 &log,
		consolidationRepo: p.ConsolidationRepo,
		shipmentRepo:      p.ShipmentRepo,
		ps:                p.PermService,
		as:                p.AuditService,
		v:                 p.Validator,
	}
}

// SelectOptions returns consolidation groups as select options for UI dropdowns.
// This is useful for forms where users need to select a consolidation group.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: Query options for filtering and pagination.
//
// Returns:
//   - []*types.SelectOption: List of consolidation options.
//   - error: If the query fails.
func (s *Service) SelectOptions(
	ctx context.Context,
	opts *ports.QueryOptions,
) ([]*types.SelectOption, error) {
	result, err := s.consolidationRepo.List(ctx, &repositories.ListConsolidationRequest{
		Filter: opts,
	})
	if err != nil {
		return nil, eris.Wrap(err, "select consolidations")
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, cg := range result.Items {
		options = append(options, &types.SelectOption{
			Value: cg.GetID(),
			Label: cg.ConsolidationNumber,
			Color: string(cg.Status),
		})
	}

	return options, nil
}

// List retrieves consolidation groups based on filtering options.
// It checks permissions before returning results.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - opts: Request options for filtering and pagination.
//
// Returns:
//   - *ports.ListResult[*consolidation.ConsolidationGroup]: Paginated consolidation groups.
//   - error: If permissions fail or query fails.
func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListConsolidationRequest,
) (*ports.ListResult[*consolidation.ConsolidationGroup], error) {
	log := s.l.With().
		Str("operation", "List").
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read consolidation groups",
		)
	}

	// * List consolidation groups
	entities, err := s.consolidationRepo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list consolidation groups")
		return nil, eris.Wrap(err, "failed to list consolidation groups")
	}

	return entities, nil
}

// Get retrieves a single consolidation group by ID.
// It checks permissions before returning the group.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - id: The ID of the consolidation group.
//   - userID: The ID of the user making the request.
//   - orgID: Organization ID for tenant filtering.
//   - buID: Business Unit ID for tenant filtering.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The requested consolidation group.
//   - error: If permissions fail or group not found.
func (s *Service) Get(
	ctx context.Context,
	id, userID, orgID, buID pulid.ID,
) (*consolidation.ConsolidationGroup, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("consolidationID", id.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check read consolidation permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this consolidation group",
		)
	}

	// * Get the consolidation group
	entity, err := s.consolidationRepo.Get(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation group")
		return nil, eris.Wrap(err, "failed to get consolidation group")
	}

	return entity, nil
}

// Create creates a new consolidation group.
// It validates permissions and logs the creation for audit purposes.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - group: The consolidation group to create.
//   - userID: The ID of the user creating the group.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The created consolidation group.
//   - error: If permissions fail or creation fails.
func (s *Service) Create(
	ctx context.Context,
	req *repositories.CreateConsolidationRequest,
) (*consolidation.ConsolidationGroup, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionCreate,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create consolidation permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a consolidation group",
		)
	}

	// ! We need to write a validation to check if the shipments are eligible for consolidation

	// * Create the consolidation group
	createdEntity, err := s.consolidationRepo.Create(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to create consolidation group")
		return nil, eris.Wrap(err, "create consolidation group")
	}

	// * Log the creation for audit
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceConsolidation,
			ResourceID:     createdEntity.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("consolidation group created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log consolidation creation")
	}

	return createdEntity, nil
}

// Update modifies an existing consolidation group.
// It validates permissions and logs the update for audit purposes.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - group: The consolidation group with updated fields.
//   - userID: The ID of the user making the update.
//
// Returns:
//   - *consolidation.ConsolidationGroup: The updated consolidation group.
//   - error: If permissions fail or update fails.
func (s *Service) Update(
	ctx context.Context,
	group *consolidation.ConsolidationGroup,
	userID pulid.ID,
) (*consolidation.ConsolidationGroup, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("consolidationID", group.ID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionUpdate,
				BusinessUnitID: group.BusinessUnitID,
				OrganizationID: group.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update consolidation permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this consolidation group",
		)
	}

	// * Get the original for audit logging
	original, err := s.consolidationRepo.Get(ctx, group.ID)
	if err != nil {
		return nil, eris.Wrap(err, "get original consolidation group")
	}

	// * Update the consolidation group
	updatedEntity, err := s.consolidationRepo.Update(ctx, group)
	if err != nil {
		log.Error().Err(err).Msg("failed to update consolidation group")
		return nil, eris.Wrap(err, "update consolidation group")
	}

	// * Log the update for audit
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceConsolidation,
			ResourceID:     updatedEntity.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("consolidation group updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log consolidation update")
	}

	return updatedEntity, nil
}

// AddShipmentToGroup adds a shipment to a consolidation group.
// It validates permissions and ensures the shipment is eligible for consolidation.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//   - shipmentID: The ID of the shipment to add.
//   - userID: The ID of the user making the request.
//   - orgID: Organization ID for tenant filtering.
//   - buID: Business Unit ID for tenant filtering.
//
// Returns:
//   - error: If permissions fail or operation fails.
func (s *Service) AddShipmentToGroup(
	ctx context.Context,
	groupID, shipmentID, userID, orgID, buID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "AddShipmentToGroup").
		Str("groupID", groupID.String()).
		Str("shipmentID", shipmentID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionUpdate,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check consolidation update permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to update this consolidation group",
		)
	}

	// * Get the consolidation group to verify it exists and check status
	group, err := s.consolidationRepo.Get(ctx, groupID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation group")
		return eris.Wrap(err, "get consolidation group")
	}

	// * Check if the group is in a valid status for adding shipments
	if group.Status == consolidation.GroupStatusCompleted ||
		group.Status == consolidation.GroupStatusCanceled {
		return errors.NewValidationError(
			"status",
			errors.ErrInvalid,
			"Cannot add shipments to a completed or canceled consolidation group",
		)
	}

	// * Get the shipment to verify it exists and check eligibility
	shp, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    shipmentID,
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return eris.Wrap(err, "get shipment")
	}

	// * Check if the shipment is eligible for consolidation
	if shp.Status != shipment.StatusNew && shp.Status != shipment.StatusReadyToBill {
		return errors.NewValidationError(
			"status",
			errors.ErrInvalid,
			"Only shipments with status New or ReadyToBill can be added to consolidation groups",
		)
	}

	// * Add the shipment to the group
	err = s.consolidationRepo.AddShipmentToGroup(ctx, groupID, shipmentID)
	if err != nil {
		log.Error().Err(err).Msg("failed to add shipment to group")
		return eris.Wrap(err, "add shipment to group")
	}

	// * Log the action for audit
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:   permission.ResourceConsolidation,
			ResourceID: groupID.String(),
			Action:     permission.ActionUpdate,
			UserID:     userID,
			CurrentState: jsonutils.MustToJSON(
				map[string]string{"shipmentID": shipmentID.String()},
			),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("shipment added to consolidation group"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment addition")
	}

	return nil
}

// RemoveShipmentFromGroup removes a shipment from a consolidation group.
// It validates permissions and ensures the group status allows removal.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//   - shipmentID: The ID of the shipment to remove.
//   - userID: The ID of the user making the request.
//   - orgID: Organization ID for tenant filtering.
//   - buID: Business Unit ID for tenant filtering.
//
// Returns:
//   - error: If permissions fail or operation fails.
func (s *Service) RemoveShipmentFromGroup(
	ctx context.Context,
	groupID, shipmentID, userID, orgID, buID pulid.ID,
) error {
	log := s.l.With().
		Str("operation", "RemoveShipmentFromGroup").
		Str("groupID", groupID.String()).
		Str("shipmentID", shipmentID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionUpdate,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check consolidation update permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to update this consolidation group",
		)
	}

	// * Get the consolidation group to verify it exists and check status
	group, err := s.consolidationRepo.Get(ctx, groupID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation group")
		return eris.Wrap(err, "get consolidation group")
	}

	// * Check if the group is in a valid status for removing shipments
	if group.Status == consolidation.GroupStatusCompleted ||
		group.Status == consolidation.GroupStatusCanceled {
		return errors.NewValidationError(
			"status",
			errors.ErrInvalid,
			"Cannot remove shipments from a completed or canceled consolidation group",
		)
	}

	// * Remove the shipment from the group
	err = s.consolidationRepo.RemoveShipmentFromGroup(ctx, groupID, shipmentID)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove shipment from group")
		return eris.Wrap(err, "remove shipment from group")
	}

	// * Log the action for audit
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:   permission.ResourceConsolidation,
			ResourceID: groupID.String(),
			Action:     permission.ActionUpdate,
			UserID:     userID,
			CurrentState: jsonutils.MustToJSON(
				map[string]string{"shipmentID": shipmentID.String()},
			),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("shipment removed from consolidation group"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment removal")
	}

	return nil
}

// GetGroupShipments retrieves all shipments in a consolidation group.
// It validates permissions before returning the shipment list.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group.
//   - userID: The ID of the user making the request.
//   - orgID: Organization ID for tenant filtering.
//   - buID: Business Unit ID for tenant filtering.
//
// Returns:
//   - []*shipment.Shipment: List of shipments in the group.
//   - error: If permissions fail or query fails.
func (s *Service) GetGroupShipments(
	ctx context.Context,
	groupID, userID, orgID, buID pulid.ID,
) ([]*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "GetGroupShipments").
		Str("groupID", groupID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionRead,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check consolidation read permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this consolidation group",
		)
	}

	// * Get the shipments in the group
	shipments, err := s.consolidationRepo.GetGroupShipments(ctx, groupID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get group shipments")
		return nil, eris.Wrap(err, "get group shipments")
	}

	return shipments, nil
}

// CancelConsolidation cancels a consolidation group and all associated shipments.
// This is a critical operation that requires proper permissions.
//
// Parameters:
//   - ctx: Context for request scope and cancellation.
//   - groupID: The ID of the consolidation group to cancel.
//   - userID: The ID of the user canceling the group.
//   - orgID: Organization ID for tenant filtering.
//   - buID: Business Unit ID for tenant filtering.
//   - reason: The reason for cancellation.
//
// Returns:
//   - error: If permissions fail or cancellation fails.
func (s *Service) CancelConsolidation(
	ctx context.Context,
	groupID, userID, orgID, buID pulid.ID,
	reason string,
) error {
	log := s.l.With().
		Str("operation", "CancelConsolidation").
		Str("groupID", groupID.String()).
		Logger()

	// * Check permissions
	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceConsolidation,
				Action:         permission.ActionUpdate,
				BusinessUnitID: buID,
				OrganizationID: orgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return eris.Wrap(err, "check consolidation update permissions")
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to cancel this consolidation group",
		)
	}

	// * Get the original group for audit logging
	original, err := s.consolidationRepo.Get(ctx, groupID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get consolidation group")
		return eris.Wrap(err, "get consolidation group")
	}

	// * Cancel the consolidation group and all shipments
	err = s.consolidationRepo.CancelConsolidation(ctx, groupID)
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel consolidation")
		return eris.Wrap(err, "cancel consolidation")
	}

	// * Log the cancellation for audit
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:   permission.ResourceConsolidation,
			ResourceID: groupID.String(),
			Action:     permission.ActionUpdate,
			UserID:     userID,
			CurrentState: jsonutils.MustToJSON(
				map[string]string{"status": "Canceled", "reason": reason},
			),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		audit.WithComment("Consolidation group canceled"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log consolidation cancellation")
	}

	return nil
}

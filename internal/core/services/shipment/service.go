package shipment

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	dedicatedlaneservice "github.com/emoss08/trenova/internal/core/services/dedicatedlane"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/jobs"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmentvalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger                     *logger.Logger
	Repo                       repositories.ShipmentRepository
	ProNumberRepo              repositories.ProNumberRepository
	PermService                services.PermissionService
	AuditService               services.AuditService
	StreamingService           services.StreamingService
	Validator                  *shipmentvalidator.Validator
	DedicatedLaneAssignService *dedicatedlaneservice.AssignmentService
	JobService                 jobs.JobServiceInterface
}

type Service struct {
	l             *zerolog.Logger
	repo          repositories.ShipmentRepository
	proNumberRepo repositories.ProNumberRepository
	ps            services.PermissionService
	as            services.AuditService
	ss            services.StreamingService
	v             *shipmentvalidator.Validator
	dlas          *dedicatedlaneservice.AssignmentService
	js            jobs.JobServiceInterface
}

//nolint:gocritic // The p parameter is passed using fx.In
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipment").
		Logger()

	return &Service{
		l:             &log,
		repo:          p.Repo,
		proNumberRepo: p.ProNumberRepo,
		ps:            p.PermService,
		as:            p.AuditService,
		ss:            p.StreamingService,
		v:             p.Validator,
		dlas:          p.DedicatedLaneAssignService,
		js:            p.JobService,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *repositories.ListShipmentOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, t := range result.Items {
		options[i] = &types.SelectOption{
			Value: t.GetID(),
			Label: t.ProNumber,
		}
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListShipmentOptions,
) (*ports.ListResult[*shipment.Shipment], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read shipments")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list shipments")
		return nil, err
	}

	return entities, nil
}

func (s *Service) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*ports.ListResult[*shipment.Shipment], error) {
	log := s.l.With().
		Str("operation", "GetPreviousRates").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceShipment,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read previous rates",
		)
	}

	entities, err := s.repo.GetPreviousRates(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get previous rates")
		return nil, err
	}

	return entities, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts *repositories.GetShipmentByIDOptions,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this shipment")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", shp.ProNumber).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionCreate,
				BusinessUnitID: shp.BusinessUnitID,
				OrganizationID: shp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a shipment")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, shp); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, shp, userID)
	if err != nil {
		return nil, err
	}

	// Check for dedicated lane auto-assignment
	if err = s.dlas.HandleDedicatedLaneOperations(ctx, createdEntity); err != nil {
		log.Error().Err(err).Msg("failed to handle dedicated lane operations")
		// Don't fail the shipment creation if dedicated lane assignment fails
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment created"),
		audit.WithCritical(),
		audit.WithCategory("operations"),
		audit.WithMetadata(map[string]any{
			"proNumber":  createdEntity.ProNumber,
			"customerID": createdEntity.CustomerID.String(),
			"bol":        createdEntity.BOL,
		}),
		audit.WithTags("shipment-creation", "customer-"+createdEntity.CustomerID.String()),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", shp.ProNumber).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionUpdate,
				BusinessUnitID: shp.BusinessUnitID,
				OrganizationID: shp.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this shipment",
		)
	}

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, shp); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    shp.ID,
		OrgID: shp.OrganizationID,
		BuID:  shp.BusinessUnitID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, shp, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCategory("operations"),
		audit.WithMetadata(map[string]any{
			"proNumber":  updatedEntity.ProNumber,
			"customerID": updatedEntity.CustomerID.String(),
			"bol":        updatedEntity.BOL,
		}),
		audit.WithTags("shipment-update", "customer-"+updatedEntity.CustomerID.String()),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment update")
	}

	return updatedEntity, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "Cancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.CanceledByID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionCancel,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to cancel this shipment",
		)
	}

	// get the original shipment
	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	if err := s.v.ValidateCancellation(original); err != nil {
		return nil, err
	}

	newEntity, err := s.repo.Cancel(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to cancel shipment")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     req.ShipmentID.String(),
			Action:         permission.ActionCancel,
			UserID:         req.CanceledByID,
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Shipment canceled"),
		audit.WithDiff(original, newEntity),
		audit.WithCategory("operations"),
		audit.WithMetadata(map[string]any{
			"proNumber":  newEntity.ProNumber,
			"customerID": newEntity.CustomerID.String(),
			"bol":        newEntity.BOL,
		}),
		audit.WithTags("shipment-cancellation", "customer-"+newEntity.CustomerID.String()),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment cancellation")
	}

	return newEntity, nil
}

func (s *Service) UnCancel(
	ctx context.Context,
	req *repositories.UnCancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "UnCancel").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceShipment,
			Action:         permission.ActionUpdate,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to un-cancel this shipment",
		)
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	log.Info().Interface("original", original).Msg("original shipment before un-cancel")
	if original.Status != shipment.StatusCanceled {
		return nil, errors.NewBusinessError("Shipment is not canceled")
	}

	updatedEntity, err := s.repo.UnCancel(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to un-cancel shipment")
		return nil, err
	}

	return updatedEntity, nil
}

func (s *Service) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "TransferOwnership").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	original, err := s.repo.GetByID(ctx, &repositories.GetShipmentByIDOptions{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment")
		return nil, err
	}

	result, err := s.ps.HasPermission(ctx, &services.PermissionCheck{
		UserID:         req.UserID,
		Resource:       permission.ResourceShipment,
		Action:         permission.ActionManage,
		BusinessUnitID: req.BuID,
		OrganizationID: req.OrgID,
	},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	isNotOwner := original.OwnerID != nil && *original.OwnerID != req.UserID
	hasNoManagePermission := !result.Allowed

	// * User must be either the current owner OR have manage permission
	if isNotOwner && hasNoManagePermission {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to transfer ownership of this shipment",
		)
	}

	log.Info().Interface("req", req).Msg("req for transfer ownership")

	updatedEntity, err := s.repo.TransferOwnership(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to transfer ownership of shipment")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         req.UserID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment ownership transferred"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCategory("operations"),
		audit.WithTags("ownership-transfer"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment ownership transfer")
	}

	return updatedEntity, nil
}

func (s *Service) Duplicate(
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) error {
	log := s.l.With().
		Str("operation", "Duplicate").
		Str("shipmentID", req.ShipmentID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceShipment,
				Action:         permission.ActionDuplicate,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to duplicate this shipment",
		)
	}

	// * Validate the request
	if err := req.Validate(ctx); err != nil {
		return err
	}

	payload := &jobs.DuplicateShipmentPayload{
		BasePayload: jobs.BasePayload{
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
			UserID:         req.UserID,
		},
		ShipmentID:               req.ShipmentID,
		Count:                    req.Count,
		OverrideDates:            req.OverrideDates,
		IncludeCommodities:       req.IncludeCommodities,
		IncludeAdditionalCharges: req.IncludeAdditionalCharges,
	}

	if _, err = s.js.Enqueue(
		jobs.JobTypeDuplicateShipment,
		payload,
		jobs.DefaultJobOptions(),
	); err != nil {
		log.Error().Err(err).Msg("failed to enqueue shipment duplication job")
		return err
	}

	return nil
}

func (s *Service) CheckForDuplicateBOLs(ctx context.Context, shp *shipment.Shipment) error {
	log := s.l.With().
		Str("operation", "CheckForDuplicateBOLs").
		Str("bol", shp.BOL).
		Logger()

	me := errors.NewMultiError()

	// Skip check if BOL is empty
	if shp.BOL == "" {
		return nil
	}

	// Determine if we should exclude the current shipment ID (during updates)
	var excludeID *pulid.ID
	if !shp.ID.IsNil() {
		excludeID = &shp.ID
		log.Debug().
			Str("excludeID", shp.ID.String()).
			Msg("excluding current shipment from duplicate check")
	}

	// Call repository function to check for duplicates
	duplicates, err := s.repo.CheckForDuplicateBOLs(
		ctx,
		shp.BOL,
		shp.OrganizationID,
		shp.BusinessUnitID,
		excludeID,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check for duplicate BOLs")
		return errors.NewBusinessError("Failed to check for duplicate BOLs").WithInternal(err)
	}

	// Add any duplicates found to the multi-error
	if len(duplicates) > 0 {
		log.Info().
			Int("duplicateCount", len(duplicates)).
			Msg("duplicate BOLs found")

		proNumbers := make([]string, 0, len(duplicates))
		for _, dup := range duplicates {
			proNumbers = append(proNumbers, dup.ProNumber)
		}

		me.Add("bol", errors.ErrInvalid, fmt.Sprintf(
			"BOL is already in use by shipment(s) with Pro Number(s): %s",
			strings.Join(proNumbers, ", "),
		))
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

func (s *Service) MarkReadyToBill(
	ctx context.Context,
	req *repositories.UpdateShipmentStatusRequest,
) (*shipment.Shipment, error) {
	log := s.l.With().
		Str("operation", "MarkReadyToBill").
		Str("shipmentID", req.GetOpts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.GetOpts.UserID,
			Resource:       permission.ResourceShipment,
			Action:         permission.ActionUpdate,
			BusinessUnitID: req.GetOpts.BuID,
			OrganizationID: req.GetOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to mark this shipment as ready to bill",
		)
	}

	// TODO(wolfred): Validate the requirements set by that particular customer on the server before allowing the shipment to be marked ready-to-bill

	updatedEntity, err := s.repo.UpdateStatus(ctx, &repositories.UpdateShipmentStatusRequest{
		GetOpts: req.GetOpts,
		Status:  shipment.StatusReadyToBill,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment status")
		return nil, err
	}

	return updatedEntity, nil
}

func (s *Service) CalculateShipmentTotals(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	log := s.l.With().Str("operation", "CalculateShipmentTotals").Logger()

	// We do not persist any data here; the calculator only needs an in-memory
	// copy of the shipment with the user-supplied fields filled in. The heavy
	// lifting is delegated to the repository which has access to the shared
	// ShipmentCalculator instance.

	// NOTE: No explicit permission check is required because we are not
	// accessing or mutating any stored resources. If that ever changes, a
	// read permission check similar to the one in List/Get can be added.

	resp, err := s.repo.CalculateShipmentTotals(ctx, shp, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to calculate shipment totals")
		return nil, err
	}

	return resp, nil
}

// LiveStream provides real-time streaming of shipment changes via CDC
func (s *Service) LiveStream(c *fiber.Ctx) error {
	s.l.Info().Msg("Starting CDC-based live stream for shipments")

	return s.ss.StreamData(c, "shipments")
}

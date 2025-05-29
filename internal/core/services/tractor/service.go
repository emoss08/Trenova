package tractor

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/tractorvalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.TractorRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *tractorvalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	v    *tractorvalidator.Validator
	repo repositories.TractorRepository
	ps   services.PermissionService
	as   services.AuditService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "tractor").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, t := range result.Items {
		options = append(options, &types.SelectOption{
			Value: t.GetID(),
			Label: t.Code,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*ports.ListResult[*tractor.Tractor], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceTractor,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read tractors")
	}

	entities, err := s.repo.List(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to list tractors")
		return nil, err
	}

	return &ports.ListResult[*tractor.Tractor]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("tractorID", req.TractorID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.UserID,
				Resource:       permission.ResourceTractor,
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

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this tractor")
	}

	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get tractor")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	lc *tractor.Tractor,
	userID pulid.ID,
) (*tractor.Tractor, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", lc.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceTractor,
				Action:         permission.ActionCreate,
				BusinessUnitID: lc.BusinessUnitID,
				OrganizationID: lc.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a tractor")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, lc); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, lc)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTractor,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Tractor created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log tractor creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	t *tractor.Tractor,
	userID pulid.ID,
) (*tractor.Tractor, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", t.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceTractor,
				Action:         permission.ActionUpdate,
				BusinessUnitID: t.BusinessUnitID,
				OrganizationID: t.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this tractor",
		)
	}

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, t); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetTractorByIDRequest{
		TractorID: t.ID,
		OrgID:     t.OrganizationID,
		BuID:      t.BusinessUnitID,
		FilterOptions: repositories.TractorFilterOptions{
			IncludeWorkerDetails:    true,
			IncludeEquipmentDetails: true,
			IncludeFleetDetails:     true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get tractor")
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, t)
	if err != nil {
		log.Error().Err(err).Msg("failed to update tractor")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTractor,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Tractor updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log tractor update")
	}

	return updatedEntity, nil
}

func (s *Service) Assignment(
	ctx context.Context,
	req repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	log := s.l.With().
		Str("operation", "Assignment").
		Str("tractorID", req.TractorID.String()).
		Logger()

	// ! We do not need to check permissions for this operation
	assignment, err := s.repo.Assignment(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to get tractor assignment")
		return nil, err
	}

	return assignment, nil
}

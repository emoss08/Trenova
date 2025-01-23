package servicetype

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/servicetypevalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.ServiceTypeRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *servicetypevalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ServiceTypeRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *servicetypevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmenttype").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) SelectOptions(ctx context.Context, opts *ports.LimitOffsetQueryOptions) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "select service types")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, st := range result.Items {
		options[i] = &types.SelectOption{
			Value: st.GetID(),
			Label: st.Code,
			Color: st.Color,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*servicetype.ServiceType], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceServiceType,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read service types")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list service types")
		return nil, eris.Wrap(err, "list service types")
	}

	return &ports.ListResult[*servicetype.ServiceType]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetServiceTypeByIDOptions) (*servicetype.ServiceType, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("serviceTypeID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceServiceType,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read service type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read this service type")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get service type")
		return nil, eris.Wrap(err, "get service type")
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, st *servicetype.ServiceType, userID pulid.ID) (*servicetype.ServiceType, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", st.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceServiceType,
				Action:         permission.ActionCreate,
				BusinessUnitID: st.BusinessUnitID,
				OrganizationID: st.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create service type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a service type")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, st); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, st)
	if err != nil {
		return nil, eris.Wrap(err, "create service type")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceServiceType,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Service Type created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log service type creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, st *servicetype.ServiceType, userID pulid.ID) (*servicetype.ServiceType, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", st.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceServiceType,
				Action:         permission.ActionUpdate,
				BusinessUnitID: st.BusinessUnitID,
				OrganizationID: st.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update service type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this service type")
	}

	// Validate the service type
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, st); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetServiceTypeByIDOptions{
		ID:    st.ID,
		OrgID: st.OrganizationID,
		BuID:  st.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get service type")
	}

	updatedEntity, err := s.repo.Update(ctx, st)
	if err != nil {
		log.Error().Err(err).Msg("failed to update service type")
		return nil, eris.Wrap(err, "update service type")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceServiceType,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Service Type updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log service type update")
	}

	return updatedEntity, nil
}

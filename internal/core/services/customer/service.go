// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/customervalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.CustomerRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *customervalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.CustomerRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *customervalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "customer").
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
	opts *repositories.ListCustomerOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, err
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, loc := range result.Items {
		options = append(options, &types.SelectOption{
			Value: loc.GetID(),
			Label: loc.Name,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts *repositories.ListCustomerOptions,
) (*ports.ListResult[*customer.Customer], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceCustomer,
				Action:         permission.ActionRead,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read customers")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list customers")
		return nil, err
	}

	return &ports.ListResult[*customer.Customer]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetCustomerByIDOptions,
) (*customer.Customer, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("customerID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceCustomer,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read this customer")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get customer")
		return nil, err
	}

	return entity, nil
}

func (s *Service) GetDocumentRequirements(
	ctx context.Context,
	cusID pulid.ID,
) ([]*repositories.CustomerDocRequirementResponse, error) {
	log := s.l.With().
		Str("operation", "GetDocumentRequirements").
		Str("customerID", cusID.String()).
		Logger()

	requirements, err := s.repo.GetDocumentRequirements(ctx, cusID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get document requirements")
		return nil, err
	}

	// ! We don't need to check permissions here because their may be times where a user doesn't have permission to read a customer but
	// ! they should be able to see the document requirements

	return requirements, nil
}

func (s *Service) Create(
	ctx context.Context,
	cus *customer.Customer,
	userID pulid.ID,
) (*customer.Customer, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", cus.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceCustomer,
				Action:         permission.ActionCreate,
				BusinessUnitID: cus.BusinessUnitID,
				OrganizationID: cus.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a location")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, cus); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, cus)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCustomer,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Customer created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log customer creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	cus *customer.Customer,
	userID pulid.ID,
) (*customer.Customer, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", cus.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceCustomer,
				Action:         permission.ActionUpdate,
				BusinessUnitID: cus.BusinessUnitID,
				OrganizationID: cus.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this customer",
		)
	}

	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, cus); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCustomerByIDOptions{
		ID:                    cus.ID,
		OrgID:                 cus.OrganizationID,
		BuID:                  cus.BusinessUnitID,
		IncludeBillingProfile: true,
		IncludeEmailProfile:   true,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, cus)
	if err != nil {
		log.Error().Err(err).Msg("failed to update customer")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceCustomer,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Customer updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log customer update")
	}

	return updatedEntity, nil
}

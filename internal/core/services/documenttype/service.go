package documenttype

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billing"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/documenttypevalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.DocumentTypeRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *documenttypevalidator.Validator
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.DocumentTypeRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *documenttypevalidator.Validator
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "documenttype").
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
		return nil, err
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, dt := range result.Items {
		options[i] = &types.SelectOption{
			Value: dt.ID.String(),
			Label: dt.Name,
			Color: dt.Color,
		}
	}

	return options, nil
}

func (s *Service) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*billing.DocumentType], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceDocumentType,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
				OrganizationID: opts.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read document types")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list document types")
		return nil, err
	}

	return &ports.ListResult[*billing.DocumentType]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(ctx context.Context, opts repositories.GetDocumentTypeByIDRequest) (*billing.DocumentType, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("documentTypeID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceDocumentType,
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
		return nil, errors.NewAuthorizationError("You do not have permission to read this document type")
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get document type")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(ctx context.Context, dt *billing.DocumentType, userID pulid.ID) (*billing.DocumentType, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("name", dt.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDocumentType,
				Action:         permission.ActionCreate,
				BusinessUnitID: dt.BusinessUnitID,
				OrganizationID: dt.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create document type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a document type")
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, dt); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, dt)
	if err != nil {
		return nil, eris.Wrap(err, "create document type")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocumentType,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Document type created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log document type creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(ctx context.Context, dt *billing.DocumentType, userID pulid.ID) (*billing.DocumentType, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("name", dt.Name).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceDocumentType,
				Action:         permission.ActionUpdate,
				BusinessUnitID: dt.BusinessUnitID,
				OrganizationID: dt.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to update this document type")
	}

	// Validate the fleet code
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, dt); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetDocumentTypeByIDRequest{
		ID:     dt.ID,
		OrgID:  dt.OrganizationID,
		BuID:   dt.BusinessUnitID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, dt)
	if err != nil {
		log.Error().Err(err).Msg("failed to update document type")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocumentType,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Document type updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log document type update")
	}

	return updatedEntity, nil
}

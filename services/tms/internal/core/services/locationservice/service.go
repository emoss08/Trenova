package locationservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          repositories.LocationRepository
	UsStateRepo   repositories.UsStateRepository
	Validator     *Validator
	AuditService  services.AuditService
	Transformer   services.DataTransformer
	CodeGenerator services.LocationCodeGenerator
}

type Service struct {
	l             *zap.Logger
	repo          repositories.LocationRepository
	usStateRepo   repositories.UsStateRepository
	validator     *Validator
	auditService  services.AuditService
	transformer   services.DataTransformer
	codeGenerator services.LocationCodeGenerator
}

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.location"),
		repo:          p.Repo,
		usStateRepo:   p.UsStateRepo,
		validator:     p.Validator,
		auditService:  p.AuditService,
		transformer:   p.Transformer,
		codeGenerator: p.CodeGenerator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.LocationSelectOptionsRequest,
) (*pagination.ListResult[*location.Location], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetLocationsByIDsRequest{
		TenantInfo:  req.TenantInfo,
		LocationIDs: req.LocationIDs,
	})
	if err != nil {
		log.Error("failed to get original locations", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update location status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*location.Location]{
			Resource:  permission.ResourceLocation,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("Location status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *location.Location,
	actor *services.RequestActor,
) (*location.Location, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	if err := s.transformer.TransformLocation(ctx, entity); err != nil {
		log.Error("failed to transform location", zap.Error(err))
		return nil, err
	}
	entity.NormalizeGeofence()

	autoGenerateCode := strings.TrimSpace(entity.Code) == ""
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.createLocation(ctx, entity, autoGenerateCode)
	if err != nil {
		log.Error("failed to create location", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceLocation,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Location created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) createLocation(
	ctx context.Context,
	entity *location.Location,
	autoGenerateCode bool,
) (*location.Location, error) {
	if !autoGenerateCode {
		created, err := s.repo.Create(ctx, entity)
		if err != nil && isLocationCodeConflict(err) {
			return nil, duplicateLocationCodeError()
		}

		return created, err
	}

	input, err := s.locationCodeInput(ctx, entity)
	if err != nil {
		return nil, err
	}

	const maxAttempts = 3

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		code, genErr := s.codeGenerator.Generate(ctx, services.LocationCodeGenerateRequest{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			Input:          input,
		})
		if genErr != nil {
			return nil, genErr
		}
		entity.Code = code

		created, createErr := s.repo.Create(ctx, entity)
		if createErr == nil {
			return created, nil
		}
		if !isLocationCodeConflict(createErr) {
			return nil, createErr
		}

		lastErr = createErr
		if attempt == maxAttempts {
			break
		}
		if waitErr := waitBeforeLocationCodeRetry(ctx, attempt); waitErr != nil {
			return nil, waitErr
		}
	}

	return nil, errortypes.NewConflictError(
		"Unable to allocate a unique location code. Retry the request.",
	).WithInternal(lastErr)
}

func (s *Service) locationCodeInput(
	ctx context.Context,
	entity *location.Location,
) (services.LocationCodeInput, error) {
	state, err := s.usStateRepo.GetByID(ctx, repositories.GetUsStateByIDRequest{
		StateID: entity.StateID,
	})
	if err != nil {
		return services.LocationCodeInput{}, err
	}

	return services.LocationCodeInput{
		Name:              entity.Name,
		City:              entity.City,
		StateAbbreviation: state.Abbreviation,
		PostalCode:        entity.PostalCode,
	}, nil
}

func isLocationCodeConflict(err error) bool {
	return dberror.IsUniqueConstraintViolation(err) &&
		dberror.ExtractConstraintName(err) == "idx_locations_code"
}

func duplicateLocationCodeError() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"code",
		errortypes.ErrDuplicate,
		"Location with this code already exists in your organization",
	)
	return multiErr
}

func waitBeforeLocationCodeRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(attempt*25) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Service) Update(
	ctx context.Context,
	entity *location.Location,
	actor *services.RequestActor,
) (*location.Location, error) {
	auditActor := actor.AuditActor()
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("principalType", string(auditActor.PrincipalType)),
		zap.String("principalID", auditActor.PrincipalID.String()),
	)

	if err := s.transformer.TransformLocation(ctx, entity); err != nil {
		log.Error("failed to transform location", zap.Error(err))
		return nil, err
	}
	entity.NormalizeGeofence()

	original, err := s.repo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original location", zap.Error(err))
		return nil, err
	}

	if strings.TrimSpace(entity.Code) == "" {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("code", errortypes.ErrRequired, "Code is required")
		return nil, multiErr
	}
	if !strings.EqualFold(strings.TrimSpace(entity.Code), original.Code) {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("code", errortypes.ErrInvalid, "Location code cannot be changed")
		return nil, multiErr
	}
	entity.Code = original.Code
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update location", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceLocation,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Location updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

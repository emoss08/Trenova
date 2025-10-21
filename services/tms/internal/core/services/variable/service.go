package variable

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/variable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger          *zap.Logger
	Repo            repositories.VariableRepository
	AuditService    services.AuditService
	QueryValidator  *QueryValidator
	FormatValidator *FormatSQLValidator
}

type Service struct {
	l    *zap.Logger
	repo repositories.VariableRepository
	as   services.AuditService
	qv   *QueryValidator
	fv   *FormatSQLValidator
}

func NewService(p ServiceParams) services.VariableService {
	return &Service{
		l:    p.Logger.Named("service.variable"),
		repo: p.Repo,
		as:   p.AuditService,
		qv:   p.QueryValidator,
		fv:   p.FormatValidator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListVariableRequest,
) (*pagination.ListResult[*variable.Variable], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetVariableByIDRequest,
) (*variable.Variable, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *variable.Variable,
	userID pulid.ID,
) (*variable.Variable, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
		zap.String("key", entity.Key),
	)

	if err := s.qv.ValidateWithTables(entity.Query); err != nil {
		return nil, errortypes.NewValidationError(
			"query",
			errortypes.ErrInvalid,
			fmt.Sprintf("Query validation failed: %s", err.Error()),
		)
	}

	entity.IsValidated = true

	multiErr := &errortypes.MultiError{}
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Variable created"),
	)
	if err != nil {
		log.Error("failed to log variable creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *variable.Variable,
	userID pulid.ID,
) (*variable.Variable, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	if !entity.CanEdit() {
		return nil, errortypes.NewValidationError(
			"variable",
			errortypes.ErrInvalid,
			"System variables cannot be modified",
		)
	}

	original, err := s.repo.GetByID(ctx, repositories.GetVariableByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if original.Query != entity.Query {
		if err = s.qv.ValidateWithTables(entity.Query); err != nil {
			return nil, errortypes.NewValidationError(
				"query",
				errortypes.ErrInvalid,
				fmt.Sprintf("Query validation failed: %s", err.Error()),
			)
		}
		entity.IsValidated = true
	}

	multiErr := &errortypes.MultiError{}
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update variable", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Variable updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log variable update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.GetVariableByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("variableID", req.ID.String()),
		zap.String("userID", req.UserID.String()),
	)

	v, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if !v.CanDelete() {
		return errortypes.NewValidationError(
			"variable",
			errortypes.ErrInvalid,
			"System variables cannot be deleted",
		)
	}

	err = s.repo.Delete(ctx, req)
	if err != nil {
		log.Error("failed to delete variable", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     req.ID.String(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(v),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Variable deleted"),
	)
	if err != nil {
		log.Error("failed to log variable deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) ValidateQuery(
	req *services.ValidateQueryRequest,
) (*services.ValidateResponse, error) {
	if err := s.qv.ValidateWithTables(req.Query); err != nil {
		return &services.ValidateResponse{ //nolint:nilerr // we do catch the error in the response
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	return &services.ValidateResponse{
		Valid: true,
	}, nil
}

func (s *Service) TestVariable(
	ctx context.Context,
	req *services.TestVariableRequest,
) (*services.TestVariableResponse, error) {
	log := s.l.With(
		zap.String("operation", "TestVariable"),
		zap.String("query", req.Query),
	)

	if err := s.qv.ValidateWithTables(req.Query); err != nil {
		return nil, err
	}

	// Create a temporary variable for testing
	testVar := &variable.Variable{
		Query:       req.Query,
		IsValidated: true,
	}

	result, err := s.repo.ResolveVariable(ctx, repositories.ResolveVariableRequest{
		Variable: testVar,
		Params:   req.TestParams,
	})
	if err != nil {
		log.Error("failed to test variable", zap.Error(err))
		return nil, err
	}

	return &services.TestVariableResponse{
		Result: result,
	}, nil
}

func (s *Service) GetVariablesByContext(
	ctx context.Context,
	req repositories.GetVariablesByContextRequest,
) ([]*variable.Variable, error) {
	return s.repo.GetVariablesByContext(ctx, req)
}

func (s *Service) ListFormats(
	ctx context.Context,
	req *repositories.ListVariableFormatRequest,
) (*pagination.ListResult[*variable.VariableFormat], error) {
	return s.repo.ListFormats(ctx, req)
}

func (s *Service) GetFormat(
	ctx context.Context,
	req repositories.GetVariableFormatByIDRequest,
) (*variable.VariableFormat, error) {
	return s.repo.GetFormatByID(ctx, req)
}

func (s *Service) CreateFormat(
	ctx context.Context,
	entity *variable.VariableFormat,
	userID pulid.ID,
) (*variable.VariableFormat, error) {
	log := s.l.With(
		zap.String("operation", "CreateFormat"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
		zap.String("name", entity.Name),
	)

	if err := s.fv.Validate(entity.FormatSQL); err != nil {
		return nil, errortypes.NewValidationError(
			"formatSQL",
			errortypes.ErrInvalid,
			fmt.Sprintf("Format SQL validation failed: %s", err.Error()),
		)
	}

	multiErr := &errortypes.MultiError{}
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	createdEntity, err := s.repo.CreateFormat(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Variable format created"),
	)
	if err != nil {
		log.Error("failed to log format creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) UpdateFormat(
	ctx context.Context,
	entity *variable.VariableFormat,
	userID pulid.ID,
) (*variable.VariableFormat, error) {
	log := s.l.With(
		zap.String("operation", "UpdateFormat"),
		zap.String("formatID", entity.ID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	if entity.IsSystem {
		return nil, errortypes.NewValidationError(
			"format",
			errortypes.ErrInvalid,
			"System formats cannot be modified",
		)
	}

	original, err := s.repo.GetFormatByID(ctx, repositories.GetVariableFormatByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if original.FormatSQL != entity.FormatSQL {
		if err = s.fv.Validate(entity.FormatSQL); err != nil {
			return nil, errortypes.NewValidationError(
				"formatSQL",
				errortypes.ErrInvalid,
				fmt.Sprintf("Format SQL validation failed: %s", err.Error()),
			)
		}
	}

	multiErr := &errortypes.MultiError{}
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updatedEntity, err := s.repo.UpdateFormat(ctx, entity)
	if err != nil {
		log.Error("failed to update format", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Variable format updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log format update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) DeleteFormat(
	ctx context.Context,
	req repositories.GetVariableFormatByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteFormat"),
		zap.String("formatID", req.ID.String()),
		zap.String("userID", req.UserID.String()),
	)

	f, err := s.repo.GetFormatByID(ctx, req)
	if err != nil {
		return err
	}

	if f.IsSystem {
		return errortypes.NewValidationError(
			"format",
			errortypes.ErrInvalid,
			"System formats cannot be deleted",
		)
	}

	err = s.repo.DeleteFormat(ctx, req)
	if err != nil {
		log.Error("failed to delete format", zap.Error(err))
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceVariable,
			ResourceID:     req.ID.String(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(f),
			OrganizationID: req.OrgID,
			BusinessUnitID: req.BuID,
		},
		audit.WithComment("Variable format deleted"),
	)
	if err != nil {
		log.Error("failed to log format deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) ValidateFormatSQL(
	req *services.ValidateFormatSQLRequest,
) (*services.ValidateResponse, error) {
	if err := s.fv.Validate(req.FormatSQL); err != nil {
		return &services.ValidateResponse{ //nolint:nilerr // we do catch the error in the response
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	return &services.ValidateResponse{
		Valid: true,
	}, nil
}

func (s *Service) TestFormat(
	ctx context.Context,
	req *services.TestFormatRequest,
) (*services.TestFormatResponse, error) {
	log := s.l.With(
		zap.String("operation", "TestFormat"),
		zap.String("formatSQL", req.FormatSQL),
		zap.String("testValue", req.TestValue),
	)

	if err := s.fv.Validate(req.FormatSQL); err != nil {
		return nil, err
	}

	result, err := s.repo.ExecuteFormatSQL(ctx, repositories.ExecuteFormatSQLRequest{
		FormatSQL: req.FormatSQL,
		Value:     req.TestValue,
	})
	if err != nil {
		log.Error("failed to test format", zap.Error(err))
		return nil, err
	}

	return &services.TestFormatResponse{
		Result: result,
	}, nil
}

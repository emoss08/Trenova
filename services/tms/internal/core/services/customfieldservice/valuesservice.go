package customfieldservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ValuesServiceParams struct {
	fx.In

	Logger         *zap.Logger
	ValueRepo      repositories.CustomFieldValueRepository
	DefinitionRepo repositories.CustomFieldDefinitionRepository
	Validator      *ValuesValidator
}

type ValuesService struct {
	l              *zap.Logger
	valueRepo      repositories.CustomFieldValueRepository
	definitionRepo repositories.CustomFieldDefinitionRepository
	validator      *ValuesValidator
}

func NewValuesService(p ValuesServiceParams) *ValuesService {
	return &ValuesService{
		l:              p.Logger.Named("customfield.values-service"),
		valueRepo:      p.ValueRepo,
		definitionRepo: p.DefinitionRepo,
		validator:      p.Validator,
	}
}

func (s *ValuesService) GetForResource(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceID string,
) (map[string]any, error) {
	values, err := s.valueRepo.GetByResource(
		ctx,
		&repositories.GetCustomFieldValuesByResourceRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
			ResourceID:   resourceID,
		},
	)
	if err != nil {
		s.l.Error(
			"failed to get custom field values",
			zap.Error(err),
			zap.String("resourceType", resourceType),
			zap.String("resourceID", resourceID),
		)
		return nil, err
	}

	result := make(map[string]any)
	for _, v := range values {
		result[v.DefinitionID.String()] = v.Value
	}

	return result, nil
}

func (s *ValuesService) GetForResources(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceIDs []string,
) (map[string]map[string]any, error) {
	if len(resourceIDs) == 0 {
		return make(map[string]map[string]any), nil
	}

	valuesMap, err := s.valueRepo.GetByResources(
		ctx,
		&repositories.GetCustomFieldValuesByResourcesRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
			ResourceIDs:  resourceIDs,
		},
	)
	if err != nil {
		s.l.Error(
			"failed to get custom field values for resources",
			zap.Error(err),
			zap.String("resourceType", resourceType),
			zap.Int("resourceCount", len(resourceIDs)),
		)
		return nil, err
	}

	result := make(map[string]map[string]any)
	for resourceID, values := range valuesMap {
		result[resourceID] = make(map[string]any)
		for _, v := range values {
			result[resourceID][v.DefinitionID.String()] = v.Value
		}
	}

	return result, nil
}

func (s *ValuesService) ValidateAndSave(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceID string,
	values map[string]any,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	s.validator.Validate(ctx, tenantInfo, resourceType, values, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	if err := s.valueRepo.Upsert(ctx, &repositories.UpsertCustomFieldValuesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Values:       values,
	}); err != nil {
		s.l.Error(
			"failed to save custom field values",
			zap.Error(err),
			zap.String("resourceType", resourceType),
			zap.String("resourceID", resourceID),
		)
		multiErr.Add(
			"customFields",
			errortypes.ErrSystemError,
			"Failed to save custom field values",
		)
		return multiErr
	}

	return nil
}

func (s *ValuesService) Delete(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceID string,
) error {
	return s.valueRepo.DeleteByResource(ctx, &repositories.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

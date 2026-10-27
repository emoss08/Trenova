package customfieldservice

import (
	"context"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const namedDateLayout = "2006-01-02"

func (s *ValuesService) GetNamed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceID string,
) (map[string]any, error) {
	definitions, err := s.definitionRepo.GetActiveByResourceType(
		ctx,
		customfieldports.GetActiveByResourceTypeRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 {
		return map[string]any{}, nil
	}
	values, err := s.valueRepo.GetByResource(
		ctx,
		&customfieldports.GetCustomFieldValuesByResourceRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
			ResourceID:   resourceID,
		},
	)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*customfield.CustomFieldDefinition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID.String()] = definition
	}
	named := make(map[string]any, len(values))
	for _, value := range values {
		definition, ok := byID[value.DefinitionID.String()]
		if !ok || value.Value == nil {
			continue
		}
		named[definition.Name] = namedValue(definition, value.Value)
	}
	return named, nil
}

func (s *ValuesService) SaveNamed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	resourceID string,
	values map[string]any,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	definitions, err := s.definitionRepo.GetActiveByResourceType(
		ctx,
		customfieldports.GetActiveByResourceTypeRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		s.l.Error("failed to fetch custom field definitions", zap.Error(err))
		multiErr.Add(
			"customFields",
			errortypes.ErrSystemError,
			"Failed to fetch custom field definitions",
		)
		return multiErr
	}
	existing, err := s.valueRepo.GetByResource(
		ctx,
		&customfieldports.GetCustomFieldValuesByResourceRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
			ResourceID:   resourceID,
		},
	)
	if err != nil {
		s.l.Error("failed to get custom field values", zap.Error(err))
		multiErr.Add(
			"customFields",
			errortypes.ErrSystemError,
			"Failed to get custom field values",
		)
		return multiErr
	}

	byName := make(map[string]*customfield.CustomFieldDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}
	merged := make(map[string]any, len(existing)+len(values))
	for _, value := range existing {
		merged[value.DefinitionID.String()] = value.Value
	}
	for name, value := range values {
		definition, ok := byName[name]
		if !ok {
			multiErr.Add(
				"customFields."+name,
				errortypes.ErrInvalid,
				"There is no active custom field named {0}",
				name,
			)
			continue
		}
		if value == nil {
			delete(merged, definition.ID.String())
			continue
		}
		value = storedValue(definition, value)
		s.validator.validateValue(definition, value, multiErr)
		merged[definition.ID.String()] = value
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	if err = s.valueRepo.Upsert(ctx, &customfieldports.UpsertCustomFieldValuesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Values:       merged,
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

func storedValue(definition *customfield.CustomFieldDefinition, value any) any {
	if definition.FieldType != customfield.FieldTypeNumber {
		return value
	}
	text, ok := value.(string)
	if !ok {
		return value
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return value
	}
	return number
}

func namedValue(definition *customfield.CustomFieldDefinition, value any) any {
	if definition.FieldType != customfield.FieldTypeDate {
		return value
	}
	switch date := value.(type) {
	case string:
		if parsed, err := time.Parse(time.RFC3339, date); err == nil {
			return parsed.UTC().Format(namedDateLayout)
		}
		return date
	case float64:
		return time.Unix(int64(date), 0).UTC().Format(namedDateLayout)
	case int64:
		return time.Unix(date, 0).UTC().Format(namedDateLayout)
	}
	return value
}

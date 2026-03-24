package customfieldservice

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ValuesValidatorParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.CustomFieldDefinitionRepository
}

type ValuesValidator struct {
	l    *zap.Logger
	repo repositories.CustomFieldDefinitionRepository
}

func NewValuesValidator(p ValuesValidatorParams) *ValuesValidator {
	return &ValuesValidator{
		l:    p.Logger.Named("customfield.values-validator"),
		repo: p.Repo,
	}
}

func (v *ValuesValidator) Validate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resourceType string,
	customFields map[string]any,
	multiErr *errortypes.MultiError,
) {
	definitions, err := v.repo.GetActiveByResourceType(
		ctx,
		repositories.GetActiveByResourceTypeRequest{
			TenantInfo:   tenantInfo,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		v.l.Error(
			"failed to fetch custom field definitions",
			zap.Error(err),
			zap.String("resourceType", resourceType),
		)
		multiErr.Add(
			"customFields",
			errortypes.ErrSystemError,
			"Failed to fetch custom field definitions",
		)
		return
	}

	if len(definitions) == 0 && len(customFields) == 0 {
		return
	}

	defMap := make(map[string]*customfield.CustomFieldDefinition)
	for _, def := range definitions {
		defMap[def.ID.String()] = def
	}

	for _, def := range definitions {
		if def.IsRequired {
			if _, exists := customFields[def.ID.String()]; !exists {
				multiErr.Add(
					fmt.Sprintf("customFields.%s", def.ID.String()),
					errortypes.ErrRequired,
					fmt.Sprintf("%s is required", def.Label),
				)
			}
		}
	}

	for fieldID, value := range customFields {
		def, exists := defMap[fieldID]
		if !exists {
			multiErr.Add(
				"customFields",
				errortypes.ErrInvalid,
				fmt.Sprintf("Unknown custom field ID: %s", fieldID),
			)
			continue
		}
		v.validateValue(def, value, multiErr)
	}
}

func (v *ValuesValidator) validateValue(
	def *customfield.CustomFieldDefinition,
	value any,
	multiErr *errortypes.MultiError,
) {
	fieldPath := fmt.Sprintf("customFields.%s", def.ID.String())

	if value == nil {
		if def.IsRequired {
			multiErr.Add(
				fieldPath,
				errortypes.ErrRequired,
				fmt.Sprintf("%s is required", def.Label),
			)
		}
		return
	}

	switch def.FieldType {
	case customfield.FieldTypeText:
		v.validateText(def, value, fieldPath, multiErr)
	case customfield.FieldTypeNumber:
		v.validateNumber(def, value, fieldPath, multiErr)
	case customfield.FieldTypeDate:
		v.validateDate(value, fieldPath, multiErr)
	case customfield.FieldTypeBoolean:
		v.validateBoolean(value, fieldPath, multiErr)
	case customfield.FieldTypeSelect:
		v.validateSelect(def, value, fieldPath, multiErr)
	case customfield.FieldTypeMultiSelect:
		v.validateMultiSelect(def, value, fieldPath, multiErr)
	}
}

func (v *ValuesValidator) validateText(
	def *customfield.CustomFieldDefinition,
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	str, ok := value.(string)
	if !ok {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", def.Label),
		)
		return
	}

	if def.ValidationRules == nil {
		return
	}

	rules := def.ValidationRules
	if rules.MinLength != nil && len(str) < *rules.MinLength {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be at least %d characters", def.Label, *rules.MinLength),
		)
	}

	if rules.MaxLength != nil && len(str) > *rules.MaxLength {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be at most %d characters", def.Label, *rules.MaxLength),
		)
	}

	if rules.Pattern != nil && *rules.Pattern != "" {
		matched, err := regexp.MatchString(*rules.Pattern, str)
		if err != nil || !matched {
			multiErr.Add(
				fieldPath,
				errortypes.ErrInvalid,
				fmt.Sprintf("%s does not match the required pattern", def.Label),
			)
		}
	}
}

func (v *ValuesValidator) validateNumber(
	def *customfield.CustomFieldDefinition,
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	var num float64
	switch n := value.(type) {
	case float64:
		num = n
	case int:
		num = float64(n)
	case int64:
		num = float64(n)
	default:
		multiErr.Add(fieldPath, errortypes.ErrInvalid, fmt.Sprintf("%s must be a number", def.Label))
		return
	}

	if def.ValidationRules == nil {
		return
	}

	rules := def.ValidationRules
	if rules.Min != nil && num < float64(*rules.Min) {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be at least %d", def.Label, *rules.Min),
		)
	}

	if rules.Max != nil && num > float64(*rules.Max) {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be at most %d", def.Label, *rules.Max),
		)
	}
}

func (v *ValuesValidator) validateDate(
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	switch d := value.(type) {
	case string:
		if _, parseErr := time.Parse(time.RFC3339, d); parseErr != nil {
			if _, dateErr := time.Parse("2006-01-02", d); dateErr != nil {
				multiErr.Add(fieldPath, errortypes.ErrInvalid, "Invalid date format. Use ISO 8601 format (e.g., 2024-01-15 or 2024-01-15T10:30:00Z)")
			}
		}
	case float64:
		if d < 0 {
			multiErr.Add(fieldPath, errortypes.ErrInvalid, "Invalid timestamp")
		}
	case int64:
		if d < 0 {
			multiErr.Add(fieldPath, errortypes.ErrInvalid, "Invalid timestamp")
		}
	default:
		multiErr.Add(fieldPath, errortypes.ErrInvalid, "Date must be a string (ISO 8601) or unix timestamp")
	}
}

func (v *ValuesValidator) validateBoolean(
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	if _, ok := value.(bool); !ok {
		multiErr.Add(fieldPath, errortypes.ErrInvalid, "Value must be a boolean")
	}
}

func (v *ValuesValidator) validateSelect(
	def *customfield.CustomFieldDefinition,
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	str, ok := value.(string)
	if !ok {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be a string", def.Label),
		)
		return
	}

	valid := false
	for _, opt := range def.Options {
		if opt.Value == str {
			valid = true
			break
		}
	}

	if !valid {
		multiErr.Add(fieldPath, errortypes.ErrInvalid,
			fmt.Sprintf("%s is not a valid option for %s", str, def.Label))
	}
}

func (v *ValuesValidator) validateMultiSelect(
	def *customfield.CustomFieldDefinition,
	value any,
	fieldPath string,
	multiErr *errortypes.MultiError,
) {
	arr, ok := value.([]any)
	if !ok {
		multiErr.Add(
			fieldPath,
			errortypes.ErrInvalid,
			fmt.Sprintf("%s must be an array", def.Label),
		)
		return
	}

	validOptions := make(map[string]bool)
	for _, opt := range def.Options {
		validOptions[opt.Value] = true
	}

	for i, item := range arr {
		str, isString := item.(string)
		if !isString {
			multiErr.Add(
				fieldPath,
				errortypes.ErrInvalid,
				fmt.Sprintf("Item %d must be a string", i),
			)
			continue
		}

		if !validOptions[str] {
			multiErr.Add(fieldPath, errortypes.ErrInvalid,
				fmt.Sprintf("%s is not a valid option for %s", str, def.Label))
		}
	}
}

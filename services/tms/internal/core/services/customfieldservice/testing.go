package customfieldservice

import (
	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*customfield.CustomFieldDefinition]().
			WithModelName("CustomFieldDefinition").
			Build(),
	}
}

package customfieldservice

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*customfield.CustomFieldDefinition]().
			WithModelName("CustomFieldDefinition").
			Build(),
	}
}

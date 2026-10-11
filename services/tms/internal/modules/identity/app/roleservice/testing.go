package roleservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*permission.Role]().
			WithModelName("Role").
			Build(),
	}
}

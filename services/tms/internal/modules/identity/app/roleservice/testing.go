package roleservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*permission.Role]().
			WithModelName("Role").
			Build(),
	}
}

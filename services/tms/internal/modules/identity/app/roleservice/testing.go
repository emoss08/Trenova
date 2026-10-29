package roleservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*rbac.Role]().
			WithModelName("Role").
			Build(),
	}
}

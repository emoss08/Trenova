package accounttypeservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*accounttype.AccountType]().
			WithModelName("AccountType").
			Build(),
	}
}

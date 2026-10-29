package accounttypeservice

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*accounttype.AccountType]().
			WithModelName("AccountType").
			Build(),
	}
}

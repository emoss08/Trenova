package userservice

import "github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"

func NewTestValidator() *Validator {
	return newValidator(nil)
}

func NewTestValidatorWithRepository(users tenantports.UserRepository) *Validator {
	return newValidator(users)
}

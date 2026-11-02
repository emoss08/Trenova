package userservice

import "github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"

func NewTestValidatorWithRepository(users tenantports.UserRepository) *Validator {
	return NewValidator(ValidatorParams{Repo: users})
}

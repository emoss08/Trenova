package rptmetavalidator

import (
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/rptmeta"
)

type VariableValidator struct{}

func NewVariableValidator() *VariableValidator {
	return &VariableValidator{}
}

func (vv *VariableValidator) Validate(variable *rptmeta.Variable, multiErr *errors.MultiError) {
	variable.Validate(multiErr)
}

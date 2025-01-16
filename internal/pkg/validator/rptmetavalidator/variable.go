package rptmetavalidator

import (
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/rptmeta"
)

type VariableValidator struct{}

func NewVariableValidator() *VariableValidator {
	return &VariableValidator{}
}

func (vv *VariableValidator) Validate(variable *rptmeta.Variable, multiErr *errors.MultiError) {
	variable.Validate(multiErr)
}

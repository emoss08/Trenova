/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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

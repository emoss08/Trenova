/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package validator

type ValidationContext struct {
	IsCreate bool
	IsUpdate bool
}

func NewValidationContext(valCtx *ValidationContext) *ValidationContext {
	return &ValidationContext{
		IsCreate: valCtx.IsCreate,
		IsUpdate: valCtx.IsUpdate,
	}
}

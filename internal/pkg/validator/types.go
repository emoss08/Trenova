package validator

import "context"

type ValidationContext struct {
	IsCreate bool
	IsUpdate bool
}

func NewValidationContext(ctx context.Context, valCtx *ValidationContext) *ValidationContext {
	return &ValidationContext{
		IsCreate: valCtx.IsCreate,
		IsUpdate: valCtx.IsUpdate,
	}
}

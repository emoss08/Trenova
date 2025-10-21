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

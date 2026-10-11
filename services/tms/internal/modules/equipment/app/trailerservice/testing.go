package trailerservice

import (
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*trailer.Trailer]().
			WithModelName("Trailer").
			Build(),
	}
}

package workerservice

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func NewTestValidator() *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*worker.Worker]().
			WithModelName("Worker").
			Build(),
	}
}

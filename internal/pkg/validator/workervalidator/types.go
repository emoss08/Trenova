package workervalidator

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/rotisserie/eris"
)

var ErrWorkerProfileRequired = eris.New(
	"worker profile is required for assignment eligibility check",
)

type ValidationContext struct {
	IsCreate bool
	IsUpdate bool
	Original *worker.Worker
}

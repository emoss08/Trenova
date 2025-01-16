package workervalidator

import (
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain/worker"
)

var ErrWorkerProfileRequired = eris.New("worker profile is required for assignment eligibility check")

type ValidationContext struct {
	IsCreate bool
	IsUpdate bool
	Original *worker.Worker
}

// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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

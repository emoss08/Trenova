package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *service) CreateManual(
	ctx context.Context,
	req *services.CreateManualServiceFailureRequest,
	actor *services.RequestActor,
) (*servicefailure.ServiceFailure, error) {
	return nil, manualServiceFailureDisabledError()
}

func manualServiceFailureDisabledError() error {
	return errortypes.NewValidationError(
		"source",
		errortypes.ErrInvalidOperation,
		"Manual service failure creation is disabled",
	)
}

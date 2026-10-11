package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *service) CreateManual(
	ctx context.Context,
	req *servicefailureports.CreateManualServiceFailureRequest,
	actor *permissionports.RequestActor,
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

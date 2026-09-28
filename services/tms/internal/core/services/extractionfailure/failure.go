package extractionfailure

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func Retryable(err error) bool {
	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, services.ErrModelSchemaValidation),
		errors.Is(err, services.ErrRequiredProviderUnavailable),
		errors.Is(err, services.ErrNoProviderConfigured),
		errortypes.IsBusinessError(err),
		errortypes.IsError(err),
		errortypes.IsMultiError(err),
		errortypes.IsNotFoundError(err):
		return false
	default:
		return true
	}
}

func Message(err error) string {
	switch {
	case errors.Is(err, services.ErrModelSchemaValidation):
		return "The model's answer did not match the extraction schema"
	case errors.Is(err, services.ErrRequiredProviderUnavailable):
		return "The provider being evaluated is no longer enabled for document extraction"
	case errortypes.IsBusinessError(err):
		return err.Error()
	default:
		return "The model could not be reached after several attempts"
	}
}

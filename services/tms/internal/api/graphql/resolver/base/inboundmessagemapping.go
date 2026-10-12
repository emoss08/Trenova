package base

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// optionalMatch reads a matched record for display. A record that is gone or
// out of the reader's reach is no match, not a failed inbox; anything else is
// a real failure and stays one.
func OptionalMatch[T any](value T, err error) (T, error) {
	if err != nil {
		var zero T
		if errortypes.IsNotFoundError(err) {
			return zero, nil
		}

		return zero, err
	}

	return value, nil
}

func RequireLoaders(ctx context.Context) error {
	if !loaders.Available(ctx) {
		return errortypes.NewDatabaseError("Request loaders are not configured")
	}

	return nil
}

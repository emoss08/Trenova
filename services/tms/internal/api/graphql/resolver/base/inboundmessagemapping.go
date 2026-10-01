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

func RequestLoaders(ctx context.Context) (*loaders.Loaders, error) {
	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, errortypes.NewDatabaseError("Request loaders are not configured")
	}

	return l, nil
}

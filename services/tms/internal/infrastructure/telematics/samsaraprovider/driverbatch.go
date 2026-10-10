package samsaraprovider

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

func fetchDriverChunks[T any](
	ctx context.Context,
	driverIDs []string,
	chunkSize int,
	fetch func(context.Context, []string) (T, error),
	merge func(T),
) error {
	requestable := requestableDriverIDs(driverIDs)
	if len(requestable) == 0 {
		return nil
	}

	var failedIDs []string
	var errs []error
	for chunk := range slices.Chunk(requestable, chunkSize) {
		result, err := fetch(ctx, chunk)
		if err != nil {
			if ctx.Err() != nil {
				return err
			}
			failedIDs = append(failedIDs, chunk...)
			errs = append(errs, err)
			continue
		}
		merge(result)
	}

	if len(failedIDs) == 0 {
		return nil
	}
	return &services.ProviderDriverBatchError{
		DriverIDs: failedIDs,
		Err:       errors.Join(errs...),
	}
}

func requestableDriverIDs(driverIDs []string) []string {
	out := make([]string, 0, len(driverIDs))
	for _, driverID := range driverIDs {
		if strings.TrimSpace(driverID) != "" {
			out = append(out, driverID)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func mergeByDriver[V any](dst, src map[string][]V) {
	for driverID, values := range src {
		existing, ok := dst[driverID]
		if !ok {
			dst[driverID] = values
			continue
		}
		dst[driverID] = append(existing, values...)
	}
}

func isDriverBatchError(err error) bool {
	_, ok := errors.AsType[*services.ProviderDriverBatchError](err)
	return ok
}

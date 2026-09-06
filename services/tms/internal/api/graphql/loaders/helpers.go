package loaders

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.opentelemetry.io/otel"
)

const (
	tracerName    = "trenova.graphql.loaders"
	batchWait     = time.Millisecond
	batchCapacity = pagination.MaxLimit
)

type entityWithID interface {
	GetID() pulid.ID
}

type fetchByIDsFunc[T entityWithID] func(context.Context, []pulid.ID) ([]T, error)

type batchFetchFunc[T any] func(context.Context, []string) ([]T, []error)

func loaderOptions() []dataloadgen.Option {
	return []dataloadgen.Option{
		dataloadgen.WithWait(batchWait),
		dataloadgen.WithBatchCapacity(batchCapacity),
		dataloadgen.WithTracer(otel.Tracer(tracerName)),
	}
}

func newLoader[T any](fetch batchFetchFunc[T]) *dataloadgen.Loader[string, T] {
	return dataloadgen.NewLoader(fetch, loaderOptions()...)
}

func batchByIDFunc[T entityWithID](
	fetch fetchByIDsFunc[T],
	notFoundMessage string,
) batchFetchFunc[T] {
	return func(ctx context.Context, keys []string) ([]T, []error) {
		values := make([]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		entities, err := fetch(ctx, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		fillEntityResults(values, errs, ids, indexesByID, entities, notFoundMessage)

		return values, errs
	}
}

func parseBatchKeys(keys []string, errs []error) ([]pulid.ID, map[pulid.ID][]int) {
	ids := make([]pulid.ID, 0, len(keys))
	indexesByID := make(map[pulid.ID][]int, len(keys))

	for idx, key := range keys {
		id, err := parseLoaderID(key)
		if err != nil {
			errs[idx] = err
			continue
		}

		if _, ok := indexesByID[id]; !ok {
			ids = append(ids, id)
		}
		indexesByID[id] = append(indexesByID[id], idx)
	}

	return ids, indexesByID
}

func fillEntityResults[T entityWithID](
	values []T,
	errs []error,
	ids []pulid.ID,
	indexesByID map[pulid.ID][]int,
	entities []T,
	notFoundMessage string,
) {
	entitiesByID := make(map[pulid.ID]T, len(entities))
	for _, entity := range entities {
		entitiesByID[entity.GetID()] = entity
	}

	for _, id := range ids {
		entity, found := entitiesByID[id]
		for _, idx := range indexesByID[id] {
			if !found {
				errs[idx] = errortypes.NewNotFoundError(notFoundMessage)
				continue
			}
			values[idx] = entity
		}
	}
}

func fillMissingErrors(errs []error, err error) {
	for idx := range errs {
		if errs[idx] == nil {
			errs[idx] = err
		}
	}
}

func parseLoaderID(value string) (pulid.ID, error) {
	return pulid.MustParse(value)
}

type fetchCountsByIDFunc func(context.Context, []pulid.ID) (map[pulid.ID]int, error)

type fetchGroupsByIDFunc[T any] func(context.Context, []pulid.ID) (map[pulid.ID][]T, error)

func batchCountFunc(fetch fetchCountsByIDFunc) batchFetchFunc[int] {
	return func(ctx context.Context, keys []string) ([]int, []error) {
		values := make([]int, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		counts, err := fetch(ctx, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = counts[id]
			}
		}

		return values, errs
	}
}

func batchGroupFunc[T any](fetch fetchGroupsByIDFunc[T]) batchFetchFunc[[]T] {
	return func(ctx context.Context, keys []string) ([][]T, []error) {
		values := make([][]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		groups, err := fetch(ctx, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			group := groups[id]
			if group == nil {
				group = []T{}
			}
			for _, idx := range indexesByID[id] {
				values[idx] = group
			}
		}

		return values, errs
	}
}

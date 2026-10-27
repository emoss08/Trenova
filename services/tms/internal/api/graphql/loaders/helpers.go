package loaders

import (
	"context"
	"strconv"
	"strings"
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
	return loaderOptionsWithCapacity(batchCapacity)
}

func loaderOptionsWithCapacity(capacity int) []dataloadgen.Option {
	return loaderOptionsWith(capacity, batchWait)
}

func loaderOptionsWith(capacity int, wait time.Duration) []dataloadgen.Option {
	return []dataloadgen.Option{
		dataloadgen.WithWait(wait),
		dataloadgen.WithBatchCapacity(capacity),
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

type fetchValuesByIDFunc[T any] func(context.Context, []pulid.ID) (map[pulid.ID]T, error)

type fetchGroupsByIDFunc[T any] func(context.Context, []pulid.ID) (map[pulid.ID][]T, error)

func batchCountFunc(fetch fetchCountsByIDFunc) batchFetchFunc[int] {
	return batchValueFunc(fetchValuesByIDFunc[int](fetch))
}

func batchValueFunc[T any](fetch fetchValuesByIDFunc[T]) batchFetchFunc[T] {
	return func(ctx context.Context, keys []string) ([]T, []error) {
		values := make([]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := parseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		found, err := fetch(ctx, ids)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = found[id]
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

type fetchFirstByIDFunc[T any] func(
	ctx context.Context,
	ids []pulid.ID,
	first int,
) (map[pulid.ID][]T, error)

func firstKey(id string, first int) string {
	return strconv.Itoa(first) + ":" + id
}

func parseFirstKey(key string) (pulid.ID, int, error) {
	prefix, id, found := strings.Cut(key, ":")
	if !found {
		return pulid.Nil, 0, errortypes.NewValidationError(
			"first",
			errortypes.ErrInvalid,
			"First is missing from the loader key",
		)
	}
	first, err := strconv.Atoi(prefix)
	if err != nil || first < 1 {
		return pulid.Nil, 0, errortypes.NewValidationError(
			"first",
			errortypes.ErrInvalid,
			"First must be a positive number",
		)
	}
	parsed, err := parseLoaderID(id)
	if err != nil {
		return pulid.Nil, 0, err
	}

	return parsed, first, nil
}

type firstWindow struct {
	ids         []pulid.ID
	indexesByID map[pulid.ID][]int
}

func batchFirstFunc[T any](fetch fetchFirstByIDFunc[T]) batchFetchFunc[[]T] {
	return func(ctx context.Context, keys []string) ([][]T, []error) {
		values := make([][]T, len(keys))
		errs := make([]error, len(keys))

		windows := make(map[int]*firstWindow)
		firsts := make([]int, 0, 1)
		for idx, key := range keys {
			id, first, err := parseFirstKey(key)
			if err != nil {
				errs[idx] = err
				continue
			}
			w, ok := windows[first]
			if !ok {
				w = &firstWindow{indexesByID: make(map[pulid.ID][]int)}
				windows[first] = w
				firsts = append(firsts, first)
			}
			if _, seen := w.indexesByID[id]; !seen {
				w.ids = append(w.ids, id)
			}
			w.indexesByID[id] = append(w.indexesByID[id], idx)
		}

		for _, first := range firsts {
			w := windows[first]
			groups, err := fetch(ctx, w.ids, first)
			for _, id := range w.ids {
				group := groups[id]
				if group == nil {
					group = []T{}
				}
				for _, idx := range w.indexesByID[id] {
					if err != nil {
						errs[idx] = err
						continue
					}
					values[idx] = group
				}
			}
		}

		return values, errs
	}
}

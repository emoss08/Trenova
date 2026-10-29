package loadercore

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
	BatchWait     = time.Millisecond
	BatchCapacity = pagination.MaxLimit
)

type EntityWithID interface {
	GetID() pulid.ID
}

type FetchByIDsFunc[T EntityWithID] func(context.Context, []pulid.ID) ([]T, error)

type BatchFetchFunc[T any] func(context.Context, []string) ([]T, []error)

func loaderOptions() []dataloadgen.Option {
	return loaderOptionsWithCapacity(BatchCapacity)
}

func loaderOptionsWithCapacity(capacity int) []dataloadgen.Option {
	return LoaderOptionsWith(capacity, BatchWait)
}

func LoaderOptionsWith(capacity int, wait time.Duration) []dataloadgen.Option {
	return []dataloadgen.Option{
		dataloadgen.WithWait(wait),
		dataloadgen.WithBatchCapacity(capacity),
		dataloadgen.WithTracer(otel.Tracer(tracerName)),
	}
}

func NewLoader[T any](fetch BatchFetchFunc[T]) *dataloadgen.Loader[string, T] {
	return dataloadgen.NewLoader(fetch, loaderOptions()...)
}

func BatchByIDFunc[T EntityWithID](
	fetch FetchByIDsFunc[T],
	notFoundMessage string,
) BatchFetchFunc[T] {
	return func(ctx context.Context, keys []string) ([]T, []error) {
		values := make([]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		entities, err := fetch(ctx, ids)
		if err != nil {
			FillMissingErrors(errs, err)
			return values, errs
		}

		fillEntityResults(values, errs, ids, indexesByID, entities, notFoundMessage)

		return values, errs
	}
}

func ParseBatchKeys(
	keys []string,
	errs []error,
) (ids []pulid.ID, indexesByID map[pulid.ID][]int) {
	ids = make([]pulid.ID, 0, len(keys))
	indexesByID = make(map[pulid.ID][]int, len(keys))

	for idx, key := range keys {
		id, err := ParseLoaderID(key)
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

func fillEntityResults[T EntityWithID](
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

func FillMissingErrors(errs []error, err error) {
	for idx := range errs {
		if errs[idx] == nil {
			errs[idx] = err
		}
	}
}

func ParseLoaderID(value string) (pulid.ID, error) {
	return pulid.MustParse(value)
}

type FetchCountsByIDFunc func(context.Context, []pulid.ID) (map[pulid.ID]int, error)

type FetchValuesByIDFunc[T any] func(context.Context, []pulid.ID) (map[pulid.ID]T, error)

type FetchGroupsByIDFunc[T any] func(context.Context, []pulid.ID) (map[pulid.ID][]T, error)

func BatchCountFunc(fetch FetchCountsByIDFunc) BatchFetchFunc[int] {
	return BatchValueFunc(FetchValuesByIDFunc[int](fetch))
}

func BatchValueFunc[T any](fetch FetchValuesByIDFunc[T]) BatchFetchFunc[T] {
	return func(ctx context.Context, keys []string) ([]T, []error) {
		values := make([]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		found, err := fetch(ctx, ids)
		if err != nil {
			FillMissingErrors(errs, err)
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

func BatchGroupFunc[T any](fetch FetchGroupsByIDFunc[T]) BatchFetchFunc[[]T] {
	return func(ctx context.Context, keys []string) ([][]T, []error) {
		values := make([][]T, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		groups, err := fetch(ctx, ids)
		if err != nil {
			FillMissingErrors(errs, err)
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

type FetchFirstByIDFunc[T any] func(
	ctx context.Context,
	ids []pulid.ID,
	first int,
) (map[pulid.ID][]T, error)

func FirstKey(id string, first int) string {
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
	parsed, err := ParseLoaderID(id)
	if err != nil {
		return pulid.Nil, 0, err
	}

	return parsed, first, nil
}

type firstWindow struct {
	ids         []pulid.ID
	indexesByID map[pulid.ID][]int
}

func BatchFirstFunc[T any](fetch FetchFirstByIDFunc[T]) BatchFetchFunc[[]T] {
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

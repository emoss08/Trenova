package loaders

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
)

type entityWithID interface {
	GetID() pulid.ID
}

type fetchByIDsFunc[T entityWithID] func(context.Context, []pulid.ID) ([]T, error)

func batchByIDFunc[T entityWithID](
	fetch fetchByIDsFunc[T],
	notFoundMessage string,
) dataloader.BatchFunc[string, T] {
	return func(ctx context.Context, keys []string) []*dataloader.Result[T] {
		results, ids, indexesByID := parseBatchKeys[T](keys)
		if len(ids) == 0 {
			return results
		}

		entities, err := fetch(ctx, ids)
		if err != nil {
			fillMissingResults(results, err)
			return results
		}

		fillEntityResults(results, ids, indexesByID, entities, notFoundMessage)
		return results
	}
}

func parseBatchKeys[T any](
	keys []string,
) ([]*dataloader.Result[T], []pulid.ID, map[pulid.ID][]int) {
	results := make([]*dataloader.Result[T], len(keys))
	ids := make([]pulid.ID, 0, len(keys))
	indexesByID := make(map[pulid.ID][]int, len(keys))

	for idx, key := range keys {
		id, err := parseLoaderID(key)
		if err != nil {
			results[idx] = &dataloader.Result[T]{Error: err}
			continue
		}

		if _, ok := indexesByID[id]; !ok {
			ids = append(ids, id)
		}
		indexesByID[id] = append(indexesByID[id], idx)
	}

	return results, ids, indexesByID
}

func fillEntityResults[T entityWithID](
	results []*dataloader.Result[T],
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
		entity, ok := entitiesByID[id]
		resultErr := error(nil)
		if !ok {
			resultErr = errortypes.NewNotFoundError(notFoundMessage)
		}
		for _, idx := range indexesByID[id] {
			results[idx] = &dataloader.Result[T]{
				Data:  entity,
				Error: resultErr,
			}
		}
	}
}

func fillMissingResults[T any](results []*dataloader.Result[T], err error) {
	for idx := range results {
		if results[idx] == nil {
			results[idx] = &dataloader.Result[T]{Error: err}
		}
	}
}

func parseLoaderID(value string) (pulid.ID, error) {
	return pulid.MustParse(value)
}

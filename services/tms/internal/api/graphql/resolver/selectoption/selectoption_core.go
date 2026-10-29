package selectoption

import (
	"strconv"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MetaKeyCode   = "code"
	MetaKeyStatus = "status"
)

type Request struct {
	TenantInfo  pagination.TenantInfo
	SelectQuery *pagination.SelectQueryRequest
	IDs         []pulid.ID
	Filters     map[string]any
}

type Item struct {
	Option *gqlmodel.SelectOption
	Cursor pagination.PositionCursor
}

// selectOptionCustomerFilter extracts an optional customerId scope from select-option
// filters (used to restrict the shipment picker to an order's customer).
func CustomerFilter(filters map[string]any) pulid.ID {
	return IDFilter(filters, "customerId")
}

func IDFilter(filters map[string]any, key string) pulid.ID {
	value, ok := filters[key]
	if !ok {
		return pulid.Nil
	}
	str, ok := value.(string)
	if !ok || str == "" {
		return pulid.Nil
	}
	id, err := pulid.Parse(str)
	if err != nil {
		return pulid.Nil
	}
	return id
}

func BoolFilter(filters map[string]any, key string) bool {
	value, ok := filters[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	default:
		return false
	}
}

func Int16Filter(filters map[string]any, key string) int16 {
	value, ok := filters[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return int16(v)
	case int:
		return int16(v) //nolint:gosec // callers pass calendar years
	case int64:
		return int16(v) //nolint:gosec // callers pass calendar years
	case string:
		parsed, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return 0
		}
		return int16(parsed)
	default:
		return 0
	}
}

func StringFilter(filters map[string]any, key string) string {
	value, ok := filters[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

func ListConnection[T any](
	result *pagination.ListResult[T],
	page pagination.Info,
	mapper func(T) Item,
) (*gqlmodel.SelectOptionConnection, error) {
	items := make([]Item, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapper(item))
	}

	return selectOptionPage(items, selectOptionTotal{
		count:       result.Total,
		exact:       result.TotalExact,
		hasNextPage: pagination.HasMore(result, page),
	})
}

func Connection(
	items []Item,
	total int,
	offset int,
) (*gqlmodel.SelectOptionConnection, error) {
	return selectOptionPage(items, selectOptionTotal{
		count:       total,
		exact:       true,
		hasNextPage: offset+len(items) < total,
	})
}

type selectOptionTotal struct {
	count       int
	exact       bool
	hasNextPage bool
}

func selectOptionPage(
	items []Item,
	total selectOptionTotal,
) (*gqlmodel.SelectOptionConnection, error) {
	edges := make([]*gqlmodel.SelectOptionEdge, len(items))
	for i, item := range items {
		cursor, err := pagination.EncodePositionCursor(item.Cursor)
		if err != nil {
			return nil, err
		}
		edges[i] = &gqlmodel.SelectOptionEdge{
			Node:   item.Option,
			Cursor: cursor,
		}
	}

	return &gqlmodel.SelectOptionConnection{
		Edges: edges,
		PageInfo: gqlconnection.PageInfo(
			total.hasNextPage,
			gqlconnection.LastEdgeCursor(edges, func(edge *gqlmodel.SelectOptionEdge) string {
				return edge.Cursor
			}),
		),
		TotalCount:      new(total.count),
		TotalCountExact: total.exact,
	}, nil
}

func OrderedItems[T any](
	ids []pulid.ID,
	entities []T,
	id func(T) pulid.ID,
	mapper func(T) Item,
) []Item {
	byID := indexByID(entities, id)

	items := make([]Item, 0, len(entities))
	for _, requestedID := range ids {
		entity, ok := byID[requestedID]
		if ok {
			items = append(items, mapper(entity))
		}
	}

	return items
}

func RequiredItems[T any](
	ids []pulid.ID,
	entities []T,
	id func(T) pulid.ID,
	mapper func(T) Item,
	entityName string,
) ([]Item, error) {
	byID := indexByID(entities, id)

	items := make([]Item, 0, len(ids))
	for _, requestedID := range ids {
		entity, ok := byID[requestedID]
		if !ok {
			return nil, errortypes.NewNotFoundError(
				"{0} not found within your organization",
				entityName,
			)
		}
		items = append(items, mapper(entity))
	}

	return items, nil
}

func indexByID[T any](entities []T, id func(T) pulid.ID) map[pulid.ID]T {
	byID := make(map[pulid.ID]T, len(entities))
	for _, entity := range entities {
		byID[id(entity)] = entity
	}

	return byID
}

func ItemFor(
	option *gqlmodel.SelectOption,
	createdAt int64,
	id pulid.ID,
) Item {
	return Item{
		Option: option,
		Cursor: pagination.PositionCursor{
			At: createdAt,
			ID: id,
		},
	}
}

func OptionalIDMeta(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}

	return id.String()
}

package resolver

import (
	"errors"
	"sync"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

var ErrUnsupportedEntity = errors.New("formula entity is not a shipment")

type walkMemo struct {
	entity  any
	once    sync.Once
	ordered []*shipment.Stop
	rated   []*shipment.Stop
}

func Memoize(entity any) any {
	if _, ok := entity.(*walkMemo); ok {
		return entity
	}
	return &walkMemo{entity: entity}
}

func Unwrap(entity any) any {
	if memo, ok := entity.(*walkMemo); ok {
		return memo.entity
	}
	return entity
}

func shipmentOf(entity any) (*shipment.Shipment, error) {
	switch typed := Unwrap(entity).(type) {
	case *shipment.Shipment:
		if typed == nil {
			return nil, ErrNilPointer
		}
		return typed, nil
	case shipment.Shipment:
		return &typed, nil
	default:
		return nil, ErrUnsupportedEntity
	}
}

func stopsOf(entity any) (ordered, rated []*shipment.Stop, err error) {
	if memo, ok := entity.(*walkMemo); ok {
		entry, shipErr := shipmentOf(memo.entity)
		if shipErr != nil {
			return nil, nil, shipErr
		}
		memo.once.Do(func() {
			memo.ordered = entry.OrderedStops()
			memo.rated = shipment.RatedStopsOf(memo.ordered)
		})
		return memo.ordered, memo.rated, nil
	}

	entry, err := shipmentOf(entity)
	if err != nil {
		return nil, nil, err
	}
	ordered = entry.OrderedStops()
	return ordered, shipment.RatedStopsOf(ordered), nil
}

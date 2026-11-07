package positioningtest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
)

type None struct{}

var _ driverpositioningports.LiveReader = None{}

func (None) ListLive(
	context.Context,
	*driverpositioningports.LiveRequest,
) ([]*driverpositioning.Positioning, error) {
	return nil, nil
}

type Fixed struct {
	Rows []*driverpositioning.Positioning
}

func (f Fixed) ListLive(
	context.Context,
	*driverpositioningports.LiveRequest,
) ([]*driverpositioning.Positioning, error) {
	return f.Rows, nil
}

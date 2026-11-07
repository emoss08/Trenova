package freightoutlookports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxAreas         = 40
	MaxAreaCustomers = 3
)

type Request struct {
	TenantInfo pagination.TenantInfo
}

type AreaCustomer struct {
	CustomerID pulid.ID
	Name       string
	Share      float64
}

type Area struct {
	Balance    demandforecast.AreaBalance
	Point      dispatchmarket.Point
	ValueCents int64
	Customers  []AreaCustomer
}

type Positioning struct {
	*driverpositioning.Positioning
	WorkerName string
}

type Outlook struct {
	Quality      *demandforecast.Quality
	GeneratedAt  int64
	Starts       []int64
	Network      []demandforecast.NetworkSlot
	Areas        []*Area
	AreaCount    int
	Steps        []freightmove.Step
	Pace         []freightmove.Pace
	Days         []*demandforecast.Day
	Positionings []*Positioning
}

type AreaRequest struct {
	TenantInfo pagination.TenantInfo
	Cell       int64
	SlotStart  int64
}

type AreaDetail struct {
	Quality             *demandforecast.Quality
	GeneratedAt         int64
	Area                *Area
	SlotStart           int64
	Reach               []freightmove.ReachRow
	Steps               []freightmove.Step
	Pace                []freightmove.Pace
	Recent              []int32
	RecentLow           int
	RecentHigh          int
	Positionings        []*Positioning
	DeliveryLocationIDs []pulid.ID
}

type Service interface {
	Outlook(ctx context.Context, req *Request) (*Outlook, error)
	Area(ctx context.Context, req *AreaRequest) (*AreaDetail, error)
}

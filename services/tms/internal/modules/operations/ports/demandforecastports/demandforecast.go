package demandforecastports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxOpenLanes     = 2_000
	MaxDepartures    = 2_000
	MaxAreaLocations = 100
)

type LocationCount struct {
	CustomerID pulid.ID
	LocationID pulid.ID
	Week       int64
	Slot       int16
	Loads      int32
}

type LocationSummary struct {
	LocationID         pulid.ID
	Latitude           float64
	Longitude          float64
	City               string
	State              string
	Loads              int32
	MedianRevenueCents int64
}

type HistoryRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Until      int64
}

type HistoryRows struct {
	Counts    []LocationCount
	Leads     []demandforecast.LeadCount
	Locations []LocationSummary
}

type ReplaceRequest struct {
	TenantInfo pagination.TenantInfo
	Fitted     *demandforecast.Fitted
	Quality    *demandforecast.Quality
	Days       []*demandforecast.Day
}

type CellsRequest struct {
	TenantInfo pagination.TenantInfo
	Cells      []int64
}

type Model struct {
	Quality *demandforecast.Quality
	Rates   []*demandforecast.Rate
	Areas   []*demandforecast.Area
	Curves  []*demandforecast.BookingCurve
}

type BookedRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	Until      int64
}

type BookedSlot struct {
	CustomerID pulid.ID
	Latitude   float64
	Longitude  float64
	SlotStart  int64
	Total      int64
	Open       int64
}

type Repository interface {
	History(ctx context.Context, req *HistoryRequest) (*HistoryRows, error)
	Replace(ctx context.Context, req *ReplaceRequest) error
	Quality(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	ModelForCells(ctx context.Context, req *CellsRequest) (*Model, error)
	Model(ctx context.Context, tenantInfo pagination.TenantInfo) (*Model, error)
	Days(ctx context.Context, tenantInfo pagination.TenantInfo) ([]*demandforecast.Day, error)
	ListBooked(ctx context.Context, req *BookedRequest) ([]BookedSlot, error)
	ListDriverStarts(ctx context.Context, req *DriverStartsRequest) ([]DriverStart, error)
	ListOpenLanes(ctx context.Context, req *WindowRequest) ([]OpenLane, error)
	ListDepartures(ctx context.Context, req *WindowRequest) ([]Departure, error)
	ListAreaLocations(ctx context.Context, req *AreaLocationsRequest) ([]pulid.ID, error)
	CustomerNames(ctx context.Context, req *CustomerNamesRequest) (map[pulid.ID]string, error)
	ListTenants(ctx context.Context, req *ListTenantsRequest) ([]pagination.TenantInfo, error)
}

type AreaLocationsRequest struct {
	TenantInfo pagination.TenantInfo
	Cell       int64
}

type ListTenantsRequest struct {
	After pagination.TenantInfo
	Limit int
}

type CustomerNamesRequest struct {
	TenantInfo  pagination.TenantInfo
	CustomerIDs []pulid.ID
}

type DriverStartsRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	WorkerIDs  []pulid.ID
}

type DriverStart struct {
	WorkerID      pulid.ID
	Name          string
	Latitude      float64
	Longitude     float64
	FreeAt        int64
	PositioningID pulid.ID
}

type WindowRequest struct {
	TenantInfo pagination.TenantInfo
	From       int64
	Until      int64
}

type OpenLane struct {
	MoveID               pulid.ID
	ProNumber            string
	OriginLatitude       float64
	OriginLongitude      float64
	PickupStart          int64
	PickupEnd            int64
	DestinationLatitude  float64
	DestinationLongitude float64
	DeliveryEnd          int64
}

type Departure struct {
	MoveID           pulid.ID
	ProNumber        string
	WorkerID         pulid.ID
	OriginLatitude   float64
	OriginLongitude  float64
	PickupStart      int64
	PickupEnd        int64
	DestinationLabel string
}

type ForecasterRequest struct {
	TenantInfo    pagination.TenantInfo
	Now           int64
	AllCells      bool
	Horizon       int64
	Cells         []int64
	Supply        []demandforecast.Driver
	Solving       []demandforecast.Booked
	MarginPercent int64
	CapCents      int64
}

type Service interface {
	Fit(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	Quality(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	Forecaster(ctx context.Context, req *ForecasterRequest) (*demandforecast.Forecaster, error)
	Location(ctx context.Context, tenantInfo pagination.TenantInfo) *time.Location
	Economics(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (margin, capCents int64, err error)
}

type TenantLister interface {
	ListTenants(ctx context.Context, req *ListTenantsRequest) ([]pagination.TenantInfo, error)
}

type FitStarter interface {
	StartFitting(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

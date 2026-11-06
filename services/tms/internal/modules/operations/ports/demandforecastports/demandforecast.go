package demandforecastports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultOutlookHours = 48
	MaxOutlookHours     = 96
	DefaultOutlookLimit = 20
	MaxOutlookLimit     = 50
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
	Latitude  float64
	Longitude float64
	SlotStart int64
	Total     int64
	Open      int64
}

type Repository interface {
	History(ctx context.Context, req *HistoryRequest) (*HistoryRows, error)
	Replace(ctx context.Context, req *ReplaceRequest) error
	Quality(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	ModelForCells(ctx context.Context, req *CellsRequest) (*Model, error)
	Model(ctx context.Context, tenantInfo pagination.TenantInfo) (*Model, error)
	ListBooked(ctx context.Context, req *BookedRequest) ([]BookedSlot, error)
	ListDriverStarts(ctx context.Context, req *DriverStartsRequest) ([]DriverStart, error)
	CustomerNames(ctx context.Context, req *CustomerNamesRequest) (map[pulid.ID]string, error)
	ListTenants(ctx context.Context, req *ListTenantsRequest) ([]pagination.TenantInfo, error)
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
}

type DriverStart struct {
	WorkerID  pulid.ID
	Latitude  float64
	Longitude float64
	FreeAt    int64
}

type ForecasterRequest struct {
	TenantInfo    pagination.TenantInfo
	Now           int64
	Cells         []int64
	Supply        []demandforecast.Driver
	Solving       []demandforecast.Booked
	MarginPercent int64
	CapCents      int64
}

type OutlookRequest struct {
	TenantInfo pagination.TenantInfo
	Hours      int
	Limit      int
}

type OutlookWindow struct {
	Cell             int64
	AreaLabel        string
	WindowStart      int64
	WindowEnd        int64
	TypicalLow       int
	TypicalHigh      int
	Booked           int
	ExpectedUnbooked float64
	DriversExpected  int
	Shortfall        float64
	ValueCents       int64
	Customers        []demandforecast.CustomerShare
	Explanation      string
}

type Outlook struct {
	Quality     *demandforecast.Quality
	GeneratedAt int64
	Windows     []*OutlookWindow
}

type Service interface {
	Fit(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	Quality(ctx context.Context, tenantInfo pagination.TenantInfo) (*demandforecast.Quality, error)
	Forecaster(ctx context.Context, req *ForecasterRequest) (*demandforecast.Forecaster, error)
	Outlook(ctx context.Context, req *OutlookRequest) (*Outlook, error)
}

type TenantLister interface {
	ListTenants(ctx context.Context, req *ListTenantsRequest) ([]pagination.TenantInfo, error)
}

type FitStarter interface {
	StartFitting(ctx context.Context, tenantInfo pagination.TenantInfo) error
}

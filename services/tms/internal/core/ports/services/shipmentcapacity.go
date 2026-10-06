package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CapacityUnitKind string

const (
	CapacityUnitDriver  = CapacityUnitKind("Driver")
	CapacityUnitCarrier = CapacityUnitKind("Carrier")
)

type CapacityGroup string

const (
	CapacityGroupReadyNow       = CapacityGroup("ReadyNow")
	CapacityGroupWithinTwoHours = CapacityGroup("WithinTwoHours")
	CapacityGroupTrucksPosted   = CapacityGroup("TrucksPosted")
	CapacityGroupUsuallyAccept  = CapacityGroup("UsuallyAccept")
)

type CapacityRing struct {
	Value float64
	Max   float64
	Low   bool
}

type CapacityUnit struct {
	ID                pulid.ID
	Kind              CapacityUnitKind
	Name              string
	Initials          string
	Group             CapacityGroup
	FreeAt            *int64
	City              string
	UnitLabel         string
	Ring              *CapacityRing
	BadgeCount        *int
	RatePerMile       decimal.NullDecimal
	AcceptancePercent *float64
	DriveRemainingMs  *int64
	TractorID         pulid.ID
}

type DriverCapacitySummary struct {
	Ready          int
	WithinTwoHours int
	Short          int
	Uncovered      int
}

type CarrierCapacitySummary struct {
	Posting            int
	Untendered         int
	AwaitingAcceptance int
	AvgRatePerMile     decimal.NullDecimal
}

type ShipmentCapacity struct {
	Kind     CapacityUnitKind
	Units    []*CapacityUnit
	Drivers  *DriverCapacitySummary
	Carriers *CarrierCapacitySummary
}

type CapacityMatch struct {
	ShipmentID      pulid.ID
	MoveID          pulid.ID
	ProNumber       string
	OriginCity      string
	DestinationCity string
	PickupAt        int64
	Revenue         decimal.Decimal
	DeadheadMiles   *float64
	Quote           decimal.NullDecimal
	MarginPercent   *float64
	FitPercent      *float64
}

type CapacityMatchesRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       CapacityUnitKind
	UnitID     pulid.ID
	Limit      int
}

type DriverCoverageSuggestion struct {
	WorkerID         pulid.ID
	TractorID        pulid.ID
	MoveID           pulid.ID
	Name             string
	Initials         string
	UnitLabel        string
	DistanceMiles    *float64
	DriveRemainingMs *int64
	FitPercent       *float64
}

type CarrierCoverageSuggestion struct {
	CarrierID         pulid.ID
	MoveID            pulid.ID
	Name              string
	Initials          string
	MCNumber          string
	Quote             decimal.Decimal
	RatePerMile       decimal.Decimal
	AcceptancePercent *float64
	Posted            bool
}

type ShipmentCoverageSuggestions struct {
	Drivers  []*DriverCoverageSuggestion
	Carriers []*CarrierCoverageSuggestion
}

type TenderShipmentItem struct {
	ShipmentID pulid.ID
	CarrierID  pulid.ID
}

type TenderShipmentsRequest struct {
	TenantInfo pagination.TenantInfo
	Items      []TenderShipmentItem
}

type TenderShipmentSuccess struct {
	ShipmentID  pulid.ID
	TenderID    pulid.ID
	CarrierID   pulid.ID
	CarrierName string
}

type TenderShipmentFailure struct {
	ShipmentID pulid.ID
	Message    string
}

type TenderShipmentsResult struct {
	Tendered []*TenderShipmentSuccess
	Failed   []*TenderShipmentFailure
}

type ShipmentCapacityReader interface {
	Capacity(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		kind CapacityUnitKind,
	) (*ShipmentCapacity, error)
	Matches(ctx context.Context, req *CapacityMatchesRequest) ([]*CapacityMatch, error)
}

type ShipmentCoverageSuggester interface {
	CoverageSuggestions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) (*ShipmentCoverageSuggestions, error)
}

type ShipmentTenderer interface {
	TenderShipments(
		ctx context.Context,
		req *TenderShipmentsRequest,
	) (*TenderShipmentsResult, error)
}

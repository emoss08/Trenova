package shipmentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DispatchBoardFilter struct {
	TenantInfo     pagination.TenantInfo
	WindowStart    int64
	WindowEnd      int64
	DayStartUnix   int64
	FleetCodeIDs   []pulid.ID
	CustomerIDs    []pulid.ID
	ServiceTypeIDs []pulid.ID
	MoveStatuses   []shipment.MoveStatus
	WorkerIDs      []pulid.ID
	MoveIDs        []pulid.ID
	Query          string
	IncludeCovered bool
	Limit          int
}

type BoardMove struct {
	MoveID                pulid.ID            `json:"moveId"`
	ShipmentID            pulid.ID            `json:"shipmentId"`
	ProNumber             string              `json:"proNumber"`
	BOL                   string              `json:"bol"`
	MoveStatus            shipment.MoveStatus `json:"moveStatus"`
	ShipmentStatus        shipment.Status     `json:"shipmentStatus"`
	Sequence              int64               `json:"sequence"`
	MoveCount             int                 `json:"moveCount"`
	Loaded                bool                `json:"loaded"`
	Distance              *float64            `json:"distance"`
	CustomerID            pulid.ID            `json:"customerId"`
	CustomerName          string              `json:"customerName"`
	ServiceTypeID         pulid.ID            `json:"serviceTypeId"`
	ServiceTypeCode       string              `json:"serviceTypeCode"`
	RequiredTractorTypeID pulid.ID            `json:"requiredTractorTypeId"`
	RequiredTrailerTypeID pulid.ID            `json:"requiredTrailerTypeId"`
	TemperatureMin        *int16              `json:"temperatureMin"`
	TemperatureMax        *int16              `json:"temperatureMax"`
	HasHazmat             bool                `json:"hasHazmat"`
	HasActiveHold         bool                `json:"hasActiveHold"`
	Revenue               *float64            `json:"revenue"`

	OriginStopID       pulid.ID `json:"originStopId"`
	OriginLocationID   pulid.ID `json:"originLocationId"`
	OriginName         string   `json:"originName"`
	OriginCity         string   `json:"originCity"`
	OriginState        string   `json:"originState"`
	OriginLatitude     *float64 `json:"originLatitude"`
	OriginLongitude    *float64 `json:"originLongitude"`
	OriginWindowStart  int64    `json:"originWindowStart"`
	OriginWindowEnd    *int64   `json:"originWindowEnd"`
	OriginActualArrive *int64   `json:"originActualArrival"`

	DestinationStopID       pulid.ID `json:"destinationStopId"`
	DestinationLocationID   pulid.ID `json:"destinationLocationId"`
	DestinationName         string   `json:"destinationName"`
	DestinationCity         string   `json:"destinationCity"`
	DestinationState        string   `json:"destinationState"`
	DestinationLatitude     *float64 `json:"destinationLatitude"`
	DestinationLongitude    *float64 `json:"destinationLongitude"`
	DestinationWindowStart  int64    `json:"destinationWindowStart"`
	DestinationWindowEnd    *int64   `json:"destinationWindowEnd"`
	DestinationActualArrive *int64   `json:"destinationActualArrival"`

	AssignmentID          pulid.ID               `json:"assignmentId"`
	AssignedWorkerID      pulid.ID               `json:"assignedWorkerId"`
	AssignedWorkerName    string                 `json:"assignedWorkerName"`
	AssignedTractorID     pulid.ID               `json:"assignedTractorId"`
	AssignedTractorCode   string                 `json:"assignedTractorCode"`
	AssignedTrailerID     pulid.ID               `json:"assignedTrailerId"`
	AssignedTrailerCode   string                 `json:"assignedTrailerCode"`
	AssignmentAckStatus   shipment.AssignmentAck `json:"assignmentAckStatus"`
	PreviousMoveTrailerID pulid.ID               `json:"previousMoveTrailerId"`

	CoverageType        string   `json:"coverageType"`
	CarrierAssignmentID pulid.ID `json:"carrierAssignmentId"`
	AssignedCarrierID   pulid.ID `json:"assignedCarrierId"`
	AssignedCarrierName string   `json:"assignedCarrierName"`
	CarrierTotalCost    *float64 `json:"carrierTotalCost"`
}

type BoardDriver struct {
	WorkerID             pulid.ID          `json:"workerId"`
	FirstName            string            `json:"firstName"`
	LastName             string            `json:"lastName"`
	WorkerType           worker.WorkerType `json:"workerType"`
	DriverType           worker.DriverType `json:"driverType"`
	FleetCodeID          pulid.ID          `json:"fleetCodeId"`
	FleetCodeName        string            `json:"fleetCodeName"`
	City                 string            `json:"city"`
	StateAbbr            string            `json:"stateAbbreviation"`
	PostalCode           string            `json:"postalCode"`
	ProfilePicURL        string            `json:"profilePicUrl"`
	AssignmentNote       string            `json:"assignmentBlocked"`
	AvailableForDispatch bool              `json:"availableForDispatch"`

	TractorID       pulid.ID `json:"tractorId"`
	TractorCode     string   `json:"tractorCode"`
	TractorTypeID   pulid.ID `json:"tractorTypeId"`
	TractorFleetID  pulid.ID `json:"tractorFleetId"`
	TractorStatusOK bool     `json:"tractorAvailable"`

	OpenAssignments int `json:"openAssignments"`
}

type WorkerCommitment struct {
	WorkerID       pulid.ID            `json:"workerId"`
	MoveID         pulid.ID            `json:"moveId"`
	ShipmentID     pulid.ID            `json:"shipmentId"`
	ProNumber      string              `json:"proNumber"`
	MoveStatus     shipment.MoveStatus `json:"moveStatus"`
	WindowStart    int64               `json:"windowStart"`
	WindowEnd      int64               `json:"windowEnd"`
	DestinationID  pulid.ID            `json:"destinationLocationId"`
	DestinationCty string              `json:"destinationCity"`
	DestinationSt  string              `json:"destinationState"`
	DestLatitude   *float64            `json:"destinationLatitude"`
	DestLongitude  *float64            `json:"destinationLongitude"`
	TrailerID      pulid.ID            `json:"trailerId"`
}

type WorkerTimeOff struct {
	WorkerID  pulid.ID       `json:"workerId"`
	StartDate int64          `json:"startDate"`
	EndDate   int64          `json:"endDate"`
	Type      worker.PTOType `json:"type"`
}

type WorkerWorkload struct {
	WorkerID     pulid.ID `json:"workerId"`
	MoveCount    int      `json:"moveCount"`
	TotalMiles   float64  `json:"totalMiles"`
	TotalRevenue float64  `json:"totalRevenue"`
	LastEndedAt  int64    `json:"lastEndedAt"`
}

type WorkerLaneExperience struct {
	WorkerID   pulid.ID `json:"workerId"`
	CustomerID pulid.ID `json:"customerId"`
	MoveCount  int      `json:"moveCount"`
}

type WorkerOnTimeStats struct {
	WorkerID            pulid.ID `json:"workerId"`
	CompletedStops      int      `json:"completedStops"`
	OnTimeStops         int      `json:"onTimeStops"`
	DriverFaultFailures int      `json:"driverFaultFailures"`
}

type WorkerAcceptanceStats struct {
	WorkerID             pulid.ID `json:"workerId"`
	Accepted             int      `json:"accepted"`
	Declined             int      `json:"declined"`
	AvgAckLatencySeconds float64  `json:"avgAckLatencySeconds"`
}

type BoardSummary struct {
	UncoveredMoves   int     `json:"uncoveredMoves"`
	CoveredMoves     int     `json:"coveredMoves"`
	LateMoves        int     `json:"lateMoves"`
	AtRiskMoves      int     `json:"atRiskMoves"`
	UnseatedDrivers  int     `json:"unseatedDrivers"`
	AvailableDrivers int     `json:"availableDrivers"`
	AssignedToday    int     `json:"assignedToday"`
	AverageDeadhead  float64 `json:"averageDeadheadMiles"`
	UtilizationPct   float64 `json:"utilizationPercent"`
}

type ListWorkerWindowsRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerIDs   []pulid.ID
	WindowStart int64
	WindowEnd   int64
}

type ListWorkloadRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerIDs  []pulid.ID
	Since      int64
}

type ListLaneExperienceRequest struct {
	TenantInfo  pagination.TenantInfo
	WorkerIDs   []pulid.ID
	CustomerIDs []pulid.ID
	Since       int64
}

type ListEquipmentByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	TractorIDs []pulid.ID
	TrailerIDs []pulid.ID
}

type DispatchConsoleRepository interface {
	ListBoardMoves(ctx context.Context, filter *DispatchBoardFilter) ([]*BoardMove, error)
	ListBoardDrivers(ctx context.Context, filter *DispatchBoardFilter) ([]*BoardDriver, error)
	ListWorkerCommitments(
		ctx context.Context,
		req *ListWorkerWindowsRequest,
	) ([]*WorkerCommitment, error)
	ListWorkerTimeOff(ctx context.Context, req *ListWorkerWindowsRequest) ([]*WorkerTimeOff, error)
	ListWorkerWorkload(ctx context.Context, req *ListWorkloadRequest) ([]*WorkerWorkload, error)
	ListWorkerLaneExperience(
		ctx context.Context,
		req *ListLaneExperienceRequest,
	) ([]*WorkerLaneExperience, error)
	ListWorkerOnTimeStats(
		ctx context.Context,
		req *ListWorkloadRequest,
	) ([]*WorkerOnTimeStats, error)
	ListWorkerAcceptanceStats(
		ctx context.Context,
		req *ListWorkloadRequest,
	) ([]*WorkerAcceptanceStats, error)
	GetBoardSummary(ctx context.Context, filter *DispatchBoardFilter) (*BoardSummary, error)
	ListWorkersByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerIDs []pulid.ID,
	) ([]*worker.Worker, error)
	ListEquipmentByIDs(
		ctx context.Context,
		req *ListEquipmentByIDsRequest,
	) ([]*tractor.Tractor, []*trailer.Trailer, error)
}

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	MoveID                pulid.ID            `bun:"move_id"           json:"moveId"`
	ShipmentID            pulid.ID            `bun:"shipment_id"       json:"shipmentId"`
	ProNumber             string              `bun:"pro_number"        json:"proNumber"`
	BOL                   string              `bun:"bol"               json:"bol"`
	MoveStatus            shipment.MoveStatus `bun:"move_status"       json:"moveStatus"`
	ShipmentStatus        shipment.Status     `bun:"shipment_status"   json:"shipmentStatus"`
	Sequence              int64               `bun:"sequence"          json:"sequence"`
	MoveCount             int                 `bun:"move_count"        json:"moveCount"`
	Loaded                bool                `bun:"loaded"            json:"loaded"`
	Distance              *float64            `bun:"distance"          json:"distance"`
	CustomerID            pulid.ID            `bun:"customer_id"       json:"customerId"`
	CustomerName          string              `bun:"customer_name"     json:"customerName"`
	ServiceTypeID         pulid.ID            `bun:"service_type_id"   json:"serviceTypeId"`
	ServiceTypeCode       string              `bun:"service_type_code" json:"serviceTypeCode"`
	RequiredTractorTypeID pulid.ID            `bun:"tractor_type_id"   json:"requiredTractorTypeId"`
	RequiredTrailerTypeID pulid.ID            `bun:"trailer_type_id"   json:"requiredTrailerTypeId"`
	TemperatureMin        *int16              `bun:"temperature_min"   json:"temperatureMin"`
	TemperatureMax        *int16              `bun:"temperature_max"   json:"temperatureMax"`
	HasHazmat             bool                `bun:"has_hazmat"        json:"hasHazmat"`
	HasActiveHold         bool                `bun:"has_active_hold"   json:"hasActiveHold"`
	Revenue               *float64            `bun:"revenue"           json:"revenue"`

	OriginStopID       pulid.ID `bun:"origin_stop_id"       json:"originStopId"`
	OriginLocationID   pulid.ID `bun:"origin_location_id"   json:"originLocationId"`
	OriginName         string   `bun:"origin_name"          json:"originName"`
	OriginCity         string   `bun:"origin_city"          json:"originCity"`
	OriginState        string   `bun:"origin_state"         json:"originState"`
	OriginLatitude     *float64 `bun:"origin_latitude"      json:"originLatitude"`
	OriginLongitude    *float64 `bun:"origin_longitude"     json:"originLongitude"`
	OriginWindowStart  int64    `bun:"origin_window_start"  json:"originWindowStart"`
	OriginWindowEnd    *int64   `bun:"origin_window_end"    json:"originWindowEnd"`
	OriginActualArrive *int64   `bun:"origin_actual_arrive" json:"originActualArrival"`

	DestinationStopID      pulid.ID `bun:"destination_stop_id"      json:"destinationStopId"`
	DestinationLocationID  pulid.ID `bun:"destination_location_id"  json:"destinationLocationId"`
	DestinationName        string   `bun:"destination_name"         json:"destinationName"`
	DestinationCity        string   `bun:"destination_city"         json:"destinationCity"`
	DestinationState       string   `bun:"destination_state"        json:"destinationState"`
	DestinationLatitude    *float64 `bun:"destination_latitude"     json:"destinationLatitude"`
	DestinationLongitude   *float64 `bun:"destination_longitude"    json:"destinationLongitude"`
	DestinationWindowStart int64    `bun:"destination_window_start" json:"destinationWindowStart"`
	DestinationWindowEnd   *int64   `bun:"destination_window_end"   json:"destinationWindowEnd"`

	AssignmentID          pulid.ID               `bun:"assignment_id"         json:"assignmentId"`
	AssignedWorkerID      pulid.ID               `bun:"assigned_worker_id"    json:"assignedWorkerId"`
	AssignedWorkerName    string                 `bun:"assigned_worker_name"  json:"assignedWorkerName"`
	AssignedTractorID     pulid.ID               `bun:"assigned_tractor_id"   json:"assignedTractorId"`
	AssignedTractorCode   string                 `bun:"assigned_tractor_code" json:"assignedTractorCode"`
	AssignedTrailerID     pulid.ID               `bun:"assigned_trailer_id"   json:"assignedTrailerId"`
	AssignedTrailerCode   string                 `bun:"assigned_trailer_code" json:"assignedTrailerCode"`
	AssignmentAckStatus   shipment.AssignmentAck `bun:"assignment_ack_status" json:"assignmentAckStatus"`
	PreviousMoveTrailerID pulid.ID               `bun:"previous_trailer_id"   json:"previousMoveTrailerId"`

	CoverageType        string   `bun:"coverage_type"         json:"coverageType"`
	CarrierAssignmentID pulid.ID `bun:"carrier_assignment_id" json:"carrierAssignmentId"`
	AssignedCarrierID   pulid.ID `bun:"assigned_carrier_id"   json:"assignedCarrierId"`
	AssignedCarrierName string   `bun:"assigned_carrier_name" json:"assignedCarrierName"`
	CarrierTotalCost    *float64 `bun:"carrier_total_cost"    json:"carrierTotalCost"`
}

type BoardDriver struct {
	WorkerID             pulid.ID          `bun:"worker_id"              json:"workerId"`
	FirstName            string            `bun:"first_name"             json:"firstName"`
	LastName             string            `bun:"last_name"              json:"lastName"`
	WorkerType           worker.WorkerType `bun:"worker_type"            json:"workerType"`
	DriverType           worker.DriverType `bun:"driver_type"            json:"driverType"`
	FleetCodeID          pulid.ID          `bun:"fleet_code_id"          json:"fleetCodeId"`
	FleetCodeName        string            `bun:"fleet_code_name"        json:"fleetCodeName"`
	City                 string            `bun:"city"                   json:"city"`
	StateAbbr            string            `bun:"state_abbr"             json:"stateAbbreviation"`
	PostalCode           string            `bun:"postal_code"            json:"postalCode"`
	ProfilePicURL        string            `bun:"profile_pic_url"        json:"profilePicUrl"`
	AssignmentNote       string            `bun:"assignment_note"        json:"assignmentBlocked"`
	AvailableForDispatch bool              `bun:"available_for_dispatch" json:"availableForDispatch"`

	TractorID       pulid.ID `bun:"tractor_id"        json:"tractorId"`
	TractorCode     string   `bun:"tractor_code"      json:"tractorCode"`
	TractorTypeID   pulid.ID `bun:"tractor_type_id"   json:"tractorTypeId"`
	TractorFleetID  pulid.ID `bun:"tractor_fleet_id"  json:"tractorFleetId"`
	TractorStatusOK bool     `bun:"tractor_status_ok" json:"tractorAvailable"`

	OpenAssignments int `bun:"open_assignments" json:"openAssignments"`
}

type WorkerCommitment struct {
	WorkerID       pulid.ID            `bun:"worker_id"        json:"workerId"`
	MoveID         pulid.ID            `bun:"move_id"          json:"moveId"`
	ShipmentID     pulid.ID            `bun:"shipment_id"      json:"shipmentId"`
	ProNumber      string              `bun:"pro_number"       json:"proNumber"`
	MoveStatus     shipment.MoveStatus `bun:"move_status"      json:"moveStatus"`
	WindowStart    int64               `bun:"window_start"     json:"windowStart"`
	WindowEnd      int64               `bun:"window_end"       json:"windowEnd"`
	DestinationID  pulid.ID            `bun:"destination_id"   json:"destinationLocationId"`
	DestinationCty string              `bun:"destination_city" json:"destinationCity"`
	DestinationSt  string              `bun:"destination_st"   json:"destinationState"`
	DestLatitude   *float64            `bun:"dest_latitude"    json:"destinationLatitude"`
	DestLongitude  *float64            `bun:"dest_longitude"   json:"destinationLongitude"`
	TrailerID      pulid.ID            `bun:"trailer_id"       json:"trailerId"`
}

type WorkerTimeOff struct {
	WorkerID  pulid.ID       `bun:"worker_id"  json:"workerId"`
	StartDate int64          `bun:"start_date" json:"startDate"`
	EndDate   int64          `bun:"end_date"   json:"endDate"`
	Type      worker.PTOType `bun:"pto_type"   json:"type"`
}

type WorkerWorkload struct {
	WorkerID     pulid.ID `bun:"worker_id"     json:"workerId"`
	MoveCount    int      `bun:"move_count"    json:"moveCount"`
	TotalMiles   float64  `bun:"total_miles"   json:"totalMiles"`
	TotalRevenue float64  `bun:"total_revenue" json:"totalRevenue"`
	LastEndedAt  int64    `bun:"last_ended_at" json:"lastEndedAt"`
}

type WorkerLaneExperience struct {
	WorkerID   pulid.ID `bun:"worker_id"   json:"workerId"`
	CustomerID pulid.ID `bun:"customer_id" json:"customerId"`
	MoveCount  int      `bun:"move_count"  json:"moveCount"`
}

type WorkerOnTimeStats struct {
	WorkerID            pulid.ID `bun:"worker_id"             json:"workerId"`
	CompletedStops      int      `bun:"completed_stops"       json:"completedStops"`
	OnTimeStops         int      `bun:"on_time_stops"         json:"onTimeStops"`
	DriverFaultFailures int      `bun:"driver_fault_failures" json:"driverFaultFailures"`
}

type WorkerAcceptanceStats struct {
	WorkerID             pulid.ID `bun:"worker_id"               json:"workerId"`
	Accepted             int      `bun:"accepted"                json:"accepted"`
	Declined             int      `bun:"declined"                json:"declined"`
	AvgAckLatencySeconds float64  `bun:"avg_ack_latency_seconds" json:"avgAckLatencySeconds"`
}

type BoardSummary struct {
	UncoveredMoves   int     `bun:"uncovered_moves"   json:"uncoveredMoves"`
	CoveredMoves     int     `bun:"covered_moves"     json:"coveredMoves"`
	LateMoves        int     `bun:"late_moves"        json:"lateMoves"`
	AtRiskMoves      int     `bun:"at_risk_moves"     json:"atRiskMoves"`
	UnseatedDrivers  int     `bun:"unseated_drivers"  json:"unseatedDrivers"`
	AvailableDrivers int     `bun:"available_drivers" json:"availableDrivers"`
	AssignedToday    int     `bun:"assigned_today"    json:"assignedToday"`
	AverageDeadhead  float64 `bun:"average_deadhead"  json:"averageDeadheadMiles"`
	UtilizationPct   float64 `bun:"-"                 json:"utilizationPercent"`
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

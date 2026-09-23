package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// telematicsReader is the slice of the telematics service the tracking tools
// read: last positions per tractor and hours per driver. Nothing here polls
// a provider; it reads what the poller last wrote.
type telematicsReader interface {
	ListVehiclePositions(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		maxAgeSeconds int64,
	) ([]*telematics.VehiclePosition, error)
	ListWorkerHOSStates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerIDs []pulid.ID,
		limit int,
	) ([]*telematics.WorkerHOSState, error)
	GetWorkerHOSState(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*telematics.WorkerHOSState, error)
}

// boardMoveLister is the one console read tracking needs: who is on a move.
type boardMoveLister interface {
	ListBoardMoves(
		ctx context.Context,
		filter *repositories.DispatchBoardFilter,
	) ([]*repositories.BoardMove, error)
}

const (
	// positionLookbackSeconds is how far back a position may be and still be
	// shown. A day-old ping is still the best evidence of where a parked
	// truck is; the snapshot marks it stale so nobody plans on it.
	positionLookbackSeconds = int64(24 * 3600)
	defaultPositionAgeMin   = 240
	maxPositionAgeMin       = 24 * 60
	maxHOSStates            = 500
)

type getShipmentTrackingTool struct {
	shipments  repositories.ShipmentRepository
	board      boardMoveLister
	telematics telematicsReader
}

func newGetShipmentTrackingTool(
	shipments repositories.ShipmentRepository,
	board boardMoveLister,
	telematics telematicsReader,
) serviceports.AgentQueryTool {
	return &getShipmentTrackingTool{shipments: shipments, board: board, telematics: telematics}
}

func (t *getShipmentTrackingTool) Name() string { return "get_shipment_tracking" }

func (t *getShipmentTrackingTool) Description() string {
	return "Where a shipment is and whether it will be on time: each stop's window, actual " +
		"arrival and lateness, the tractor's last position, and an arrival estimate. It " +
		"also shows who is on each move with what truck and the driver's remaining " +
		"hours. The next-stop estimate is straight-line at a planning speed, so call it " +
		"an estimate and never a promise. Start here for any \"where is\", \"is it " +
		"late\" or \"when will it arrive\" question; get_shipment has the commercial " +
		"detail instead."
}

func (t *getShipmentTrackingTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment's id, from search_shipments or list_shipments, " +
					"the page you are on, or this run's subject. Give this or proNumber.",
			},
			"proNumber": map[string]any{
				"type":        "string",
				"description": "The PRO number, when that is what the person gave you.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *getShipmentTrackingTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *getShipmentTrackingTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	sp, err := t.load(ctx, tenant, params.Params)
	if err != nil {
		return nil, err
	}

	moveIDs := make([]pulid.ID, 0, len(sp.Moves))
	for _, move := range sp.Moves {
		if move != nil {
			moveIDs = append(moveIDs, move.ID)
		}
	}

	assignments, err := t.assignments(ctx, tenant, moveIDs)
	if err != nil {
		return nil, err
	}

	tractorIDs := make([]pulid.ID, 0, len(assignments))
	workerIDs := make([]pulid.ID, 0, len(assignments))
	for _, board := range assignments {
		if !board.AssignedTractorID.IsNil() {
			tractorIDs = append(tractorIDs, board.AssignedTractorID)
		}
		if !board.AssignedWorkerID.IsNil() {
			workerIDs = append(workerIDs, board.AssignedWorkerID)
		}
	}

	positions, err := t.positions(ctx, tenant, tractorIDs)
	if err != nil {
		return nil, err
	}
	hos, err := t.hours(ctx, tenant, workerIDs)
	if err != nil {
		return nil, err
	}

	return shipmenttracking.Build(shipmenttracking.Input{
		Shipment:    sp,
		Assignments: assignments,
		Positions:   positions,
		HOS:         hos,
		Now:         clockFor(params).Now,
		Timezone:    params.Timezone,
	}), nil
}

// load reads the shipment by id, or by PRO number when that is what was
// given; a PRO that matches more than one record is refused with the
// candidates, because guessing between two loads is how the wrong customer
// gets a call.
func (t *getShipmentTrackingTool) load(
	ctx context.Context,
	tenant pagination.TenantInfo,
	params map[string]any,
) (*shipment.Shipment, error) {
	options := repositories.ShipmentOptions{
		ExpandShipmentDetails: true,
		IncludeCustomer:       true,
	}

	if raw := optionalString(params, "shipmentId"); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", err)
		}

		return t.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID:              id,
			TenantInfo:      tenant,
			ShipmentOptions: options,
		})
	}

	pro := optionalString(params, "proNumber")
	if pro == "" {
		return nil, fmt.Errorf("give shipmentId or proNumber")
	}

	result, err := t.shipments.List(ctx, &repositories.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: 10},
			Query:      pro,
		},
		ShipmentOptions: options,
	})
	if err != nil {
		return nil, err
	}

	var exact *shipment.Shipment
	others := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if strings.EqualFold(item.ProNumber, pro) {
			if exact != nil {
				return nil, fmt.Errorf(
					"more than one shipment carries PRO %q; use shipmentId to say which", pro,
				)
			}
			exact = item
			continue
		}
		others = append(others, item.ProNumber)
	}
	if exact == nil {
		if len(others) > 0 {
			return nil, fmt.Errorf(
				"no shipment has PRO %q; close matches: %s", pro, strings.Join(others, ", "),
			)
		}

		return nil, fmt.Errorf("no shipment has PRO %q", pro)
	}

	return exact, nil
}

func (t *getShipmentTrackingTool) assignments(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveIDs []pulid.ID,
) (map[pulid.ID]*repositories.BoardMove, error) {
	out := make(map[pulid.ID]*repositories.BoardMove, len(moveIDs))
	if t.board == nil || len(moveIDs) == 0 {
		return out, nil
	}

	moves, err := t.board.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:     tenant,
		MoveIDs:        moveIDs,
		IncludeCovered: true,
		Limit:          len(moveIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("read the moves' assignments: %w", err)
	}
	for _, move := range moves {
		if move != nil {
			out[move.MoveID] = move
		}
	}

	return out, nil
}

func (t *getShipmentTrackingTool) positions(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tractorIDs []pulid.ID,
) (map[pulid.ID]*telematics.VehiclePosition, error) {
	out := make(map[pulid.ID]*telematics.VehiclePosition, len(tractorIDs))
	if t.telematics == nil || len(tractorIDs) == 0 {
		return out, nil
	}

	wanted := make(map[pulid.ID]bool, len(tractorIDs))
	for _, id := range tractorIDs {
		wanted[id] = true
	}

	positions, err := t.telematics.ListVehiclePositions(ctx, tenant, positionLookbackSeconds)
	if err != nil {
		return nil, fmt.Errorf("read vehicle positions: %w", err)
	}
	for _, position := range positions {
		if position != nil && wanted[position.TractorID] {
			out[position.TractorID] = position
		}
	}

	return out, nil
}

func (t *getShipmentTrackingTool) hours(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerIDs []pulid.ID,
) (map[pulid.ID]*telematics.WorkerHOSState, error) {
	out := make(map[pulid.ID]*telematics.WorkerHOSState, len(workerIDs))
	if t.telematics == nil || len(workerIDs) == 0 {
		return out, nil
	}

	states, err := t.telematics.ListWorkerHOSStates(ctx, tenant, workerIDs, len(workerIDs))
	if err != nil {
		return nil, fmt.Errorf("read hours of service: %w", err)
	}
	for _, state := range states {
		if state != nil {
			out[state.WorkerID] = state
		}
	}

	return out, nil
}

func tenantOf(params serviceports.QueryToolParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

type vehiclePositionRow struct {
	TractorID         string  `json:"tractorId"`
	Tractor           string  `json:"tractor,omitempty"`
	Driver            string  `json:"driver,omitempty"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	FormattedLocation string  `json:"formattedLocation,omitempty"`
	SpeedMph          float64 `json:"speedMph"`
	EngineState       string  `json:"engineState,omitempty"`
	RecordedAt        string  `json:"recordedAt"`
	AgeMinutes        int64   `json:"ageMinutes"`
}

type listVehiclePositionsTool struct {
	telematics telematicsReader
	access     fieldAccess
}

func newListVehiclePositionsTool(
	telematics telematicsReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listVehiclePositionsTool{telematics: telematics, access: newFieldAccess(permissions)}
}

func (t *listVehiclePositionsTool) Name() string { return "list_vehicle_positions" }

func (t *listVehiclePositionsTool) Description() string {
	return "The last known position of every tractor the telematics feed reports: " +
		"coordinates, the nearest place name, speed, engine state and how old the " +
		"reading is. Use it for \"where is truck 104\" or \"which trucks are near " +
		"Dallas\". A reading older than an hour is where the truck was, not where " +
		"it is. Drivers have no position of their own; a driver is where the " +
		"tractor they are logged into is."
}

func (t *listVehiclePositionsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tractorIds": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Optional: only these tractors, by id from list_tractors.",
			},
			"maxAgeMinutes": map[string]any{
				"type": "integer",
				"description": fmt.Sprintf(
					"Ignore readings older than this. Default %d, at most %d.",
					defaultPositionAgeMin, maxPositionAgeMin,
				),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listVehiclePositionsTool) PermissionResource() permission.Resource {
	return permission.ResourceTractor
}

func (t *listVehiclePositionsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	maxAge := optionalInt(params.Params, "maxAgeMinutes", defaultPositionAgeMin)
	if maxAge <= 0 || maxAge > maxPositionAgeMin {
		maxAge = defaultPositionAgeMin
	}

	wanted := map[pulid.ID]bool{}
	if raw, ok := params.Params["tractorIds"]; ok && raw != nil {
		ids, err := pulidList(raw, "tractorIds")
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			wanted[id] = true
		}
	}

	criteria := filtercatalog.NewCriteria("vehicle positions").At(clockFor(params))
	criteria.Field("readings newer than", fmt.Sprintf("%d minutes", maxAge))
	if len(wanted) > 0 {
		criteria.Field("tractors", fmt.Sprintf("%d named", len(wanted)))
	}

	positions, err := t.telematics.ListVehiclePositions(
		ctx, tenantOf(params), int64(maxAge)*60,
	)
	if err != nil {
		return nil, err
	}

	now := clockFor(params).Now
	// The map is a tractor's; who is driving it is a worker's, and a reader
	// who may not open workers is not told who is where from the map instead.
	nameDrivers := t.access.mayRead(ctx, params, permission.ResourceWorker)

	rows := make([]vehiclePositionRow, 0, len(positions))
	for _, position := range positions {
		if position == nil || (len(wanted) > 0 && !wanted[position.TractorID]) {
			continue
		}
		row := toVehiclePositionRow(position, now, params.Timezone)
		if !nameDrivers {
			row.Driver = ""
		}
		rows = append(rows, row)
	}

	return searchResult(criteria, rows, len(rows)), nil
}

func toVehiclePositionRow(
	position *telematics.VehiclePosition,
	now int64,
	timezone string,
) vehiclePositionRow {
	row := vehiclePositionRow{
		TractorID:         position.TractorID.String(),
		Latitude:          position.Latitude,
		Longitude:         position.Longitude,
		FormattedLocation: position.FormattedLocation,
		SpeedMph:          position.SpeedMph,
		EngineState:       string(position.EngineState),
		RecordedAt:        timeutils.FormatUnixDateTimeIn(position.RecordedAt, timezone),
		AgeMinutes:        max(0, (now-position.RecordedAt)/60),
	}
	if position.Tractor != nil {
		row.Tractor = position.Tractor.Code
		if position.Tractor.PrimaryWorker != nil {
			row.Driver = strings.TrimSpace(
				position.Tractor.PrimaryWorker.FirstName + " " + position.Tractor.PrimaryWorker.LastName,
			)
		}
	}

	return row
}

type workerHOSRow struct {
	WorkerID              string `json:"workerId"`
	Name                  string `json:"name,omitempty"`
	DutyStatus            string `json:"dutyStatus"`
	DriveRemainingMinutes int64  `json:"driveRemainingMinutes"`
	ShiftRemainingMinutes int64  `json:"shiftRemainingMinutes"`
	CycleRemainingMinutes int64  `json:"cycleRemainingMinutes"`
	BreakRemainingMinutes int64  `json:"breakRemainingMinutes"`
	CurrentTractorID      string `json:"currentTractorId,omitempty"`
	RecordedAt            string `json:"recordedAt"`
	Stale                 bool   `json:"stale"`
	Note                  string `json:"note,omitempty"`
}

type getWorkerHOSTool struct {
	telematics telematicsReader
}

func newGetWorkerHOSTool(telematics telematicsReader) serviceports.AgentQueryTool {
	return &getWorkerHOSTool{telematics: telematics}
}

func (t *getWorkerHOSTool) Name() string { return "get_worker_hos" }

func (t *getWorkerHOSTool) Description() string {
	return "A driver's hours of service as the ELD last reported them: duty status and " +
		"the drive, shift, cycle and break time remaining, in minutes. Use it before " +
		"promising a delivery time or proposing an assignment. A reading older than " +
		"twelve hours is marked stale and should not be planned on."
}

func (t *getWorkerHOSTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The driver's id, from search_worker or list_workers.",
			},
		},
		"required":             []string{"workerId"},
		"additionalProperties": false,
	}
}

func (t *getWorkerHOSTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *getWorkerHOSTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}

	state, err := t.telematics.GetWorkerHOSState(ctx, tenantOf(params), workerID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf(
			"no hours of service are on file for worker %s; the driver may not be "+
				"mapped to the telematics provider",
			workerID.String(),
		)
	}

	now := clockFor(params).Now
	row := workerHOSRow{
		WorkerID:              state.WorkerID.String(),
		DutyStatus:            string(state.DutyStatus),
		DriveRemainingMinutes: state.DriveRemainingMs / 60_000,
		ShiftRemainingMinutes: state.ShiftRemainingMs / 60_000,
		CycleRemainingMinutes: state.CycleRemainingMs / 60_000,
		BreakRemainingMinutes: state.BreakRemainingMs / 60_000,
		RecordedAt:            timeutils.FormatUnixDateTimeIn(state.RecordedAt, params.Timezone),
		Stale:                 now-state.RecordedAt > 12*3600,
	}
	if !state.CurrentTractorID.IsNil() {
		row.CurrentTractorID = state.CurrentTractorID.String()
	}
	if state.Worker != nil {
		row.Name = strings.TrimSpace(state.Worker.FirstName + " " + state.Worker.LastName)
	}
	if row.Stale {
		row.Note = "This reading is more than twelve hours old; treat the clocks as unknown."
	}

	return row, nil
}

// pulidList reads an array of ids, refusing a malformed one by position.
func pulidList(raw any, key string) ([]pulid.ID, error) {
	values, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an array of ids", key)
	}

	ids := make([]pulid.ID, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("parameter %q[%d] must be a non-empty id", key, index)
		}
		id, err := pulid.Parse(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("parameter %q[%d] is not a valid id: %w", key, index, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

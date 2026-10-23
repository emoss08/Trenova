package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
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
}

// boardMoveLister is the one console read tracking needs: who is on a move.
type boardMoveLister interface {
	ListBoardMoves(
		ctx context.Context,
		filter *shipmentports.DispatchBoardFilter,
	) ([]*shipmentports.BoardMove, error)
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

type stopEtaReader interface {
	ListByShipmentIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) ([]*stopeta.StopETA, error)
}

type getShipmentTrackingTool struct {
	shipments  shipmentports.ShipmentRepository
	board      boardMoveLister
	telematics telematicsReader
	stopEtas   stopEtaReader
}

func newGetShipmentTrackingTool(
	shipments shipmentports.ShipmentRepository,
	board boardMoveLister,
	reader telematicsReader,
	stopEtas stopEtaReader,
) agentports.AgentQueryTool {
	return &getShipmentTrackingTool{
		shipments:  shipments,
		board:      board,
		telematics: reader,
		stopEtas:   stopEtas,
	}
}

func (t *getShipmentTrackingTool) Name() string { return "shipment_get_tracking" }

func (t *getShipmentTrackingTool) Description() string {
	return "Where a shipment is and whether it will be on time: each stop's window, actual " +
		"arrival and lateness, the tractor's last position, and an arrival estimate. It " +
		"also shows who is on each move with what truck and the driver's remaining " +
		"hours. Each open stop carries its projected arrival, slack and verdict, worked out " +
		"from the truck's position, the carrier's check calls or the last departure with " +
		"routed drive time, dwell and the driver's hours; estimate.basis says what it rests " +
		"on and estimate.inputStale says when that is old. Call it an estimate and never a " +
		"promise. Start here for any \"where is\", \"is it " +
		"late\" or \"when will it arrive\" question; shipment_get has the commercial " +
		"detail instead."
}

func (t *getShipmentTrackingTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramShipmentID: shipmentRefProperty(),
			paramProNumber:  proNumberProperty(),
		},
		"additionalProperties": false,
	}
}

func (t *getShipmentTrackingTool) SearchTerms() []string {
	return []string{"track", "where is", "eta", "on time", "running late"}
}

func shipmentRefProperty() map[string]any {
	return agenttoolschema.OfResource(map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The shipment's id, from shipment_list, the page you are " +
			"on, or this run's subject. Give this or proNumber.",
	}, permission.ResourceShipment)
}

func proNumberProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: "The PRO number, when that is what the person gave you.",
	}
}

func (t *getShipmentTrackingTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceShipment,
	})
}

func (t *getShipmentTrackingTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	sp, err := loadShipment(ctx, t.shipments, tenant, params.Params)
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

	etas, err := t.stopEtas.ListByShipmentIDs(ctx, tenant, []pulid.ID{sp.ID})
	if err != nil {
		return nil, err
	}

	return shipmenttracking.Build(shipmenttracking.Input{
		Shipment:    sp,
		Assignments: assignments,
		Positions:   positions,
		HOS:         hos,
		StopETAs:    stopeta.IndexByStop(etas),
		Now:         clockFor(params).Now,
		Timezone:    params.Timezone,
	}), nil
}

// loadShipment reads the shipment by id, or by PRO number when that is what was
// given; a PRO that matches more than one record is refused with the
// candidates, because guessing between two loads is how the wrong customer
// gets a call.
func loadShipment(
	ctx context.Context,
	shipments shipmentports.ShipmentRepository,
	tenant pagination.TenantInfo,
	params map[string]any,
) (*shipment.Shipment, error) {
	options := shipmentports.ShipmentOptions{
		ExpandShipmentDetails: true,
		IncludeCustomer:       true,
	}

	if raw := optionalString(params, "shipmentId"); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", err)
		}

		return shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:              id,
			TenantInfo:      tenant,
			ShipmentOptions: options,
		})
	}

	pro := optionalString(params, "proNumber")
	if pro == "" {
		return nil, fmt.Errorf("give shipmentId or proNumber")
	}

	result, err := shipments.List(ctx, &shipmentports.ListShipmentsRequest{
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
) (map[pulid.ID]*shipmentports.BoardMove, error) {
	out := make(map[pulid.ID]*shipmentports.BoardMove, len(moveIDs))
	if t.board == nil || len(moveIDs) == 0 {
		return out, nil
	}

	moves, err := t.board.ListBoardMoves(ctx, &shipmentports.DispatchBoardFilter{
		TenantInfo:     tenant,
		MoveIDs:        moveIDs,
		MoveStatuses:   shipment.MoveStatuses(),
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

func tenantOf(params *agentports.QueryToolParams) pagination.TenantInfo {
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
	reader telematicsReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listVehiclePositionsTool{telematics: reader, access: newFieldAccess(permissions)}
}

func (t *listVehiclePositionsTool) Name() string { return "equipment_list_positions" }

func (t *listVehiclePositionsTool) SearchTerms() []string {
	return []string{"truck location", "tractor position", "gps", "telematics", "live position"}
}

func (t *listVehiclePositionsTool) Description() string {
	return "Where every tractor is right now, from the telematics feed: its last known " +
		"position, the nearest place name, speed, engine state and how old the reading " +
		"is. Use it for \"where is truck 104\" or \"which trucks are near " +
		"Dallas\". A reading older than an hour is where the truck was, not where " +
		"it is. Drivers have no position of their own; a driver is where the " +
		"tractor they are logged into is."
}

func (t *listVehiclePositionsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tractorIds": agenttoolschema.OfResource(map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyType: toolschema.TypeString,
				},
				toolschema.KeyDescription: "Optional: only these tractors, by id from equipment_list_tractors.",
			}, permission.ResourceTractor),
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

func (t *listVehiclePositionsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceTractor,
	})
}

func (t *listVehiclePositionsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
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

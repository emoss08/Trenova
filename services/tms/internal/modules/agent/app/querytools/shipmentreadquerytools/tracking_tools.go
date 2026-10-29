package shipmentreadquerytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

// telematicsReader is the slice of the telematics service the tracking tools
// read: last positions per tractor and hours per driver. Nothing here polls
// a provider; it reads what the poller last wrote.
type telematicsReader interface {
	ListVehiclePositions(
		ctx context.Context,
		req *telematicsports.ListVehiclePositionsRequest,
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
		filter *dispatchboardports.DispatchBoardFilter,
	) ([]*dispatchboardports.BoardMove, error)
}

const (
	// positionLookbackSeconds is how far back a position may be and still be
	// shown. A day-old ping is still the best evidence of where a parked
	// truck is; the snapshot marks it stale so nobody plans on it.
	positionLookbackSeconds = int64(24 * 3600)
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

func NewGetShipmentTrackingTool(
	shipments shipmentports.ShipmentRepository,
	board boardMoveLister,
	reader telematicsReader,
	stopEtas stopEtaReader,
) agenttoolports.AgentQueryTool {
	return &getShipmentTrackingTool{
		shipments:  shipments,
		board:      board,
		telematics: reader,
		stopEtas:   stopEtas,
	}
}

func (t *getShipmentTrackingTool) Name() string { return "shipment_get_tracking" }

func (t *getShipmentTrackingTool) Description() string {
	return "Where shipments are and whether each will be on time; pass every load you want " +
		"tracked in one call, up to 25. One load comes with each stop's window, actual " +
		"arrival and lateness, the tractor's last position, who is on each move with what " +
		"truck and the driver's remaining hours; several come as a table of each load's " +
		"next stop, projected arrival, verdict, driver, position and flags. A late or " +
		"overdue stop's lateBy says how far past its window it is the way to say it " +
		"(\"22h 00m\"). The estimate is worked out from the truck's position, check calls " +
		"or the last departure with drive time and hours; estimate.basis says what it rests " +
		"on and inputStale says when that is old. Call it an estimate, never a promise. " +
		"Start here for \"where is\", \"is it late\" or \"when will it arrive\"; " +
		"shipment_get has the commercial detail."
}

func (t *getShipmentTrackingTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramShipmentIDs:        shipmentRefsProperty(),
			listtool.ParamProNumber: proNumberProperty(),
		},
		"additionalProperties": false,
	}
}

func (t *getShipmentTrackingTool) SearchTerms() []string {
	return []string{"track", "where is", "eta", "on time", "running late"}
}

func proNumberProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The PRO number, when that is what the person gave you " +
			"for one load.",
	}
}

func (t *getShipmentTrackingTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceShipment,
	})
}

func (t *getShipmentTrackingTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	ids, pro, err := shipmentRefs(params.Params)
	if err != nil {
		return nil, err
	}
	tenant := querytool.TenantOf(params)
	if len(ids) > 1 {
		entities, missing, loadErr := loadShipments(ctx, t.shipments, tenant, ids,
			shipmentports.ShipmentOptions{ExpandShipmentDetails: true, IncludeCustomer: true})
		if loadErr != nil {
			return nil, loadErr
		}
		snapshots, snapErr := t.snapshots(ctx, params, entities)
		if snapErr != nil {
			return nil, snapErr
		}

		return trackingTable(snapshots, missing), nil
	}

	var id pulid.ID
	if len(ids) == 1 {
		id = ids[0]
	}
	sp, err := loadShipment(ctx, t.shipments, tenant, id, pro)
	if err != nil {
		return nil, err
	}
	snapshots, err := t.snapshots(ctx, params, []*shipment.Shipment{sp})
	if err != nil {
		return nil, err
	}

	return snapshots[0], nil
}

func (t *getShipmentTrackingTool) snapshots(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	entities []*shipment.Shipment,
) ([]*shipmenttracking.Snapshot, error) {
	if len(entities) == 0 {
		return []*shipmenttracking.Snapshot{}, nil
	}
	tenant := querytool.TenantOf(params)
	moveIDs := make([]pulid.ID, 0, len(entities)*2)
	shipmentIDs := make([]pulid.ID, 0, len(entities))
	for _, sp := range entities {
		shipmentIDs = append(shipmentIDs, sp.ID)
		for _, move := range sp.Moves {
			if move != nil {
				moveIDs = append(moveIDs, move.ID)
			}
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

	positions, err := t.positions(ctx, tenant, sliceutils.Dedupe(tractorIDs))
	if err != nil {
		return nil, err
	}
	hos, err := t.hours(ctx, tenant, sliceutils.Dedupe(workerIDs))
	if err != nil {
		return nil, err
	}

	etas, err := t.stopEtas.ListByShipmentIDs(ctx, tenant, shipmentIDs)
	if err != nil {
		return nil, err
	}
	byStop := stopeta.IndexByStop(etas)

	now := querytool.ClockFor(params).Now
	snapshots := make([]*shipmenttracking.Snapshot, 0, len(entities))
	for _, sp := range entities {
		snapshots = append(snapshots, shipmenttracking.Build(shipmenttracking.Input{
			Shipment:    sp,
			Assignments: assignments,
			Positions:   positions,
			HOS:         hos,
			StopETAs:    byStop,
			Now:         now,
			Timezone:    params.Timezone,
		}))
	}

	return snapshots, nil
}

func loadShipment(
	ctx context.Context,
	shipments shipmentports.ShipmentRepository,
	tenant pagination.TenantInfo,
	id pulid.ID,
	pro string,
) (*shipment.Shipment, error) {
	options := shipmentports.ShipmentOptions{
		ExpandShipmentDetails: true,
		IncludeCustomer:       true,
	}

	if id.IsNotNil() {
		return shipments.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
			ID:              id,
			TenantInfo:      tenant,
			ShipmentOptions: options,
		})
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
					"more than one shipment carries PRO %q; use shipmentIds to say which", pro,
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
) (map[pulid.ID]*dispatchboardports.BoardMove, error) {
	out := make(map[pulid.ID]*dispatchboardports.BoardMove, len(moveIDs))
	if t.board == nil || len(moveIDs) == 0 {
		return out, nil
	}

	moves, err := t.board.ListBoardMoves(ctx, &dispatchboardports.DispatchBoardFilter{
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

	positions, err := t.telematics.ListVehiclePositions(
		ctx,
		&telematicsports.ListVehiclePositionsRequest{
			TenantInfo:    tenant,
			TractorIDs:    tractorIDs,
			MaxAgeSeconds: positionLookbackSeconds,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read vehicle positions: %w", err)
	}
	for _, position := range positions {
		if position != nil {
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

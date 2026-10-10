package shipmentetaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

const positionLookbackSeconds = int64(24 * 60 * 60)

type BoardMoveLister interface {
	ListBoardMoves(
		ctx context.Context,
		filter *repositories.DispatchBoardFilter,
	) ([]*repositories.BoardMove, error)
}

type TelematicsReader interface {
	ListVehiclePositions(
		ctx context.Context,
		req *repositories.ListVehiclePositionsRequest,
	) ([]*telematics.VehiclePosition, error)
	ListWorkerHOSStates(
		ctx context.Context,
		req *repositories.ListWorkerHOSStatesRequest,
	) ([]*telematics.WorkerHOSState, error)
}

type ServiceFailureLister interface {
	ListUnresolvedByShipmentIDs(
		ctx context.Context,
		req *repositories.ServiceFailuresByShipmentIDsRequest,
	) ([]*servicefailure.ServiceFailure, error)
}

type Params struct {
	fx.In

	Shipments       repositories.ShipmentTrackingRepository
	Console         repositories.DispatchConsoleRepository
	Telematics      repositories.TelematicsRepository
	ServiceFailures repositories.ServiceFailureRepository
}

type Dependencies struct {
	Shipments       repositories.ShipmentTrackingRepository
	Board           BoardMoveLister
	Telematics      TelematicsReader
	ServiceFailures ServiceFailureLister
	Now             func() time.Time
}

type Service struct {
	shipments repositories.ShipmentTrackingRepository
	board     BoardMoveLister
	telem     TelematicsReader
	failures  ServiceFailureLister
	now       func() time.Time
}

var (
	_ services.ShipmentEtaReader      = (*Service)(nil)
	_ services.ShipmentTrackingReader = (*Service)(nil)
)

func New(p Params) services.ShipmentEtaReader {
	return fromParams(p)
}

func NewTrackingReader(p Params) services.ShipmentTrackingReader {
	return fromParams(p)
}

func fromParams(p Params) *Service {
	return NewWithDependencies(&Dependencies{
		Shipments:       p.Shipments,
		Board:           p.Console,
		Telematics:      p.Telematics,
		ServiceFailures: p.ServiceFailures,
		Now:             time.Now,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		shipments: d.Shipments,
		board:     d.Board,
		telem:     d.Telematics,
		failures:  d.ServiceFailures,
		now:       d.Now,
	}
}

type trackingSources struct {
	assignments map[pulid.ID]*repositories.BoardMove
	positions   map[pulid.ID]*telematics.VehiclePosition
	hos         map[pulid.ID]*telematics.WorkerHOSState
	reasons     map[pulid.ID]string
}

func (s *Service) EtasByShipmentIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*services.ShipmentEta, error) {
	out := make(map[pulid.ID]*services.ShipmentEta, len(shipmentIDs))
	err := s.eachSnapshot(ctx, &snapshotRequest{
		tenant:   tenantInfo,
		ids:      shipmentIDs,
		timezone: time.UTC.String(),
	}, func(entity *shipment.Shipment, snapshot *shipmenttracking.Snapshot, reason string) {
		if eta := EtaOf(entity, snapshot, reason); eta != nil {
			out[entity.ID] = eta
		}
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) TrackingSnapshots(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
	timezone string,
) (map[pulid.ID]*shipmenttracking.Snapshot, error) {
	out := make(map[pulid.ID]*shipmenttracking.Snapshot, len(shipmentIDs))
	err := s.eachSnapshot(ctx, &snapshotRequest{
		tenant:   tenantInfo,
		ids:      shipmentIDs,
		timezone: timezone,
	}, func(entity *shipment.Shipment, snapshot *shipmenttracking.Snapshot, _ string) {
		out[entity.ID] = snapshot
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

type snapshotRequest struct {
	tenant   pagination.TenantInfo
	ids      []pulid.ID
	timezone string
}

func (s *Service) eachSnapshot(
	ctx context.Context,
	req *snapshotRequest,
	visit func(*shipment.Shipment, *shipmenttracking.Snapshot, string),
) error {
	if len(req.ids) == 0 {
		return nil
	}

	entities, err := s.shipments.ListTrackingShipments(
		ctx,
		&repositories.ListTrackingShipmentsRequest{
			TenantInfo:  req.tenant,
			ShipmentIDs: req.ids,
		},
	)
	if err != nil {
		return err
	}

	sources, err := s.collect(ctx, req.tenant, entities)
	if err != nil {
		return err
	}

	now := s.now().Unix()
	for _, entity := range entities {
		snapshot := shipmenttracking.Build(shipmenttracking.Input{
			Shipment:    entity,
			Assignments: sources.assignments,
			Positions:   sources.positions,
			HOS:         sources.hos,
			Now:         now,
			Timezone:    req.timezone,
		})
		visit(entity, snapshot, sources.reasons[entity.ID])
	}

	return nil
}

func (s *Service) collect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entities []*shipment.Shipment,
) (*trackingSources, error) {
	sources := &trackingSources{
		assignments: make(map[pulid.ID]*repositories.BoardMove),
		positions:   make(map[pulid.ID]*telematics.VehiclePosition),
		hos:         make(map[pulid.ID]*telematics.WorkerHOSState),
		reasons:     make(map[pulid.ID]string),
	}

	moveIDs := make([]pulid.ID, 0, len(entities))
	shipmentIDs := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		shipmentIDs = append(shipmentIDs, entity.ID)
		for _, move := range entity.Moves {
			if move != nil && move.Status != shipment.MoveStatusCanceled {
				moveIDs = append(moveIDs, move.ID)
			}
		}
	}
	if len(shipmentIDs) == 0 {
		return sources, nil
	}

	if err := s.collectAssignments(ctx, tenantInfo, moveIDs, sources); err != nil {
		return nil, err
	}
	if err := s.collectTelematics(ctx, tenantInfo, sources); err != nil {
		return nil, err
	}

	failures, err := s.failures.ListUnresolvedByShipmentIDs(
		ctx,
		&repositories.ServiceFailuresByShipmentIDsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, failure := range failures {
		if _, seen := sources.reasons[failure.ShipmentID]; seen {
			continue
		}
		if reason := FailureReason(failure); reason != "" {
			sources.reasons[failure.ShipmentID] = reason
		}
	}

	return sources, nil
}

func (s *Service) collectAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
	sources *trackingSources,
) error {
	if len(moveIDs) == 0 {
		return nil
	}

	moves, err := s.board.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo:     tenantInfo,
		MoveIDs:        moveIDs,
		MoveStatuses:   shipment.MoveStatuses(),
		IncludeCovered: true,
		Limit:          len(moveIDs),
	})
	if err != nil {
		return err
	}
	for _, move := range moves {
		if move != nil {
			sources.assignments[move.MoveID] = move
		}
	}

	return nil
}

func (s *Service) collectTelematics(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources *trackingSources,
) error {
	tractorIDs := make([]pulid.ID, 0, len(sources.assignments))
	workerIDs := make([]pulid.ID, 0, len(sources.assignments))
	for _, move := range sources.assignments {
		if move.AssignedTractorID.IsNotNil() {
			tractorIDs = append(tractorIDs, move.AssignedTractorID)
		}
		if move.AssignedWorkerID.IsNotNil() {
			workerIDs = append(workerIDs, move.AssignedWorkerID)
		}
	}

	if len(tractorIDs) > 0 {
		positions, err := s.telem.ListVehiclePositions(
			ctx,
			&repositories.ListVehiclePositionsRequest{
				TenantInfo:    tenantInfo,
				TractorIDs:    tractorIDs,
				MaxAgeSeconds: positionLookbackSeconds,
			},
		)
		if err != nil {
			return err
		}
		for _, position := range positions {
			if position != nil {
				sources.positions[position.TractorID] = position
			}
		}
	}

	if len(workerIDs) > 0 {
		states, err := s.telem.ListWorkerHOSStates(ctx, &repositories.ListWorkerHOSStatesRequest{
			TenantInfo: tenantInfo,
			WorkerIDs:  workerIDs,
			Limit:      len(workerIDs),
		})
		if err != nil {
			return err
		}
		for _, state := range states {
			if state != nil {
				sources.hos[state.WorkerID] = state
			}
		}
	}

	return nil
}

func FailureReason(failure *servicefailure.ServiceFailure) string {
	if failure == nil {
		return ""
	}
	label := ""
	if failure.ReasonCode != nil {
		label = failure.ReasonCode.Label
	}

	return stringutils.FirstNonEmptyTrimmed(label, failure.Notes)
}

func EtaOf(
	entity *shipment.Shipment,
	snapshot *shipmenttracking.Snapshot,
	failureReason string,
) *services.ShipmentEta {
	if snapshot == nil || snapshot.NextStop == nil || entity.Status == shipment.StatusCanceled {
		return nil
	}

	eta := &services.ShipmentEta{
		ShipmentID: entity.ID,
		Verdict:    shipmenttracking.VerdictUnknown,
	}
	if estimate := snapshot.Estimate; estimate != nil {
		eta.Verdict = estimate.Verdict
		if estimate.EstimatedArrival > 0 {
			arrival := estimate.EstimatedArrival
			eta.EstimatedArrival = &arrival
			if estimate.WindowEnd > 0 {
				slack := estimate.SlackMinutes
				eta.SlackMinutes = &slack
			}
		}
	}

	if snapshot.NextStop.Overdue {
		eta.Verdict = shipmenttracking.VerdictLate
		if eta.SlackMinutes == nil {
			slack := -snapshot.NextStop.LateMinutes
			eta.SlackMinutes = &slack
		}
	}
	if entity.Status == shipment.StatusDelayed && eta.Verdict == shipmenttracking.VerdictUnknown {
		eta.Verdict = shipmenttracking.VerdictLate
	}

	if eta.Verdict == shipmenttracking.VerdictLate ||
		eta.Verdict == shipmenttracking.VerdictAtRisk {
		eta.Reason = failureReason
		if eta.Reason == "" && len(snapshot.Flags) > 0 {
			eta.Reason = snapshot.Flags[0]
		}
	}

	return eta
}

package shipmentetaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

type trackingSources struct {
	assignments map[pulid.ID]*shipmentports.BoardMove
	positions   map[pulid.ID]*telematics.VehiclePosition
	hos         map[pulid.ID]*telematics.WorkerHOSState
	exempt      map[pulid.ID]bool
	checkCalls  map[pulid.ID]*checkcall.CheckCall
}

func newTrackingSources() *trackingSources {
	return &trackingSources{
		assignments: make(map[pulid.ID]*shipmentports.BoardMove),
		positions:   make(map[pulid.ID]*telematics.VehiclePosition),
		hos:         make(map[pulid.ID]*telematics.WorkerHOSState),
		exempt:      make(map[pulid.ID]bool),
		checkCalls:  make(map[pulid.ID]*checkcall.CheckCall),
	}
}

func openMoveIDs(entities []*shipment.Shipment) []pulid.ID {
	moveIDs := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		for _, move := range entity.Moves {
			if move != nil && move.Status != shipment.MoveStatusCanceled {
				moveIDs = append(moveIDs, move.ID)
			}
		}
	}
	return moveIDs
}

func (s *Service) collect(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entities []*shipment.Shipment,
) (*trackingSources, error) {
	sources := newTrackingSources()
	moveIDs := openMoveIDs(entities)
	if len(moveIDs) == 0 {
		return sources, nil
	}

	if err := s.collectAssignments(ctx, tenantInfo, moveIDs, sources); err != nil {
		return nil, err
	}
	if err := s.collectTelematics(ctx, tenantInfo, sources); err != nil {
		return nil, err
	}

	return sources, nil
}

func (s *Service) collectForRefresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entities []*shipment.Shipment,
) (*trackingSources, error) {
	sources, err := s.collect(ctx, tenantInfo, entities)
	if err != nil {
		return nil, err
	}

	moveIDs := openMoveIDs(entities)
	if len(moveIDs) == 0 {
		return sources, nil
	}

	calls, err := s.calls.LatestForMoves(ctx, tenantInfo, moveIDs)
	if err != nil {
		return nil, err
	}
	sources.checkCalls = calls

	if err = s.collectExemptions(ctx, tenantInfo, sources); err != nil {
		return nil, err
	}

	return sources, nil
}

func (s *Service) collectAssignments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveIDs []pulid.ID,
	sources *trackingSources,
) error {
	moves, err := s.board.ListBoardMoves(ctx, &shipmentports.DispatchBoardFilter{
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
	workerIDs := assignedWorkerIDs(sources)
	for _, move := range sources.assignments {
		if move.AssignedTractorID.IsNotNil() {
			tractorIDs = append(tractorIDs, move.AssignedTractorID)
		}
	}

	if len(tractorIDs) > 0 {
		positions, err := s.telem.ListVehiclePositions(
			ctx,
			&telematicsports.ListVehiclePositionsRequest{
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
		states, err := s.telem.ListWorkerHOSStates(ctx, &telematicsports.ListWorkerHOSStatesRequest{
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

func (s *Service) collectExemptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sources *trackingSources,
) error {
	workerIDs := assignedWorkerIDs(sources)
	if len(workerIDs) == 0 {
		return nil
	}

	workers, err := s.workers.ListWorkersByIDs(ctx, tenantInfo, workerIDs)
	if err != nil {
		return err
	}
	for _, w := range workers {
		if w != nil && w.Profile != nil && (w.Profile.ELDExempt || w.Profile.ShortHaulExempt) {
			sources.exempt[w.ID] = true
		}
	}

	return nil
}

func assignedWorkerIDs(sources *trackingSources) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(sources.assignments))
	workerIDs := make([]pulid.ID, 0, len(sources.assignments))
	for _, move := range sources.assignments {
		if move.AssignedWorkerID.IsNil() {
			continue
		}
		if _, ok := seen[move.AssignedWorkerID]; ok {
			continue
		}
		seen[move.AssignedWorkerID] = struct{}{}
		workerIDs = append(workerIDs, move.AssignedWorkerID)
	}
	return workerIDs
}

func (s *Service) failureReasons(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]string, error) {
	reasons := make(map[pulid.ID]string, len(shipmentIDs))
	if len(shipmentIDs) == 0 {
		return reasons, nil
	}

	failures, err := s.failures.ListUnresolvedByShipmentIDs(
		ctx,
		&servicefailureports.ServiceFailuresByShipmentIDsRequest{
			TenantInfo:  tenantInfo,
			ShipmentIDs: shipmentIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, failure := range failures {
		if _, seen := reasons[failure.ShipmentID]; seen {
			continue
		}
		if reason := FailureReason(failure); reason != "" {
			reasons[failure.ShipmentID] = reason
		}
	}

	return reasons, nil
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

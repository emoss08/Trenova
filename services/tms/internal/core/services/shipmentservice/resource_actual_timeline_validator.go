package shipmentservice

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type resourceActualField string

const (
	resourceActualFieldArrival   = resourceActualField("actualArrival")
	resourceActualFieldDeparture = resourceActualField("actualDeparture")
)

type resourceTimelineKind string

const (
	resourceTimelineKindTractor       = resourceTimelineKind("tractor")
	resourceTimelineKindPrimaryWorker = resourceTimelineKind("primary worker")
)

type resourceTimelineKey struct {
	kind resourceTimelineKind
	id   pulid.ID
}

type resourceTimelineEvent struct {
	key       resourceTimelineKey
	moveIndex int
	stopIndex int
	field     resourceActualField
	timestamp int64
}

type resourceTimelineWindow struct {
	key          resourceTimelineKey
	moveIndex    int
	stopIndex    int
	arrival      int64
	departure    int64
	hasArrival   bool
	hasDeparture bool
}

type shipmentStopKey struct {
	moveID    pulid.ID
	stopID    pulid.ID
	moveIndex int
	stopIndex int
}

type timelineWindowConflictKey struct {
	field       string
	actualField resourceActualField
	timestamp   int64
	start       int64
	end         int64
	moveIndex   int
	stopIndex   int
}

func validateResourceActualTimeline(
	ctx context.Context,
	assignmentRepo repositories.AssignmentRepository,
	original *shipment.Shipment,
	entity *shipment.Shipment,
	isCreate bool,
) *errortypes.MultiError {
	if entity == nil {
		return nil
	}

	allEvents, changedEvents := collectResourceTimelineEvents(original, entity, isCreate)
	if len(changedEvents) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	payloadTimeline := buildPayloadTimeline(allEvents)
	payloadWindows := buildPayloadWindows(entity)
	windowConflicts := make(map[timelineWindowConflictKey]map[resourceTimelineKind]struct{})
	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	for _, event := range changedEvents {
		if overlap := findPayloadOverlap(payloadWindows[event.key], event); overlap != nil {
			addTimelineConflict(
				multiErr,
				event,
				"previous",
				resourceActualFieldDeparture,
				overlap.departure,
			)
		}

		if previous := findPayloadNeighbor(payloadTimeline[event.key], event, repositories.ActualTimelineDirectionPrevious); previous != nil {
			addTimelineConflict(
				multiErr,
				event,
				"previous",
				previous.field,
				previous.timestamp,
			)
		}

		if next := findPayloadNeighbor(payloadTimeline[event.key], event, repositories.ActualTimelineDirectionNext); next != nil {
			addTimelineConflict(
				multiErr,
				event,
				"next",
				next.field,
				next.timestamp,
			)
		}

		if err := validateExternalTimelineEvent(
			ctx,
			assignmentRepo,
			tenantInfo,
			entity.ID,
			event,
			multiErr,
			windowConflicts,
		); err != nil {
			multiErr.Add(
				stopFieldPath(event.moveIndex, event.stopIndex, string(event.field)),
				errortypes.ErrInvalid,
				err.Error(),
			)
		}
	}

	addCombinedTimelineWindowConflicts(multiErr, windowConflicts)

	if !multiErr.HasErrors() {
		return nil
	}

	return multiErr
}

func collectResourceTimelineEvents(
	original *shipment.Shipment,
	entity *shipment.Shipment,
	isCreate bool,
) ([]*resourceTimelineEvent, []*resourceTimelineEvent) {
	allEvents := make([]*resourceTimelineEvent, 0)
	changedEvents := make([]*resourceTimelineEvent, 0)
	originalStopActuals := buildOriginalStopActuals(original)

	for moveIndex, move := range entity.Moves {
		if move == nil || move.IsCanceled() || move.Assignment == nil {
			continue
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil || stop.IsCanceled() {
				continue
			}

			stopKey := buildStopKey(move, stop, moveIndex, stopIndex)
			currentEvents := collectStopEvents(move, stop, moveIndex, stopIndex)
			allEvents = append(allEvents, currentEvents...)

			if isCreate || stopActualsChanged(originalStopActuals[stopKey], stop) {
				changedEvents = append(changedEvents, currentEvents...)
			}
		}
	}

	return allEvents, changedEvents
}

type stopActuals struct {
	actualArrival   *int64
	actualDeparture *int64
}

func buildOriginalStopActuals(original *shipment.Shipment) map[shipmentStopKey]stopActuals {
	actuals := make(map[shipmentStopKey]stopActuals)
	if original == nil {
		return actuals
	}

	for moveIndex, move := range original.Moves {
		if move == nil {
			continue
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil {
				continue
			}

			actuals[buildStopKey(move, stop, moveIndex, stopIndex)] = stopActuals{
				actualArrival:   stop.ActualArrival,
				actualDeparture: stop.ActualDeparture,
			}
		}
	}

	return actuals
}

func buildStopKey(
	move *shipment.ShipmentMove,
	stop *shipment.Stop,
	moveIndex int,
	stopIndex int,
) shipmentStopKey {
	return shipmentStopKey{
		moveID:    move.ID,
		stopID:    stop.ID,
		moveIndex: moveIndex,
		stopIndex: stopIndex,
	}
}

func stopActualsChanged(original stopActuals, current *shipment.Stop) bool {
	return !timestampsEqual(original.actualArrival, current.ActualArrival) ||
		!timestampsEqual(original.actualDeparture, current.ActualDeparture)
}

func timestampsEqual(left, right *int64) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func collectStopEvents(
	move *shipment.ShipmentMove,
	stop *shipment.Stop,
	moveIndex int,
	stopIndex int,
) []*resourceTimelineEvent {
	events := make([]*resourceTimelineEvent, 0, 4)
	keys := make([]resourceTimelineKey, 0, 2)
	if move.Assignment.TractorID != nil {
		keys = append(keys, resourceTimelineKey{
			kind: resourceTimelineKindTractor,
			id:   *move.Assignment.TractorID,
		})
	}
	if move.Assignment.PrimaryWorkerID != nil {
		keys = append(keys, resourceTimelineKey{
			kind: resourceTimelineKindPrimaryWorker,
			id:   *move.Assignment.PrimaryWorkerID,
		})
	}

	for _, key := range keys {
		if stop.ActualArrival != nil {
			events = append(events, &resourceTimelineEvent{
				key:       key,
				moveIndex: moveIndex,
				stopIndex: stopIndex,
				field:     resourceActualFieldArrival,
				timestamp: *stop.ActualArrival,
			})
		}
		if stop.ActualDeparture != nil {
			events = append(events, &resourceTimelineEvent{
				key:       key,
				moveIndex: moveIndex,
				stopIndex: stopIndex,
				field:     resourceActualFieldDeparture,
				timestamp: *stop.ActualDeparture,
			})
		}
	}

	return events
}

func buildPayloadTimeline(events []*resourceTimelineEvent) map[resourceTimelineKey][]*resourceTimelineEvent {
	byResource := make(map[resourceTimelineKey][]*resourceTimelineEvent)

	for _, event := range events {
		byResource[event.key] = append(byResource[event.key], event)
	}

	for key := range byResource {
		sort.Slice(byResource[key], func(i, j int) bool {
			left := byResource[key][i]
			right := byResource[key][j]
			if left.timestamp != right.timestamp {
				return left.timestamp < right.timestamp
			}
			if left.moveIndex != right.moveIndex {
				return left.moveIndex < right.moveIndex
			}
			if left.stopIndex != right.stopIndex {
				return left.stopIndex < right.stopIndex
			}
			return left.field < right.field
		})
	}

	return byResource
}

func buildPayloadWindows(entity *shipment.Shipment) map[resourceTimelineKey][]*resourceTimelineWindow {
	byResource := make(map[resourceTimelineKey][]*resourceTimelineWindow)

	for moveIndex, move := range entity.Moves {
		if move == nil || move.IsCanceled() || move.Assignment == nil {
			continue
		}

		keys := make([]resourceTimelineKey, 0, 2)
		if move.Assignment.TractorID != nil {
			keys = append(keys, resourceTimelineKey{kind: resourceTimelineKindTractor, id: *move.Assignment.TractorID})
		}
		if move.Assignment.PrimaryWorkerID != nil {
			keys = append(keys, resourceTimelineKey{kind: resourceTimelineKindPrimaryWorker, id: *move.Assignment.PrimaryWorkerID})
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil || stop.IsCanceled() {
				continue
			}

			for _, key := range keys {
				window := &resourceTimelineWindow{
					key:          key,
					moveIndex:    moveIndex,
					stopIndex:    stopIndex,
					hasArrival:   stop.ActualArrival != nil,
					hasDeparture: stop.ActualDeparture != nil,
				}
				if stop.ActualArrival != nil {
					window.arrival = *stop.ActualArrival
				}
				if stop.ActualDeparture != nil {
					window.departure = *stop.ActualDeparture
				}
				byResource[key] = append(byResource[key], window)
			}
		}
	}

	return byResource
}

func findPayloadOverlap(
	windows []*resourceTimelineWindow,
	candidate *resourceTimelineEvent,
) *resourceTimelineWindow {
	for _, window := range windows {
		if window.moveIndex == candidate.moveIndex || !window.hasArrival || !window.hasDeparture {
			continue
		}
		if window.arrival <= candidate.timestamp && candidate.timestamp <= window.departure {
			return window
		}
	}

	return nil
}

func findPayloadNeighbor(
	events []*resourceTimelineEvent,
	candidate *resourceTimelineEvent,
	direction repositories.ActualTimelineDirection,
) *resourceTimelineEvent {
	if len(events) == 0 {
		return nil
	}

	candidateIndex := -1
	for idx, event := range events {
		if event == candidate {
			candidateIndex = idx
			break
		}
	}
	if candidateIndex == -1 {
		return nil
	}

	switch direction {
	case repositories.ActualTimelineDirectionPrevious:
		if candidateIndex == 0 {
			return nil
		}
		previous := events[candidateIndex-1]
		if previous.timestamp >= candidate.timestamp {
			return previous
		}
	case repositories.ActualTimelineDirectionNext:
		if candidateIndex >= len(events)-1 {
			return nil
		}
		next := events[candidateIndex+1]
		if next.timestamp <= candidate.timestamp {
			return next
		}
	}

	return nil
}

func validateExternalTimelineEvent(
	ctx context.Context,
	assignmentRepo repositories.AssignmentRepository,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	event *resourceTimelineEvent,
	multiErr *errortypes.MultiError,
	windowConflicts map[timelineWindowConflictKey]map[resourceTimelineKind]struct{},
) error {
	req := repositories.FindNearestActualTimelineEventRequest{
		TenantInfo:        tenantInfo,
		ExcludeShipmentID: shipmentID,
		Timestamp:         event.timestamp,
		Direction:         repositories.ActualTimelineDirectionPrevious,
	}

	previous, err := findNearestExternalTimelineEvent(ctx, assignmentRepo, event.key, req)
	if err != nil {
		return err
	}
	if previous != nil && previous.Timestamp >= event.timestamp {
		addTimelineConflict(multiErr, event, "previous", resourceActualField(previous.EventType), previous.Timestamp)
	}

	req.Direction = repositories.ActualTimelineDirectionNext
	next, err := findNearestExternalTimelineEvent(ctx, assignmentRepo, event.key, req)
	if err != nil {
		return err
	}
	if next != nil && next.Timestamp <= event.timestamp {
		addTimelineConflict(multiErr, event, "next", resourceActualField(next.EventType), next.Timestamp)
	}

	overlap, err := findOverlappingExternalTimelineWindow(
		ctx,
		assignmentRepo,
		event.key,
		repositories.FindOverlappingActualTimelineWindowRequest{
			TenantInfo:        tenantInfo,
			ExcludeShipmentID: shipmentID,
			Timestamp:         event.timestamp,
		},
	)
	if err != nil {
		return err
	}
	if overlap != nil {
		recordTimelineWindowConflict(windowConflicts, event, overlap)
	}

	return nil
}

func findNearestExternalTimelineEvent(
	ctx context.Context,
	assignmentRepo repositories.AssignmentRepository,
	key resourceTimelineKey,
	req repositories.FindNearestActualTimelineEventRequest,
) (*repositories.ActualTimelineEvent, error) {
	switch key.kind {
	case resourceTimelineKindTractor:
		return assignmentRepo.FindNearestActualEventByTractorID(ctx, req, key.id)
	case resourceTimelineKindPrimaryWorker:
		return assignmentRepo.FindNearestActualEventByPrimaryWorkerID(ctx, req, key.id)
	default:
		return nil, nil
	}
}

func findOverlappingExternalTimelineWindow(
	ctx context.Context,
	assignmentRepo repositories.AssignmentRepository,
	key resourceTimelineKey,
	req repositories.FindOverlappingActualTimelineWindowRequest,
) (*repositories.ActualTimelineWindow, error) {
	switch key.kind {
	case resourceTimelineKindTractor:
		return assignmentRepo.FindOverlappingActualWindowByTractorID(ctx, req, key.id)
	case resourceTimelineKindPrimaryWorker:
		return assignmentRepo.FindOverlappingActualWindowByPrimaryWorkerID(ctx, req, key.id)
	default:
		return nil, nil
	}
}

func addTimelineConflict(
	multiErr *errortypes.MultiError,
	event *resourceTimelineEvent,
	neighborPosition string,
	neighborField resourceActualField,
	neighborTimestamp int64,
) {
	multiErr.Add(
		stopFieldPath(event.moveIndex, event.stopIndex, string(event.field)),
		errortypes.ErrInvalidOperation,
		fmt.Sprintf(
			"%s time cannot be at %s because %s stop actual %s is %s",
			humanizeActualField(event.field),
			timeutils.UnixToHumanReadable(event.timestamp),
			neighborPosition,
			humanizeActualField(neighborField),
			timeutils.UnixToHumanReadable(neighborTimestamp),
		),
	)
}

func recordTimelineWindowConflict(
	windowConflicts map[timelineWindowConflictKey]map[resourceTimelineKind]struct{},
	event *resourceTimelineEvent,
	window *repositories.ActualTimelineWindow,
) {
	key := timelineWindowConflictKey{
		field:       stopFieldPath(event.moveIndex, event.stopIndex, string(event.field)),
		actualField: event.field,
		timestamp:   event.timestamp,
		start:       window.StartTimestamp,
		end:         window.EndTimestamp,
		moveIndex:   event.moveIndex,
		stopIndex:   event.stopIndex,
	}

	if _, ok := windowConflicts[key]; !ok {
		windowConflicts[key] = make(map[resourceTimelineKind]struct{}, 2)
	}
	windowConflicts[key][event.key.kind] = struct{}{}
}

func addCombinedTimelineWindowConflicts(
	multiErr *errortypes.MultiError,
	windowConflicts map[timelineWindowConflictKey]map[resourceTimelineKind]struct{},
) {
	for key, kinds := range windowConflicts {
		verb := "is"
		if len(kinds) > 1 {
			verb = "are"
		}

		multiErr.Add(
			key.field,
			errortypes.ErrInvalidOperation,
			fmt.Sprintf(
				"%s time cannot be at %s because this %s %s already in use from %s to %s",
				humanizeActualField(key.actualField),
				timeutils.UnixToHumanReadable(key.timestamp),
				humanizeResourceKinds(kinds),
				verb,
				timeutils.UnixToHumanReadable(key.start),
				timeutils.UnixToHumanReadable(key.end),
			),
		)
	}
}

func humanizeResourceKinds(kinds map[resourceTimelineKind]struct{}) string {
	ordered := make([]string, 0, len(kinds))
	if _, ok := kinds[resourceTimelineKindTractor]; ok {
		ordered = append(ordered, string(resourceTimelineKindTractor))
	}
	if _, ok := kinds[resourceTimelineKindPrimaryWorker]; ok {
		ordered = append(ordered, string(resourceTimelineKindPrimaryWorker))
	}

	return strings.Join(ordered, " and ")
}

func humanizeActualField(field resourceActualField) string {
	switch field {
	case resourceActualFieldArrival:
		return "Arrival"
	case resourceActualFieldDeparture:
		return "departure"
	default:
		return string(field)
	}
}

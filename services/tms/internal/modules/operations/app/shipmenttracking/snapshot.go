package shipmenttracking

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

// A snapshot answers "where is this shipment and is it going to be on time"
// from what the system already holds: the stops with their windows and
// actuals, the assignment on each move, the assigned tractor's last position
// and the driver's hours. Nothing here is a source of truth; it reads the
// sources of truth into one page, which is what a person on the desk does
// by hand across four screens.

const (
	// positionStaleAfterSeconds is how old a position may be before it is
	// reported as stale: a truck that has not reported in an hour is not
	// where its last ping says.
	positionStaleAfterSeconds = int64(3600)
	hosStaleAfterSeconds      = int64(12 * 3600)

	minutesPerHour = 60
	msPerMinute    = 60_000
)

type Input struct {
	Shipment *shipment.Shipment
	// Assignments carries the board's view of each move, keyed by move id:
	// who is on it and with what, or which carrier has it.
	Assignments map[pulid.ID]*dispatchboardports.BoardMove
	// Positions is the last known position per tractor.
	Positions map[pulid.ID]*telematics.VehiclePosition
	// HOS is the hours-of-service state per worker.
	HOS map[pulid.ID]*telematics.WorkerHOSState
	// StopETAs is the stored projection per stop id, the same rows the board,
	// the shipment page and the API read.
	StopETAs map[pulid.ID]*stopeta.StopETA
	Now      int64
	Timezone string
}

type Snapshot struct {
	ShipmentID string         `json:"shipmentId"`
	ProNumber  string         `json:"proNumber"`
	BOL        string         `json:"bol,omitempty"`
	Status     string         `json:"status"`
	Customer   string         `json:"customer,omitempty"`
	Freight    string         `json:"freight,omitempty"`
	Moves      []MoveSnapshot `json:"moves"`
	// NextStop is the first stop not yet completed, which is where the
	// truck is headed or sitting.
	NextStop *StopSnapshot     `json:"nextStop,omitempty"`
	Position *PositionSnapshot `json:"position,omitempty"`
	Driver   *DriverSnapshot   `json:"driver,omitempty"`
	Estimate *ArrivalEstimate  `json:"estimate,omitempty"`
	// Flags are the things a person would want said first: a stop already
	// late, a position too old to trust, a move nobody is covering.
	Flags    []string          `json:"flags"`
	Summary  string            `json:"summary"`
	Canceled *CanceledSnapshot `json:"canceled,omitempty"`
}

type CanceledSnapshot struct {
	At     int64  `json:"at,omitempty"`
	By     string `json:"by,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type MoveSnapshot struct {
	ID       string  `json:"id"`
	Sequence int64   `json:"sequence"`
	Status   string  `json:"status"`
	Loaded   bool    `json:"loaded"`
	Distance float64 `json:"distanceMiles,omitempty"`
	// Coverage is driver, carrier or unassigned.
	Coverage    string         `json:"coverage"`
	Driver      string         `json:"driver,omitempty"`
	DriverID    string         `json:"driverId,omitempty"`
	Tractor     string         `json:"tractor,omitempty"`
	TractorID   string         `json:"tractorId,omitempty"`
	Trailer     string         `json:"trailer,omitempty"`
	Carrier     string         `json:"carrier,omitempty"`
	Acknowledge string         `json:"assignmentAck,omitempty"`
	Stops       []StopSnapshot `json:"stops"`
}

type StopSnapshot struct {
	ID       string `json:"id"`
	MoveID   string `json:"moveId"`
	Sequence int64  `json:"sequence"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Location string `json:"location,omitempty"`
	City     string `json:"city,omitempty"`
	State    string `json:"state,omitempty"`
	// Windows and actuals are epoch seconds with a rendered text beside
	// each, so the model has both what to compare and what to say.
	ScheduledStart     int64  `json:"scheduledStart"`
	ScheduledStartText string `json:"scheduledStartText"`
	ScheduledEnd       int64  `json:"scheduledEnd,omitempty"`
	ScheduledEndText   string `json:"scheduledEndText,omitempty"`
	Appointment        bool   `json:"appointment"`
	ActualArrival      int64  `json:"actualArrival,omitempty"`
	ActualArrivalText  string `json:"actualArrivalText,omitempty"`
	ActualDeparture    int64  `json:"actualDeparture,omitempty"`
	// Late is set when the arrival came after the window closed; Overdue
	// when the window has closed and nothing has arrived. LateMinutes is
	// how far past in either case, and LateBy says it as a person does.
	Late        bool   `json:"late"`
	Overdue     bool   `json:"overdue"`
	LateMinutes int64  `json:"lateMinutes,omitempty"`
	LateBy      string `json:"lateBy,omitempty"`
	Next        bool   `json:"next"`
	// The projected arrival from the stored ETA, for a stop not yet reached.
	EstimatedArrival     int64  `json:"estimatedArrival,omitempty"`
	EstimatedArrivalText string `json:"estimatedArrivalText,omitempty"`
	SlackMinutes         *int64 `json:"slackMinutes,omitempty"`
	EtaVerdict           string `json:"etaVerdict,omitempty"`

	latitude  *float64
	longitude *float64
}

type PositionSnapshot struct {
	TractorID         string  `json:"tractorId"`
	Tractor           string  `json:"tractor,omitempty"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	FormattedLocation string  `json:"formattedLocation,omitempty"`
	SpeedMph          float64 `json:"speedMph"`
	EngineState       string  `json:"engineState,omitempty"`
	RecordedAt        int64   `json:"recordedAt"`
	RecordedAtText    string  `json:"recordedAtText"`
	AgeMinutes        int64   `json:"ageMinutes"`
	Stale             bool    `json:"stale"`
}

type DriverSnapshot struct {
	WorkerID              string `json:"workerId"`
	Name                  string `json:"name,omitempty"`
	DutyStatus            string `json:"dutyStatus,omitempty"`
	DriveRemainingMinutes int64  `json:"driveRemainingMinutes"`
	ShiftRemainingMinutes int64  `json:"shiftRemainingMinutes"`
	CycleRemainingMinutes int64  `json:"cycleRemainingMinutes"`
	RecordedAt            int64  `json:"recordedAt"`
	Stale                 bool   `json:"stale"`
}

// ArrivalEstimate is the stored projection for the next stop: routed drive
// time where the organization has routing, the driver's hours, dwell at the
// stops before it, and the carrier's own estimate when that is the best there
// is. Basis says in words what it was worked out from.
type ArrivalEstimate struct {
	StopID               string  `json:"stopId"`
	StopLabel            string  `json:"stopLabel"`
	MilesRemaining       float64 `json:"milesRemaining"`
	DriveMinutes         int64   `json:"driveMinutes"`
	RestMinutes          int64   `json:"restMinutes,omitempty"`
	EstimatedArrival     int64   `json:"estimatedArrival"`
	EstimatedArrivalText string  `json:"estimatedArrivalText"`
	WindowEnd            int64   `json:"windowEnd,omitempty"`
	SlackMinutes         int64   `json:"slackMinutes"`
	// Verdict is OnTime, AtRisk, Late or Unknown.
	Verdict        Verdict `json:"verdict"`
	Source         string  `json:"source,omitempty"`
	DriveMethod    string  `json:"driveMethod,omitempty"`
	InputStale     bool    `json:"inputStale"`
	InputAt        int64   `json:"inputAt,omitempty"`
	ComputedAt     int64   `json:"computedAt,omitempty"`
	ComputedAtText string  `json:"computedAtText,omitempty"`
	Basis          string  `json:"basis"`
}

type Verdict = stopeta.Verdict

const (
	VerdictOnTime  = stopeta.VerdictOnTime
	VerdictAtRisk  = stopeta.VerdictAtRisk
	VerdictLate    = stopeta.VerdictLate
	VerdictUnknown = stopeta.VerdictUnknown
)

func Build(in Input) *Snapshot {
	sp := in.Shipment
	snapshot := &Snapshot{
		ShipmentID: sp.ID.String(),
		ProNumber:  sp.ProNumber,
		BOL:        sp.BOL,
		Status:     string(sp.Status),
		Flags:      make([]string, 0, 4),
	}
	if sp.Customer != nil {
		snapshot.Customer = sp.Customer.Name
	}
	snapshot.Canceled = canceledOf(sp)
	snapshot.Freight = freightText(sp.Weight, sp.Pieces)

	moves := append([]*shipment.ShipmentMove(nil), sp.Moves...)
	sort.SliceStable(moves, func(i, j int) bool { return moves[i].Sequence < moves[j].Sequence })

	snapshot.Moves = make([]MoveSnapshot, 0, len(moves))
	var nextStop *StopSnapshot
	var nextMove *MoveSnapshot
	for _, move := range moves {
		if move == nil {
			continue
		}
		ms := buildMove(move, in)
		snapshot.Moves = append(snapshot.Moves, ms)
	}
	for i := range snapshot.Moves {
		ms := &snapshot.Moves[i]
		for j := range ms.Stops {
			stop := &ms.Stops[j]
			if stop.Late {
				snapshot.Flags = append(snapshot.Flags, fmt.Sprintf(
					"Stop %d (%s at %s) arrived %s late",
					stop.Sequence, stop.Type, stopPlace(stop), stop.LateBy,
				))
			}
			if stop.Overdue {
				snapshot.Flags = append(snapshot.Flags, fmt.Sprintf(
					"Stop %d (%s at %s) is %s past its window with no arrival recorded",
					stop.Sequence, stop.Type, stopPlace(stop), stop.LateBy,
				))
			}
			if nextStop == nil && stop.Status != string(shipment.StopStatusCompleted) &&
				stop.Status != string(shipment.StopStatusCanceled) {
				stop.Next = true
				nextStop = stop
				nextMove = ms
			}
		}
		if ms.Coverage == string(shipment.MoveCoverageTypeUnassigned) &&
			ms.Status != string(shipment.MoveStatusCompleted) &&
			ms.Status != string(shipment.MoveStatusCanceled) {
			snapshot.Flags = append(snapshot.Flags, fmt.Sprintf(
				"Move %d has no driver or carrier", ms.Sequence,
			))
		}
	}

	if nextStop != nil {
		copied := *nextStop
		snapshot.NextStop = &copied
	}
	if nextMove != nil {
		snapshot.Position = positionFor(nextMove, in)
		snapshot.Driver = driverFor(nextMove, in)
		if snapshot.Position != nil && snapshot.Position.Stale {
			snapshot.Flags = append(snapshot.Flags, fmt.Sprintf(
				"The last position is %d minutes old", snapshot.Position.AgeMinutes,
			))
		}
		if snapshot.Position == nil &&
			nextMove.Coverage == string(shipment.MoveCoverageTypeDriver) {
			snapshot.Flags = append(snapshot.Flags,
				"No telematics position is on file for the assigned tractor")
		}
		snapshot.Estimate = estimate(nextStop, in)
	}

	snapshot.Summary = summarize(snapshot)

	return snapshot
}

func buildMove(move *shipment.ShipmentMove, in Input) MoveSnapshot {
	ms := MoveSnapshot{
		ID:       move.ID.String(),
		Sequence: move.Sequence,
		Status:   string(move.Status),
		Loaded:   move.Loaded,
		Coverage: string(move.CoverageType),
		Stops:    make([]StopSnapshot, 0, len(move.Stops)),
	}
	if ms.Coverage == "" {
		ms.Coverage = string(shipment.MoveCoverageTypeUnassigned)
	}
	if move.Distance != nil {
		ms.Distance = *move.Distance
	}

	if board := in.Assignments[move.ID]; board != nil {
		ms.Driver = board.AssignedWorkerName
		if !board.AssignedWorkerID.IsNil() {
			ms.DriverID = board.AssignedWorkerID.String()
		}
		ms.Tractor = board.AssignedTractorCode
		if !board.AssignedTractorID.IsNil() {
			ms.TractorID = board.AssignedTractorID.String()
		}
		ms.Trailer = board.AssignedTrailerCode
		ms.Carrier = board.AssignedCarrierName
		ms.Acknowledge = string(board.AssignmentAckStatus)
		if board.CoverageType != "" {
			ms.Coverage = board.CoverageType
		}
	}
	if ms.Coverage == string(shipment.MoveCoverageTypeUnassigned) {
		switch {
		case ms.Carrier != "":
			ms.Coverage = string(shipment.MoveCoverageTypeCarrier)
		case ms.Driver != "":
			ms.Coverage = string(shipment.MoveCoverageTypeDriver)
		}
	}

	stops := append([]*shipment.Stop(nil), move.Stops...)
	sort.SliceStable(stops, func(i, j int) bool { return stops[i].Sequence < stops[j].Sequence })
	for _, stop := range stops {
		if stop == nil {
			continue
		}
		built := buildStop(stop, move.ID, in)
		applyStopEta(&built, in)
		ms.Stops = append(ms.Stops, built)
	}

	return ms
}

func buildStop(stop *shipment.Stop, moveID pulid.ID, in Input) StopSnapshot {
	ss := StopSnapshot{
		ID:                 stop.ID.String(),
		MoveID:             moveID.String(),
		Sequence:           stop.Sequence,
		Type:               string(stop.Type),
		Status:             string(stop.Status),
		ScheduledStart:     stop.ScheduledWindowStart,
		ScheduledStartText: timeutils.FormatUnixDateTimeIn(stop.ScheduledWindowStart, in.Timezone),
		Appointment:        stop.ScheduleType == shipment.StopScheduleTypeAppointment,
	}
	if stop.ScheduledWindowEnd != nil {
		ss.ScheduledEnd = *stop.ScheduledWindowEnd
		ss.ScheduledEndText = timeutils.FormatUnixDateTimeIn(*stop.ScheduledWindowEnd, in.Timezone)
	}
	if stop.ActualArrival != nil {
		ss.ActualArrival = *stop.ActualArrival
		ss.ActualArrivalText = timeutils.FormatUnixDateTimeIn(*stop.ActualArrival, in.Timezone)
	}
	if stop.ActualDeparture != nil {
		ss.ActualDeparture = *stop.ActualDeparture
	}
	if stop.Location != nil {
		ss.Location = stop.Location.Name
		ss.City = stop.Location.City
		if stop.Location.State != nil {
			ss.State = stop.Location.State.Abbreviation
		}
		ss.latitude = stop.Location.Latitude
		ss.longitude = stop.Location.Longitude
	}

	cutoff := stop.EffectiveScheduledCutoff()
	switch {
	case cutoff <= 0:
	case stop.ActualArrival != nil && *stop.ActualArrival > cutoff:
		ss.Late = true
		ss.LateMinutes = (*stop.ActualArrival - cutoff) / minutesPerHour
	case stop.ActualArrival == nil && !stop.IsCompleted() && !stop.IsCanceled() && in.Now > cutoff:
		ss.Overdue = true
		ss.LateMinutes = (in.Now - cutoff) / minutesPerHour
	}
	if ss.LateMinutes > 0 {
		ss.LateBy = timeutils.FormatLongDurationMs(ss.LateMinutes * msPerMinute)
	}

	return ss
}

func positionFor(move *MoveSnapshot, in Input) *PositionSnapshot {
	if move.TractorID == "" {
		return nil
	}
	tractorID, err := pulid.Parse(move.TractorID)
	if err != nil {
		return nil
	}
	position := in.Positions[tractorID]
	if position == nil {
		return nil
	}

	age := (in.Now - position.RecordedAt) / minutesPerHour
	if age < 0 {
		age = 0
	}

	return &PositionSnapshot{
		TractorID:         move.TractorID,
		Tractor:           move.Tractor,
		Latitude:          position.Latitude,
		Longitude:         position.Longitude,
		FormattedLocation: position.FormattedLocation,
		SpeedMph:          position.SpeedMph,
		EngineState:       string(position.EngineState),
		RecordedAt:        position.RecordedAt,
		RecordedAtText:    timeutils.FormatUnixDateTimeIn(position.RecordedAt, in.Timezone),
		AgeMinutes:        age,
		Stale:             PositionStale(position.RecordedAt, in.Now),
	}
}

func driverFor(move *MoveSnapshot, in Input) *DriverSnapshot {
	if move.DriverID == "" {
		return nil
	}
	workerID, err := pulid.Parse(move.DriverID)
	if err != nil {
		return nil
	}
	state := in.HOS[workerID]
	if state == nil {
		return nil
	}

	return &DriverSnapshot{
		WorkerID:              move.DriverID,
		Name:                  move.Driver,
		DutyStatus:            string(state.DutyStatus),
		DriveRemainingMinutes: state.DriveRemainingMs / msPerMinute,
		ShiftRemainingMinutes: state.ShiftRemainingMs / msPerMinute,
		CycleRemainingMinutes: state.CycleRemainingMs / msPerMinute,
		RecordedAt:            state.RecordedAt,
		Stale:                 HOSStale(state.RecordedAt, in.Now),
	}
}

func estimate(stop *StopSnapshot, in Input) *ArrivalEstimate {
	if stop == nil || stop.ActualArrival > 0 {
		return nil
	}

	out := &ArrivalEstimate{
		StopID:    stop.ID,
		StopLabel: fmt.Sprintf("stop %d, %s at %s", stop.Sequence, stop.Type, stopPlace(stop)),
		WindowEnd: stop.ScheduledEnd,
		Verdict:   VerdictUnknown,
	}
	if out.WindowEnd == 0 {
		out.WindowEnd = stop.ScheduledStart
	}

	row := etaFor(stop, in)
	if row == nil {
		out.Basis = "No arrival has been projected for this stop yet. Projections are " +
			"refreshed when the truck reports, a check call comes in or a stop changes."
		return out
	}

	out.MilesRemaining = row.MilesRemaining
	out.DriveMinutes = row.DriveMinutes
	out.RestMinutes = row.RestMinutes
	out.Verdict = row.Verdict
	out.Source = row.Basis.String()
	out.DriveMethod = row.DriveMethod.String()
	out.InputStale = row.InputStale
	if row.InputAt != nil {
		out.InputAt = *row.InputAt
	}
	out.ComputedAt = row.ComputedAt
	out.ComputedAtText = timeutils.FormatUnixDateTimeIn(row.ComputedAt, in.Timezone)
	out.Basis = row.Reason
	if row.WindowEnd != nil {
		out.WindowEnd = *row.WindowEnd
	}
	if row.EstimatedArrival != nil {
		out.EstimatedArrival = *row.EstimatedArrival
		out.EstimatedArrivalText = timeutils.FormatUnixDateTimeIn(
			*row.EstimatedArrival,
			in.Timezone,
		)
	}
	if row.SlackMinutes != nil {
		out.SlackMinutes = *row.SlackMinutes
	}

	return out
}

func etaFor(stop *StopSnapshot, in Input) *stopeta.StopETA {
	if len(in.StopETAs) == 0 {
		return nil
	}
	id, err := pulid.Parse(stop.ID)
	if err != nil {
		return nil
	}

	return in.StopETAs[id]
}

func applyStopEta(stop *StopSnapshot, in Input) {
	if stop.ActualArrival > 0 {
		return
	}
	row := etaFor(stop, in)
	if row == nil {
		return
	}
	stop.EtaVerdict = row.Verdict.String()
	if row.EstimatedArrival != nil {
		stop.EstimatedArrival = *row.EstimatedArrival
		stop.EstimatedArrivalText = timeutils.FormatUnixDateTimeIn(
			*row.EstimatedArrival,
			in.Timezone,
		)
	}
	if row.SlackMinutes != nil {
		slack := *row.SlackMinutes
		stop.SlackMinutes = &slack
	}
}

func stopPlace(stop *StopSnapshot) string {
	parts := make([]string, 0, 2)
	if stop.Location != "" {
		parts = append(parts, stop.Location)
	}
	if stop.City != "" {
		city := stop.City
		if stop.State != "" {
			city += ", " + stop.State
		}
		parts = append(parts, city)
	}
	if len(parts) == 0 {
		return "an unnamed location"
	}

	return strings.Join(parts, " in ")
}

func summarize(s *Snapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PRO %s is %s", s.ProNumber, s.Status)
	if s.Customer != "" {
		fmt.Fprintf(&b, " for %s", s.Customer)
	}
	b.WriteString(".")

	switch {
	case s.NextStop == nil:
		b.WriteString(" Every stop is complete.")
	default:
		fmt.Fprintf(&b, " Next: %s %s, due by %s.",
			strings.ToLower(s.NextStop.Type), stopPlace(s.NextStop), dueText(s.NextStop))
	}
	if s.Position != nil {
		fmt.Fprintf(&b, " Last seen %s at %s", positionText(s.Position), s.Position.RecordedAtText)
		if s.Position.Stale {
			b.WriteString(" (stale)")
		}
		b.WriteString(".")
	}
	if s.Estimate != nil && s.Estimate.Verdict != VerdictUnknown {
		fmt.Fprintf(
			&b,
			" Estimated arrival %s, %s.",
			s.Estimate.EstimatedArrivalText,
			verdictText(s.Estimate),
		)
	}

	return b.String()
}

func dueText(stop *StopSnapshot) string {
	if stop.ScheduledEndText != "" {
		return stop.ScheduledEndText
	}

	return stop.ScheduledStartText
}

func positionText(p *PositionSnapshot) string {
	if p.FormattedLocation != "" {
		return "near " + p.FormattedLocation
	}

	return fmt.Sprintf("at %.4f, %.4f", p.Latitude, p.Longitude)
}

func verdictText(e *ArrivalEstimate) string {
	switch e.Verdict {
	case VerdictLate:
		return fmt.Sprintf("%d minutes after the window closes", -e.SlackMinutes)
	case VerdictAtRisk:
		return fmt.Sprintf("only %d minutes of slack", e.SlackMinutes)
	default:
		return fmt.Sprintf("%d minutes of slack", e.SlackMinutes)
	}
}

func canceledOf(sp *shipment.Shipment) *CanceledSnapshot {
	if !sp.IsCanceled() {
		return nil
	}

	canceled := &CanceledSnapshot{Reason: strings.TrimSpace(sp.CancelReason)}
	if sp.CanceledAt != nil {
		canceled.At = *sp.CanceledAt
	}
	if sp.CanceledBy != nil {
		canceled.By = strings.TrimSpace(sp.CanceledBy.Name)
	}

	return canceled
}

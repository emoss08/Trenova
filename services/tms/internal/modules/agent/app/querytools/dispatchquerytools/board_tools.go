package dispatchquerytools

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

// boardReader is the dispatch console's one read: the board a dispatcher
// looks at, moves on one side and drivers on the other, already scored for
// urgency, coverage, hours and position.
type boardReader interface {
	GetBoard(
		ctx context.Context,
		req *dispatchconsoleservice.GetBoardRequest,
	) (*dispatchconsoleservice.Board, error)
}

const (
	paramBoardLate    = "late"
	paramBoardDrivers = "drivers"

	defaultBoardHours = 24
	maxBoardHours     = 7 * 24
	defaultBoardLimit = 40
	maxBoardLimit     = 100
)

type boardMoveRow struct {
	MoveID          string `json:"moveId"`
	ShipmentID      string `json:"shipmentId"`
	ProNumber       string `json:"proNumber"`
	Customer        string `json:"customer,omitempty"`
	ShipmentStatus  string `json:"shipmentStatus"`
	MoveStatus      string `json:"moveStatus"`
	Urgency         string `json:"urgency"`
	LateTo          string `json:"lateTo,omitempty"`
	LateBy          string `json:"lateBy,omitempty"`
	lateSeconds     int64
	MinutesToPickup int64   `json:"minutesToPickup"`
	Covered         bool    `json:"covered"`
	Coverage        string  `json:"coverage"`
	Driver          string  `json:"driver,omitempty"`
	DriverID        string  `json:"driverId,omitempty"`
	Tractor         string  `json:"tractor,omitempty"`
	Carrier         string  `json:"carrier,omitempty"`
	AssignmentAck   string  `json:"assignmentAck,omitempty"`
	Origin          string  `json:"origin"`
	OriginWindow    string  `json:"originWindow"`
	OriginArrived   string  `json:"originArrived,omitempty"`
	Destination     string  `json:"destination"`
	DestWindow      string  `json:"destinationWindow"`
	DistanceMiles   float64 `json:"distanceMiles,omitempty"`
	HasHazmat       bool    `json:"hasHazmat,omitempty"`
	HasActiveHold   bool    `json:"hasActiveHold,omitempty"`
	LiveTender      string  `json:"liveTender,omitempty"`
	TenderExpiresAt string  `json:"tenderExpiresAt,omitempty"`
}

type boardDriverRow struct {
	WorkerID              string   `json:"workerId"`
	Name                  string   `json:"name"`
	Availability          string   `json:"availability"`
	DutyStatus            string   `json:"dutyStatus,omitempty"`
	DriveRemainingMinutes int64    `json:"driveRemainingMinutes"`
	ShiftRemainingMinutes int64    `json:"shiftRemainingMinutes"`
	HOSStale              bool     `json:"hosStale"`
	Tractor               string   `json:"tractor,omitempty"`
	Location              string   `json:"location,omitempty"`
	PositionAt            string   `json:"positionAt,omitempty"`
	OpenAssignments       int      `json:"openAssignments"`
	ProjectedAvailable    string   `json:"projectedAvailable,omitempty"`
	Findings              []string `json:"findings,omitempty"`
}

type boardView struct {
	Window     string           `json:"window"`
	Summary    *boardSummaryRow `json:"summary"`
	Moves      []boardMoveRow   `json:"moves"`
	Drivers    []boardDriverRow `json:"drivers,omitempty"`
	Note       string           `json:"note"`
	Counts     map[string]int   `json:"urgencyCounts"`
	HasMore    bool             `json:"hasMore"`
	NextCursor string           `json:"nextCursor,omitempty"`

	lateOnly bool
	window   querytool.OffsetWindow
}

func (v *boardView) PageRows() int { return len(v.Moves) }

// CutPage is the board with its first rows moves, resuming at the move after
// them; the drivers and the summary stay whole.
func (v *boardView) CutPage(rows int) (any, bool) {
	if rows <= 0 || rows >= len(v.Moves) {
		return nil, false
	}
	cut := *v
	cut.Moves = v.Moves[:rows]
	cut.HasMore = true
	cut.NextCursor = pagination.EncodeKeyCursor(
		v.window.Scope, strconv.Itoa(v.window.Offset+rows))
	cut.tally()

	return &cut, true
}

// tally counts the moves listed by urgency and by what each is late to do,
// and says so in the note.
func (v *boardView) tally() {
	v.Counts = make(map[string]int, 5)
	late := lateTally{}
	for idx := range v.Moves {
		v.Counts[v.Moves[idx].Urgency]++
		late.add(&v.Moves[idx])
	}
	listed := "moves"
	if v.lateOnly {
		listed = "late moves"
	}
	drivers := ""
	if v.Drivers != nil {
		drivers = fmt.Sprintf(" and %d drivers", len(v.Drivers))
	}
	more := ""
	if v.HasMore {
		more = " More moves follow on the next page (nextCursor as after)."
	}
	v.Note = fmt.Sprintf(
		"%d %s%s listed, the furthest behind first, then the soonest due.%s Urgency "+
			"counts among the moves listed: %s; summary counts every move in the window, so "+
			"quote it for totals. Urgency follows each move's next stop, the pickup until it "+
			"is picked up and then the delivery; lateTo says what each is late to do: %d late "+
			"to pick up, %d late to deliver.%s",
		len(v.Moves), listed, drivers, more, countsText(v.Counts),
		late.pickUp, late.deliver, late.delayedText(),
	)
}

// boardSummaryRow counts every move in the window, not only the moves listed.
// The console's "late" and "at risk" are coverage alarms for uncovered moves;
// called lateMoves beside an urgency count of Late an agent read 6 against 34
// as the board disagreeing with itself, so each is named for what it counts.
type boardSummaryRow struct {
	UncoveredMoves      int     `json:"uncoveredMoves"`
	CoveredMoves        int     `json:"coveredMoves"`
	OverdueMoves        int     `json:"overdueMoves"`
	DueNowMoves         int     `json:"dueNowMoves"`
	DueTodayMoves       int     `json:"dueTodayMoves"`
	UncoveredPastPickup int     `json:"uncoveredPastPickup"`
	UncoveredPickupSoon int     `json:"uncoveredPickupSoon"`
	AvailableDrivers    int     `json:"availableDrivers"`
	UnseatedDrivers     int     `json:"unseatedDrivers"`
	AssignedToday       int     `json:"assignedToday"`
	UtilizationPct      float64 `json:"utilizationPercent"`
}

type getDispatchBoardTool struct {
	board boardReader
}

func NewGetDispatchBoardTool(board boardReader) agenttoolports.AgentQueryTool {
	return &getDispatchBoardTool{board: board}
}

func (t *getDispatchBoardTool) Name() string { return "dispatch_get_board" }

func (t *getDispatchBoardTool) Description() string {
	return "The dispatch board: moves in the window, furthest behind first, with urgency and " +
		"coverage, and drivers with availability, hours left and position. Urgency (Late, " +
		"Now, Today, Tomorrow, Planned) follows the next stop: pickup window until picked " +
		"up, then delivery window. The one call for \"what is late\" (late), \"what is " +
		"uncovered\" (uncoveredOnly) and \"who is free\". Say a late move as its lateTo " +
		"(late to pick up or to deliver) and lateBy (how far past the window). Answer a " +
		"rundown from the board as it stands, not by tracking each move; check " +
		"shipment_get_tracking before acting on a late move, since arrivals are sometimes " +
		"recorded late. summary counts every move in the window, listed or not; on a board, " +
		"show it with the metrics movesLate, movesDueNow, movesDueToday and movesUncovered. " +
		"To act on one load, find it with shipment_list."
}

func (t *getDispatchBoardTool) ParamSchema() map[string]any {
	properties := map[string]any{
		"hoursAhead": map[string]any{
			"type": "integer",
			"description": fmt.Sprintf(
				"How far ahead the window reaches, in hours. Default %d, at most %d.",
				defaultBoardHours, maxBoardHours,
			),
		},
		"uncoveredOnly": map[string]any{
			"type":        "boolean",
			"description": "Only moves with no driver or carrier. Default false.",
		},
		paramBoardLate: map[string]any{
			"type": "boolean",
			"description": "Only the moves late now: past the pickup window's close with no " +
				"arrival, or past the delivery window's close once picked up, the same rule as " +
				"shipment_list's late. A move inside its window is due now, not late. Every " +
				"late move and no others, a page at a time. Default false.",
		},
		paramBoardDrivers: map[string]any{
			"type": "boolean",
			"description": "List the drivers beside the moves. Default true, and false with " +
				"late; leave them out when the question is only about loads.",
		},
		"query": map[string]any{
			"type":        "string",
			"description": "Optional text matched against PRO numbers, customers and places.",
		},
	}
	for name, property := range querytool.CursorPageSchema(defaultBoardLimit, maxBoardLimit) {
		properties[name] = property
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
}

func (t *getDispatchBoardTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceShipmentMove,
	})
}

func (t *getDispatchBoardTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	hours := querytool.OptionalInt(params.Params, "hoursAhead", defaultBoardHours)
	if hours <= 0 || hours > maxBoardHours {
		hours = defaultBoardHours
	}
	window, err := querytool.ReadOffsetWindow(
		t.Name(),
		params.Params,
		defaultBoardLimit,
		maxBoardLimit,
	)
	if err != nil {
		return nil, err
	}
	late := querytool.OptionalBool(params.Params, paramBoardLate)
	drivers := !late
	if raw, given := params.Params[paramBoardDrivers].(bool); given {
		drivers = raw
	}

	now := querytool.ClockFor(params).Now
	board, err := t.board.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo:     querytool.TenantOf(params),
		WindowStart:    now,
		WindowEnd:      now + int64(hours)*3600,
		Query:          querytool.OptionalString(params.Params, "query"),
		IncludeCovered: !querytool.OptionalBool(params.Params, "uncoveredOnly"),
		Limit:          window.Limit + 1,
		LateOnly:       late,
		DueFirst:       true,
		Offset:         window.Offset,
	})
	if err != nil {
		return nil, err
	}

	view := boardViewOf(board, params.Timezone, window.Limit)
	view.lateOnly = late
	view.window = window
	if view.HasMore {
		view.NextCursor = window.Next()
	}
	if !drivers {
		view.Drivers = nil
	}
	view.tally()

	return &view, nil
}

// boardViewOf is a page of the board, at most limit moves in the board's
// order, which the read made due-first: the furthest behind lead.
func boardViewOf(board *dispatchconsoleservice.Board, timezone string, limit int) boardView {
	view := boardView{
		Window: fmt.Sprintf(
			"%s to %s",
			timeutils.FormatUnixDateTimeIn(board.WindowStart, timezone),
			timeutils.FormatUnixDateTimeIn(board.WindowEnd, timezone),
		),
		Moves:   make([]boardMoveRow, 0, len(board.Moves)),
		Drivers: make([]boardDriverRow, 0, len(board.Drivers)),
	}
	if board.Summary != nil {
		view.Summary = &boardSummaryRow{
			UncoveredMoves:      board.Summary.UncoveredMoves,
			CoveredMoves:        board.Summary.CoveredMoves,
			OverdueMoves:        board.Summary.OverdueMoves,
			DueNowMoves:         board.Summary.DueNowMoves,
			DueTodayMoves:       board.Summary.DueTodayMoves,
			UncoveredPastPickup: board.Summary.LateMoves,
			UncoveredPickupSoon: board.Summary.AtRiskMoves,
			AvailableDrivers:    board.Summary.AvailableDrivers,
			UnseatedDrivers:     board.Summary.UnseatedDrivers,
			AssignedToday:       board.Summary.AssignedToday,
			UtilizationPct:      board.Summary.UtilizationPct,
		}
	}

	for _, move := range board.Moves {
		if move == nil || move.BoardMove == nil {
			continue
		}
		if len(view.Moves) == limit {
			view.HasMore = true
			break
		}
		row := toBoardMoveRow(move, timezone)
		row.LateTo, row.lateSeconds = lateness(move.BoardMove, board.WindowStart)
		row.LateBy = lateFor(row.lateSeconds)
		view.Moves = append(view.Moves, row)
	}
	for _, driver := range board.Drivers {
		if driver == nil || driver.BoardDriver == nil {
			continue
		}
		view.Drivers = append(view.Drivers, toBoardDriverRow(driver, timezone))
	}

	return view
}

const (
	lateToPickUp    = "pick up"
	lateToDeliver   = "deliver"
	millisPerSecond = 1000
)

// lateness is what a move is late to do at now, and how many seconds past the
// window's close it is: pick up, when its pickup window has closed with no
// arrival recorded, or deliver, when it was picked up and its delivery
// window has closed. A move inside its pickup window is due, not late, as
// the board's urgency and the shipment list's late both read it.
func lateness(move *dispatchboardports.BoardMove, now int64) (lateTo string, lateSeconds int64) {
	if move.OriginActualArrive == nil || *move.OriginActualArrive <= 0 {
		pickupClose := closeOf(move.OriginWindowStart, move.OriginWindowEnd)
		if move.OriginWindowStart > 0 && pickupClose < now {
			return lateToPickUp, now - pickupClose
		}

		return "", 0
	}

	deadline := closeOf(move.DestinationWindowStart, move.DestinationWindowEnd)
	if deadline > 0 && deadline < now {
		return lateToDeliver, now - deadline
	}

	return "", 0
}

func closeOf(start int64, end *int64) int64 {
	if end != nil && *end > 0 {
		return *end
	}

	return start
}

func lateFor(seconds int64) string {
	if seconds <= 0 {
		return ""
	}

	return timeutils.FormatLongDurationMs(seconds * millisPerSecond)
}

// lateTally counts the board's late moves by what they are late to do, and
// keeps the shipments marked Delayed for the note, which list shortening
// never cuts.
type lateTally struct {
	pickUp  int
	deliver int
	delayed []string
}

func (l *lateTally) add(row *boardMoveRow) {
	switch row.LateTo {
	case lateToPickUp:
		l.pickUp++
	case lateToDeliver:
		l.deliver++
	}
	if row.ShipmentStatus == string(shipmentvocab.StatusDelayed) &&
		!slices.Contains(l.delayed, row.ProNumber) {
		l.delayed = append(l.delayed, row.ProNumber)
	}
}

func (l *lateTally) delayedText() string {
	if len(l.delayed) == 0 {
		return ""
	}

	return " Shipments marked Delayed: " + strings.Join(l.delayed, ", ") + "."
}

func toBoardMoveRow(move *dispatchconsoleservice.BoardMove, timezone string) boardMoveRow {
	m := move.BoardMove
	row := boardMoveRow{
		MoveID:          m.MoveID.String(),
		ShipmentID:      m.ShipmentID.String(),
		ProNumber:       m.ProNumber,
		Customer:        m.CustomerName,
		ShipmentStatus:  string(m.ShipmentStatus),
		MoveStatus:      string(m.MoveStatus),
		Urgency:         string(move.Urgency),
		MinutesToPickup: move.MinutesToPickup,
		Covered:         move.IsCovered,
		Coverage:        m.CoverageType,
		Driver:          m.AssignedWorkerName,
		Tractor:         m.AssignedTractorCode,
		Carrier:         m.AssignedCarrierName,
		AssignmentAck:   string(m.AssignmentAckStatus),
		Origin:          placeText(m.OriginName, m.OriginCity, m.OriginState),
		OriginWindow:    windowText(m.OriginWindowStart, m.OriginWindowEnd, timezone),
		Destination:     placeText(m.DestinationName, m.DestinationCity, m.DestinationState),
		DestWindow:      windowText(m.DestinationWindowStart, m.DestinationWindowEnd, timezone),
		HasHazmat:       m.HasHazmat,
		HasActiveHold:   m.HasActiveHold,
	}
	if row.Coverage == "" {
		row.Coverage = "unassigned"
	}
	if !m.AssignedWorkerID.IsNil() {
		row.DriverID = m.AssignedWorkerID.String()
	}
	if m.Distance != nil {
		row.DistanceMiles = *m.Distance
	}
	if m.OriginActualArrive != nil && *m.OriginActualArrive > 0 {
		row.OriginArrived = timeutils.FormatUnixDateTimeIn(*m.OriginActualArrive, timezone)
	}
	if move.LiveTender != nil {
		row.LiveTender = fmt.Sprintf(
			"%s, offer %d of %d with %s",
			move.LiveTender.Status, move.LiveTender.CurrentRank, move.LiveTender.OfferCount,
			move.LiveTender.CurrentCarrierName,
		)
		if move.LiveTender.CurrentOfferExpiresAt != nil {
			row.TenderExpiresAt = timeutils.FormatUnixDateTimeIn(
				*move.LiveTender.CurrentOfferExpiresAt, timezone,
			)
		}
	}

	return row
}

func toBoardDriverRow(driver *dispatchconsoleservice.BoardDriver, timezone string) boardDriverRow {
	d := driver.BoardDriver
	row := boardDriverRow{
		WorkerID:              d.WorkerID.String(),
		Name:                  strings.TrimSpace(d.FirstName + " " + d.LastName),
		Availability:          string(driver.Availability),
		DutyStatus:            driver.DutyStatus,
		DriveRemainingMinutes: driver.DriveRemainingMs / 60_000,
		ShiftRemainingMinutes: driver.ShiftRemainingMs / 60_000,
		HOSStale:              driver.HOSIsStale,
		Tractor:               d.TractorCode,
		Location:              driver.FormattedLocation,
		OpenAssignments:       d.OpenAssignments,
	}
	if driver.PositionAt > 0 {
		row.PositionAt = timeutils.FormatUnixDateTimeIn(driver.PositionAt, timezone)
	}
	if driver.ProjectedTimeAvailable > 0 {
		row.ProjectedAvailable = timeutils.FormatUnixDateTimeIn(
			driver.ProjectedTimeAvailable,
			timezone,
		)
	}
	for _, finding := range driver.Findings {
		if finding.Message != "" {
			row.Findings = append(row.Findings, finding.Message)
		}
	}

	return row
}

func placeText(name, city, state string) string {
	parts := make([]string, 0, 2)
	if name != "" {
		parts = append(parts, name)
	}
	if city != "" {
		if state != "" {
			city += ", " + state
		}
		parts = append(parts, city)
	}

	return strings.Join(parts, " in ")
}

func windowText(start int64, end *int64, timezone string) string {
	if start <= 0 {
		return "unscheduled"
	}
	text := timeutils.FormatUnixDateTimeIn(start, timezone)
	if end != nil && *end > start {
		text += " to " + timeutils.FormatUnixDateTimeIn(*end, timezone)
	}

	return text
}

func countsText(counts map[string]int) string {
	order := []string{"Late", "Now", "Today", "Tomorrow", "Planned"}
	parts := make([]string, 0, len(order))
	for _, key := range order {
		if counts[key] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
		}
	}
	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ", ")
}

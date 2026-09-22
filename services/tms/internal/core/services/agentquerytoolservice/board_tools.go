package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
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
	defaultBoardHours = 24
	maxBoardHours     = 7 * 24
	defaultBoardLimit = 40
	maxBoardLimit     = 100
)

type boardMoveRow struct {
	MoveID          string  `json:"moveId"`
	ShipmentID      string  `json:"shipmentId"`
	ProNumber       string  `json:"proNumber"`
	Customer        string  `json:"customer,omitempty"`
	ShipmentStatus  string  `json:"shipmentStatus"`
	MoveStatus      string  `json:"moveStatus"`
	Urgency         string  `json:"urgency"`
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
	Window  string           `json:"window"`
	Summary *boardSummaryRow `json:"summary"`
	Moves   []boardMoveRow   `json:"moves"`
	Drivers []boardDriverRow `json:"drivers"`
	Note    string           `json:"note"`
	Counts  map[string]int   `json:"urgencyCounts"`
}

type boardSummaryRow struct {
	UncoveredMoves   int     `json:"uncoveredMoves"`
	CoveredMoves     int     `json:"coveredMoves"`
	LateMoves        int     `json:"lateMoves"`
	AtRiskMoves      int     `json:"atRiskMoves"`
	AvailableDrivers int     `json:"availableDrivers"`
	UnseatedDrivers  int     `json:"unseatedDrivers"`
	AssignedToday    int     `json:"assignedToday"`
	UtilizationPct   float64 `json:"utilizationPercent"`
}

type getDispatchBoardTool struct {
	board boardReader
}

func newGetDispatchBoardTool(board boardReader) serviceports.AgentQueryTool {
	return &getDispatchBoardTool{board: board}
}

func (t *getDispatchBoardTool) Name() string { return "get_dispatch_board" }

func (t *getDispatchBoardTool) Description() string {
	return "The dispatch board as the desk sees it: every move in the window with its " +
		"urgency (Late, Now, Today, Tomorrow, Planned), whether it is covered and by " +
		"whom, the pickup and delivery windows, and every driver with availability, " +
		"hours remaining and last position. This is the one call for \"what is late\", " +
		"\"what is uncovered\" and \"who is free\". Late means the pickup window has " +
		"already opened with nobody there; check get_shipment_tracking on a late move " +
		"before deciding it is a problem, since arrivals are sometimes recorded late."
}

func (t *getDispatchBoardTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
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
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against PRO numbers, customers and places.",
			},
			"limit": map[string]any{
				"type": "integer",
				"description": fmt.Sprintf(
					"Moves to return. Default %d, at most %d.", defaultBoardLimit, maxBoardLimit,
				),
			},
		},
		"additionalProperties": false,
	}
}

func (t *getDispatchBoardTool) PermissionResource() permission.Resource {
	return permission.ResourceShipmentMove
}

func (t *getDispatchBoardTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	hours := optionalInt(params.Params, "hoursAhead", defaultBoardHours)
	if hours <= 0 || hours > maxBoardHours {
		hours = defaultBoardHours
	}
	limit := optionalInt(params.Params, "limit", defaultBoardLimit)
	if limit <= 0 || limit > maxBoardLimit {
		limit = defaultBoardLimit
	}

	now := clockFor(params).Now
	board, err := t.board.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo:     tenantOf(params),
		WindowStart:    now,
		WindowEnd:      now + int64(hours)*3600,
		Query:          optionalString(params.Params, "query"),
		IncludeCovered: !optionalBool(params.Params, "uncoveredOnly"),
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}

	return boardViewOf(board, params.Timezone), nil
}

func boardViewOf(board *dispatchconsoleservice.Board, timezone string) boardView {
	view := boardView{
		Window: fmt.Sprintf(
			"%s to %s",
			timeutils.FormatUnixDateTimeIn(board.WindowStart, timezone),
			timeutils.FormatUnixDateTimeIn(board.WindowEnd, timezone),
		),
		Moves:   make([]boardMoveRow, 0, len(board.Moves)),
		Drivers: make([]boardDriverRow, 0, len(board.Drivers)),
		Counts:  make(map[string]int, 5),
	}
	if board.Summary != nil {
		view.Summary = &boardSummaryRow{
			UncoveredMoves:   board.Summary.UncoveredMoves,
			CoveredMoves:     board.Summary.CoveredMoves,
			LateMoves:        board.Summary.LateMoves,
			AtRiskMoves:      board.Summary.AtRiskMoves,
			AvailableDrivers: board.Summary.AvailableDrivers,
			UnseatedDrivers:  board.Summary.UnseatedDrivers,
			AssignedToday:    board.Summary.AssignedToday,
			UtilizationPct:   board.Summary.UtilizationPct,
		}
	}

	for _, move := range board.Moves {
		if move == nil || move.BoardMove == nil {
			continue
		}
		view.Counts[string(move.Urgency)]++
		view.Moves = append(view.Moves, toBoardMoveRow(move, timezone))
	}
	for _, driver := range board.Drivers {
		if driver == nil || driver.BoardDriver == nil {
			continue
		}
		view.Drivers = append(view.Drivers, toBoardDriverRow(driver, timezone))
	}

	view.Note = fmt.Sprintf(
		"%d moves and %d drivers in the window. Urgency counts: %s. Late is a pickup "+
			"window already open with no arrival recorded.",
		len(view.Moves), len(view.Drivers), countsText(view.Counts),
	)

	return view
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
		row.ProjectedAvailable = timeutils.FormatUnixDateTimeIn(driver.ProjectedTimeAvailable, timezone)
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

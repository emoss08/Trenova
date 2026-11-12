package recordcaseservice

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
)

const (
	driveLimitMinutes = 11 * 60
	shiftLimitMinutes = 14 * 60
	cycleLimitMinutes = 70 * 60
	routePadding      = 0.06
	lowHoursFraction  = 0.25
)

func shipmentSubject(tracking shipmenttrackingports.ShipmentTrackingReader) subjectReader {
	return func(ctx context.Context, read *subjectRead) (*subject, error) {
		snapshots, err := tracking.TrackingSnapshots(
			ctx,
			read.tenant,
			[]pulid.ID{read.id},
			read.zone.String(),
		)
		if err != nil {
			return nil, err
		}
		snapshot := snapshots[read.id]
		if snapshot == nil {
			return &subject{}, nil
		}

		againstTheWindowAsItIs(snapshot)

		return (&shipmentCase{read: read, snapshot: snapshot, stops: allStops(snapshot)}).build(), nil
	}
}

// againstTheWindowAsItIs judges each projected arrival against the stop's
// window as the record holds it now. A projection keeps the window it was
// worked out against until the next one is made, so a window somebody just
// moved would otherwise be drawn beside a lateness measured from the old one:
// "arrives 8:23 PM, 6h 34m after the window closes" under a window that
// closes at 9:04 PM. Nothing here predicts anything: the arrival is the
// projection's, the window is the record's, and the difference is arithmetic.
func againstTheWindowAsItIs(snapshot *shipmenttracking.Snapshot) {
	for i := range snapshot.Moves {
		for j := range snapshot.Moves[i].Stops {
			stop := &snapshot.Moves[i].Stops[j]
			if stop.EstimatedArrival <= 0 || stop.ScheduledEnd <= 0 || stop.ActualArrival > 0 {
				continue
			}
			slack := (stop.ScheduledEnd - stop.EstimatedArrival) / secondsPerMinute
			stop.SlackMinutes = &slack
			stop.EtaVerdict = string(
				verdictFor(shipmenttracking.Verdict(stop.EtaVerdict), slack),
			)
		}
	}

	estimate, next := snapshot.Estimate, snapshot.NextStop
	if estimate == nil || next == nil || estimate.EstimatedArrival <= 0 || next.ScheduledEnd <= 0 {
		return
	}
	estimate.WindowEnd = next.ScheduledEnd
	estimate.SlackMinutes = (next.ScheduledEnd - estimate.EstimatedArrival) / secondsPerMinute
	estimate.Verdict = verdictFor(estimate.Verdict, estimate.SlackMinutes)
}

func verdictFor(stored shipmenttracking.Verdict, slack int64) shipmenttracking.Verdict {
	switch {
	case slack < 0:
		return shipmenttracking.VerdictLate
	case stored == shipmenttracking.VerdictLate:
		return shipmenttracking.VerdictOnTime
	default:
		return stored
	}
}

type shipmentCase struct {
	read     *subjectRead
	snapshot *shipmenttracking.Snapshot
	stops    []*shipmenttracking.StopSnapshot
}

func allStops(snapshot *shipmenttracking.Snapshot) []*shipmenttracking.StopSnapshot {
	stops := make([]*shipmenttracking.StopSnapshot, 0, 4)
	for i := range snapshot.Moves {
		for j := range snapshot.Moves[i].Stops {
			stops = append(stops, &snapshot.Moves[i].Stops[j])
		}
	}

	return stops
}

func (c *shipmentCase) build() *subject {
	out := &subject{
		head:     c.head(),
		absorbs:  map[watchtower.SourceKind]struct{}{},
		customer: c.snapshot.Customer,
	}
	if route := c.route(); route != nil {
		out.exhibits = append(out.exhibits, *route)
	}
	if hours := c.hours(); hours != nil {
		out.exhibits = append(out.exhibits, *hours)
	}
	if stops := c.timeline(); stops != nil {
		out.exhibits = append(out.exhibits, *stops)
		if c.snapshot.Estimate != nil && c.snapshot.Estimate.EstimatedArrival > 0 {
			out.absorbs[watchtower.SourceEtaAtRisk] = struct{}{}
		}
	}
	out.unchanged = c.unchanged()
	out.starters = c.starters()

	return out
}

func (c *shipmentCase) head() recordcase.Head {
	head := recordcase.Head{StatusTone: recordcase.ToneNeutral}
	if c.snapshot.Customer != "" {
		head.References = append(head.References, c.snapshot.Customer)
	}
	if c.snapshot.BOL != "" {
		head.References = append(head.References, "BOL "+c.snapshot.BOL)
	}
	if len(c.stops) >= 2 {
		from, to := cityOf(c.stops[0]), cityOf(c.stops[len(c.stops)-1])
		if from != "" && to != "" {
			head.Title = from + " → " + to
		}
	}

	status := textcase.HumanizeCamelCaseSentence(c.snapshot.Status)
	head.Status = status
	if estimate := c.snapshot.Estimate; estimate != nil {
		switch estimate.Verdict {
		case shipmenttracking.VerdictLate:
			head.Status, head.StatusTone = status+" · projected late", recordcase.ToneBad
		case shipmenttracking.VerdictAtRisk:
			head.Status, head.StatusTone = status+" · at risk", recordcase.ToneWarning
		case shipmenttracking.VerdictOnTime:
			head.StatusTone = recordcase.ToneGood
			head.Status = status + " · on time"
			if c.read.done {
				head.Status = status + " · back on time"
			}
		case shipmenttracking.VerdictUnknown:
		}
	}

	if crew := c.crew(); crew != "" {
		head.Details = append(head.Details, crew)
	}
	if next := c.snapshot.NextStop; next != nil {
		detail := stopKind(next.Type) + " " +
			c.read.at(next.Timezone).window(next.ScheduledStart, next.ScheduledEnd)
		if next.Location != "" {
			detail += " at " + next.Location
		}
		head.Details = append(head.Details, detail)
	}

	return head
}

func (c *shipmentCase) move() *shipmenttracking.MoveSnapshot {
	if len(c.snapshot.Moves) == 0 {
		return nil
	}
	if next := c.snapshot.NextStop; next != nil {
		for i := range c.snapshot.Moves {
			if c.snapshot.Moves[i].ID == next.MoveID {
				return &c.snapshot.Moves[i]
			}
		}
	}

	return &c.snapshot.Moves[len(c.snapshot.Moves)-1]
}

func (c *shipmentCase) crew() string {
	move := c.move()
	if move == nil {
		return ""
	}
	if move.Carrier != "" {
		return joined(move.Carrier, move.Trailer)
	}

	return joined(move.Driver, move.Tractor, move.Trailer)
}

func cityOf(stop *shipmenttracking.StopSnapshot) string {
	switch {
	case stop.City != "" && stop.State != "":
		return stop.City + ", " + stop.State
	case stop.City != "":
		return stop.City
	default:
		return stop.Location
	}
}

func stopDone(stop *shipmenttracking.StopSnapshot) bool {
	return stop.ActualDeparture > 0 || stop.ActualArrival > 0 || stop.Status == "Completed"
}

type geoPoint struct {
	label string
	lat   float64
	lon   float64
	state recordcase.PointState
}

func (c *shipmentCase) route() *recordcase.Exhibit {
	points := make([]geoPoint, 0, len(c.stops)+1)
	for _, stop := range c.stops {
		lat, lon, ok := stop.Coordinates()
		if !ok {
			continue
		}
		state := recordcase.PointAhead
		if stopDone(stop) {
			state = recordcase.PointDone
		}
		points = append(points, geoPoint{
			label: stringutils.FirstNonEmptyTrimmed(stop.City, stop.Location),
			lat:   lat,
			lon:   lon,
			state: state,
		})
	}
	position := c.snapshot.Position
	if len(points) < 2 && position == nil {
		return nil
	}

	if step := c.projectedStep(); step != nil && step.Latitude != nil && step.Longitude != nil &&
		len(points) >= 2 {
		proposed := geoPoint{
			label: step.Label,
			lat:   *step.Latitude,
			lon:   *step.Longitude,
			state: recordcase.PointProposed,
		}
		last := len(points) - 1
		points = append(points[:last], proposed, points[last])
	}

	extent := points
	if position != nil {
		extent = append(append([]geoPoint{}, points...), geoPoint{
			lat: position.Latitude,
			lon: position.Longitude,
		})
	}
	project := projector(extent)

	route := &recordcase.Route{Points: make([]recordcase.RoutePoint, 0, len(points))}
	for _, point := range points {
		x, y := project(point.lat, point.lon)
		route.Points = append(route.Points, recordcase.RoutePoint{
			Label:     point.label,
			X:         x,
			Y:         y,
			Latitude:  point.lat,
			Longitude: point.lon,
			State:     point.state,
		})
	}

	exhibit := &recordcase.Exhibit{
		Key:    "position",
		Kind:   recordcase.ExhibitRoute,
		Title:  "Where it is",
		Source: "Planned route",
		Route:  route,
	}
	if position != nil {
		x, y := project(position.Latitude, position.Longitude)
		route.Truck = &recordcase.RouteMarker{
			X:         x,
			Y:         y,
			Latitude:  position.Latitude,
			Longitude: position.Longitude,
		}
		exhibit.Source = joined("Telematics", position.Tractor)
		exhibit.At = position.RecordedAt
		exhibit.Read = true
		exhibit.Note = c.positionNote(position)
		exhibit.Claims = append(exhibit.Claims, exhibit.Note)
		if position.Stale {
			exhibit.Claims = append(exhibit.Claims, fmt.Sprintf(
				"the position is %s old", span(position.AgeMinutes),
			))
		}
	}
	if breakAt, ok := c.breakDue(); ok {
		route.Callout = "break due ~" + c.read.clock(breakAt)
		route.CalloutTone = recordcase.ToneWarning
	}

	return exhibit
}

func projector(points []geoPoint) func(lat, lon float64) (x, y float64) {
	minLat, maxLat := math.Inf(1), math.Inf(-1)
	minLon, maxLon := math.Inf(1), math.Inf(-1)
	for _, point := range points {
		minLat, maxLat = math.Min(minLat, point.lat), math.Max(maxLat, point.lat)
		minLon, maxLon = math.Min(minLon, point.lon), math.Max(maxLon, point.lon)
	}
	latSpan, lonSpan := maxLat-minLat, maxLon-minLon
	usable := 1 - 2*routePadding

	return func(lat, lon float64) (x, y float64) {
		x, y = 0.5, 0.5
		if lonSpan > 0 {
			x = routePadding + usable*(lon-minLon)/lonSpan
		}
		if latSpan > 0 {
			y = routePadding + usable*(maxLat-lat)/latSpan
		}

		return x, y
	}
}

func (c *shipmentCase) positionNote(position *shipmenttracking.PositionSnapshot) string {
	parts := make([]string, 0, 3)
	if position.FormattedLocation != "" {
		parts = append(parts, position.FormattedLocation)
	}
	if position.SpeedMph >= 1 {
		parts = append(parts, fmt.Sprintf("moving %.0f mph", position.SpeedMph))
	} else {
		parts = append(parts, "stopped")
	}
	if estimate := c.snapshot.Estimate; estimate != nil && estimate.MilesRemaining > 0 {
		miles := fmt.Sprintf("%.0f miles", estimate.MilesRemaining)
		if next := c.snapshot.NextStop; next != nil && next.City != "" {
			miles += " to " + next.City
		}
		parts = append(parts, miles)
	}

	return strings.Join(parts, " · ")
}

func (c *shipmentCase) breakDue() (int64, bool) {
	driver, estimate := c.snapshot.Driver, c.snapshot.Estimate
	if driver == nil || estimate == nil || driver.Stale || driver.DriveRemainingMinutes <= 0 {
		return 0, false
	}
	if estimate.DriveMinutes <= driver.DriveRemainingMinutes {
		return 0, false
	}

	return driver.RecordedAt + driver.DriveRemainingMinutes*60, true
}

func (c *shipmentCase) hours() *recordcase.Exhibit {
	driver := c.snapshot.Driver
	if driver == nil {
		return nil
	}
	title := "Driver's hours"
	if driver.Name != "" {
		title = driver.Name + "'s hours"
	}

	driveTone := recordcase.ToneNeutral
	_, short := c.breakDue()
	if short || fraction(driver.DriveRemainingMinutes, driveLimitMinutes) < lowHoursFraction {
		driveTone = recordcase.ToneWarning
	}
	exhibit := &recordcase.Exhibit{
		Key:    "hours",
		Kind:   recordcase.ExhibitMeters,
		Title:  title,
		Source: "ELD · hours of service",
		At:     driver.RecordedAt,
		Read:   true,
		Meters: []recordcase.Meter{
			meter("Driving left", driver.DriveRemainingMinutes, driveLimitMinutes, driveTone),
			meter(
				"On duty left",
				driver.ShiftRemainingMinutes,
				shiftLimitMinutes,
				recordcase.ToneNeutral,
			),
			meter(
				"Cycle left",
				driver.CycleRemainingMinutes,
				cycleLimitMinutes,
				recordcase.ToneNeutral,
			),
		},
	}
	if breakAt, ok := c.breakDue(); ok {
		exhibit.Note = "Break due by " + c.read.clock(breakAt) + "."
	}
	if driver.Stale {
		exhibit.Note = strings.TrimSpace(exhibit.Note + " These hours are more than 12 hours old.")
	}
	name := stringutils.FirstNonEmptyTrimmed(driver.Name, "The driver")
	exhibit.Claims = []string{
		fmt.Sprintf("%s has %s of driving left of 11h", name, span(driver.DriveRemainingMinutes)),
		fmt.Sprintf("%s of on-duty time left of 14h", span(driver.ShiftRemainingMinutes)),
		fmt.Sprintf("%s left in the 70h cycle", span(driver.CycleRemainingMinutes)),
		exhibit.Note,
	}
	if estimate := c.snapshot.Estimate; estimate != nil && estimate.DriveMinutes > 0 {
		exhibit.Claims = append(exhibit.Claims, fmt.Sprintf(
			"%s of driving remains to the next stop", span(estimate.DriveMinutes),
		))
	}

	return exhibit
}

func meter(label string, remaining, limit int64, tone recordcase.Tone) recordcase.Meter {
	return recordcase.Meter{
		Label:    label,
		Value:    fmt.Sprintf("%s of %dh", span(remaining), limit/60),
		Fraction: fraction(remaining, limit),
		Tone:     tone,
	}
}

func (c *shipmentCase) projectedStep() *recordcase.ProjectedStep {
	if c.read.chosen == nil || c.read.chosen.Projection == nil {
		return nil
	}

	return c.read.chosen.Projection.Step
}

func (c *shipmentCase) timeline() *recordcase.Exhibit {
	if len(c.stops) == 0 {
		return nil
	}
	exhibit := &recordcase.Exhibit{
		Key:      "stops",
		Kind:     recordcase.ExhibitTimeline,
		Title:    "Stops",
		Source:   "Scheduled windows",
		Timeline: make([]recordcase.TimelineRow, 0, len(c.stops)+1),
	}
	if estimate := c.snapshot.Estimate; estimate != nil && estimate.EstimatedArrival > 0 {
		exhibit.Source = "Projected from position and hours"
		exhibit.At = estimate.ComputedAt
		exhibit.Read = estimate.ComputedAt > 0
	}

	last := len(c.stops) - 1
	for index, stop := range c.stops {
		if index == last && index > 0 {
			if step := c.projectedStep(); step != nil {
				exhibit.Timeline = append(exhibit.Timeline, recordcase.TimelineRow{
					Label:    step.Label,
					When:     step.When,
					State:    recordcase.PointProposed,
					Proposed: true,
				})
			}
		}
		row := c.stopRow(stop)
		if index == last && !stopDone(stop) {
			c.applyOutcome(&row)
		}
		exhibit.Timeline = append(exhibit.Timeline, row)
		exhibit.Claims = append(exhibit.Claims, stopClaim(c.read, stop))
	}

	return exhibit
}

func (c *shipmentCase) stopRow(stop *shipmenttracking.StopSnapshot) recordcase.TimelineRow {
	row := recordcase.TimelineRow{
		Label: joined(stopKind(stop.Type), strings.Join(nonEmpty(stop.Location, stop.City), ", ")),
		State: recordcase.PointAhead,
		Tone:  recordcase.ToneNeutral,
	}
	read := c.read.at(stop.Timezone)
	switch {
	case stop.ActualDeparture > 0:
		row.State = recordcase.PointDone
		row.When = "Departed " + read.clock(stop.ActualDeparture)
	case stop.ActualArrival > 0:
		row.State = recordcase.PointDone
		row.When = "Arrived " + read.clock(stop.ActualArrival)
	default:
		row.Sub = "Window " + read.window(stop.ScheduledStart, stop.ScheduledEnd)
		if stop.EstimatedArrival > 0 {
			row.When = read.clock(stop.EstimatedArrival)
			row.Tone = verdictTone(stop.EtaVerdict)
		} else if stop.Overdue {
			row.When = stop.LateBy + " overdue"
			row.Tone = recordcase.ToneBad
		}
	}
	if stop.Late {
		row.Tone = recordcase.ToneBad
	}

	return row
}

func (c *shipmentCase) applyOutcome(row *recordcase.TimelineRow) {
	chosen := c.read.chosen
	if chosen == nil || chosen.Kind == recordcase.OptionKindLeave || c.read.stopped {
		return
	}
	if chosen.Projection != nil && chosen.Projection.Window != "" {
		row.Sub = "Window " + chosen.Projection.Window + " (proposed)"
	}
	if chosen.Outcome == "" {
		return
	}
	row.When, row.Tone = chosen.Outcome, chosen.Tone
	if !c.read.done {
		row.When += " if approved"
	}
}

func verdictTone(verdict string) recordcase.Tone {
	switch shipmenttracking.Verdict(verdict) {
	case shipmenttracking.VerdictLate:
		return recordcase.ToneBad
	case shipmenttracking.VerdictAtRisk:
		return recordcase.ToneWarning
	case shipmenttracking.VerdictOnTime:
		return recordcase.ToneGood
	case shipmenttracking.VerdictUnknown:
		return recordcase.ToneNeutral
	default:
		return recordcase.ToneNeutral
	}
}

func nonEmpty(values ...string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}

	return kept
}

func stopKind(kind string) string {
	return textcase.HumanizeCamelCaseSentence(kind)
}

func stopClaim(read *subjectRead, stop *shipmenttracking.StopSnapshot) string {
	read = read.at(stop.Timezone)
	claim := fmt.Sprintf("%s at %s, window %s",
		stopKind(stop.Type),
		strings.Join(nonEmpty(stop.Location, stop.City), ", "),
		read.window(stop.ScheduledStart, stop.ScheduledEnd),
	)
	switch {
	case stop.ActualDeparture > 0:
		claim += ", departed " + read.clock(stop.ActualDeparture)
	case stop.ActualArrival > 0:
		claim += ", arrived " + read.clock(stop.ActualArrival)
	case stop.EstimatedArrival > 0:
		claim += ", projected arrival " + read.clock(stop.EstimatedArrival)
		if stop.SlackMinutes != nil && *stop.SlackMinutes < 0 {
			claim += ", " + span(-*stop.SlackMinutes) + " after the window closes"
		}
	}

	return claim
}

func (c *shipmentCase) unchanged() string {
	estimate := c.snapshot.Estimate
	if estimate == nil || estimate.EstimatedArrival <= 0 {
		return ""
	}
	there := c.read
	if next := c.snapshot.NextStop; next != nil {
		there = c.read.at(next.Timezone)
	}
	arrives := "Arrives about " + there.clock(estimate.EstimatedArrival)
	switch estimate.Verdict {
	case shipmenttracking.VerdictLate:
		if estimate.SlackMinutes < 0 {
			return arrives + ", " + span(-estimate.SlackMinutes) + " after the window closes."
		}

		return arrives + ", after the window closes."
	case shipmenttracking.VerdictAtRisk:
		return arrives + ", with " + span(estimate.SlackMinutes) + " to spare."
	case shipmenttracking.VerdictOnTime, shipmenttracking.VerdictUnknown:
		return ""
	default:
		return ""
	}
}

func (c *shipmentCase) starters() []string {
	starters := []string{"What happens if nothing changes?"}
	if move := c.move(); move != nil && move.Driver != "" {
		starters = append(starters, "What else does "+firstName(move.Driver)+" have today?")
	}
	if c.snapshot.Customer != "" {
		starters = append(starters, "Draft an update for "+c.snapshot.Customer)
	}

	return starters
}

func firstName(name string) string {
	if first, _, ok := strings.Cut(strings.TrimSpace(name), " "); ok {
		return first
	}

	return name
}

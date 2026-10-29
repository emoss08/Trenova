package stopetavolume

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	etaChunk             = 5_000
	etaPositionFresh     = int64(3600)
	etaCheckCallStale    = int64(4 * 3600)
	etaStaleWindow       = 6 * volumegen.Hour
	etaStaleShare        = 0.3
	etaDriveSecondsLimit = 11 * volumegen.Hour
	etaResetSeconds      = 10 * volumegen.Hour
	etaBreakAfter        = 8 * volumegen.Hour
	etaBreakSeconds      = 30 * volumegen.Minute
)

type etaPosition struct {
	lat, lon   float64
	recordedAt int64
}

type etaItem struct {
	move *volumegen.MoveRecord
	stop *volumegen.StopRecord
}

type etaOrigin struct {
	basis    stopeta.Basis
	readyAt  int64
	inputAt  int64
	stale    bool
	chained  bool
	atStop   bool
	reported int64
	miles    float64
}

type etaCursor struct {
	readyAt int64
	miles   float64
	driveS  int64
	restS   int64
	sinceRs int64
	sinceBk int64
	routed  bool
	planned bool
	clamped bool
	breaks  int
	resets  int
}

type etaBuilder struct {
	g         *volumegen.Generator
	now       int64
	threshold int64
	dwell     int64
	positions map[pulid.ID]*etaPosition
	rows      []*stopeta.StopETA
	verdicts  map[stopeta.Verdict]int
}

type etaPhase struct {
	*volumegen.Generator
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&etaPhase{Generator: g}).seed(ctx)
}

func (g *etaPhase) seed(ctx context.Context) error {
	control, err := g.DispatchControl(ctx)
	if err != nil {
		return err
	}
	b := &etaBuilder{
		g:         g.Generator,
		now:       volumegen.Unix(g.Now),
		threshold: int64(control.EtaThresholdMinutes(nil)),
		dwell:     int64(control.EtaDwell()),
		positions: make(map[pulid.ID]*etaPosition),
		verdicts:  make(map[stopeta.Verdict]int),
	}
	if err = b.loadPositions(ctx); err != nil {
		return err
	}

	for _, rec := range g.Shipments {
		switch {
		case stopeta.TrackedShipment(rec.Status):
			b.project(rec)
		case rec.DeliveredAt > 0 && b.now-rec.DeliveredAt <= etaStaleWindow &&
			g.Fake.Chance(etaStaleShare):
			b.stale(rec)
		}
	}

	total := len(b.rows)
	for begin := 0; begin < total; begin += etaChunk {
		end := min(begin+etaChunk, total)
		chunk := b.rows[begin:end]
		if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			return volumegen.InsertRows(ctx, tx, stopetadb.StopETAs.Handle, chunk)
		}); err != nil {
			return err
		}
	}

	g.Report.Add("Stop ETAs", total)
	for _, v := range []stopeta.Verdict{
		stopeta.VerdictOnTime, stopeta.VerdictAtRisk, stopeta.VerdictLate, stopeta.VerdictUnknown,
	} {
		g.Opts.Logf("  • stop ETAs %s: %d", v, b.verdicts[v])
	}

	return nil
}

func (b *etaBuilder) loadPositions(ctx context.Context) error {
	ids := make([]pulid.ID, 0, len(b.g.Crews))
	for _, c := range b.g.Crews {
		ids = append(ids, c.TractorID)
	}
	if len(ids) == 0 {
		return nil
	}
	vp := telematicsdb.VehiclePositions
	rows, err := vp.Pick(vp.TractorID, vp.Latitude, vp.Longitude, vp.RecordedAt).
		Where(vp.TractorID.In(ids...)).
		All(ctx, b.g.DB)
	if err != nil {
		return fmt.Errorf("load vehicle positions: %w", err)
	}
	for _, row := range rows {
		b.positions[row.TractorID] = &etaPosition{
			lat:        row.Latitude,
			lon:        row.Longitude,
			recordedAt: row.RecordedAt,
		}
	}

	return nil
}

func openStop(move *volumegen.MoveRecord, stop *volumegen.StopRecord) bool {
	if move.Status == shipmentvocab.MoveStatusCompleted ||
		move.Status == shipmentvocab.MoveStatusCanceled {
		return false
	}
	return stop.Status != shipmentvocab.StopStatusCompleted &&
		stop.Status != shipmentvocab.StopStatusCanceled &&
		stop.ActualDeparture == 0
}

func (b *etaBuilder) project(rec *volumegen.ShipmentRecord) {
	computed := b.now - b.g.Fake.Between64(0, 9*volumegen.Minute)
	var open []etaItem
	var lastDeparted *volumegen.StopRecord
	for _, move := range rec.Moves {
		for _, stop := range move.Stops {
			if openStop(move, stop) {
				open = append(open, etaItem{move: move, stop: stop})
				continue
			}
			if len(open) == 0 && stop.ActualDeparture > 0 &&
				stop.Status != shipmentvocab.StopStatusCanceled {
				lastDeparted = stop
			}
		}
	}
	if len(open) == 0 {
		return
	}

	o := b.origin(open[0], lastDeparted, computed)
	c := &etaCursor{readyAt: o.readyAt}
	for i, item := range open {
		row := b.baseRow(rec, item, o, computed)
		arrival, method := b.arrive(c, o, i, item, computed)
		row.DriveMethod = method
		if arrival > 0 {
			start := max(arrival, item.stop.WindowStart)
			departure := start + b.dwell*volumegen.Minute
			if item.stop.Arrived() {
				departure = max(departure, computed)
			}
			row.EstimatedArrival = volumegen.Ptr(arrival)
			row.EstimatedDeparture = volumegen.Ptr(departure)
			c.readyAt = departure
		}
		row.MilesRemaining = math.Round(c.miles*10) / 10
		row.DriveMinutes = c.driveS / volumegen.Minute
		row.RestMinutes = c.restS / volumegen.Minute
		row.InputStale = o.stale || c.clamped
		b.judge(row, item.stop, computed)
		delayed := rec.Status == shipmentvocab.StatusDelayed && !row.Arrived &&
			row.Verdict == stopeta.VerdictUnknown
		if delayed {
			row.Verdict = stopeta.VerdictLate
		}
		row.Reason = b.reason(row, o, c, item, computed, delayed)
		b.add(row)
	}
}

func (b *etaBuilder) origin(
	first etaItem,
	lastDeparted *volumegen.StopRecord,
	computed int64,
) etaOrigin {
	f := b.g.Fake
	if first.stop.Arrived() {
		return etaOrigin{
			basis:   stopeta.BasisAtStop,
			readyAt: computed,
			inputAt: first.stop.ActualArrival,
			chained: true,
			atStop:  true,
		}
	}

	move := first.move
	var position *etaPosition
	if move.Crew != nil {
		position = b.positions[move.Crew.TractorID]
	}
	if position != nil && computed-position.recordedAt <= etaPositionFresh {
		return etaOrigin{
			basis:   stopeta.BasisPosition,
			readyAt: computed,
			inputAt: position.recordedAt,
			chained: true,
			miles:   roadMiles(position.lat, position.lon, first.stop.Location.City),
		}
	}
	if move.CarrierAssignment != nil {
		switch f.Weighted([]int{35, 40, 25}) {
		case 0:
			reportedAt := computed - f.Between64(10*volumegen.Minute, 3*volumegen.Hour)
			reported := max(
				first.stop.WindowStart+f.Between64(-45*volumegen.Minute, 3*volumegen.Hour),
				computed+15*volumegen.Minute,
			)
			return etaOrigin{
				basis:    stopeta.BasisCarrierReported,
				readyAt:  computed,
				inputAt:  reportedAt,
				chained:  true,
				reported: reported,
			}
		case 1:
			reportedAt := computed - f.Between64(15*volumegen.Minute, 6*volumegen.Hour)
			return etaOrigin{
				basis:   stopeta.BasisCheckCall,
				readyAt: computed,
				inputAt: reportedAt,
				stale:   computed-reportedAt > etaCheckCallStale,
				chained: true,
				miles:   first.stop.Miles * (0.2 + f.Float()*0.7),
			}
		}
	}
	if position != nil {
		return etaOrigin{
			basis:   stopeta.BasisPosition,
			readyAt: computed,
			inputAt: position.recordedAt,
			stale:   true,
			chained: true,
			miles:   roadMiles(position.lat, position.lon, first.stop.Location.City),
		}
	}
	if lastDeparted != nil {
		return etaOrigin{
			basis:   stopeta.BasisLastDeparture,
			readyAt: lastDeparted.ActualDeparture,
			inputAt: lastDeparted.ActualDeparture,
			chained: true,
			miles:   first.stop.Miles,
		}
	}

	return etaOrigin{basis: stopeta.BasisNone}
}

func (b *etaBuilder) baseRow(
	rec *volumegen.ShipmentRecord,
	item etaItem,
	o etaOrigin,
	computed int64,
) *stopeta.StopETA {
	row := &stopeta.StopETA{
		ID:               volumegen.IDAt("seta_", computed),
		OrganizationID:   b.g.Refs.OrgID,
		BusinessUnitID:   b.g.Refs.BuID,
		ShipmentID:       rec.ID,
		ShipmentMoveID:   item.move.ID,
		StopID:           item.stop.ID,
		MoveSequence:     item.move.Sequence,
		StopSequence:     item.stop.Sequence,
		Arrived:          item.stop.Arrived(),
		Verdict:          stopeta.VerdictUnknown,
		Basis:            o.basis,
		DriveMethod:      stopeta.DriveMethodNone,
		ThresholdMinutes: b.threshold,
		ComputedAt:       computed,
	}
	if cutoff := cutoffOf(item.stop); cutoff > 0 {
		row.WindowEnd = volumegen.Ptr(cutoff)
	}
	if o.inputAt > 0 {
		row.InputAt = volumegen.Ptr(o.inputAt)
	}

	return row
}

func (b *etaBuilder) arrive(
	c *etaCursor,
	o etaOrigin,
	index int,
	item etaItem,
	computed int64,
) (int64, stopeta.DriveMethod) {
	if !o.chained {
		return 0, stopeta.DriveMethodNone
	}
	if index == 0 && o.atStop {
		return item.stop.ActualArrival, stopeta.DriveMethodNone
	}
	if index == 0 && o.basis == stopeta.BasisCarrierReported {
		return o.reported, stopeta.DriveMethodReported
	}

	miles := item.stop.Miles
	if index == 0 {
		miles = o.miles
	}
	method := stopeta.DriveMethodPlanning
	speed := 50.0
	if b.g.Fake.Chance(0.85) {
		method = stopeta.DriveMethodRouted
		speed = 52.0
		c.routed = true
	} else {
		c.planned = true
	}
	drive := int64(miles / speed * float64(volumegen.Hour))
	elapsed := drive
	if item.move.Crew != nil {
		elapsed += c.rest(drive)
	}
	c.miles += miles
	c.driveS += drive

	arrival := c.readyAt + elapsed
	if arrival < computed {
		c.clamped = true
		arrival = computed
	}

	return arrival, method
}

func (c *etaCursor) rest(drive int64) int64 {
	added := int64(0)
	for left := drive; left > 0; {
		step := min(left, etaDriveSecondsLimit-c.sinceRs, etaBreakAfter-c.sinceBk)
		left -= step
		c.sinceRs += step
		c.sinceBk += step
		switch {
		case c.sinceRs >= etaDriveSecondsLimit && left > 0:
			added += etaResetSeconds
			c.resets++
			c.sinceRs, c.sinceBk = 0, 0
		case c.sinceBk >= etaBreakAfter && left > 0:
			added += etaBreakSeconds
			c.breaks++
			c.sinceBk = 0
		}
	}
	c.restS += added

	return added
}

func (b *etaBuilder) judge(row *stopeta.StopETA, stop *volumegen.StopRecord, computed int64) {
	cutoff := cutoffOf(stop)
	if stop.Arrived() {
		if cutoff <= 0 {
			row.Verdict = stopeta.VerdictOnTime
			return
		}
		slack := floorMinutes(cutoff - stop.ActualArrival)
		row.SlackMinutes = volumegen.Ptr(slack)
		row.Verdict = stopeta.VerdictOnTime
		if stop.ActualArrival > cutoff {
			row.Verdict = stopeta.VerdictLate
		}
		return
	}
	if cutoff > 0 && computed > cutoff {
		reference := computed
		if row.EstimatedArrival != nil {
			reference = *row.EstimatedArrival
		}
		row.SlackMinutes = volumegen.Ptr(floorMinutes(cutoff - reference))
		row.Verdict = stopeta.VerdictLate
		return
	}
	if row.EstimatedArrival == nil {
		return
	}
	if cutoff <= 0 {
		row.Verdict = stopeta.VerdictOnTime
		return
	}
	slack := floorMinutes(cutoff - *row.EstimatedArrival)
	row.SlackMinutes = volumegen.Ptr(slack)
	switch {
	case *row.EstimatedArrival > cutoff:
		row.Verdict = stopeta.VerdictLate
	case slack < b.threshold:
		row.Verdict = stopeta.VerdictAtRisk
	default:
		row.Verdict = stopeta.VerdictOnTime
	}
}

func (b *etaBuilder) stale(rec *volumegen.ShipmentRecord) {
	move := rec.Moves[len(rec.Moves)-1]
	stop := move.Stops[len(move.Stops)-1]
	if !stop.Arrived() {
		return
	}
	computed := min(
		stop.ActualArrival+b.g.Fake.Between64(volumegen.Minute, 20*volumegen.Minute),
		b.now,
	)
	row := b.baseRow(rec, etaItem{move: move, stop: stop}, etaOrigin{
		basis:   stopeta.BasisAtStop,
		inputAt: stop.ActualArrival,
	}, computed)
	departure := max(stop.ActualArrival, stop.WindowStart) + b.dwell*volumegen.Minute
	row.EstimatedArrival = volumegen.Ptr(stop.ActualArrival)
	row.EstimatedDeparture = volumegen.Ptr(departure)
	b.judge(row, stop, computed)
	row.Reason = "The truck is at the stop."
	b.add(row)
}

func (b *etaBuilder) add(row *stopeta.StopETA) {
	b.verdicts[row.Verdict]++
	b.rows = append(b.rows, row)
}

func (b *etaBuilder) reason(
	row *stopeta.StopETA,
	o etaOrigin,
	c *etaCursor,
	item etaItem,
	computed int64,
	delayed bool,
) string {
	parts := make([]string, 0, 4)
	age := timeutils.FormatLongDurationMs((computed - o.inputAt) * 1000)
	switch o.basis {
	case stopeta.BasisAtStop:
		parts = append(parts, "The truck is at the stop.")
	case stopeta.BasisPosition:
		if o.stale {
			parts = append(
				parts,
				fmt.Sprintf("From the truck's last position, which is %s old.", age),
			)
		} else {
			parts = append(parts, fmt.Sprintf("From the truck's position %s ago.", age))
		}
	case stopeta.BasisCarrierReported:
		parts = append(
			parts,
			fmt.Sprintf("The carrier gave this arrival on a check call %s ago.", age),
		)
	case stopeta.BasisCheckCall:
		parts = append(parts, fmt.Sprintf("From the position on a check call %s ago.", age))
	case stopeta.BasisLastDeparture:
		parts = append(parts,
			fmt.Sprintf("From the last departure %s ago; there is no position since.", age))
	case stopeta.BasisNone:
		parts = append(parts, "There is no position, check call or departure to estimate from.")
	}
	if row.EstimatedArrival != nil && (row.DriveMethod == stopeta.DriveMethodRouted ||
		row.DriveMethod == stopeta.DriveMethodPlanning) {
		method := "planning speed"
		switch {
		case c.routed && c.planned:
			method = "partly routed, partly at planning speed"
		case c.routed:
			method = "routed"
		}
		parts = append(parts, fmt.Sprintf("%.0f miles, %s of driving (%s).",
			c.miles, timeutils.FormatLongDurationMs(c.driveS*1000), method))
	}
	if c.restS > 0 && item.move.Crew != nil {
		pieces := make([]string, 0, 2)
		if c.breaks > 0 {
			pieces = append(pieces, plural(c.breaks, "30 minute break", "30 minute breaks"))
		}
		if c.resets > 0 {
			pieces = append(pieces, plural(c.resets, "10 hour reset", "10 hour resets"))
		}
		parts = append(parts,
			fmt.Sprintf("Includes %s for hours of service.", strings.Join(pieces, ", ")))
	}
	if cutoff := cutoffOf(item.stop); !item.stop.Arrived() && cutoff > 0 && computed > cutoff {
		parts = append(parts, fmt.Sprintf("The window closed %s ago with no arrival recorded.",
			timeutils.FormatLongDurationMs((computed-cutoff)*1000)))
	}
	if delayed {
		parts = append(parts, "The shipment is marked delayed.")
	}

	return stringutils.TruncateRunes(strings.Join(parts, " "), stopeta.MaxReasonLength)
}

func cutoffOf(stop *volumegen.StopRecord) int64 {
	if stop.WindowEnd > 0 {
		return stop.WindowEnd
	}
	return stop.WindowStart
}

func roadMiles(lat, lon float64, to *volumegen.CityRecord) float64 {
	from := &volumegen.CityRecord{Lat: lat, Lon: lon}
	return math.Round(volumegen.HaversineMiles(from, to)*1.18*10) / 10
}

func floorMinutes(seconds int64) int64 {
	if seconds >= 0 {
		return seconds / volumegen.Minute
	}
	return -((-seconds + volumegen.Minute - 1) / volumegen.Minute)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "one " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

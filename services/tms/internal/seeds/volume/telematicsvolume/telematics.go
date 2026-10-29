package telematicsvolume

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	tmProvider       = "Samsara"
	tmChunk          = 10_000
	tmHourMs         = int64(3_600_000)
	tmMetersPerMile  = 1609.344
	tmRulesetCycle   = "USA 70 hour / 8 day"
	tmRulesetShift   = "US Interstate Property"
	tmRulesetRestart = "34-hour Restart"
	tmRulesetBreak   = "Property (off-duty/sleeper)"
	tmRulesetJuris   = "US"
)

var tmViolationTypes = []struct {
	kind        string
	description string
	weight      int
}{
	{"shiftDrivingHours", "Driving beyond the 11 hour driving limit", 40},
	{"shiftHours", "Driving beyond the 14 hour on-duty window", 30},
	{"restbreakMissed", "30 minute break not taken within 8 hours of driving", 22},
	{"cycleHoursOn", "Driving beyond the 70 hour / 8 day cycle", 8},
}

type tmMove struct {
	rec  *volumegen.ShipmentRecord
	move *volumegen.MoveRecord
}

type tmTractor struct {
	id        pulid.ID
	vehicleID string
	crew      *volumegen.Crew
	home      *volumegen.CityRecord
	moves     []tmMove
	state     tmState
}

type tmState struct {
	lat, lon   float64
	heading    float64
	speed      float64
	engine     telematics.EngineState
	place      string
	recordedAt int64
	duty       telematics.DutyStatus
}

type tmBuilder struct {
	g        *volumegen.Generator
	now      int64
	tractors []*tmTractor
	byID     map[pulid.ID]*tmTractor
	zones    map[string]*time.Location
	counts   map[string]int
}

type tmPhase struct {
	*volumegen.Generator
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&tmPhase{Generator: g}).seed(ctx)
}

func (g *tmPhase) seed(ctx context.Context) error {
	b := &tmBuilder{
		g:      g.Generator,
		now:    volumegen.Unix(g.Now),
		byID:   make(map[pulid.ID]*tmTractor),
		zones:  make(map[string]*time.Location),
		counts: make(map[string]int),
	}
	b.indexFleet()
	if len(b.tractors) == 0 {
		return nil
	}
	for _, t := range b.tractors {
		b.locate(t)
	}

	if err := g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if err := b.writePositions(ctx, tx); err != nil {
			return err
		}
		if err := b.writeHOS(ctx, tx); err != nil {
			return err
		}
		return b.writeFeedState(ctx, tx)
	}); err != nil {
		return err
	}
	if err := b.writeViolations(ctx); err != nil {
		return err
	}
	if err := b.writeEvents(ctx); err != nil {
		return err
	}

	for _, entity := range []struct{ table, label string }{
		{"telematics_vehicle_positions", "Telematics vehicle positions"},
		{"worker_hos_states", "Driver HOS states"},
		{"worker_hos_violations", "Driver HOS violations"},
		{"telematics_events", "Telematics events"},
	} {
		g.Report.Add(entity.label, b.counts[entity.table])
	}

	return nil
}

func (b *tmBuilder) indexFleet() {
	f := b.g.Fake
	homes := make(map[pulid.ID]*volumegen.CityRecord, len(b.g.Drivers))
	for _, d := range b.g.Drivers {
		homes[d.WorkerID] = d.Home
	}
	for _, c := range b.g.Crews {
		if _, ok := b.byID[c.TractorID]; ok {
			continue
		}
		t := &tmTractor{
			id:        c.TractorID,
			vehicleID: "28" + f.Digits(13),
			crew:      c,
			home:      homes[c.WorkerID],
		}
		if t.home == nil {
			t.home = f.City()
		}
		b.tractors = append(b.tractors, t)
		b.byID[t.id] = t
	}
	for _, rec := range b.g.Shipments {
		for _, move := range rec.Moves {
			if move.Crew == nil || len(move.Stops) == 0 {
				continue
			}
			if t, ok := b.byID[move.Crew.TractorID]; ok {
				t.moves = append(t.moves, tmMove{rec: rec, move: move})
			}
		}
	}
}

func (b *tmBuilder) locate(t *tmTractor) {
	f := b.g.Fake
	var active, last *tmMove
	for i := range t.moves {
		m := &t.moves[i]
		switch {
		case m.move.Status == shipmentvocab.MoveStatusInTransit && m.move.StartedAt > 0:
			if active == nil || m.move.StartedAt > active.move.StartedAt {
				active = m
			}
		case m.move.CompletedAt > 0 && m.move.CompletedAt <= b.now:
			if last == nil || m.move.CompletedAt > last.move.CompletedAt {
				last = m
			}
		}
	}

	switch {
	case active != nil:
		b.enRoute(t, active.move)
	case last != nil:
		stop := last.move.Stops[len(last.move.Stops)-1]
		lat, lon := f.Jitter(stop.Location.City)
		t.state = tmState{
			lat:     lat,
			lon:     lon,
			heading: float64(f.Intn(360)),
			engine: volumegen.Pick(
				f,
				[]telematics.EngineState{telematics.EngineStateOff, telematics.EngineStateIdle},
			),
			place:      placeOf(stop.Location.City),
			recordedAt: b.now - f.Between64(volumegen.Minute, 50*volumegen.Minute),
			duty: volumegen.Pick(
				f,
				[]telematics.DutyStatus{
					telematics.DutyStatusOffDuty,
					telematics.DutyStatusSleeperBed,
				},
			),
		}
	default:
		lat, lon := f.Jitter(t.home)
		t.state = tmState{
			lat:        lat,
			lon:        lon,
			heading:    float64(f.Intn(360)),
			engine:     telematics.EngineStateOff,
			place:      placeOf(t.home),
			recordedAt: b.now - f.Between64(5*volumegen.Minute, 3*volumegen.Day),
			duty:       telematics.DutyStatusOffDuty,
		}
	}
}

func (b *tmBuilder) enRoute(t *tmTractor, move *volumegen.MoveRecord) {
	f := b.g.Fake
	departed := -1
	for i, stop := range move.Stops {
		if stop.Arrived() && stop.ActualDeparture == 0 {
			lat, lon := f.Jitter(stop.Location.City)
			t.state = tmState{
				lat:        lat,
				lon:        lon,
				heading:    float64(f.Intn(360)),
				engine:     telematics.EngineStateIdle,
				place:      placeOf(stop.Location.City),
				recordedAt: b.now - f.Between64(30, 10*volumegen.Minute),
				duty:       telematics.DutyStatusOnDuty,
			}
			return
		}
		if stop.ActualDeparture > 0 {
			departed = i
		}
	}

	if departed < 0 || departed+1 >= len(move.Stops) {
		stop := move.Stops[0]
		lat, lon := f.Jitter(stop.Location.City)
		t.state = tmState{
			lat: lat, lon: lon, heading: float64(f.Intn(360)),
			engine: telematics.EngineStateOn, speed: float64(f.Between(48, 66)),
			place:      placeOf(stop.Location.City),
			recordedAt: b.now - f.Between64(15, 8*volumegen.Minute),
			duty:       telematics.DutyStatusDriving,
		}
		return
	}

	from, to := move.Stops[departed], move.Stops[departed+1]
	drive := math.Max(to.Miles/50*float64(volumegen.Hour), float64(volumegen.Hour))
	share := math.Min(math.Max(float64(b.now-from.ActualDeparture)/drive, 0.05), 0.95)
	a, c := from.Location.City, to.Location.City
	lat := a.Lat + (c.Lat-a.Lat)*share
	lon := a.Lon + (c.Lon-a.Lon)*share
	near := volumegen.NearestCity(a, c, share)
	t.state = tmState{
		lat:        lat,
		lon:        lon,
		heading:    bearing(a, c),
		engine:     telematics.EngineStateOn,
		speed:      float64(f.Between(48, 68)),
		place:      fmt.Sprintf("%d mi from %s", f.Between(2, 30), placeOf(near)),
		recordedAt: b.now - f.Between64(15, 8*volumegen.Minute),
		duty:       telematics.DutyStatusDriving,
	}
	if f.Chance(0.12) {
		t.state.speed = 0
		t.state.engine = telematics.EngineStateIdle
		t.state.duty = telematics.DutyStatusOnDuty
	}
}

func (b *tmBuilder) writePositions(ctx context.Context, tx *keel.DB) error {
	f := b.g.Fake
	rows := make([]*telematics.VehiclePosition, 0, len(b.tractors))
	for _, t := range b.tractors {
		s := t.state
		odometer := int64(float64(f.Between(80_000, 900_000)) * tmMetersPerMile)
		fuel := math.Round((15+f.Float()*85)*10) / 10
		rows = append(rows, &telematics.VehiclePosition{
			OrganizationID:    b.g.Refs.OrgID,
			BusinessUnitID:    b.g.Refs.BuID,
			TractorID:         t.id,
			Provider:          tmProvider,
			ProviderVehicleID: t.vehicleID,
			Latitude:          math.Round(s.lat*1e6) / 1e6,
			Longitude:         math.Round(s.lon*1e6) / 1e6,
			HeadingDegrees:    s.heading,
			SpeedMph:          s.speed,
			EngineState:       s.engine,
			FuelPercent:       volumegen.Ptr(fuel),
			OdometerMeters:    volumegen.Ptr(odometer),
			FormattedLocation: s.place,
			RecordedAt:        s.recordedAt,
			ReceivedAt:        min(s.recordedAt+f.Between64(2, 40), b.now),
		})
	}
	if err := volumegen.InsertRows(
		ctx,
		tx,
		telematicsdb.VehiclePositions.Handle,
		rows,
	); err != nil {
		return err
	}
	b.counts["telematics_vehicle_positions"] += len(rows)

	return nil
}

func (b *tmBuilder) writeHOS(ctx context.Context, tx *keel.DB) error {
	f := b.g.Fake
	rows := make([]*telematics.WorkerHOSState, 0, len(b.tractors)*11/10)
	add := func(t *tmTractor, workerID pulid.ID, duty telematics.DutyStatus) {
		drive := f.Between64(0, 11) * tmHourMs
		shift := min(drive+f.Between64(0, 3)*tmHourMs, 14*tmHourMs)
		cycle := f.Between64(8, 70) * tmHourMs
		if duty == telematics.DutyStatusOffDuty || duty == telematics.DutyStatusSleeperBed {
			drive, shift = 11*tmHourMs, 14*tmHourMs
		}
		cycleStart := b.now - f.Between64(1, 7)*volumegen.Day
		state := &telematics.WorkerHOSState{
			OrganizationID:      b.g.Refs.OrgID,
			BusinessUnitID:      b.g.Refs.BuID,
			WorkerID:            workerID,
			Provider:            tmProvider,
			ProviderDriverID:    f.Digits(8),
			DutyStatus:          duty,
			DriveRemainingMs:    drive,
			ShiftRemainingMs:    shift,
			CycleRemainingMs:    cycle,
			CycleTomorrowMs:     min(cycle+f.Between64(0, 10)*tmHourMs, 70*tmHourMs),
			BreakRemainingMs:    f.Between64(0, 8) * tmHourMs,
			CycleStartedAt:      volumegen.Ptr(cycleStart),
			CurrentVehicleID:    t.vehicleID,
			RulesetCycle:        tmRulesetCycle,
			RulesetShift:        tmRulesetShift,
			RulesetRestart:      tmRulesetRestart,
			RulesetBreak:        tmRulesetBreak,
			RulesetJurisdiction: tmRulesetJuris,
			CurrentTractorID:    t.id,
			RecordedAt:          t.state.recordedAt,
			ReceivedAt:          min(t.state.recordedAt+f.Between64(2, 40), b.now),
		}
		rows = append(rows, state)
	}
	for _, t := range b.tractors {
		add(t, t.crew.WorkerID, t.state.duty)
		if t.crew.SecondaryWorkerID != nil {
			mate := telematics.DutyStatusSleeperBed
			if t.state.duty != telematics.DutyStatusDriving {
				mate = telematics.DutyStatusOffDuty
			}
			add(t, *t.crew.SecondaryWorkerID, mate)
		}
	}
	if err := volumegen.InsertRows(ctx, tx, telematicsdb.WorkerHOSStates.Handle, rows); err != nil {
		return err
	}
	b.counts["worker_hos_states"] += len(rows)

	return nil
}

func (b *tmBuilder) writeFeedState(ctx context.Context, tx *keel.DB) error {
	fs := telematicsdb.FeedStates
	at := b.now - b.g.Fake.Between64(10, 90)
	if _, err := fs.Upsert(ctx, tx, []*telematics.FeedState{{
		OrganizationID: b.g.Refs.OrgID,
		BusinessUnitID: b.g.Refs.BuID,
		Provider:       tmProvider,
		FeedType:       telematics.FeedTypeVehicleStats,
		Cursor:         fmt.Sprintf("%x", at),
		LastPolledAt:   at,
		LastSuccessAt:  at,
	}}, keel.OnConflict(fs.OrganizationID, fs.BusinessUnitID, fs.Provider, fs.FeedType).
		DoNothing()); err != nil {
		return fmt.Errorf("upsert telematics feed state: %w", err)
	}

	return nil
}

func (b *tmBuilder) writeViolations(ctx context.Context) error {
	f := b.g.Fake
	weights := make([]int, len(tmViolationTypes))
	for i, v := range tmViolationTypes {
		weights[i] = v.weight
	}
	start := volumegen.Unix(b.g.Start)
	months := max(b.g.Opts.HistoryMonths, 1)
	rows := make([]*telematics.WorkerHOSViolation, 0, len(b.tractors)*months/2)
	seen := make(map[string]struct{})
	for _, t := range b.tractors {
		zone := b.zone(t.home)
		for m := range months {
			for range f.Weighted([]int{55, 30, 12, 3}) {
				at := start + int64(m)*30*volumegen.Day + f.Between64(0, 30*volumegen.Day)
				if at >= b.now-volumegen.Hour {
					continue
				}
				v := tmViolationTypes[f.Weighted(weights)]
				key := fmt.Sprintf("%s|%s|%d", t.crew.WorkerID, v.kind, at)
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				local := time.Unix(at, 0).In(zone)
				dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone).
					Unix()
				duration := f.Between64(2, 95) * volumegen.Minute
				rows = append(rows, &telematics.WorkerHOSViolation{
					OrganizationID:   b.g.Refs.OrgID,
					BusinessUnitID:   b.g.Refs.BuID,
					WorkerID:         t.crew.WorkerID,
					ViolationType:    v.kind,
					ViolationStartAt: at,
					Description:      v.description,
					DurationMs:       duration * 1000,
					DayStartAt:       volumegen.Ptr(dayStart),
					DayEndAt:         volumegen.Ptr(dayStart + volumegen.Day - 1),
					DetectedAt: min(
						at+duration+f.Between64(volumegen.Minute, 2*volumegen.Hour),
						b.now,
					),
				})
			}
		}
	}

	for begin := 0; begin < len(rows); begin += tmChunk {
		chunk := rows[begin:min(begin+tmChunk, len(rows))]
		if err := b.g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			return volumegen.InsertRows(ctx, tx, telematicsdb.WorkerHOSViolations.Handle, chunk)
		}); err != nil {
			return err
		}
	}
	b.counts["worker_hos_violations"] += len(rows)

	return nil
}

func (b *tmBuilder) zone(c *volumegen.CityRecord) *time.Location {
	if loc, ok := b.zones[c.Timezone]; ok {
		return loc
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		loc = time.UTC
	}
	b.zones[c.Timezone] = loc
	return loc
}

func placeOf(c *volumegen.CityRecord) string {
	return c.Name + ", " + c.State
}

func bearing(a, b *volumegen.CityRecord) float64 {
	lat1, lat2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dLon := (b.Lon - a.Lon) * math.Pi / 180
	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	deg := math.Mod(math.Atan2(y, x)*180/math.Pi+360, 360)
	return math.Round(deg*10) / 10
}

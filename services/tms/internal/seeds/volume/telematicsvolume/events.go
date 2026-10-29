package telematicsvolume

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/google/uuid"
)

const (
	tmDuplicateRate = 0.04
	tmUnmatchedRate = 0.012
	tmRefusedOpen   = 0.1
	tmRefusedClosed = 0.005
	tmAlertRate     = 0.1
	tmReasonRefused = "An earlier stop on the move is still open, so this visit was not recorded on its stop."
	tmReasonNoStop  = "No stop on the truck's current move is at this geofence."
)

var tmAlerts = []string{
	"HarshBrake",
	"HarshAcceleration",
	"Speeding",
	"HarshTurn",
	"CollisionWarning",
}

type tmEventDraft struct {
	t        *tmTractor
	move     *volumegen.MoveRecord
	kind     telematics.EventType
	at       int64
	loc      *volumegen.LocationRecord
	stop     *volumegen.StopRecord
	visit    shipmentvocab.VisitKind
	outcome  telematics.StopOutcome
	reason   string
	alert    string
	severity string
}

func (b *tmBuilder) writeEvents(ctx context.Context) error {
	rows := make([]*telematics.TelematicsEvent, 0, len(b.g.Shipments)*5)
	for _, t := range b.tractors {
		for _, m := range t.moves {
			rows = b.moveEvents(rows, t, m.move)
		}
	}

	for begin := 0; begin < len(rows); begin += tmChunk {
		end := min(begin+tmChunk, len(rows))
		chunk := rows[begin:end]
		if err := b.g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			return volumegen.InsertRows(ctx, tx, telematicsdb.TelematicsEvents.Handle, chunk)
		}); err != nil {
			return err
		}
		b.g.Progress("telematics events", end, len(rows))
	}
	b.counts["telematics_events"] += len(rows)

	return nil
}

func (b *tmBuilder) moveEvents(
	rows []*telematics.TelematicsEvent,
	t *tmTractor,
	move *volumegen.MoveRecord,
) []*telematics.TelematicsEvent {
	f := b.g.Fake
	open := move.Status != shipmentvocab.MoveStatusCompleted &&
		move.Status != shipmentvocab.MoveStatusCanceled
	waiting := -1
	for i, stop := range move.Stops {
		if !stop.Arrived() {
			if waiting < 0 {
				waiting = i
			}
			continue
		}
		rows = b.visit(rows, t, move, stop, shipmentvocab.VisitArrival, stop.ActualArrival)
		if stop.ActualDeparture > 0 {
			rows = b.visit(rows, t, move, stop, shipmentvocab.VisitDeparture, stop.ActualDeparture)
		}
	}

	if move.StartedAt <= 0 {
		return rows
	}
	until := b.now
	if move.CompletedAt > 0 {
		until = move.CompletedAt
	}
	during := func() int64 {
		return f.Between64(move.StartedAt, max(until-volumegen.Minute, move.StartedAt))
	}

	if stop, at, ok := b.refusal(move, open, waiting, during); ok {
		rows = append(rows, b.event(&tmEventDraft{
			t:       t,
			move:    move,
			kind:    telematics.EventTypeGeofenceEntry,
			at:      at,
			loc:     stop.Location,
			stop:    stop,
			visit:   shipmentvocab.VisitArrival,
			outcome: telematics.StopOutcomeRefused,
			reason:  tmReasonRefused,
		}))
	}
	if f.Chance(tmUnmatchedRate) && len(b.g.Locations) > 0 {
		rows = append(rows, b.event(&tmEventDraft{
			t:       t,
			move:    move,
			kind:    telematics.EventTypeGeofenceEntry,
			at:      during(),
			loc:     volumegen.Pick(f, b.g.Locations),
			visit:   shipmentvocab.VisitArrival,
			outcome: telematics.StopOutcomeUnmatched,
			reason:  tmReasonNoStop,
		}))
	}
	if f.Chance(tmAlertRate) {
		rows = append(rows, b.event(&tmEventDraft{
			t:        t,
			move:     move,
			kind:     telematics.EventTypeAlertIncident,
			at:       during(),
			alert:    volumegen.Pick(f, tmAlerts),
			severity: volumegen.Pick(f, []string{"low", "low", "medium", "high"}),
		}))
	}

	return rows
}

func (b *tmBuilder) refusal(
	move *volumegen.MoveRecord,
	open bool,
	waiting int,
	during func() int64,
) (*volumegen.StopRecord, int64, bool) {
	f := b.g.Fake
	switch {
	case open && waiting >= 0:
		if !f.Chance(tmRefusedOpen) {
			return nil, 0, false
		}
		return move.Stops[min(waiting+1, len(move.Stops)-1)], during(), true
	case !open && len(move.Stops) > 1:
		if !f.Chance(tmRefusedClosed) {
			return nil, 0, false
		}
		stop := move.Stops[f.Between(1, len(move.Stops)-1)]
		if !stop.Arrived() {
			return nil, 0, false
		}
		at := max(
			stop.ActualArrival-f.Between64(5*volumegen.Minute, 2*volumegen.Hour),
			move.StartedAt,
		)
		return stop, at, true
	default:
		return nil, 0, false
	}
}

func (b *tmBuilder) visit(
	rows []*telematics.TelematicsEvent,
	t *tmTractor,
	move *volumegen.MoveRecord,
	stop *volumegen.StopRecord,
	visit shipmentvocab.VisitKind,
	at int64,
) []*telematics.TelematicsEvent {
	kind := telematics.EventTypeGeofenceEntry
	if visit == shipmentvocab.VisitDeparture {
		kind = telematics.EventTypeGeofenceExit
	}
	d := &tmEventDraft{
		t:       t,
		move:    move,
		kind:    kind,
		at:      at,
		loc:     stop.Location,
		stop:    stop,
		visit:   visit,
		outcome: telematics.StopOutcomeRecorded,
	}
	rows = append(rows, b.event(d))
	if b.g.Fake.Chance(tmDuplicateRate) {
		dup := *d
		dup.at = min(at+b.g.Fake.Between64(20, 4*volumegen.Minute), b.now)
		dup.outcome = telematics.StopOutcomeDuplicate
		rows = append(rows, b.event(&dup))
	}

	return rows
}

func (b *tmBuilder) event(d *tmEventDraft) *telematics.TelematicsEvent {
	f := b.g.Fake
	id := volumegen.IDAt("tlev_", d.at)
	payload := map[string]any{
		"eventType":      string(d.kind),
		"happenedAtTime": time.Unix(d.at, 0).UTC().Format(time.RFC3339),
		"vehicle": map[string]any{
			"id":   d.t.vehicleID,
			"name": d.t.vehicleID[len(d.t.vehicleID)-4:],
		},
	}
	e := &telematics.TelematicsEvent{
		ID:             id,
		OrganizationID: b.g.Refs.OrgID,
		BusinessUnitID: b.g.Refs.BuID,
		Provider:       tmProvider,
		EventID:        uuid.NewSHA1(uuid.NameSpaceOID, []byte(id.String())).String(),
		EventType:      d.kind,
		OccurredAt:     d.at,
		TractorID:      d.t.id,
		WorkerID:       d.t.crew.WorkerID,
		CreatedAt:      min(d.at+f.Between64(1, 45), b.now),
	}
	if d.loc != nil {
		e.LocationID = d.loc.ID
		e.AddressName = d.loc.Name
		payload["address"] = map[string]any{
			"name":        d.loc.Name,
			"externalIds": map[string]any{"trenovaLocationId": d.loc.ID.String()},
		}
	}
	if d.alert != "" {
		payload["alertType"] = d.alert
		payload["severity"] = d.severity
	}
	if d.outcome != "" {
		e.StopOutcome = d.outcome
		e.StopVisit = d.visit
		e.ShipmentMoveID = d.move.ID
		e.StopOutcomeReason = d.reason
		if d.stop != nil {
			e.StopID = d.stop.ID
		}
	}
	e.Payload = payload

	return e
}

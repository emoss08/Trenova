// Package agentwait is work an agent parked until something happens in the
// world: a truck reaching a stop, a reply arriving, an appointment coming
// round, free time running out, a driver's hours running low, or a time.
package agentwait

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	IDPrefix            = "awt_"
	MaxDescriptionRunes = 200
	MaxThenRunes        = 1000
	MaxOutcomeRunes     = 1000
	MaxMinutesBefore    = 24 * 60
	MaxDriveMinutes     = 11 * 60
	MinLifetimeSeconds  = 60
	MaxLifetimeSeconds  = 7 * 24 * 60 * 60
	DefaultLifetimeSecs = 24 * 60 * 60
	MaxOpenPerOwner     = 10
	tableName           = "agent_waits"
)

type Kind string

const (
	KindTime            = Kind("Time")
	KindStopArrival     = Kind("StopArrival")
	KindStopDeparture   = Kind("StopDeparture")
	KindReply           = Kind("Reply")
	KindAppointmentNear = Kind("AppointmentNear")
	KindFreeTimeEnding  = Kind("FreeTimeEnding")
	KindHOSDriveBelow   = Kind("HOSDriveBelow")
)

func AllKinds() []Kind {
	return []Kind{
		KindTime,
		KindStopArrival,
		KindStopDeparture,
		KindReply,
		KindAppointmentNear,
		KindFreeTimeEnding,
		KindHOSDriveBelow,
	}
}

func (k Kind) IsValid() bool {
	switch k {
	case KindTime, KindStopArrival, KindStopDeparture, KindReply,
		KindAppointmentNear, KindFreeTimeEnding, KindHOSDriveBelow:
		return true
	default:
		return false
	}
}

// Timed reports a wait that comes due at a time it can work out: a timer
// wakes it, and it reads the record again before it decides. Drive time is
// one: while the driver drives, the clock runs down a second a second, so the
// moment it crosses is known ahead of the next poll.
func (k Kind) Timed() bool {
	return k == KindTime || k == KindAppointmentNear || k == KindFreeTimeEnding ||
		k == KindHOSDriveBelow
}

// Visit reports a wait on a truck reaching or leaving a stop.
func (k Kind) Visit() bool {
	return k == KindStopArrival || k == KindStopDeparture
}

type Status string

const (
	StatusWaiting   = Status("Waiting")
	StatusMet       = Status("Met")
	StatusTimedOut  = Status("TimedOut")
	StatusCancelled = Status("Cancelled")
	StatusFailed    = Status("Failed")
)

func AllStatuses() []Status {
	return []Status{StatusWaiting, StatusMet, StatusTimedOut, StatusCancelled, StatusFailed}
}

func (s Status) Open() bool { return s == StatusWaiting }

// Condition is what the wait is for. Which fields it holds depends on the
// kind; Validate says which.
type Condition struct {
	At                    int64    `json:"at,omitempty"`
	ShipmentMoveID        pulid.ID `json:"shipmentMoveId,omitempty"`
	StopID                pulid.ID `json:"stopId,omitempty"`
	ShipmentID            pulid.ID `json:"shipmentId,omitempty"`
	CarrierID             pulid.ID `json:"carrierId,omitempty"`
	CustomerID            pulid.ID `json:"customerId,omitempty"`
	WorkerID              pulid.ID `json:"workerId,omitempty"`
	DetentionOccurrenceID pulid.ID `json:"detentionOccurrenceId,omitempty"`
	MinutesBefore         int      `json:"minutesBefore,omitempty"`
	DriveMinutesBelow     int      `json:"driveMinutesBelow,omitempty"`
	// SeenInsideAt is when the truck was last seen inside the stop's geofence,
	// for a departure watched from polled positions: leaving is being outside
	// after being inside.
	SeenInsideAt int64 `json:"seenInsideAt,omitempty"`
}

var (
	_ bun.BeforeAppendModelHook          = (*Wait)(nil)
	_ validationframework.TenantedEntity = (*Wait)(nil)
)

type Wait struct {
	bun.BaseModel `bun:"table:agent_waits,alias:awt" json:"-"`

	ID                pulid.ID          `json:"id"                    bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID    pulid.ID          `json:"organizationId"        bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"        bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Kind              Kind              `json:"kind"                  bun:"kind,type:VARCHAR(50),notnull"`
	Condition         *Condition        `json:"condition"             bun:"condition,type:JSONB,notnull"`
	WatchID           pulid.ID          `json:"watchId,omitempty"     bun:"watch_id,type:VARCHAR(100),nullzero"`
	Description       string            `json:"description"           bun:"description,type:TEXT,notnull"`
	Then              string            `json:"nextStep,omitempty"    bun:"then_note,type:TEXT,nullzero"`
	AgentDefinitionID pulid.ID          `json:"agentDefinitionId"     bun:"agent_definition_id,type:VARCHAR(100),notnull"`
	ThreadID          pulid.ID          `json:"threadId,omitempty"    bun:"thread_id,type:VARCHAR(100),nullzero"`
	UserID            pulid.ID          `json:"userId,omitempty"      bun:"user_id,type:VARCHAR(100),nullzero"`
	RunID             pulid.ID          `json:"runId,omitempty"       bun:"run_id,type:VARCHAR(100),nullzero"`
	SubjectType       agent.SubjectType `json:"subjectType,omitempty" bun:"subject_type,type:VARCHAR(50),nullzero"`
	SubjectID         pulid.ID          `json:"subjectId,omitempty"   bun:"subject_id,type:VARCHAR(100),nullzero"`
	// Taint is the outside content the work had read when it set the wait.
	// The work picks up with it, so a wait never launders what was read.
	Taint         *agent.RunTaint `json:"-"                       bun:"taint,type:JSONB,nullzero"`
	Status        Status          `json:"status"                  bun:"status,type:VARCHAR(50),notnull"`
	DueAt         *int64          `json:"dueAt,omitempty"         bun:"due_at,type:BIGINT,nullzero"`
	ExpiresAt     int64           `json:"expiresAt"               bun:"expires_at,type:BIGINT,notnull"`
	ResolvedAt    *int64          `json:"resolvedAt,omitempty"    bun:"resolved_at,type:BIGINT,nullzero"`
	Outcome       string          `json:"outcome,omitempty"       bun:"outcome,type:TEXT,nullzero"`
	ResumedTurnID pulid.ID        `json:"resumedTurnId,omitempty" bun:"resumed_turn_id,type:VARCHAR(100),nullzero"`
	ResumedRunID  pulid.ID        `json:"resumedRunId,omitempty"  bun:"resumed_run_id,type:VARCHAR(100),nullzero"`
	WorkflowID    string          `json:"-"                       bun:"workflow_id,type:VARCHAR(200),nullzero"`
	Version       int64           `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt     int64           `json:"createdAt"               bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt     int64           `json:"updatedAt"               bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
}

// Conversational reports a wait a conversation set, which picks up as a turn
// of that conversation. Any other was set by a background run and picks up as
// a run of the same agent on the same record.
func (w *Wait) Conversational() bool { return w.ThreadID.IsNotNil() }

func (w *Wait) Normalize() {
	w.Description = strings.TrimSpace(w.Description)
	w.Then = strings.TrimSpace(w.Then)
	if w.Condition == nil {
		w.Condition = &Condition{}
	}
	w.WatchID = w.Condition.watched(w.Kind)
}

// watched is the record whose change can satisfy the wait, when one can.
func (c *Condition) watched(kind Kind) pulid.ID {
	switch kind {
	case KindStopArrival, KindStopDeparture:
		return c.ShipmentMoveID
	case KindReply:
		switch {
		case c.ShipmentID.IsNotNil():
			return c.ShipmentID
		case c.CarrierID.IsNotNil():
			return c.CarrierID
		default:
			return c.CustomerID
		}
	case KindAppointmentNear:
		return c.StopID
	case KindFreeTimeEnding:
		return c.DetentionOccurrenceID
	case KindHOSDriveBelow:
		return c.WorkerID
	case KindTime:
		return pulid.Nil
	default:
		return pulid.Nil
	}
}

func (w *Wait) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(w,
		validation.Field(&w.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&w.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&w.AgentDefinitionID, validation.Required.Error("Agent is required")),
	))
	if !w.Kind.IsValid() {
		multiErr.Add("until", errortypes.ErrInvalid, "Unknown kind of wait")
	}
	if w.ThreadID.IsNil() && (w.RunID.IsNil() || w.SubjectID.IsNil()) {
		multiErr.Add("owner", errortypes.ErrInvalid,
			"A wait belongs to a conversation, or to a run and the record it works on")
	}
	validateText(multiErr, "description", w.Description, MaxDescriptionRunes, true)
	validateText(multiErr, "then", w.Then, MaxThenRunes, false)
	if w.Condition != nil {
		w.Condition.validate(w.Kind, multiErr)
	}
}

func validateText(multiErr *errortypes.MultiError, field, value string, limit int, required bool) {
	switch {
	case required && value == "":
		multiErr.Add(field, errortypes.ErrRequired, "Say in a few words what the wait is for")
	case utf8.RuneCountInString(value) > limit:
		multiErr.Add(field, errortypes.ErrInvalid,
			fmt.Sprintf("Keep it to %d characters", limit))
	}
}

func (c *Condition) validate(kind Kind, multiErr *errortypes.MultiError) {
	require := func(field string, id pulid.ID) {
		if id.IsNil() {
			multiErr.Add(field, errortypes.ErrRequired, "This wait needs "+field)
		}
	}
	minutes := func(field string, value, limit int) {
		if value < 0 || value > limit {
			multiErr.Add(field, errortypes.ErrInvalid,
				fmt.Sprintf("Use a number of minutes from 0 to %d", limit))
		}
	}

	switch kind {
	case KindTime:
		if c.At <= 0 {
			multiErr.Add("at", errortypes.ErrRequired, "Say when to pick the work up")
		}
	case KindStopArrival, KindStopDeparture:
		require("shipmentMoveId", c.ShipmentMoveID)
	case KindReply:
		named := 0
		for _, id := range []pulid.ID{c.ShipmentID, c.CarrierID, c.CustomerID} {
			if id.IsNotNil() {
				named++
			}
		}
		if named != 1 {
			multiErr.Add("shipmentId", errortypes.ErrInvalid,
				"Name exactly one of shipmentId, carrierId or customerId to wait for a reply about")
		}
	case KindAppointmentNear:
		require("shipmentMoveId", c.ShipmentMoveID)
		require("stopId", c.StopID)
		minutes("minutesBefore", c.MinutesBefore, MaxMinutesBefore)
	case KindFreeTimeEnding:
		require("detentionOccurrenceId", c.DetentionOccurrenceID)
		minutes("minutesBefore", c.MinutesBefore, MaxMinutesBefore)
	case KindHOSDriveBelow:
		require("workerId", c.WorkerID)
		if c.DriveMinutesBelow < 1 || c.DriveMinutesBelow > MaxDriveMinutes {
			multiErr.Add("driveHoursBelow", errortypes.ErrInvalid,
				"Use a number of drive hours above 0 and at most 11")
		}
	}
}

// ResumeNote is what the agent reads when it picks the work up: what it
// waited for, what came of it, and what it said it would do then. It is the
// system's account, not the person's words.
func (w *Wait) ResumeNote() string {
	var b strings.Builder
	b.WriteString("[Notice from the system, not from the person: a wait you set has ended.]\n")
	b.WriteString("You waited for: ")
	b.WriteString(w.Description)
	b.WriteString("\n")
	if w.Status == StatusTimedOut {
		b.WriteString("It did not happen before the wait ran out. ")
	} else if w.Status == StatusMet {
		b.WriteString("It happened. ")
	}
	if w.Outcome != "" {
		b.WriteString(w.Outcome)
	}
	b.WriteString("\n")
	if w.Then != "" {
		b.WriteString("When you set it you planned to: ")
		b.WriteString(w.Then)
		b.WriteString("\n")
	}
	b.WriteString("Pick the work up from here. Read the records again before you act, and say " +
		"what you did. Wait id: ")
	b.WriteString(w.ID.String())

	return b.String()
}

func (w *Wait) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew(IDPrefix)
		}
		if w.CreatedAt == 0 {
			w.CreatedAt = now
		}
		w.UpdatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}

func (w *Wait) GetID() pulid.ID { return w.ID }

func (w *Wait) GetOrganizationID() pulid.ID { return w.OrganizationID }

func (w *Wait) GetBusinessUnitID() pulid.ID { return w.BusinessUnitID }

func (w *Wait) GetTableName() string { return tableName }

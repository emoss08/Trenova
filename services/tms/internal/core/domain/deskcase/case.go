package deskcase

import (
	"database/sql/driver"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
)

// State is where a case stands, worked out every time it is read from the
// record it is about, its open waits and its snooze. It is never stored: a
// payment that announces nothing still settles the invoice's case the next
// time anyone looks.
type State string

const (
	StateWorking = State("Working")
	StateWaiting = State("Waiting")
	StateSnoozed = State("Snoozed")
	StateSettled = State("Settled")
)

// WaitingOn says who a waiting case is waiting on. A reply from the carrier
// or the customer is the common one; anything else the agent set a wait for
// (a truck reaching a stop, a time) is an event.
type WaitingOn string

const (
	WaitingOnCarrier  = WaitingOn("Carrier")
	WaitingOnCustomer = WaitingOn("Customer")
	WaitingOnReply    = WaitingOn("Reply")
	WaitingOnEvent    = WaitingOn("Event")
)

// SnoozeAnchor is what a snooze follows. A snooze to the appointment follows
// that stop's window, and ends when the truck reaches it; a snooze to the ETA
// follows the shipment's estimate. Both are read again whenever the case is,
// so a moved appointment moves the snooze with it.
type SnoozeAnchor string

const (
	SnoozeTime        = SnoozeAnchor("Time")
	SnoozeAppointment = SnoozeAnchor("Appointment")
	SnoozeETA         = SnoozeAnchor("ETA")
)

func (a SnoozeAnchor) IsValid() bool {
	switch a {
	case SnoozeTime, SnoozeAppointment, SnoozeETA:
		return true
	default:
		return false
	}
}

// Value stores no anchor as NULL, which is what the column's check reads as
// a conversation that is not snoozed.
func (a SnoozeAnchor) Value() (driver.Value, error) {
	if a == "" {
		return nil, nil //nolint:nilnil // nil is a valid value for a driver.Value
	}

	return string(a), nil
}

func AllSnoozeAnchors() []SnoozeAnchor {
	return []SnoozeAnchor{SnoozeTime, SnoozeAppointment, SnoozeETA}
}

// Subjects are the kinds of record a conversation can be a case about.
var subjects = []agent.SubjectType{
	agent.SubjectShipment,
	agent.SubjectInvoice,
	agent.SubjectInvoiceDispute,
}

func Subjects() []agent.SubjectType {
	return subjects
}

// IsSubject reports whether a conversation about this kind of record is a
// case.
func IsSubject(subject agent.SubjectType) bool {
	return slices.Contains(subjects, subject)
}

// Ref names one record a case is about.
type Ref struct {
	Type agent.SubjectType
	ID   pulid.ID
}

// Record is what a case shows about its record: what it is called, where it
// stands, and whether it has closed. Closing is what settles the case: a
// shipment invoiced or cancelled, an invoice paid or voided, a dispute
// resolved or withdrawn.
type Record struct {
	Type   agent.SubjectType `json:"type"`
	ID     pulid.ID          `json:"id"`
	Label  string            `json:"label"`
	Status string            `json:"status"`
	Closed bool              `json:"closed"`
	// ClosedAs is how it closed: Invoiced, Canceled, Paid, Voided, or the
	// dispute's resolution.
	ClosedAs string `json:"closedAs,omitempty"`
	// InvoiceID is the invoice a dispute is on, which is where the dispute
	// is opened.
	InvoiceID pulid.ID `json:"invoiceId,omitempty"`
	// BillToID is the customer billed for the record, whose checklist
	// template applies: a shipment's bill-to when it has one.
	BillToID pulid.ID `json:"-"`
	// CustomerID and CarrierIDs are who a case on this record can wait on.
	CustomerID pulid.ID   `json:"customerId,omitempty"`
	CarrierIDs []pulid.ID `json:"carrierIds,omitempty"`
	// NextStopID and NextAppointmentAt are the shipment's next stop not yet
	// reached and the start of its window: what a snooze to the appointment
	// follows.
	NextStopID        pulid.ID `json:"-"`
	NextAppointmentAt *int64   `json:"-"`
}

const statusMissing = "Missing"

// Missing is the record a case names when it is gone: deleted, or not this
// organization's. The case stays readable and says so.
func Missing(ref Ref) *Record {
	return &Record{Type: ref.Type, ID: ref.ID, Status: statusMissing}
}

// Gone reports a record the case names that could not be read.
func (r *Record) Gone() bool {
	return r == nil || r.Status == statusMissing
}

// OpenWait is one open wait of the conversation, as much as the case needs
// to say who it is waiting on.
type OpenWait struct {
	Party WaitingOn
	DueAt *int64
}

// Snooze is the conversation's snooze as stored.
type Snooze struct {
	Until  *int64
	Anchor SnoozeAnchor
	StopID pulid.ID
}

func (s Snooze) Set() bool {
	return s.Until != nil && s.Anchor != ""
}

// Summary is what the Desk's rail shows of a case.
type Summary struct {
	State        State        `json:"state"`
	WaitingOn    WaitingOn    `json:"waitingOn,omitempty"`
	OpenWaits    int          `json:"openWaits"`
	NextWaitDue  *int64       `json:"nextWaitDue,omitempty"`
	SnoozedUntil *int64       `json:"snoozedUntil,omitempty"`
	SnoozeAnchor SnoozeAnchor `json:"snoozeAnchor,omitempty"`
	Record       *Record      `json:"record"`
}

// Inputs is everything a case's state is worked out from.
type Inputs struct {
	Record *Record
	Waits  []OpenWait
	Snooze Snooze
	// ETA is the shipment's estimated arrival, read only for a snooze that
	// follows it.
	ETA *int64
	Now int64
}

// Resolve works out a case's state. A closed record settles the case
// whatever else is true; a snooze still in the future holds it next; an open
// wait makes it waiting; anything else is being worked.
func Resolve(in *Inputs) *Summary {
	out := &Summary{Record: in.Record, OpenWaits: len(in.Waits)}
	out.WaitingOn, out.NextWaitDue = waitingOn(in.Waits)

	switch until := snoozeEnd(in); {
	case in.Record != nil && in.Record.Closed:
		out.State = StateSettled
	case until != nil && *until > in.Now:
		out.State = StateSnoozed
		out.SnoozedUntil = until
		out.SnoozeAnchor = in.Snooze.Anchor
	case len(in.Waits) > 0:
		out.State = StateWaiting
	default:
		out.State = StateWorking
	}

	return out
}

// snoozeEnd is when the snooze ends now, read against the record: an
// appointment snooze ends when its stop is reached and otherwise follows the
// stop's window; an ETA snooze follows the latest estimate, and keeps the
// time it was set to while there is none.
func snoozeEnd(in *Inputs) *int64 {
	if !in.Snooze.Set() {
		return nil
	}

	switch in.Snooze.Anchor {
	case SnoozeAppointment:
		if in.Record == nil || in.Snooze.StopID.IsNil() {
			return in.Snooze.Until
		}
		if in.Record.NextStopID != in.Snooze.StopID {
			return nil
		}
		if in.Record.NextAppointmentAt != nil {
			return in.Record.NextAppointmentAt
		}
	case SnoozeETA:
		if in.ETA != nil {
			return in.ETA
		}
	case SnoozeTime:
	}

	return in.Snooze.Until
}

// waitingOn names the party a case waits on: a carrier or customer reply
// before any other, then a reply about the record, then anything else. The
// soonest due time among the waits says when the next one comes due.
func waitingOn(waits []OpenWait) (best WaitingOn, due *int64) {
	if len(waits) == 0 {
		return "", nil
	}

	best = waits[0].Party
	for _, wait := range waits {
		if partyRank(wait.Party) < partyRank(best) {
			best = wait.Party
		}
		if wait.DueAt != nil && (due == nil || *wait.DueAt < *due) {
			due = wait.DueAt
		}
	}

	return best, due
}

func partyRank(party WaitingOn) int {
	switch party {
	case WaitingOnCustomer:
		return 0
	case WaitingOnCarrier:
		return 1
	case WaitingOnReply:
		return 2
	case WaitingOnEvent:
		return 3
	default:
		return 4
	}
}

// Party is someone a case can wait on: the record's customer or one of its
// carriers.
type Party struct {
	Kind WaitingOn `json:"kind"`
	ID   pulid.ID  `json:"id"`
	Name string    `json:"name"`
}

// View is a case as its conversation shows it: where it stands, what stands
// between the record and what comes next, and who it can wait on.
type View struct {
	Summary   *Summary   `json:"summary"`
	Checklist *Checklist `json:"checklist,omitempty"`
	Parties   []Party    `json:"parties"`
}

package recordcase

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
)

type Record struct {
	Kind     recordkind.Kind
	ID       pulid.ID
	Label    string
	Readable bool
}

type Deadline struct {
	At    int64
	Label string
	Note  string
}

type Head struct {
	References []string
	Title      string
	Status     string
	StatusTone Tone
	Details    []string
	Deadline   *Deadline
}

type TrustCell struct {
	Key  TrustKey
	Text string
}

type Finding struct {
	AgentDefinitionID pulid.ID
	FoundAt           int64
	Sources           int
	Raised            string
	Computed          string
	ReadingStatus     ReadingStatus
	Segments          []Segment
	Trust             []TrustCell
}

type RoutePoint struct {
	Label     string
	X         float64
	Y         float64
	Latitude  float64
	Longitude float64
	State     PointState
}

type RouteMarker struct {
	X         float64
	Y         float64
	Latitude  float64
	Longitude float64
}

type Route struct {
	Points      []RoutePoint
	Truck       *RouteMarker
	Callout     string
	CalloutTone Tone
}

type Meter struct {
	Label    string
	Value    string
	Fraction float64
	Tone     Tone
}

type Message struct {
	Quote      string
	From       string
	ReplyAgent pulid.ID
	Reply      string
	ReplySent  bool
}

type TimelineRow struct {
	Label    string
	Sub      string
	When     string
	Tone     Tone
	State    PointState
	Proposed bool
}

type Fact struct {
	Label string
	Value string
	At    *int64
	Tone  Tone
}

type Exhibit struct {
	Key      string
	Number   int
	Kind     ExhibitKind
	Title    string
	Source   string
	At       int64
	Read     bool
	Route    *Route
	Meters   []Meter
	Message  *Message
	Timeline []TimelineRow
	Facts    []Fact
	Note     string
	Path     string
	Claims   []string
}

type Decision struct {
	Kind              DecisionKind
	ID                pulid.ID
	Title             string
	Phase             DecisionPhase
	RunID             pulid.ID
	AgentDefinitionID pulid.ID
	ProposedAt        int64
	StepCount         int
	CompletedSteps    int
	FailedStep        *int
	FailureError      string
	CommitsAt         *int64
	DecidedAt         *int64
	DecidedByUserID   pulid.ID
	Note              string
	ItemID            pulid.ID
	Others            int
}

// Ownership is who a case belongs to and how its reader stands to it.
type Ownership struct {
	// OwnerID is nil while the case is unowned.
	OwnerID pulid.ID
	// Auto says the owner came from the record rather than a person's choice.
	Auto bool
	// Covering says the reader works the case in its owner's place.
	Covering  bool
	CanWork   bool
	CanClaim  bool
	CanAssign bool
}

// HandOff is the condition an agent can be asked to work when nobody has
// worked the case yet.
type HandOff struct {
	// ItemID is the Watchtower item that is handed to an agent.
	ItemID pulid.ID
	// Kind is what kind of condition it is, which decides what a person can
	// do about it themselves: a missing proof of delivery can be uploaded.
	Kind watchtower.SourceKind
	// Working says an agent has it and has not finished.
	Working bool
	// AgentDefinitionID is the agent that has it, or last had it.
	AgentDefinitionID pulid.ID
	// Note is what that agent said when it finished and proposed nothing;
	// markdown.
	Note string
	// Play is the first move this kind of condition has, which the person
	// sends themselves; empty when the kind has none.
	Play watchtowerplay.Kind
	// Request is the last play a person sent for it; nil when none has been.
	Request *watchtower.Request
}

// Away says a case was put away: its plan is done, or nothing more can be
// done for now, and what is still open about the record is snoozed. It comes
// back at Until, or sooner when something new is raised about the record.
type Away struct {
	Until int64
}

type Case struct {
	Record    Record
	Ownership Ownership
	// Away is nil while the case is in front of people.
	Away *Away
	// HandOff is nil when nothing open about the record can be given to an
	// agent to work.
	HandOff   *HandOff
	Head      Head
	Finding   *Finding
	Options   []*Option
	Chosen    *Option
	Drafting  bool
	DraftNote string
	Decision  *Decision
	Exhibits  []Exhibit
	Note      string
	Starters  []string
	Open      int
	Severity  watchtower.Severity
	UpdatedAt int64
	// Stream is the workflow whose stream carries what the agent on the case
	// is writing right now: the run it was handed to, or the one drafting the
	// chosen option's plan. Empty when no agent is working the case. It is
	// read by the case's stream endpoint and never sent to a client.
	Stream string
}

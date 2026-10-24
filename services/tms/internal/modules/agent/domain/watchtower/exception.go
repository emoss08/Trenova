package watchtower

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Status string

const (
	StatusOpen         = Status("Open")
	StatusAcknowledged = Status("Acknowledged")
	StatusInProgress   = Status("InProgress")
	StatusSnoozed      = Status("Snoozed")
	StatusResolved     = Status("Resolved")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusOpen, StatusAcknowledged, StatusInProgress, StatusSnoozed, StatusResolved:
		return true
	default:
		return false
	}
}

func (s Status) IsOpen() bool {
	return s.IsValid() && s != StatusResolved
}

func OpenStatuses() []Status {
	return []Status{StatusOpen, StatusAcknowledged, StatusInProgress, StatusSnoozed}
}

type ResolutionCode string

const (
	ResolutionFixed            = ResolutionCode("Fixed")
	ResolutionCarrierDelay     = ResolutionCode("CarrierDelay")
	ResolutionShipperDelay     = ResolutionCode("ShipperDelay")
	ResolutionReceiverDelay    = ResolutionCode("ReceiverDelay")
	ResolutionCustomerChange   = ResolutionCode("CustomerChange")
	ResolutionRescheduled      = ResolutionCode("Rescheduled")
	ResolutionWeather          = ResolutionCode("Weather")
	ResolutionEquipment        = ResolutionCode("Equipment")
	ResolutionDriverIssue      = ResolutionCode("DriverIssue")
	ResolutionDocumentReceived = ResolutionCode("DocumentReceived")
	ResolutionDataCorrected    = ResolutionCode("DataCorrected")
	ResolutionDuplicate        = ResolutionCode("Duplicate")
	ResolutionNoActionNeeded   = ResolutionCode("NoActionNeeded")
	ResolutionOther            = ResolutionCode("Other")
	ResolutionCleared          = ResolutionCode("Cleared")
)

func (c ResolutionCode) String() string { return string(c) }

func SelectableResolutionCodes() []ResolutionCode {
	return []ResolutionCode{
		ResolutionFixed,
		ResolutionCarrierDelay,
		ResolutionShipperDelay,
		ResolutionReceiverDelay,
		ResolutionCustomerChange,
		ResolutionRescheduled,
		ResolutionWeather,
		ResolutionEquipment,
		ResolutionDriverIssue,
		ResolutionDocumentReceived,
		ResolutionDataCorrected,
		ResolutionDuplicate,
		ResolutionNoActionNeeded,
		ResolutionOther,
	}
}

type ResolutionCodeOption struct {
	Code         ResolutionCode `json:"code"`
	Label        string         `json:"label"`
	RequiresNote bool           `json:"requiresNote"`
}

func SelectableResolutionCodeOptions() []*ResolutionCodeOption {
	codes := SelectableResolutionCodes()
	options := make([]*ResolutionCodeOption, 0, len(codes))
	for _, code := range codes {
		options = append(options, &ResolutionCodeOption{
			Code:         code,
			Label:        code.Label(),
			RequiresNote: code.RequiresNote(),
		})
	}
	return options
}

func (c ResolutionCode) IsValid() bool {
	return c == ResolutionCleared || c.IsSelectable()
}

func (c ResolutionCode) IsSelectable() bool {
	for _, code := range SelectableResolutionCodes() {
		if code == c {
			return true
		}
	}
	return false
}

func (c ResolutionCode) RequiresNote() bool {
	return c == ResolutionOther
}

func (c ResolutionCode) Label() string {
	switch c {
	case ResolutionFixed:
		return "Fixed"
	case ResolutionCarrierDelay:
		return "Carrier delay"
	case ResolutionShipperDelay:
		return "Shipper delay"
	case ResolutionReceiverDelay:
		return "Receiver delay"
	case ResolutionCustomerChange:
		return "Customer change"
	case ResolutionRescheduled:
		return "Rescheduled"
	case ResolutionWeather:
		return "Weather"
	case ResolutionEquipment:
		return "Equipment"
	case ResolutionDriverIssue:
		return "Driver issue"
	case ResolutionDocumentReceived:
		return "Document received"
	case ResolutionDataCorrected:
		return "Data corrected"
	case ResolutionDuplicate:
		return "Duplicate"
	case ResolutionNoActionNeeded:
		return "No action needed"
	case ResolutionOther:
		return "Other"
	case ResolutionCleared:
		return "Cleared by the source"
	default:
		return string(c)
	}
}

type EventKind string

const (
	EventCreated        = EventKind("Created")
	EventAssigned       = EventKind("Assigned")
	EventUnassigned     = EventKind("Unassigned")
	EventAcknowledged   = EventKind("Acknowledged")
	EventStarted        = EventKind("Started")
	EventSnoozed        = EventKind("Snoozed")
	EventWoke           = EventKind("Woke")
	EventDueChanged     = EventKind("DueChanged")
	EventCommented      = EventKind("Commented")
	EventEscalated      = EventKind("Escalated")
	EventResolved       = EventKind("Resolved")
	EventReopened       = EventKind("Reopened")
	EventSourceResolved = EventKind("SourceResolved")
	EventSourceReopened = EventKind("SourceReopened")
	EventHandedOff      = EventKind("HandedOff")
)

func (k EventKind) String() string { return string(k) }

func AllEventKinds() []EventKind {
	return []EventKind{
		EventCreated, EventAssigned, EventUnassigned, EventAcknowledged, EventStarted,
		EventSnoozed, EventWoke, EventDueChanged, EventCommented, EventEscalated,
		EventResolved, EventReopened, EventSourceResolved, EventSourceReopened,
		EventHandedOff,
	}
}

func (k EventKind) IsValid() bool {
	switch k {
	case EventCreated, EventAssigned, EventUnassigned, EventAcknowledged, EventStarted,
		EventSnoozed, EventWoke, EventDueChanged, EventCommented, EventEscalated,
		EventResolved, EventReopened, EventSourceResolved, EventSourceReopened,
		EventHandedOff:
		return true
	default:
		return false
	}
}

const (
	MaxNoteLength        = 2000
	MaxCommentLength     = 4000
	MaxSnoozeSeconds     = int64(30 * 24 * 3600)
	MaxDueAheadSeconds   = int64(90 * 24 * 3600)
	MaxEventValueLength  = 200
	DefaultEventPageSize = 100
	MaxEventPageSize     = 500
)

type Event struct {
	bun.BaseModel `bun:"table:watchtower_item_events,alias:wtie" json:"-"`
	keel.Table    `keel:"watchtower_item_events"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:wte_"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	ItemID         pulid.ID  `json:"itemId"         bun:"item_id,type:VARCHAR(100),notnull" keel:"item_id"`
	Kind           EventKind `json:"kind"           bun:"kind,type:VARCHAR(30),notnull" keel:"kind"`
	ActorID        pulid.ID  `json:"actorId"        bun:"actor_id,type:VARCHAR(100),nullzero" keel:"actor_id,nullzero"`
	FromValue      string    `json:"fromValue"      bun:"from_value,type:VARCHAR(200),nullzero" keel:"from_value,nullzero"`
	ToValue        string    `json:"toValue"        bun:"to_value,type:VARCHAR(200),nullzero" keel:"to_value,nullzero"`
	Note           string    `json:"note"           bun:"note,type:TEXT,nullzero" keel:"note,nullzero"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,type:BIGINT,notnull" keel:"created_at,autocreate"`
}

func NewEvent(item *Item, kind EventKind, actorID pulid.ID) *Event {
	return &Event{
		ID:             pulid.MustNew("wte_"),
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
		ItemID:         item.ID,
		Kind:           kind,
		ActorID:        actorID,
		CreatedAt:      timeutils.NowUnix(),
	}
}

func (e *Event) WithChange(from, to string) *Event {
	e.FromValue = truncateValue(from)
	e.ToValue = truncateValue(to)
	return e
}

func (e *Event) WithNote(note string) *Event {
	e.Note = strings.TrimSpace(note)
	return e
}

func (e *Event) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wte_")
		}
		if e.CreatedAt == 0 {
			e.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}

func truncateValue(value string) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= MaxEventValueLength {
		return value
	}
	return string([]rune(value)[:MaxEventValueLength])
}

func (i *Item) requireOpen() error {
	if i.IsResolved() {
		return errortypes.NewBusinessError("This exception is resolved; reopen it first")
	}
	return nil
}

func (i *Item) Acknowledge(now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if i.Status != StatusOpen {
		return errortypes.NewBusinessError("Only an open exception can be acknowledged")
	}
	i.Status = StatusAcknowledged
	i.AcknowledgedAt = &now
	return nil
}

func (i *Item) Start(now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if i.Status == StatusInProgress {
		return errortypes.NewBusinessError("This exception is already in progress")
	}
	if i.AcknowledgedAt == nil {
		i.AcknowledgedAt = &now
	}
	i.Status = StatusInProgress
	i.SnoozedUntil = nil
	return nil
}

func (i *Item) Snooze(until, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if until <= now {
		return errortypes.NewValidationError(
			"snoozedUntil",
			errortypes.ErrInvalid,
			"Snooze until a time in the future",
		)
	}
	if until-now > MaxSnoozeSeconds {
		return errortypes.NewValidationError(
			"snoozedUntil",
			errortypes.ErrInvalid,
			"An exception can be snoozed for at most 30 days",
		)
	}
	i.Status = StatusSnoozed
	i.SnoozedUntil = &until
	return nil
}

func (i *Item) Wake() bool {
	if i.Status != StatusSnoozed || i.IsResolved() {
		return false
	}
	i.SnoozedUntil = nil
	if i.AcknowledgedAt != nil {
		i.Status = StatusAcknowledged
	} else {
		i.Status = StatusOpen
	}
	return true
}

func (i *Item) Unsnooze() error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if !i.Wake() {
		return errortypes.NewBusinessError("This exception is not snoozed")
	}
	return nil
}

func (i *Item) SetDue(dueAt *int64, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if dueAt != nil {
		if *dueAt <= 0 {
			return errortypes.NewValidationError(
				"dueAt",
				errortypes.ErrInvalid,
				"Due time is invalid",
			)
		}
		if *dueAt-now > MaxDueAheadSeconds {
			return errortypes.NewValidationError(
				"dueAt",
				errortypes.ErrInvalid,
				"An exception can be due at most 90 days out",
			)
		}
	}
	i.DueAt = dueAt
	i.DueLabel = ""
	i.DueSource = DueSourcePerson
	return nil
}

func (i *Item) Assign(assigneeID pulid.ID) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if i.AssigneeID == assigneeID {
		return errortypes.NewBusinessError("This exception is already assigned that way")
	}
	i.AssigneeID = assigneeID
	return nil
}

func (i *Item) Resolve(code ResolutionCode, note string, actorID pulid.ID, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	if !code.IsSelectable() {
		return errortypes.NewValidationError(
			"resolutionCode",
			errortypes.ErrInvalid,
			"Choose a resolution code",
		)
	}
	note = strings.TrimSpace(note)
	if code.RequiresNote() && note == "" {
		return errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"Say how it was resolved",
		)
	}
	if len([]rune(note)) > MaxNoteLength {
		return errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrInvalid,
			"The note cannot be longer than 2000 characters",
		)
	}
	i.Status = StatusResolved
	i.ResolutionCode = code
	i.ResolutionNote = note
	i.ResolvedByID = actorID
	i.ResolvedAt = &now
	i.SnoozedUntil = nil
	i.SourceClearedAt = nil
	return nil
}

func (i *Item) Reopen() error {
	if !i.IsResolved() {
		return errortypes.NewBusinessError("This exception is not resolved")
	}
	i.Status = StatusOpen
	i.ResolvedAt = nil
	i.ResolutionCode = ""
	i.ResolutionNote = ""
	i.ResolvedByID = pulid.Nil
	i.SourceClearedAt = nil
	i.AcknowledgedAt = nil
	i.EscalationLevel = 0
	i.LastEscalatedAt = nil
	return nil
}

func ValidateComment(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", errortypes.NewValidationError("body", errortypes.ErrRequired, "Write a comment")
	}
	if len([]rune(body)) > MaxCommentLength {
		return "", errortypes.NewValidationError(
			"body",
			errortypes.ErrInvalid,
			"A comment cannot be longer than 4000 characters",
		)
	}
	return body, nil
}

const NoDueSortKey = int64(9_000_000_000_000)

func (i *Item) DueSortKey() int64 {
	if i.DueAt == nil {
		return NoDueSortKey
	}
	return *i.DueAt
}

func (i *Item) IsOverdue(now int64) bool {
	return !i.IsResolved() && i.Status != StatusSnoozed && i.DueAt != nil && *i.DueAt < now
}

func (s Severity) Raised() Severity {
	switch s {
	case SeverityInfo:
		return SeverityWarning
	case SeverityWarning, SeverityCritical:
		return SeverityCritical
	default:
		return SeverityCritical
	}
}

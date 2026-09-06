package worker

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var ErrInvalidEmploymentEventKind = errors.New("invalid employment event kind")

type EmploymentEventKind string

const (
	EmploymentEventHired          = EmploymentEventKind("Hired")
	EmploymentEventProbationEnded = EmploymentEventKind("ProbationEnded")
	EmploymentEventPromoted       = EmploymentEventKind("Promoted")
	EmploymentEventTransferred    = EmploymentEventKind("Transferred")
	EmploymentEventLeaveStarted   = EmploymentEventKind("LeaveStarted")
	EmploymentEventLeaveEnded     = EmploymentEventKind("LeaveEnded")
	EmploymentEventSuspended      = EmploymentEventKind("Suspended")
	EmploymentEventReinstated     = EmploymentEventKind("Reinstated")
	EmploymentEventTerminated     = EmploymentEventKind("Terminated")
	EmploymentEventRehired        = EmploymentEventKind("Rehired")
	EmploymentEventRateChanged    = EmploymentEventKind("RateChanged")
)

func (k EmploymentEventKind) String() string { return string(k) }

func (k EmploymentEventKind) IsValid() bool {
	switch k {
	case EmploymentEventHired, EmploymentEventProbationEnded, EmploymentEventPromoted,
		EmploymentEventTransferred, EmploymentEventLeaveStarted, EmploymentEventLeaveEnded,
		EmploymentEventSuspended, EmploymentEventReinstated, EmploymentEventTerminated,
		EmploymentEventRehired, EmploymentEventRateChanged:
		return true
	default:
		return false
	}
}

func EmploymentEventKindFromString(s string) (EmploymentEventKind, error) {
	kind := EmploymentEventKind(s)
	if !kind.IsValid() {
		return "", ErrInvalidEmploymentEventKind
	}
	return kind, nil
}

// EndsEmployment reports whether the event closes the worker's employment.
func (k EmploymentEventKind) EndsEmployment() bool {
	return k == EmploymentEventTerminated
}

// StartsEmployment reports whether the event opens (or reopens) employment.
func (k EmploymentEventKind) StartsEmployment() bool {
	return k == EmploymentEventHired || k == EmploymentEventRehired
}

// TakesOffDuty reports whether the event pulls the worker out of dispatch
// without ending employment; PutsOnDuty is its counterpart.
func (k EmploymentEventKind) TakesOffDuty() bool {
	return k == EmploymentEventSuspended || k == EmploymentEventLeaveStarted
}

func (k EmploymentEventKind) PutsOnDuty() bool {
	return k == EmploymentEventReinstated || k == EmploymentEventLeaveEnded
}

// RequiresReason lists the events an office must justify.
func (k EmploymentEventKind) RequiresReason() bool {
	switch k {
	case EmploymentEventTerminated, EmploymentEventSuspended, EmploymentEventLeaveStarted,
		EmploymentEventRateChanged:
		return true
	default:
		return false
	}
}

// Counterpart returns the event that must precede this one for it to make
// sense (LeaveEnded needs an open LeaveStarted, Reinstated an open Suspended).
func (k EmploymentEventKind) Counterpart() (EmploymentEventKind, bool) {
	switch k {
	case EmploymentEventLeaveEnded:
		return EmploymentEventLeaveStarted, true
	case EmploymentEventReinstated:
		return EmploymentEventSuspended, true
	default:
		return "", false
	}
}

// Value keys used inside FromValues / ToValues. They are stable identifiers the
// client renders with labels; new kinds add keys here rather than ad hoc.
const (
	EmploymentValueHireDate        = "hireDate"
	EmploymentValueTerminationDate = "terminationDate"
	EmploymentValueStatus          = "status"
	EmploymentValueFleetCodeID     = "fleetCodeId"
	EmploymentValueFleetCode       = "fleetCode"
	EmploymentValueManagerID       = "managerId"
	EmploymentValueManager         = "manager"
	EmploymentValueDriverType      = "driverType"
	EmploymentValueWorkerType      = "workerType"
	EmploymentValueRate            = "rate"
	EmploymentValueRateUnit        = "rateUnit"
	EmploymentValueCanBeAssigned   = "canBeAssigned"
	EmploymentValueLeaveType       = "leaveType"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerEmploymentEvent)(nil)
	_ validationframework.TenantedEntity = (*WorkerEmploymentEvent)(nil)
)

type WorkerEmploymentEvent struct {
	bun.BaseModel `bun:"table:worker_employment_events,alias:wee" json:"-"`

	ID             pulid.ID            `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID            `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	Kind           EmploymentEventKind `json:"kind"           bun:"kind,type:worker_employment_event_kind_enum,notnull"`
	EffectiveAt    int64               `json:"effectiveAt"    bun:"effective_at,type:BIGINT,notnull"`
	Reason         string              `json:"reason"         bun:"reason,type:VARCHAR(255),nullzero"`
	Notes          string              `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	FromValues     map[string]string   `json:"fromValues"     bun:"from_values,type:JSONB,notnull,default:'{}'"`
	ToValues       map[string]string   `json:"toValues"       bun:"to_values,type:JSONB,notnull,default:'{}'"`
	DocumentID     pulid.ID            `json:"documentId"     bun:"document_id,type:VARCHAR(100),nullzero"`
	RecordedByID   pulid.ID            `json:"recordedById"   bun:"recorded_by_id,type:VARCHAR(100),nullzero"`
	AmendedByID    pulid.ID            `json:"amendedById"    bun:"amended_by_id,type:VARCHAR(100),nullzero"`
	AmendedAt      *int64              `json:"amendedAt"      bun:"amended_at,type:BIGINT,nullzero"`
	AmendmentNote  string              `json:"amendmentNote"  bun:"amendment_note,type:VARCHAR(255),nullzero"`
	Version        int64               `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker     *Worker            `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document   *document.Document `json:"document,omitempty"   bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	RecordedBy *tenant.User       `json:"recordedBy,omitempty" bun:"rel:belongs-to,join:recorded_by_id=id"`
	AmendedBy  *tenant.User       `json:"amendedBy,omitempty"  bun:"rel:belongs-to,join:amended_by_id=id"`
}

func (e *WorkerEmploymentEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&e.Kind,
			validation.Required.Error("Event kind is required"),
			domainvalidation.ValidEnum[EmploymentEventKind]("Unknown employment event kind"),
		),
		validation.Field(&e.EffectiveAt,
			validation.Required.Error("Effective date is required"),
			validation.Min(int64(1)).Error("Effective date must be a valid date"),
		),
		validation.Field(&e.Reason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
		validation.Field(&e.AmendmentNote,
			validation.Length(0, 255).Error("Amendment note cannot exceed 255 characters"),
		),
	))

	if e.Kind.RequiresReason() && e.Reason == "" {
		multiErr.Add("reason", errortypes.ErrRequired, "A reason is required for this event")
	}
}

func (e *WorkerEmploymentEvent) IsAmended() bool {
	return e.AmendedAt != nil && *e.AmendedAt > 0
}

func (e *WorkerEmploymentEvent) GetID() pulid.ID { return e.ID }

func (e *WorkerEmploymentEvent) GetCreatedAt() int64 { return e.CreatedAt }

func (e *WorkerEmploymentEvent) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *WorkerEmploymentEvent) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *WorkerEmploymentEvent) GetTableName() string { return "worker_employment_events" }

func (e *WorkerEmploymentEvent) GetResourceType() string { return "worker_employment_event" }

func (e *WorkerEmploymentEvent) GetResourceID() string { return e.ID.String() }

func (e *WorkerEmploymentEvent) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("wee_")
		}
		if e.FromValues == nil {
			e.FromValues = map[string]string{}
		}
		if e.ToValues == nil {
			e.ToValues = map[string]string{}
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		if e.FromValues == nil {
			e.FromValues = map[string]string{}
		}
		if e.ToValues == nil {
			e.ToValues = map[string]string{}
		}
		e.UpdatedAt = now
	}

	return nil
}

// EmploymentState is the slice of a worker the events act on, kept small so
// the transition rules are testable without a database.
type EmploymentState struct {
	Status          domaintypes.Status
	CanBeAssigned   bool
	HireDate        int64
	TerminationDate *int64
	// OpenLeave / OpenSuspension mirror the most recent unmatched event.
	OpenLeave      bool
	OpenSuspension bool
}

func EmploymentStateOf(wrk *Worker, history []*WorkerEmploymentEvent) EmploymentState {
	state := EmploymentState{}
	if wrk == nil {
		return state
	}
	state.Status = wrk.Status
	state.CanBeAssigned = wrk.CanBeAssigned
	if wrk.Profile != nil {
		state.HireDate = wrk.Profile.HireDate
		state.TerminationDate = wrk.Profile.TerminationDate
	}
	for _, event := range history {
		switch event.Kind {
		case EmploymentEventLeaveStarted:
			state.OpenLeave = true
		case EmploymentEventLeaveEnded:
			state.OpenLeave = false
		case EmploymentEventSuspended:
			state.OpenSuspension = true
		case EmploymentEventReinstated:
			state.OpenSuspension = false
		case EmploymentEventTerminated:
			state.OpenLeave = false
			state.OpenSuspension = false
		case EmploymentEventHired, EmploymentEventRehired, EmploymentEventProbationEnded,
			EmploymentEventPromoted, EmploymentEventTransferred, EmploymentEventRateChanged:
		}
	}
	return state
}

// CanRecord checks an event against the worker's current employment state and
// returns a field-level reason when it does not fit. The rules keep the
// timeline coherent: you cannot terminate someone twice, rehire an active
// worker, or end a leave that never started.
func (k EmploymentEventKind) CanRecord(state EmploymentState) error {
	employed := state.Status == domaintypes.StatusActive
	switch k {
	case EmploymentEventHired:
		if state.HireDate > 0 && employed {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"This worker already has a hire on record. Use Rehired for a returning worker",
			)
		}
	case EmploymentEventRehired:
		if employed {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"Only an inactive worker can be rehired",
			)
		}
	case EmploymentEventTerminated:
		if !employed {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"This worker is already inactive",
			)
		}
	case EmploymentEventSuspended:
		if !employed {
			return inactiveError()
		}
		if state.OpenSuspension {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"This worker is already suspended",
			)
		}
	case EmploymentEventReinstated:
		if !state.OpenSuspension {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"There is no suspension to lift",
			)
		}
	case EmploymentEventLeaveStarted:
		if !employed {
			return inactiveError()
		}
		if state.OpenLeave {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"This worker is already on leave",
			)
		}
	case EmploymentEventLeaveEnded:
		if !state.OpenLeave {
			return errortypes.NewValidationError(
				"kind",
				errortypes.ErrInvalidOperation,
				"There is no leave to end",
			)
		}
	case EmploymentEventProbationEnded, EmploymentEventPromoted, EmploymentEventTransferred,
		EmploymentEventRateChanged:
		if !employed {
			return inactiveError()
		}
	}
	return nil
}

func inactiveError() error {
	return errortypes.NewValidationError(
		"kind",
		errortypes.ErrInvalidOperation,
		"This worker is inactive. Record a rehire first",
	)
}

// Apply mutates the worker for the events that change employment state and
// returns whether anything changed. Informational kinds leave the worker
// untouched; the values they carry live on the event.
func (e *WorkerEmploymentEvent) Apply(wrk *Worker) bool {
	if wrk == nil {
		return false
	}
	switch e.Kind {
	case EmploymentEventHired, EmploymentEventRehired:
		wrk.Status = domaintypes.StatusActive
		wrk.CanBeAssigned = true
		wrk.AvailableForDispatch = true
		if wrk.Profile != nil {
			wrk.Profile.HireDate = e.EffectiveAt
			wrk.Profile.TerminationDate = nil
		}
		return true
	case EmploymentEventTerminated:
		wrk.Status = domaintypes.StatusInactive
		wrk.CanBeAssigned = false
		wrk.AvailableForDispatch = false
		wrk.LeaveType = ""
		if wrk.Profile != nil {
			effective := e.EffectiveAt
			wrk.Profile.TerminationDate = &effective
		}
		return true
	case EmploymentEventLeaveStarted:
		wrk.CanBeAssigned = false
		wrk.AvailableForDispatch = false
		if value, ok := e.ToValues[EmploymentValueLeaveType]; ok && LeaveType(value).IsValid() {
			wrk.LeaveType = LeaveType(value)
		} else {
			wrk.LeaveType = LeaveTypeOther
		}
		return true
	case EmploymentEventSuspended:
		wrk.CanBeAssigned = false
		wrk.AvailableForDispatch = false
		return true
	case EmploymentEventLeaveEnded:
		wrk.CanBeAssigned = true
		wrk.AvailableForDispatch = true
		wrk.LeaveType = ""
		return true
	case EmploymentEventReinstated:
		wrk.CanBeAssigned = true
		wrk.AvailableForDispatch = true
		return true
	case EmploymentEventTransferred:
		changed := false
		if id, ok := e.ToValues[EmploymentValueFleetCodeID]; ok {
			wrk.FleetCodeID = pulid.ID(id)
			changed = true
		}
		if id, ok := e.ToValues[EmploymentValueManagerID]; ok {
			wrk.ManagerID = pulid.ID(id)
			changed = true
		}
		return changed
	case EmploymentEventPromoted:
		changed := false
		if value, ok := e.ToValues[EmploymentValueDriverType]; ok && DriverType(value).IsValid() {
			wrk.DriverType = DriverType(value)
			changed = true
		}
		if value, ok := e.ToValues[EmploymentValueWorkerType]; ok && WorkerType(value).IsValid() {
			wrk.Type = WorkerType(value)
			changed = true
		}
		return changed
	case EmploymentEventProbationEnded, EmploymentEventRateChanged:
		return false
	default:
		return false
	}
}

// Tenure returns whole days of service between hire and either termination or
// now. A zero hire date yields zero.
func Tenure(hireDate int64, terminationDate *int64, now int64) time.Duration {
	if hireDate <= 0 {
		return 0
	}
	end := now
	if terminationDate != nil && *terminationDate > 0 && *terminationDate < now {
		end = *terminationDate
	}
	if end <= hireDate {
		return 0
	}
	return time.Duration(end-hireDate) * time.Second
}

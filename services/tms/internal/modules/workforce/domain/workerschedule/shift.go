package workerschedule

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	ErrInvalidAvailability  = errors.New("invalid availability preference")
	ErrInvalidShiftSwap     = errors.New("invalid shift swap status")
	daysOfWeekMaskLength    = 7
	defaultShiftStartMinute = int16(360)
)

// AvailabilityPreference is what a driver would rather work. It is a
// statement, never a constraint: dispatch is free to override it, and the rota
// shows where it did so the override is visible rather than silent.
type AvailabilityPreference string

const (
	AvailabilityPreferred   = AvailabilityPreference("Preferred")
	AvailabilityAvailable   = AvailabilityPreference("Available")
	AvailabilityUnavailable = AvailabilityPreference("Unavailable")
)

func AvailabilityPreferenceValues() []AvailabilityPreference {
	return []AvailabilityPreference{
		AvailabilityPreferred,
		AvailabilityAvailable,
		AvailabilityUnavailable,
	}
}

func (p AvailabilityPreference) String() string { return string(p) }

func (p AvailabilityPreference) IsValid() bool {
	switch p {
	case AvailabilityPreferred, AvailabilityAvailable, AvailabilityUnavailable:
		return true
	default:
		return false
	}
}

// ShiftSwapStatus is how far a swap has got. Two acceptances are needed — the
// colleague's and a manager's — because a swap the office never saw is a shift
// nobody is covering.
type ShiftSwapStatus string

const (
	SwapProposed  = ShiftSwapStatus("Proposed")
	SwapAccepted  = ShiftSwapStatus("Accepted")
	SwapDeclined  = ShiftSwapStatus("Declined")
	SwapApproved  = ShiftSwapStatus("Approved")
	SwapRejected  = ShiftSwapStatus("Rejected")
	SwapWithdrawn = ShiftSwapStatus("Withdrawn")
)

func (s ShiftSwapStatus) String() string { return string(s) }

func (s ShiftSwapStatus) IsValid() bool {
	switch s {
	case SwapProposed, SwapAccepted, SwapDeclined, SwapApproved, SwapRejected, SwapWithdrawn:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the swap is still going somewhere.
func (s ShiftSwapStatus) IsOpen() bool { return s == SwapProposed || s == SwapAccepted }

// CanTransitionTo is the state machine. A colleague answers first and the
// office answers second; skipping the colleague would approve a swap they
// never agreed to.
func (s ShiftSwapStatus) CanTransitionTo(next ShiftSwapStatus) bool {
	switch s {
	case SwapProposed:
		return next == SwapAccepted || next == SwapDeclined || next == SwapWithdrawn
	case SwapAccepted:
		return next == SwapApproved || next == SwapRejected || next == SwapWithdrawn
	case SwapDeclined, SwapApproved, SwapRejected, SwapWithdrawn:
		return false
	default:
		return false
	}
}

var (
	_ validationframework.TenantedEntity = (*ShiftTemplate)(nil)
)

// ShiftTemplate is a repeating pattern of working days.
type ShiftTemplate struct {
	keel.Table `keel:"shift_templates"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:shft_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Status      domaintypes.Status `json:"status"      keel:"status"`
	Code        string             `json:"code"        keel:"code"`
	Name        string             `json:"name"        keel:"name"`
	Description string             `json:"description" keel:"description,nullzero"`
	Color       string             `json:"color"       keel:"color,nullzero"`
	// DaysOfWeek is seven characters indexed from Sunday: 0111110 is Monday to
	// Friday. A mask rather than an array so a pattern reads at a glance and
	// compares without unpacking.
	DaysOfWeek      string `json:"daysOfWeek"      keel:"days_of_week"`
	StartMinute     int16  `json:"startMinute"     keel:"start_minute"`
	DurationMinutes int16  `json:"durationMinutes" keel:"duration_minutes"`
	// CycleWeeks lets an A/B rotation be one template: the assignment carries
	// which week of the cycle a worker starts on.
	CycleWeeks int16 `json:"cycleWeeks" keel:"cycle_weeks"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (t *ShiftTemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 20).Error("Code cannot exceed 20 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name cannot exceed 100 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
	))

	if !isDayMask(t.DaysOfWeek) {
		multiErr.Add(
			"daysOfWeek",
			errortypes.ErrInvalid,
			"Days must be seven characters of 0 or 1, starting on Sunday",
		)
	}
	// A pattern with no working days is not a shift; it is a way to roster
	// somebody onto nothing and never notice.
	if isDayMask(t.DaysOfWeek) && !strings.Contains(t.DaysOfWeek, "1") {
		multiErr.Add("daysOfWeek", errortypes.ErrInvalid, "A shift needs at least one working day")
	}
	if t.StartMinute < 0 || t.StartMinute >= 1440 {
		multiErr.Add("startMinute", errortypes.ErrInvalid, "The start time is not a time of day")
	}
	if t.DurationMinutes <= 0 || t.DurationMinutes > 1440 {
		multiErr.Add(
			"durationMinutes",
			errortypes.ErrInvalid,
			"A shift is between a minute and a day long",
		)
	}
	if t.CycleWeeks < 1 || t.CycleWeeks > 8 {
		multiErr.Add("cycleWeeks", errortypes.ErrInvalid, "A rotation is 1 to 8 weeks")
	}
}

func isDayMask(value string) bool {
	if len(value) != daysOfWeekMaskLength {
		return false
	}
	for _, char := range value {
		if char != '0' && char != '1' {
			return false
		}
	}
	return true
}

// WorksOn reports whether the pattern covers a weekday, given which week of the
// cycle is being asked about. Weekday is 0 for Sunday.
func (t *ShiftTemplate) WorksOn(weekday int, cycleWeek int) bool {
	if !isDayMask(t.DaysOfWeek) || weekday < 0 || weekday > 6 {
		return false
	}
	// A single-week cycle works every week; a longer one works only on the
	// week the assignment lands on, which is what makes an A/B rotation
	// alternate rather than double up.
	if t.CycleWeeks > 1 && cycleWeek != 0 {
		return false
	}
	return t.DaysOfWeek[weekday] == '1'
}

// EndMinute is when the shift finishes, which may be past midnight. Callers
// that render a clock time take it modulo a day; callers computing length do
// not, because a shift that ends at 02:00 is ten hours long, not minus four.
func (t *ShiftTemplate) EndMinute() int {
	return int(t.StartMinute) + int(t.DurationMinutes)
}

func (t *ShiftTemplate) Normalise() {
	t.Code = strings.ToUpper(strings.TrimSpace(t.Code))
	t.Name = strings.TrimSpace(t.Name)
	t.Description = strings.TrimSpace(t.Description)
	t.Color = strings.TrimSpace(t.Color)
}

func (t *ShiftTemplate) GetID() pulid.ID { return t.ID }

func (t *ShiftTemplate) GetCreatedAt() int64 { return t.CreatedAt }

func (t *ShiftTemplate) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *ShiftTemplate) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *ShiftTemplate) GetTableName() string { return "shift_templates" }

func (t *ShiftTemplate) GetResourceType() string { return "shift_template" }

func (t *ShiftTemplate) GetResourceID() string { return t.ID.String() }

func (t *ShiftTemplate) BeforeCreate(_ context.Context) error {
	if t.Status == "" {
		t.Status = domaintypes.StatusActive
	}
	if t.DaysOfWeek == "" {
		t.DaysOfWeek = "0111110"
	}
	if t.CycleWeeks <= 0 {
		t.CycleWeeks = 1
	}
	if t.StartMinute <= 0 {
		t.StartMinute = defaultShiftStartMinute
	}
	if t.StartMinute == 0 {
		t.StartMinute = 360
	}
	if t.DurationMinutes == 0 {
		t.DurationMinutes = 600
	}
	if t.CycleWeeks == 0 {
		t.CycleWeeks = 1
	}

	return nil
}

func (ShiftTemplate) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_shift_templates_code",
			Message: "Shift template with this code already exists in your organization",
		},
	}
}

var (
	_ validationframework.TenantedEntity = (*WorkerShiftAssignment)(nil)
)

// WorkerShiftAssignment puts a worker on a pattern for a period.
type WorkerShiftAssignment struct {
	keel.Table `keel:"worker_shift_assignments"`

	ID              pulid.ID `json:"id"              keel:"id,pk,autoid:wsa_"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID `json:"organizationId"  keel:"organization_id,pk,tenant"`
	WorkerID        pulid.ID `json:"workerId"        keel:"worker_id"`
	ShiftTemplateID pulid.ID `json:"shiftTemplateId" keel:"shift_template_id"`

	EffectiveFrom int64  `json:"effectiveFrom" keel:"effective_from"`
	EffectiveTo   *int64 `json:"effectiveTo"   keel:"effective_to"`
	// CycleOffsetWeeks is which week of the template's cycle this worker starts
	// on, so an A/B rotation is one template and two offsets.
	CycleOffsetWeeks int16    `json:"cycleOffsetWeeks" keel:"cycle_offset_weeks"`
	Notes            string   `json:"notes"            keel:"notes,nullzero"`
	AssignedByID     pulid.ID `json:"assignedById"     keel:"assigned_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	ShiftTemplate *ShiftTemplate `json:"shiftTemplate,omitempty" keel:"belongs:shift_template_id"`
	Worker        *worker.Worker `json:"worker,omitempty"        keel:"belongs:worker_id"`
}

func (a *WorkerShiftAssignment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.ShiftTemplateID, validation.Required.Error("A shift is required")),
		validation.Field(&a.EffectiveFrom,
			validation.Required.Error("An effective date is required"),
		),
	))

	if a.EffectiveTo != nil && *a.EffectiveTo < a.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"An assignment cannot end before it begins",
		)
	}
	if a.CycleOffsetWeeks < 0 || a.CycleOffsetWeeks > 7 {
		multiErr.Add("cycleOffsetWeeks", errortypes.ErrInvalid, "The rotation offset is 0 to 7")
	}
}

func (a *WorkerShiftAssignment) GetID() pulid.ID { return a.ID }

func (a *WorkerShiftAssignment) GetCreatedAt() int64 { return a.CreatedAt }

func (a *WorkerShiftAssignment) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *WorkerShiftAssignment) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *WorkerShiftAssignment) GetTableName() string { return "worker_shift_assignments" }

func (a *WorkerShiftAssignment) GetResourceType() string { return "worker_shift_assignment" }

func (a *WorkerShiftAssignment) GetResourceID() string { return a.ID.String() }

var (
	_ validationframework.TenantedEntity = (*WorkerAvailabilityPreference)(nil)
)

// WorkerAvailabilityPreference is what a driver would rather work on a given
// weekday.
type WorkerAvailabilityPreference struct {
	keel.Table `keel:"worker_availability_preferences"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:wapf_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	DayOfWeek  int16                  `json:"dayOfWeek"  keel:"day_of_week"`
	Preference AvailabilityPreference `json:"preference" keel:"preference"`
	Note       string                 `json:"note"       keel:"note,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (WorkerShiftAssignment) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_worker_shift_assignments_open",
			Message: "Worker already has an open shift assignment",
		},
		{
			Name:    "fk_worker_shift_assignments_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_shift_assignments_template",
			Message: "Shift template does not exist or belongs to a different organization",
		},
	}
}

func (p *WorkerAvailabilityPreference) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&p.Preference,
			validation.Required.Error("A preference is required"),
			domainvalidation.ValidEnum[AvailabilityPreference]("Preference is not valid"),
		),
		validation.Field(&p.Note,
			validation.Length(0, 255).Error("Note cannot exceed 255 characters"),
		),
	))

	if p.DayOfWeek < 0 || p.DayOfWeek > 6 {
		multiErr.Add("dayOfWeek", errortypes.ErrInvalid, "That is not a day of the week")
	}
}

func (p *WorkerAvailabilityPreference) GetID() pulid.ID { return p.ID }

func (p *WorkerAvailabilityPreference) GetCreatedAt() int64 { return p.CreatedAt }

func (p *WorkerAvailabilityPreference) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *WorkerAvailabilityPreference) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *WorkerAvailabilityPreference) GetTableName() string {
	return "worker_availability_preferences"
}

func (p *WorkerAvailabilityPreference) GetResourceType() string {
	return "worker_availability_preference"
}

func (p *WorkerAvailabilityPreference) GetResourceID() string { return p.ID.String() }

func (p *WorkerAvailabilityPreference) BeforeCreate(_ context.Context) error {
	if p.Preference == "" {
		p.Preference = AvailabilityAvailable
	}

	return nil
}

func (WorkerAvailabilityPreference) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_availability_preferences_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}

var (
	_ validationframework.TenantedEntity = (*ShiftSwapRequest)(nil)
)

// ShiftSwapRequest is one driver asking another to take a day.
type ShiftSwapRequest struct {
	keel.Table `keel:"shift_swap_requests"`

	ID                   pulid.ID `json:"id"                   keel:"id,pk,autoid:sswp_"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID `json:"organizationId"       keel:"organization_id,pk,tenant"`
	RequestingWorkerID   pulid.ID `json:"requestingWorkerId"   keel:"requesting_worker_id"`
	CounterpartyWorkerID pulid.ID `json:"counterpartyWorkerId" keel:"counterparty_worker_id,nullzero"`

	Status ShiftSwapStatus `json:"status" keel:"status"`
	// ShiftDate is the day being given up; CounterpartyShiftDate is the day
	// offered back, when the swap is a trade rather than a hand-off.
	ShiftDate             int64  `json:"shiftDate"             keel:"shift_date"`
	CounterpartyShiftDate *int64 `json:"counterpartyShiftDate" keel:"counterparty_shift_date"`

	Reason       string   `json:"reason"       keel:"reason,nullzero"`
	ResponseNote string   `json:"responseNote" keel:"response_note,nullzero"`
	RespondedAt  *int64   `json:"respondedAt"  keel:"responded_at"`
	DecidedAt    *int64   `json:"decidedAt"    keel:"decided_at"`
	DecidedByID  pulid.ID `json:"decidedById"  keel:"decided_by_id,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	RequestingWorker   *worker.Worker `json:"requestingWorker,omitempty"   keel:"belongs:requesting_worker_id"`
	CounterpartyWorker *worker.Worker `json:"counterpartyWorker,omitempty" keel:"belongs:counterparty_worker_id"`
}

func (r *ShiftSwapRequest) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.RequestingWorkerID,
			validation.Required.Error("The requesting driver is required"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ShiftSwapStatus]("Status is not valid"),
		),
		validation.Field(&r.ShiftDate, validation.Required.Error("The day is required")),
		validation.Field(&r.Reason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	// Swapping with yourself changes nothing and shows on the board as cover
	// somebody arranged.
	if !r.CounterpartyWorkerID.IsNil() && r.CounterpartyWorkerID == r.RequestingWorkerID {
		multiErr.Add(
			"counterpartyWorkerId",
			errortypes.ErrInvalid,
			"A swap has to be with somebody else",
		)
	}
	// A decision is a dated act. Recording one with no date leaves a swap that
	// says it was approved and cannot say when.
	if (r.Status == SwapApproved || r.Status == SwapRejected) &&
		(r.DecidedAt == nil || *r.DecidedAt <= 0) {
		multiErr.Add("decidedAt", errortypes.ErrRequired, "A decision needs the date it was made")
	}
}

func (r *ShiftSwapRequest) GetID() pulid.ID { return r.ID }

func (r *ShiftSwapRequest) GetCreatedAt() int64 { return r.CreatedAt }

func (r *ShiftSwapRequest) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *ShiftSwapRequest) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *ShiftSwapRequest) GetTableName() string { return "shift_swap_requests" }

func (r *ShiftSwapRequest) GetResourceType() string { return "shift_swap_request" }

func (r *ShiftSwapRequest) GetResourceID() string { return r.ID.String() }

func (r *ShiftSwapRequest) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = SwapProposed
	}

	return nil
}

func (ShiftSwapRequest) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_shift_swap_requests_requester",
			Message: "Requesting worker does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_shift_swap_requests_counterparty",
			Message: "Counterparty worker does not exist or belongs to a different organization",
		},
	}
}

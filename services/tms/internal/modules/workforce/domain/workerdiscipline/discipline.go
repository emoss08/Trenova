package workerdiscipline

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	ErrInvalidDisciplinaryLevel  = errors.New("invalid disciplinary level")
	ErrInvalidDisciplinaryStatus = errors.New("invalid disciplinary status")
	ErrInvalidRecognitionKind    = errors.New("invalid recognition kind")
)

// DisciplinaryLevel is a rung on the progressive-discipline ladder.
type DisciplinaryLevel string

const (
	DisciplinaryLevelNone           = DisciplinaryLevel("")
	DisciplinaryLevelCoaching       = DisciplinaryLevel("Coaching")
	DisciplinaryLevelVerbalWarning  = DisciplinaryLevel("VerbalWarning")
	DisciplinaryLevelWrittenWarning = DisciplinaryLevel("WrittenWarning")
	DisciplinaryLevelFinalWarning   = DisciplinaryLevel("FinalWarning")
	DisciplinaryLevelSuspension     = DisciplinaryLevel("Suspension")
	DisciplinaryLevelTermination    = DisciplinaryLevel("Termination")
)

var disciplinaryLadder = []DisciplinaryLevel{
	DisciplinaryLevelCoaching,
	DisciplinaryLevelVerbalWarning,
	DisciplinaryLevelWrittenWarning,
	DisciplinaryLevelFinalWarning,
	DisciplinaryLevelSuspension,
	DisciplinaryLevelTermination,
}

func (l DisciplinaryLevel) String() string { return string(l) }

func (l DisciplinaryLevel) IsValid() bool {
	return l.Rank() > 0
}

// Rank is the rung number, 1 for Coaching through 6 for Termination; 0 for none.
func (l DisciplinaryLevel) Rank() int {
	for i, level := range disciplinaryLadder {
		if level == l {
			return i + 1
		}
	}
	return 0
}

// Next is the rung above this one; Termination has none.
func (l DisciplinaryLevel) Next() DisciplinaryLevel {
	rank := l.Rank()
	if rank == 0 {
		return DisciplinaryLevelCoaching
	}
	if rank >= len(disciplinaryLadder) {
		return DisciplinaryLevelTermination
	}
	return disciplinaryLadder[rank]
}

// EndsEmployment reports whether the rung is the last one.
func (l DisciplinaryLevel) EndsEmployment() bool { return l == DisciplinaryLevelTermination }

// Label is the rung in the words a manager would say it, for prose the server
// builds. The client keeps its own map for labels it renders itself.
func (l DisciplinaryLevel) Label() string {
	switch l {
	case DisciplinaryLevelCoaching:
		return "coaching conversation"
	case DisciplinaryLevelVerbalWarning:
		return "verbal warning"
	case DisciplinaryLevelWrittenWarning:
		return "written warning"
	case DisciplinaryLevelFinalWarning:
		return "final warning"
	case DisciplinaryLevelSuspension:
		return "suspension"
	case DisciplinaryLevelTermination:
		return "termination"
	default:
		return "disciplinary action"
	}
}

type DisciplinaryStatus string

const (
	DisciplinaryStatusActive    = DisciplinaryStatus("Active")
	DisciplinaryStatusExpired   = DisciplinaryStatus("Expired")
	DisciplinaryStatusRescinded = DisciplinaryStatus("Rescinded")
)

func (s DisciplinaryStatus) String() string { return string(s) }

func (s DisciplinaryStatus) IsValid() bool {
	switch s {
	case DisciplinaryStatusActive, DisciplinaryStatusExpired, DisciplinaryStatusRescinded:
		return true
	default:
		return false
	}
}

// DisciplinaryLookbackMonths is how long an action stays on the ladder by
// default before it rolls off.
const DisciplinaryLookbackMonths = 12

var (
	_ validationframework.TenantedEntity = (*WorkerDisciplinaryAction)(nil)
)

type WorkerDisciplinaryAction struct {
	keel.Table `keel:"worker_disciplinary_actions"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:wdac_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID           `json:"workerId"       keel:"worker_id"`
	Level          DisciplinaryLevel  `json:"level"          keel:"level"`
	Status         DisciplinaryStatus `json:"status"         keel:"status"`
	Reason         string             `json:"reason"         keel:"reason"`
	Details        string             `json:"details"        keel:"details,nullzero"`
	OccurredAt     *int64             `json:"occurredAt"     keel:"occurred_at"`
	IssuedAt       int64              `json:"issuedAt"       keel:"issued_at"`
	ExpiresAt      *int64             `json:"expiresAt"      keel:"expires_at"`
	SuspensionDays *int32             `json:"suspensionDays" keel:"suspension_days"`
	SafetyEventID  pulid.ID           `json:"safetyEventId"  keel:"safety_event_id,nullzero"`
	DocumentID     pulid.ID           `json:"documentId"     keel:"document_id,nullzero"`
	IssuedByID     pulid.ID           `json:"issuedById"     keel:"issued_by_id,nullzero"`
	AcknowledgedAt *int64             `json:"acknowledgedAt" keel:"acknowledged_at"`
	WorkerComment  string             `json:"workerComment"  keel:"worker_comment,nullzero"`
	RescindedAt    *int64             `json:"rescindedAt"    keel:"rescinded_at"`
	RescindedByID  pulid.ID           `json:"rescindedById"  keel:"rescinded_by_id,nullzero"`
	RescindReason  string             `json:"rescindReason"  keel:"rescind_reason,nullzero"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Worker      *worker.Worker                  `json:"worker,omitempty"      keel:"belongs:worker_id"`
	SafetyEvent *workersafety.WorkerSafetyEvent `json:"safetyEvent,omitempty" keel:"belongs:safety_event_id"`
	Document    *document.Document              `json:"document,omitempty"    keel:"belongs:document_id"`
	IssuedBy    *tenant.User                    `json:"issuedBy,omitempty"    keel:"belongs:issued_by_id"`
}

func (a *WorkerDisciplinaryAction) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&a.Level,
			validation.Required.Error("Level is required"),
			domainvalidation.ValidEnum[DisciplinaryLevel](
				"level must be one of: Coaching, VerbalWarning, WrittenWarning, FinalWarning, Suspension, Termination",
			),
		),
		validation.Field(
			&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DisciplinaryStatus](
				"status must be Active, Expired or Rescinded",
			),
		),
		validation.Field(&a.Reason,
			validation.Required.Error("Say why the action is being taken"),
			validation.Length(1, 4000).Error("Reason cannot exceed 4000 characters"),
		),
		validation.Field(&a.IssuedAt, validation.Required.Error("Issue date is required")),
		validation.Field(&a.RescindReason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if a.Level == DisciplinaryLevelSuspension &&
		(a.SuspensionDays == nil || *a.SuspensionDays <= 0) {
		multiErr.Add("suspensionDays", errortypes.ErrRequired, "How many days is the suspension?")
	}
	if a.Level != DisciplinaryLevelSuspension && a.SuspensionDays != nil {
		multiErr.Add("suspensionDays", errortypes.ErrInvalid, "Only suspensions carry days")
	}
	if a.ExpiresAt != nil && *a.ExpiresAt <= a.IssuedAt {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry must be after the issue date")
	}
	if a.Status == DisciplinaryStatusRescinded && strings.TrimSpace(a.RescindReason) == "" {
		multiErr.Add("rescindReason", errortypes.ErrRequired, "Say why the action is rescinded")
	}
}

// IsActive reports whether the action still counts on the ladder at now.
func (a *WorkerDisciplinaryAction) IsActive(now int64) bool {
	if a.Status != DisciplinaryStatusActive {
		return false
	}
	return a.ExpiresAt == nil || *a.ExpiresAt <= 0 || *a.ExpiresAt > now
}

// DefaultExpiry sets the roll-off date from the issue date unless one was
// chosen; terminations never roll off.
func (a *WorkerDisciplinaryAction) DefaultExpiry() {
	if a.Level.EndsEmployment() {
		a.ExpiresAt = nil
		return
	}
	if a.ExpiresAt == nil || *a.ExpiresAt <= 0 {
		expiry := timeutils.AddMonthsUTC(a.IssuedAt, DisciplinaryLookbackMonths)
		a.ExpiresAt = &expiry
	}
}

func (a *WorkerDisciplinaryAction) IsAcknowledged() bool {
	return a.AcknowledgedAt != nil && *a.AcknowledgedAt > 0
}

func (a *WorkerDisciplinaryAction) GetID() pulid.ID { return a.ID }

func (a *WorkerDisciplinaryAction) GetCreatedAt() int64 { return a.CreatedAt }

func (a *WorkerDisciplinaryAction) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *WorkerDisciplinaryAction) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *WorkerDisciplinaryAction) GetTableName() string { return "worker_disciplinary_actions" }

func (a *WorkerDisciplinaryAction) GetResourceType() string { return "worker_disciplinary_action" }

func (a *WorkerDisciplinaryAction) GetResourceID() string { return a.ID.String() }

func (a *WorkerDisciplinaryAction) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if a.Status == "" {
		a.Status = DisciplinaryStatusActive
	}
	if a.IssuedAt == 0 {
		a.IssuedAt = now
	}

	return nil
}

func (WorkerDisciplinaryAction) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_disciplinary_actions_event",
			Message: "Safety event does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_disciplinary_actions_document",
			Message: "Document does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_worker_disciplinary_actions_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}

// DisciplinaryLadder is where a worker stands: the rungs already taken that
// still count, and the rung the next action would normally be.
type DisciplinaryLadder struct {
	ActiveActions  []*WorkerDisciplinaryAction
	HighestLevel   DisciplinaryLevel
	SuggestedLevel DisciplinaryLevel
	// AtFinalStep is true when the suggested rung ends employment, so the UI
	// can warn before it is issued.
	AtFinalStep bool
}

// BuildDisciplinaryLadder folds the active actions into a suggestion: the
// rung above the highest one still in force, Coaching when the slate is clean.
func BuildDisciplinaryLadder(
	actions []*WorkerDisciplinaryAction,
	now int64,
) *DisciplinaryLadder {
	ladder := &DisciplinaryLadder{ActiveActions: make([]*WorkerDisciplinaryAction, 0, len(actions))}
	for _, action := range actions {
		if action == nil || !action.IsActive(now) {
			continue
		}
		ladder.ActiveActions = append(ladder.ActiveActions, action)
		if action.Level.Rank() > ladder.HighestLevel.Rank() {
			ladder.HighestLevel = action.Level
		}
	}
	ladder.SuggestedLevel = ladder.HighestLevel.Next()
	ladder.AtFinalStep = ladder.SuggestedLevel.EndsEmployment()
	return ladder
}

type RecognitionKind string

const (
	RecognitionKindSafetyMilestone = RecognitionKind("SafetyMilestone")
	RecognitionKindCustomerPraise  = RecognitionKind("CustomerPraise")
	RecognitionKindPerformance     = RecognitionKind("Performance")
	RecognitionKindTenure          = RecognitionKind("Tenure")
	RecognitionKindTeamPlayer      = RecognitionKind("TeamPlayer")
	RecognitionKindOther           = RecognitionKind("Other")
)

func (k RecognitionKind) String() string { return string(k) }

func RecognitionKindValues() []RecognitionKind {
	return []RecognitionKind{
		RecognitionKindSafetyMilestone,
		RecognitionKindCustomerPraise,
		RecognitionKindPerformance,
		RecognitionKindTenure,
		RecognitionKindTeamPlayer,
		RecognitionKindOther,
	}
}

func (k RecognitionKind) IsValid() bool {
	switch k {
	case RecognitionKindSafetyMilestone, RecognitionKindCustomerPraise, RecognitionKindPerformance,
		RecognitionKindTenure, RecognitionKindTeamPlayer, RecognitionKindOther:
		return true
	default:
		return false
	}
}

var (
	_ validationframework.TenantedEntity = (*WorkerRecognition)(nil)
)

type WorkerRecognition struct {
	keel.Table `keel:"worker_recognitions"`

	ID              pulid.ID        `json:"id"              keel:"id,pk,autoid:wrec_"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID  pulid.ID        `json:"organizationId"  keel:"organization_id,pk,tenant"`
	WorkerID        pulid.ID        `json:"workerId"        keel:"worker_id"`
	Kind            RecognitionKind `json:"kind"            keel:"kind"`
	Title           string          `json:"title"           keel:"title"`
	Message         string          `json:"message"         keel:"message,nullzero"`
	OccurredAt      int64           `json:"occurredAt"      keel:"occurred_at"`
	AwardedByID     pulid.ID        `json:"awardedById"     keel:"awarded_by_id,nullzero"`
	VisibleToWorker bool            `json:"visibleToWorker" keel:"visible_to_worker"`
	Version         int64           `json:"version"         keel:"version,version"`
	CreatedAt       int64           `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt       int64           `json:"updatedAt"       keel:"updated_at,autoupdate"`

	Worker    *worker.Worker `json:"worker,omitempty"    keel:"belongs:worker_id"`
	AwardedBy *tenant.User   `json:"awardedBy,omitempty" keel:"belongs:awarded_by_id"`
}

func (r *WorkerRecognition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[RecognitionKind](
				"kind must be one of: SafetyMilestone, CustomerPraise, Performance, Tenure, TeamPlayer, Other",
			),
		),
		validation.Field(&r.Title,
			validation.Required.Error("Give the recognition a title"),
			validation.Length(1, 120).Error("Title cannot exceed 120 characters"),
		),
		validation.Field(&r.Message,
			validation.Length(0, 2000).Error("Message cannot exceed 2000 characters"),
		),
		validation.Field(&r.OccurredAt, validation.Required.Error("Date is required")),
	))
}

func (r *WorkerRecognition) GetID() pulid.ID { return r.ID }

func (r *WorkerRecognition) GetCreatedAt() int64 { return r.CreatedAt }

func (r *WorkerRecognition) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *WorkerRecognition) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *WorkerRecognition) GetTableName() string { return "worker_recognitions" }

func (r *WorkerRecognition) GetResourceType() string { return "worker_recognition" }

func (r *WorkerRecognition) GetResourceID() string { return r.ID.String() }

func (r *WorkerRecognition) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if r.Kind == "" {
		r.Kind = RecognitionKindOther
	}
	if r.OccurredAt == 0 {
		r.OccurredAt = now
	}

	return nil
}

func (WorkerRecognition) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_worker_recognitions_worker",
			Message: "Worker does not exist or belongs to a different organization",
		},
	}
}

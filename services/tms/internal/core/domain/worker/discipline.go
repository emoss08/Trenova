package worker

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*WorkerDisciplinaryAction)(nil)
	_ validationframework.TenantedEntity = (*WorkerDisciplinaryAction)(nil)
)

type WorkerDisciplinaryAction struct {
	bun.BaseModel `bun:"table:worker_disciplinary_actions,alias:wdac" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID           `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	Level          DisciplinaryLevel  `json:"level"          bun:"level,type:disciplinary_level_enum,notnull"`
	Status         DisciplinaryStatus `json:"status"         bun:"status,type:disciplinary_status_enum,notnull,default:'Active'"`
	Reason         string             `json:"reason"         bun:"reason,type:TEXT,notnull"`
	Details        string             `json:"details"        bun:"details,type:TEXT,nullzero"`
	OccurredAt     *int64             `json:"occurredAt"     bun:"occurred_at,type:BIGINT,nullzero"`
	IssuedAt       int64              `json:"issuedAt"       bun:"issued_at,type:BIGINT,notnull"`
	ExpiresAt      *int64             `json:"expiresAt"      bun:"expires_at,type:BIGINT,nullzero"`
	SuspensionDays *int32             `json:"suspensionDays" bun:"suspension_days,type:INTEGER,nullzero"`
	SafetyEventID  pulid.ID           `json:"safetyEventId"  bun:"safety_event_id,type:VARCHAR(100),nullzero"`
	DocumentID     pulid.ID           `json:"documentId"     bun:"document_id,type:VARCHAR(100),nullzero"`
	IssuedByID     pulid.ID           `json:"issuedById"     bun:"issued_by_id,type:VARCHAR(100),nullzero"`
	AcknowledgedAt *int64             `json:"acknowledgedAt" bun:"acknowledged_at,type:BIGINT,nullzero"`
	WorkerComment  string             `json:"workerComment"  bun:"worker_comment,type:TEXT,nullzero"`
	RescindedAt    *int64             `json:"rescindedAt"    bun:"rescinded_at,type:BIGINT,nullzero"`
	RescindedByID  pulid.ID           `json:"rescindedById"  bun:"rescinded_by_id,type:VARCHAR(100),nullzero"`
	RescindReason  string             `json:"rescindReason"  bun:"rescind_reason,type:VARCHAR(255),nullzero"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker      *Worker            `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	SafetyEvent *WorkerSafetyEvent `json:"safetyEvent,omitempty" bun:"rel:belongs-to,join:safety_event_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document    *document.Document `json:"document,omitempty"    bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	IssuedBy    *tenant.User       `json:"issuedBy,omitempty"    bun:"rel:belongs-to,join:issued_by_id=id"`
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
		validation.Field(&a.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DisciplinaryStatus]("status must be Active, Expired or Rescinded"),
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

	if a.Level == DisciplinaryLevelSuspension && (a.SuspensionDays == nil || *a.SuspensionDays <= 0) {
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

func (a *WorkerDisciplinaryAction) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("wdac_")
		}
		if a.Status == "" {
			a.Status = DisciplinaryStatusActive
		}
		if a.IssuedAt == 0 {
			a.IssuedAt = now
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
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
	_ bun.BeforeAppendModelHook          = (*WorkerRecognition)(nil)
	_ validationframework.TenantedEntity = (*WorkerRecognition)(nil)
)

type WorkerRecognition struct {
	bun.BaseModel `bun:"table:worker_recognitions,alias:wrec" json:"-"`

	ID              pulid.ID        `json:"id"              bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID        `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID  pulid.ID        `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID        pulid.ID        `json:"workerId"        bun:"worker_id,type:VARCHAR(100),notnull"`
	Kind            RecognitionKind `json:"kind"            bun:"kind,type:recognition_kind_enum,notnull,default:'Other'"`
	Title           string          `json:"title"           bun:"title,type:VARCHAR(120),notnull"`
	Message         string          `json:"message"         bun:"message,type:TEXT,nullzero"`
	OccurredAt      int64           `json:"occurredAt"      bun:"occurred_at,type:BIGINT,notnull"`
	AwardedByID     pulid.ID        `json:"awardedById"     bun:"awarded_by_id,type:VARCHAR(100),nullzero"`
	VisibleToWorker bool            `json:"visibleToWorker" bun:"visible_to_worker,type:BOOLEAN,notnull,default:true"`
	Version         int64           `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64           `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64           `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker    *Worker      `json:"worker,omitempty"    bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	AwardedBy *tenant.User `json:"awardedBy,omitempty" bun:"rel:belongs-to,join:awarded_by_id=id"`
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

func (r *WorkerRecognition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("wrec_")
		}
		if r.Kind == "" {
			r.Kind = RecognitionKindOther
		}
		if r.OccurredAt == 0 {
			r.OccurredAt = now
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

package worker

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidTrainingCategory = errors.New("invalid training category")
	ErrInvalidTrainingDelivery = errors.New("invalid training delivery")
	ErrInvalidTrainingStatus   = errors.New("invalid training status")
	ErrInvalidTrainingHealth   = errors.New("invalid training health")
)

type TrainingCategory string

const (
	TrainingCategorySafety             = TrainingCategory("Safety")
	TrainingCategoryCompliance         = TrainingCategory("Compliance")
	TrainingCategoryEquipment          = TrainingCategory("Equipment")
	TrainingCategoryOrientation        = TrainingCategory("Orientation")
	TrainingCategoryHazardousMaterials = TrainingCategory("HazardousMaterials")
	TrainingCategoryOther              = TrainingCategory("Other")
)

func (c TrainingCategory) String() string { return string(c) }

func (c TrainingCategory) IsValid() bool {
	switch c {
	case TrainingCategorySafety, TrainingCategoryCompliance, TrainingCategoryEquipment,
		TrainingCategoryOrientation, TrainingCategoryHazardousMaterials, TrainingCategoryOther:
		return true
	default:
		return false
	}
}

// TrainingDelivery is how a course is taken. Online and Document courses can be
// finished from the driver portal by acknowledging them; Classroom and OnTheJob
// need the office to record the result.
type TrainingDelivery string

const (
	TrainingDeliveryOnline    = TrainingDelivery("Online")
	TrainingDeliveryClassroom = TrainingDelivery("Classroom")
	TrainingDeliveryOnTheJob  = TrainingDelivery("OnTheJob")
	TrainingDeliveryDocument  = TrainingDelivery("Document")
)

func (d TrainingDelivery) String() string { return string(d) }

func (d TrainingDelivery) IsValid() bool {
	switch d {
	case TrainingDeliveryOnline, TrainingDeliveryClassroom, TrainingDeliveryOnTheJob,
		TrainingDeliveryDocument:
		return true
	default:
		return false
	}
}

// SelfServe reports whether a driver can complete the course from the portal
// without the office scoring it.
func (d TrainingDelivery) SelfServe() bool {
	return d == TrainingDeliveryOnline || d == TrainingDeliveryDocument
}

type TrainingStatus string

const (
	TrainingStatusAssigned   = TrainingStatus("Assigned")
	TrainingStatusInProgress = TrainingStatus("InProgress")
	TrainingStatusCompleted  = TrainingStatus("Completed")
	TrainingStatusFailed     = TrainingStatus("Failed")
	TrainingStatusExpired    = TrainingStatus("Expired")
	TrainingStatusWaived     = TrainingStatus("Waived")
	TrainingStatusCancelled  = TrainingStatus("Cancelled")
)

func (s TrainingStatus) String() string { return string(s) }

func (s TrainingStatus) IsValid() bool {
	switch s {
	case TrainingStatusAssigned, TrainingStatusInProgress, TrainingStatusCompleted,
		TrainingStatusFailed, TrainingStatusExpired, TrainingStatusWaived, TrainingStatusCancelled:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the record still has work outstanding.
func (s TrainingStatus) IsOpen() bool {
	return s == TrainingStatusAssigned || s == TrainingStatusInProgress
}

// TrainingHealth is the evaluated state of one course slot for a worker.
type TrainingHealth string

const (
	TrainingHealthCurrent      = TrainingHealth("Current")
	TrainingHealthScheduled    = TrainingHealth("Scheduled")
	TrainingHealthDueSoon      = TrainingHealth("DueSoon")
	TrainingHealthOverdue      = TrainingHealth("Overdue")
	TrainingHealthExpiringSoon = TrainingHealth("ExpiringSoon")
	TrainingHealthExpired      = TrainingHealth("Expired")
	TrainingHealthFailed       = TrainingHealth("Failed")
	TrainingHealthMissing      = TrainingHealth("Missing")
)

func (h TrainingHealth) String() string { return string(h) }

func (h TrainingHealth) IsValid() bool {
	switch h {
	case TrainingHealthCurrent, TrainingHealthScheduled, TrainingHealthDueSoon,
		TrainingHealthOverdue, TrainingHealthExpiringSoon, TrainingHealthExpired,
		TrainingHealthFailed, TrainingHealthMissing:
		return true
	default:
		return false
	}
}

// Blocks reports whether this health leaves a required course unsatisfied.
func (h TrainingHealth) Blocks() bool {
	switch h {
	case TrainingHealthOverdue, TrainingHealthExpired, TrainingHealthFailed, TrainingHealthMissing:
		return true
	default:
		return false
	}
}

// Satisfied reports whether the course counts as done for the matrix.
func (h TrainingHealth) Satisfied() bool {
	return h == TrainingHealthCurrent || h == TrainingHealthExpiringSoon
}

const dueSoonDays = int64(7)

var (
	_ bun.BeforeAppendModelHook          = (*TrainingCourse)(nil)
	_ domaintypes.PostgresSearchable     = (*TrainingCourse)(nil)
	_ pagination.CursorEntity            = (*TrainingCourse)(nil)
	_ validationframework.TenantedEntity = (*TrainingCourse)(nil)
)

type TrainingCourse struct {
	bun.BaseModel             `bun:"table:training_courses,alias:trnc" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID                      pulid.ID            `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID            `json:"businessUnitId"          bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID            `json:"organizationId"          bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Code                    string              `json:"code"                    bun:"code,type:VARCHAR(50),notnull"`
	Name                    string              `json:"name"                    bun:"name,type:VARCHAR(100),notnull"`
	Description             string              `json:"description"             bun:"description,type:TEXT,nullzero"`
	Category                TrainingCategory    `json:"category"                bun:"category,type:training_category_enum,notnull,default:'Other'"`
	Status                  domaintypes.Status  `json:"status"                  bun:"status,type:status_enum,notnull,default:'Active'"`
	Delivery                TrainingDelivery    `json:"delivery"                bun:"delivery,type:training_delivery_enum,notnull,default:'Online'"`
	ContentURL              string              `json:"contentUrl"              bun:"content_url,type:VARCHAR(500),nullzero"`
	DurationMinutes         int32               `json:"durationMinutes"         bun:"duration_minutes,type:INTEGER,notnull"`
	PassingScore            decimal.NullDecimal `json:"passingScore"            bun:"passing_score,type:NUMERIC(5,2),nullzero"`
	ValidityMonths          *int32              `json:"validityMonths"          bun:"validity_months,type:INTEGER,nullzero"`
	RenewalWindowDays       int32               `json:"renewalWindowDays"       bun:"renewal_window_days,type:INTEGER,notnull,default:30"`
	IsRequired              bool                `json:"isRequired"              bun:"is_required,type:BOOLEAN,notnull"`
	RequiredForDriverTypes  []DriverType        `json:"requiredForDriverTypes"  bun:"required_for_driver_types,type:JSONB,notnull,default:'[]'"`
	DueDaysAfterAssignment  int32               `json:"dueDaysAfterAssignment"  bun:"due_days_after_assignment,type:INTEGER,notnull,default:30"`
	RequiresAcknowledgement bool                `json:"requiresAcknowledgement" bun:"requires_acknowledgement,type:BOOLEAN,notnull"`
	SortOrder               int32               `json:"sortOrder"               bun:"sort_order,type:INTEGER,notnull"`
	Version                 int64               `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64               `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64               `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector            string              `json:"-"                       bun:"search_vector,type:TSVECTOR,scanonly"`
}

func (c *TrainingCourse) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&c.Category,
			validation.Required.Error("Category is required"),
			domainvalidation.ValidEnum[TrainingCategory](
				"category must be one of: Safety, Compliance, Equipment, Orientation, HazardousMaterials, Other",
			),
		),
		validation.Field(&c.Delivery,
			validation.Required.Error("Delivery is required"),
			domainvalidation.ValidEnum[TrainingDelivery](
				"delivery must be one of: Online, Classroom, OnTheJob, Document",
			),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(&c.ContentURL,
			validation.Length(0, 500).Error("Link cannot exceed 500 characters"),
		),
		validation.Field(&c.DurationMinutes,
			validation.Min(int32(0)).Error("Duration cannot be negative"),
		),
		validation.Field(&c.RenewalWindowDays,
			validation.Min(int32(0)).Error("Renewal window cannot be negative"),
			validation.Max(int32(365)).Error("Renewal window cannot exceed 365 days"),
		),
		validation.Field(&c.DueDaysAfterAssignment,
			validation.Min(int32(0)).Error("Due days cannot be negative"),
			validation.Max(int32(730)).Error("Due days cannot exceed two years"),
		),
	))

	if c.ValidityMonths != nil && *c.ValidityMonths <= 0 {
		multiErr.Add("validityMonths", errortypes.ErrInvalid, "Validity must be at least one month")
	}
	if c.PassingScore.Valid {
		score := c.PassingScore.Decimal
		if score.IsNegative() || score.GreaterThan(decimal.NewFromInt(100)) {
			multiErr.Add("passingScore", errortypes.ErrInvalid, "Passing score must be between 0 and 100")
		}
	}
	if c.ContentURL != "" {
		parsed, err := url.Parse(c.ContentURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			multiErr.Add("contentUrl", errortypes.ErrInvalid, "Link must be a full http(s) address")
		}
	}
	if c.Delivery == TrainingDeliveryOnline && c.ContentURL == "" {
		multiErr.Add("contentUrl", errortypes.ErrRequired, "Online courses need a link the driver can open")
	}
	for i, driverType := range c.RequiredForDriverTypes {
		if !driverType.IsValid() {
			multiErr.Add(
				"requiredForDriverTypes["+strconv.Itoa(i)+"]",
				errortypes.ErrInvalid,
				"Unknown driver type",
			)
		}
	}
}

// AppliesTo reports whether the course is required for the worker: a required
// course with no driver-type filter applies to everyone, otherwise only to the
// listed driver types.
func (c *TrainingCourse) AppliesTo(wrk *Worker) bool {
	if c == nil || !c.IsRequired || c.Status != domaintypes.StatusActive {
		return false
	}
	if len(c.RequiredForDriverTypes) == 0 {
		return true
	}
	if wrk == nil {
		return false
	}
	return slices.Contains(c.RequiredForDriverTypes, wrk.DriverType)
}

// Recurs reports whether a completion lapses and has to be renewed.
func (c *TrainingCourse) Recurs() bool {
	return c != nil && c.ValidityMonths != nil && *c.ValidityMonths > 0
}

// ExpiryFor returns when a completion on the given day lapses, nil for
// one-time courses.
func (c *TrainingCourse) ExpiryFor(completedAt int64) *int64 {
	if !c.Recurs() {
		return nil
	}
	expiry := timeutils.AddMonthsUTC(completedAt, int(*c.ValidityMonths))
	return &expiry
}

// DueFor returns the due date for an assignment made on assignedAt.
func (c *TrainingCourse) DueFor(assignedAt int64) *int64 {
	if c == nil || c.DueDaysAfterAssignment <= 0 {
		return nil
	}
	due := assignedAt + int64(c.DueDaysAfterAssignment)*secondsPerDay
	return &due
}

// Grade decides whether a score passes. Courses without a passing score pass
// on completion; courses with one need a score.
func (c *TrainingCourse) Grade(score decimal.NullDecimal) (bool, error) {
	if !c.PassingScore.Valid {
		return true, nil
	}
	if !score.Valid {
		return false, errortypes.NewValidationError(
			"score",
			errortypes.ErrRequired,
			c.Name+" is scored; enter the result",
		)
	}
	return score.Decimal.GreaterThanOrEqual(c.PassingScore.Decimal), nil
}

func (c *TrainingCourse) NormalizeCode() {
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
}

func (c *TrainingCourse) GetID() pulid.ID { return c.ID }

func (c *TrainingCourse) GetCreatedAt() int64 { return c.CreatedAt }

func (c *TrainingCourse) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *TrainingCourse) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *TrainingCourse) GetTableName() string { return "training_courses" }

func (c *TrainingCourse) GetResourceType() string { return "training_course" }

func (c *TrainingCourse) GetResourceID() string { return c.ID.String() }

func (c *TrainingCourse) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "trnc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (c *TrainingCourse) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("trnc_")
		}
		if c.RequiredForDriverTypes == nil {
			c.RequiredForDriverTypes = []DriverType{}
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		if c.RequiredForDriverTypes == nil {
			c.RequiredForDriverTypes = []DriverType{}
		}
		c.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*WorkerTrainingRecord)(nil)
	_ validationframework.TenantedEntity = (*WorkerTrainingRecord)(nil)
)

type WorkerTrainingRecord struct {
	bun.BaseModel `bun:"table:worker_training_records,alias:wtrn" json:"-"`

	ID             pulid.ID            `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID            `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	CourseID       pulid.ID            `json:"courseId"       bun:"course_id,type:VARCHAR(100),notnull"`
	Status         TrainingStatus      `json:"status"         bun:"status,type:worker_training_status_enum,notnull,default:'Assigned'"`
	AssignedAt     int64               `json:"assignedAt"     bun:"assigned_at,type:BIGINT,notnull"`
	DueAt          *int64              `json:"dueAt"          bun:"due_at,type:BIGINT,nullzero"`
	StartedAt      *int64              `json:"startedAt"      bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt    *int64              `json:"completedAt"    bun:"completed_at,type:BIGINT,nullzero"`
	ExpiresAt      *int64              `json:"expiresAt"      bun:"expires_at,type:BIGINT,nullzero"`
	Score          decimal.NullDecimal `json:"score"          bun:"score,type:NUMERIC(5,2),nullzero"`
	Passed         *bool               `json:"passed"         bun:"passed,type:BOOLEAN,nullzero"`
	AcknowledgedAt *int64              `json:"acknowledgedAt" bun:"acknowledged_at,type:BIGINT,nullzero"`
	DocumentID     pulid.ID            `json:"documentId"     bun:"document_id,type:VARCHAR(100),nullzero"`
	AssignedByID   pulid.ID            `json:"assignedById"   bun:"assigned_by_id,type:VARCHAR(100),nullzero"`
	RecordedByID   pulid.ID            `json:"recordedById"   bun:"recorded_by_id,type:VARCHAR(100),nullzero"`
	Notes          string              `json:"notes"          bun:"notes,type:TEXT,nullzero"`
	WaivedReason   string              `json:"waivedReason"   bun:"waived_reason,type:VARCHAR(255),nullzero"`
	Version        int64               `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Course     *TrainingCourse    `json:"course,omitempty"     bun:"rel:belongs-to,join:course_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker     *Worker            `json:"worker,omitempty"     bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Document   *document.Document `json:"document,omitempty"   bun:"rel:belongs-to,join:document_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	AssignedBy *tenant.User       `json:"assignedBy,omitempty" bun:"rel:belongs-to,join:assigned_by_id=id"`
	RecordedBy *tenant.User       `json:"recordedBy,omitempty" bun:"rel:belongs-to,join:recorded_by_id=id"`
}

func (r *WorkerTrainingRecord) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.CourseID, validation.Required.Error("Course is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TrainingStatus]("status is not a known training status"),
		),
		validation.Field(&r.AssignedAt,
			validation.Required.Error("Assigned date is required"),
		),
		validation.Field(&r.WaivedReason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if r.DueAt != nil && *r.DueAt <= 0 {
		multiErr.Add("dueAt", errortypes.ErrInvalid, "Due date must be a valid date")
	}
	if r.CompletedAt != nil && *r.CompletedAt <= 0 {
		multiErr.Add("completedAt", errortypes.ErrInvalid, "Completion date must be a valid date")
	}
	if r.Score.Valid {
		score := r.Score.Decimal
		if score.IsNegative() || score.GreaterThan(decimal.NewFromInt(100)) {
			multiErr.Add("score", errortypes.ErrInvalid, "Score must be between 0 and 100")
		}
	}
	if r.Status == TrainingStatusWaived && strings.TrimSpace(r.WaivedReason) == "" {
		multiErr.Add("waivedReason", errortypes.ErrRequired, "Say why the course is waived")
	}
}

func (r *WorkerTrainingRecord) IsOpen() bool { return r.Status.IsOpen() }

func (r *WorkerTrainingRecord) IsAcknowledged() bool {
	return r.AcknowledgedAt != nil && *r.AcknowledgedAt > 0
}

// Health grades the record on its own; the summary uses the same rule and
// adds Missing for required courses with no record.
func (r *WorkerTrainingRecord) Health(now int64) TrainingHealth {
	window := int32(30)
	if r.Course != nil {
		window = r.Course.RenewalWindowDays
	}
	return EvaluateTrainingHealth(r, window, now)
}

func (r *WorkerTrainingRecord) DaysUntilDue(now int64) *int64 {
	if r.DueAt == nil || *r.DueAt <= 0 || !r.IsOpen() {
		return nil
	}
	days := DaysUntil(*r.DueAt, now)
	return &days
}

func (r *WorkerTrainingRecord) DaysUntilExpiry(now int64) *int64 {
	if r.ExpiresAt == nil || *r.ExpiresAt <= 0 {
		return nil
	}
	days := DaysUntil(*r.ExpiresAt, now)
	return &days
}

// EvaluateTrainingHealth is the pure grading rule shared by the summary, the
// portal and the reminder sweep.
func EvaluateTrainingHealth(
	record *WorkerTrainingRecord,
	renewalWindowDays int32,
	now int64,
) TrainingHealth {
	if record == nil {
		return TrainingHealthMissing
	}
	switch record.Status {
	case TrainingStatusAssigned, TrainingStatusInProgress:
		if record.DueAt == nil || *record.DueAt <= 0 {
			return TrainingHealthScheduled
		}
		days := DaysUntil(*record.DueAt, now)
		switch {
		case days < 0:
			return TrainingHealthOverdue
		case days <= dueSoonDays:
			return TrainingHealthDueSoon
		default:
			return TrainingHealthScheduled
		}
	case TrainingStatusCompleted:
		if record.ExpiresAt == nil || *record.ExpiresAt <= 0 {
			return TrainingHealthCurrent
		}
		days := DaysUntil(*record.ExpiresAt, now)
		switch {
		case days < 0:
			return TrainingHealthExpired
		case days <= int64(renewalWindowDays):
			return TrainingHealthExpiringSoon
		default:
			return TrainingHealthCurrent
		}
	case TrainingStatusWaived:
		return TrainingHealthCurrent
	case TrainingStatusExpired:
		return TrainingHealthExpired
	case TrainingStatusFailed:
		return TrainingHealthFailed
	case TrainingStatusCancelled:
		return TrainingHealthMissing
	default:
		return TrainingHealthMissing
	}
}

func (r *WorkerTrainingRecord) GetID() pulid.ID { return r.ID }

func (r *WorkerTrainingRecord) GetCreatedAt() int64 { return r.CreatedAt }

func (r *WorkerTrainingRecord) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *WorkerTrainingRecord) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *WorkerTrainingRecord) GetTableName() string { return "worker_training_records" }

func (r *WorkerTrainingRecord) GetResourceType() string { return "worker_training" }

func (r *WorkerTrainingRecord) GetResourceID() string { return r.ID.String() }

func (r *WorkerTrainingRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("wtrn_")
		}
		if r.Status == "" {
			r.Status = TrainingStatusAssigned
		}
		if r.AssignedAt == 0 {
			r.AssignedAt = now
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

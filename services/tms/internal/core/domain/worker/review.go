package worker

import (
	"context"
	"errors"
	"strconv"
	"strings"

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
	ErrInvalidReviewStatus = errors.New("invalid performance review status")
	ErrInvalidGoalStatus   = errors.New("invalid review goal status")
)

const (
	reviewRatingMin = 1
	reviewRatingMax = 5
)

// ReviewItem is one thing a review rates; the weight decides its share of
// the overall score.
type ReviewItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Weight      int32  `json:"weight"`
}

var (
	_ bun.BeforeAppendModelHook          = (*PerformanceReviewTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*PerformanceReviewTemplate)(nil)
	_ pagination.CursorEntity            = (*PerformanceReviewTemplate)(nil)
	_ validationframework.TenantedEntity = (*PerformanceReviewTemplate)(nil)
)

type PerformanceReviewTemplate struct {
	bun.BaseModel             `bun:"table:performance_review_templates,alias:prt" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                       json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Code           string             `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Name           string             `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string             `json:"description"    bun:"description,type:TEXT,nullzero"`
	Status         domaintypes.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	IsDefault      bool               `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull"`
	CadenceMonths  *int32             `json:"cadenceMonths"  bun:"cadence_months,type:INTEGER,nullzero"`
	Items          []ReviewItem       `json:"items"          bun:"items,type:JSONB,notnull,default:'[]'"`
	Version        int64              `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64              `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string             `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
}

func (t *PerformanceReviewTemplate) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
	))

	if t.CadenceMonths != nil && *t.CadenceMonths <= 0 {
		multiErr.Add("cadenceMonths", errortypes.ErrInvalid, "Cadence must be at least one month")
	}
	if len(t.Items) == 0 {
		multiErr.Add("items", errortypes.ErrRequired, "Add at least one rating item")
	}
	if t.IsDefault && t.Status != domaintypes.StatusActive {
		multiErr.Add("isDefault", errortypes.ErrInvalid, "Only an active template can be the default")
	}
	seen := make(map[string]struct{}, len(t.Items))
	for i, item := range t.Items {
		prefix := "items[" + strconv.Itoa(i) + "]."
		if strings.TrimSpace(item.Key) == "" {
			multiErr.Add(prefix+"key", errortypes.ErrRequired, "Item key is required")
		} else if _, dup := seen[item.Key]; dup {
			multiErr.Add(prefix+"key", errortypes.ErrDuplicate, "Item keys must be unique")
		}
		seen[item.Key] = struct{}{}
		if strings.TrimSpace(item.Label) == "" {
			multiErr.Add(prefix+"label", errortypes.ErrRequired, "Item label is required")
		}
		if item.Weight <= 0 {
			multiErr.Add(prefix+"weight", errortypes.ErrInvalid, "Weight must be above zero")
		}
	}
}

func (t *PerformanceReviewTemplate) NormalizeCode() {
	t.Code = strings.ToUpper(strings.TrimSpace(t.Code))
}

func (t *PerformanceReviewTemplate) GetID() pulid.ID { return t.ID }

func (t *PerformanceReviewTemplate) GetCreatedAt() int64 { return t.CreatedAt }

func (t *PerformanceReviewTemplate) GetOrganizationID() pulid.ID { return t.OrganizationID }

func (t *PerformanceReviewTemplate) GetBusinessUnitID() pulid.ID { return t.BusinessUnitID }

func (t *PerformanceReviewTemplate) GetTableName() string { return "performance_review_templates" }

func (t *PerformanceReviewTemplate) GetResourceType() string {
	return "performance_review_template"
}

func (t *PerformanceReviewTemplate) GetResourceID() string { return t.ID.String() }

func (t *PerformanceReviewTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "prt",
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

func (t *PerformanceReviewTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("prt_")
		}
		if t.Items == nil {
			t.Items = []ReviewItem{}
		}
		t.CreatedAt = now
		t.UpdatedAt = now
	case *bun.UpdateQuery:
		if t.Items == nil {
			t.Items = []ReviewItem{}
		}
		t.UpdatedAt = now
	}

	return nil
}

type ReviewStatus string

const (
	ReviewStatusDraft        = ReviewStatus("Draft")
	ReviewStatusSubmitted    = ReviewStatus("Submitted")
	ReviewStatusAcknowledged = ReviewStatus("Acknowledged")
	ReviewStatusClosed       = ReviewStatus("Closed")
)

func (s ReviewStatus) String() string { return string(s) }

func (s ReviewStatus) IsValid() bool {
	switch s {
	case ReviewStatusDraft, ReviewStatusSubmitted, ReviewStatusAcknowledged, ReviewStatusClosed:
		return true
	default:
		return false
	}
}

// IsOpen reports whether the review is still waiting on someone.
func (s ReviewStatus) IsOpen() bool {
	return s == ReviewStatusDraft || s == ReviewStatusSubmitted
}

// ReviewRating is the score given to one template item, copied from the
// template at creation so later template edits do not rewrite the review.
type ReviewRating struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Weight  int32  `json:"weight"`
	Score   *int32 `json:"score"`
	Comment string `json:"comment,omitempty"`
}

type ReviewGoalStatus string

const (
	ReviewGoalStatusOpen    = ReviewGoalStatus("Open")
	ReviewGoalStatusDone    = ReviewGoalStatus("Done")
	ReviewGoalStatusDropped = ReviewGoalStatus("Dropped")
)

func (s ReviewGoalStatus) String() string { return string(s) }

func (s ReviewGoalStatus) IsValid() bool {
	switch s {
	case ReviewGoalStatusOpen, ReviewGoalStatusDone, ReviewGoalStatusDropped:
		return true
	default:
		return false
	}
}

type ReviewGoal struct {
	ID     string           `json:"id"`
	Title  string           `json:"title"`
	DueAt  *int64           `json:"dueAt,omitempty"`
	Status ReviewGoalStatus `json:"status"`
}

var (
	_ bun.BeforeAppendModelHook          = (*PerformanceReview)(nil)
	_ validationframework.TenantedEntity = (*PerformanceReview)(nil)
)

type PerformanceReview struct {
	bun.BaseModel `bun:"table:performance_reviews,alias:prev" json:"-"`

	ID             pulid.ID            `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID            `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`
	TemplateID     pulid.ID            `json:"templateId"     bun:"template_id,type:VARCHAR(100),notnull"`
	ReviewerID     pulid.ID            `json:"reviewerId"     bun:"reviewer_id,type:VARCHAR(100),nullzero"`
	Status         ReviewStatus        `json:"status"         bun:"status,type:performance_review_status_enum,notnull,default:'Draft'"`
	Title          string              `json:"title"          bun:"title,type:VARCHAR(120),notnull"`
	PeriodStart    int64               `json:"periodStart"    bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd      int64               `json:"periodEnd"      bun:"period_end,type:BIGINT,notnull"`
	Ratings        []ReviewRating      `json:"ratings"        bun:"ratings,type:JSONB,notnull,default:'[]'"`
	OverallScore   decimal.NullDecimal `json:"overallScore"   bun:"overall_score,type:NUMERIC(4,2),nullzero"`
	Summary        string              `json:"summary"        bun:"summary,type:TEXT,nullzero"`
	Strengths      string              `json:"strengths"      bun:"strengths,type:TEXT,nullzero"`
	Improvements   string              `json:"improvements"   bun:"improvements,type:TEXT,nullzero"`
	Goals          []ReviewGoal        `json:"goals"          bun:"goals,type:JSONB,notnull,default:'[]'"`
	SubmittedAt    *int64              `json:"submittedAt"    bun:"submitted_at,type:BIGINT,nullzero"`
	AcknowledgedAt *int64              `json:"acknowledgedAt" bun:"acknowledged_at,type:BIGINT,nullzero"`
	WorkerComment  string              `json:"workerComment"  bun:"worker_comment,type:TEXT,nullzero"`
	ClosedAt       *int64              `json:"closedAt"       bun:"closed_at,type:BIGINT,nullzero"`
	ClosedByID     pulid.ID            `json:"closedById"     bun:"closed_by_id,type:VARCHAR(100),nullzero"`
	NextReviewAt   *int64              `json:"nextReviewAt"   bun:"next_review_at,type:BIGINT,nullzero"`
	Version        int64               `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker   *Worker                    `json:"worker,omitempty"   bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Template *PerformanceReviewTemplate `json:"template,omitempty" bun:"rel:belongs-to,join:template_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Reviewer *tenant.User               `json:"reviewer,omitempty" bun:"rel:belongs-to,join:reviewer_id=id"`
}

func (r *PerformanceReview) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ReviewStatus]("status must be Draft, Submitted, Acknowledged or Closed"),
		),
		validation.Field(&r.Title,
			validation.Required.Error("Give the review a title"),
			validation.Length(1, 120).Error("Title cannot exceed 120 characters"),
		),
		validation.Field(&r.PeriodStart, validation.Required.Error("Period start is required")),
		validation.Field(&r.PeriodEnd, validation.Required.Error("Period end is required")),
	))

	if r.PeriodEnd < r.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "Period must end after it starts")
	}
	for i, rating := range r.Ratings {
		prefix := "ratings[" + strconv.Itoa(i) + "]."
		if rating.Score != nil && (*rating.Score < reviewRatingMin || *rating.Score > reviewRatingMax) {
			multiErr.Add(prefix+"score", errortypes.ErrInvalid, "Scores run from 1 to 5")
		}
		if len(rating.Comment) > 2000 {
			multiErr.Add(prefix+"comment", errortypes.ErrInvalid, "Comment cannot exceed 2000 characters")
		}
	}
	for i, goal := range r.Goals {
		prefix := "goals[" + strconv.Itoa(i) + "]."
		if strings.TrimSpace(goal.Title) == "" {
			multiErr.Add(prefix+"title", errortypes.ErrRequired, "Goal needs a title")
		}
		if goal.Status != "" && !goal.Status.IsValid() {
			multiErr.Add(prefix+"status", errortypes.ErrInvalid, "Goal status must be Open, Done or Dropped")
		}
	}
}

// ValidateForSubmit adds what a draft may leave blank but a submitted review
// may not: every item rated.
func (r *PerformanceReview) ValidateForSubmit(multiErr *errortypes.MultiError) {
	r.Validate(multiErr)
	if len(r.Ratings) == 0 {
		multiErr.Add("ratings", errortypes.ErrRequired, "Rate at least one item")
	}
	for i, rating := range r.Ratings {
		if rating.Score == nil {
			multiErr.Add(
				"ratings["+strconv.Itoa(i)+"].score",
				errortypes.ErrRequired,
				"Rate {0}", rating.Label,
			)
		}
	}
	if strings.TrimSpace(r.Summary) == "" {
		multiErr.Add("summary", errortypes.ErrRequired, "Write a summary the worker will read")
	}
}

// RatingsFromTemplate copies the template's items so the review keeps its
// own history.
func RatingsFromTemplate(template *PerformanceReviewTemplate) []ReviewRating {
	if template == nil {
		return []ReviewRating{}
	}
	ratings := make([]ReviewRating, 0, len(template.Items))
	for _, item := range template.Items {
		ratings = append(ratings, ReviewRating{Key: item.Key, Label: item.Label, Weight: item.Weight})
	}
	return ratings
}

// ComputeOverallScore is the weighted mean of the rated items on the 1–5
// scale, to two decimals; invalid when nothing has been rated yet.
func ComputeOverallScore(ratings []ReviewRating) decimal.NullDecimal {
	var weighted, weights int64
	for _, rating := range ratings {
		if rating.Score == nil || rating.Weight <= 0 {
			continue
		}
		weighted += int64(*rating.Score) * int64(rating.Weight)
		weights += int64(rating.Weight)
	}
	if weights == 0 {
		return decimal.NullDecimal{}
	}
	score := decimal.NewFromInt(weighted).Div(decimal.NewFromInt(weights)).Round(2)
	return decimal.NewNullDecimal(score)
}

// NextReviewDate is when the next review is due after this one closes.
func (r *PerformanceReview) NextReviewDate(template *PerformanceReviewTemplate) *int64 {
	if template == nil || template.CadenceMonths == nil || *template.CadenceMonths <= 0 {
		return nil
	}
	next := timeutils.AddMonthsUTC(r.PeriodEnd, int(*template.CadenceMonths))
	return &next
}

func (r *PerformanceReview) IsAcknowledged() bool {
	return r.AcknowledgedAt != nil && *r.AcknowledgedAt > 0
}

func (r *PerformanceReview) GetID() pulid.ID { return r.ID }

func (r *PerformanceReview) GetCreatedAt() int64 { return r.CreatedAt }

func (r *PerformanceReview) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *PerformanceReview) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *PerformanceReview) GetTableName() string { return "performance_reviews" }

func (r *PerformanceReview) GetResourceType() string { return "performance_review" }

func (r *PerformanceReview) GetResourceID() string { return r.ID.String() }

func (r *PerformanceReview) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("prev_")
		}
		if r.Status == "" {
			r.Status = ReviewStatusDraft
		}
		if r.Ratings == nil {
			r.Ratings = []ReviewRating{}
		}
		if r.Goals == nil {
			r.Goals = []ReviewGoal{}
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		if r.Ratings == nil {
			r.Ratings = []ReviewRating{}
		}
		if r.Goals == nil {
			r.Goals = []ReviewGoal{}
		}
		r.UpdatedAt = now
	}

	return nil
}

package worker

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
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
	_ domaintypes.PostgresSearchable     = (*PerformanceReviewTemplate)(nil)
	_ pagination.CursorEntity            = (*PerformanceReviewTemplate)(nil)
	_ validationframework.TenantedEntity = (*PerformanceReviewTemplate)(nil)
)

type PerformanceReviewTemplate struct {
	keel.Table     `keel:"performance_review_templates"`
	ID             pulid.ID           `keel:"id,pk,autoid:prt_"                        json:"id"`
	BusinessUnitID pulid.ID           `keel:"business_unit_id,pk,tenant"               json:"businessUnitId"`
	OrganizationID pulid.ID           `keel:"organization_id,pk,tenant"                json:"organizationId"`
	Code           string             `keel:"code"                                     json:"code"`
	Name           string             `keel:"name"                                     json:"name"`
	Description    string             `keel:"description,nullzero"                     json:"description"`
	Status         domaintypes.Status `keel:"status"                                   json:"status"`
	IsDefault      bool               `keel:"is_default"                               json:"isDefault"`
	CadenceMonths  *int32             `keel:"cadence_months"                           json:"cadenceMonths"`
	Items          []ReviewItem       `keel:"items,jsonb"                              json:"items"`
	Version        int64              `keel:"version,version"                          json:"version"`
	CreatedAt      int64              `keel:"created_at,autocreate"                    json:"createdAt"`
	UpdatedAt      int64              `keel:"updated_at,autoupdate"                    json:"updatedAt"`
	SearchVector   string             `keel:"search_vector,tsvector,readonly,noselect" json:"-"`
}

func (t *PerformanceReviewTemplate) BeforeUpdate(_ context.Context) error {
	if t.Items == nil {
		t.Items = []ReviewItem{}
	}

	return nil
}

func (t *PerformanceReviewTemplate) BeforeCreate(_ context.Context) error {
	if t.Items == nil {
		t.Items = []ReviewItem{}
	}
	if t.Status == "" {
		t.Status = domaintypes.StatusActive
	}

	return nil
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
		multiErr.Add(
			"isDefault",
			errortypes.ErrInvalid,
			"Only an active template can be the default",
		)
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

func (t *PerformanceReviewTemplate) BeforeSave(_ context.Context) error {
	if t.Items == nil {
		t.Items = []ReviewItem{}
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

func ReviewGoalStatusValues() []ReviewGoalStatus {
	return []ReviewGoalStatus{
		ReviewGoalStatusOpen,
		ReviewGoalStatusDone,
		ReviewGoalStatusDropped,
	}
}

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
	_ validationframework.TenantedEntity = (*PerformanceReview)(nil)
)

type PerformanceReview struct {
	keel.Table `keel:"performance_reviews"`

	ID             pulid.ID            `json:"id"             keel:"id,pk,autoid:prev_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	WorkerID       pulid.ID            `json:"workerId"       keel:"worker_id"`
	TemplateID     pulid.ID            `json:"templateId"     keel:"template_id"`
	ReviewerID     pulid.ID            `json:"reviewerId"     keel:"reviewer_id,nullzero"`
	Status         ReviewStatus        `json:"status"         keel:"status"`
	Title          string              `json:"title"          keel:"title"`
	PeriodStart    int64               `json:"periodStart"    keel:"period_start"`
	PeriodEnd      int64               `json:"periodEnd"      keel:"period_end"`
	Ratings        []ReviewRating      `json:"ratings"        keel:"ratings,jsonb"`
	OverallScore   decimal.NullDecimal `json:"overallScore"   keel:"overall_score"`
	Summary        string              `json:"summary"        keel:"summary,nullzero"`
	Strengths      string              `json:"strengths"      keel:"strengths,nullzero"`
	Improvements   string              `json:"improvements"   keel:"improvements,nullzero"`
	Goals          []ReviewGoal        `json:"goals"          keel:"goals,jsonb"`
	SubmittedAt    *int64              `json:"submittedAt"    keel:"submitted_at"`
	AcknowledgedAt *int64              `json:"acknowledgedAt" keel:"acknowledged_at"`
	WorkerComment  string              `json:"workerComment"  keel:"worker_comment,nullzero"`
	ClosedAt       *int64              `json:"closedAt"       keel:"closed_at"`
	ClosedByID     pulid.ID            `json:"closedById"     keel:"closed_by_id,nullzero"`
	NextReviewAt   *int64              `json:"nextReviewAt"   keel:"next_review_at"`
	Version        int64               `json:"version"        keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Worker   *Worker                    `json:"worker,omitempty"   keel:"belongs:worker_id"`
	Template *PerformanceReviewTemplate `json:"template,omitempty" keel:"belongs:template_id"`
	Reviewer *tenant.User               `json:"reviewer,omitempty" keel:"belongs:reviewer_id"`
}

func (r *PerformanceReview) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(&r.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(
			&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ReviewStatus](
				"status must be Draft, Submitted, Acknowledged or Closed",
			),
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
		if rating.Score != nil &&
			(*rating.Score < reviewRatingMin || *rating.Score > reviewRatingMax) {
			multiErr.Add(prefix+"score", errortypes.ErrInvalid, "Scores run from 1 to 5")
		}
		if len(rating.Comment) > 2000 {
			multiErr.Add(
				prefix+"comment",
				errortypes.ErrInvalid,
				"Comment cannot exceed 2000 characters",
			)
		}
	}
	for i, goal := range r.Goals {
		prefix := "goals[" + strconv.Itoa(i) + "]."
		if strings.TrimSpace(goal.Title) == "" {
			multiErr.Add(prefix+"title", errortypes.ErrRequired, "Goal needs a title")
		}
		if goal.Status != "" && !goal.Status.IsValid() {
			multiErr.Add(
				prefix+"status",
				errortypes.ErrInvalid,
				"Goal status must be Open, Done or Dropped",
			)
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
		ratings = append(
			ratings,
			ReviewRating{Key: item.Key, Label: item.Label, Weight: item.Weight},
		)
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

func (r *PerformanceReview) BeforeCreate(_ context.Context) error {
	if r.Status == "" {
		r.Status = ReviewStatusDraft
	}
	if r.Ratings == nil {
		r.Ratings = []ReviewRating{}
	}
	if r.Goals == nil {
		r.Goals = []ReviewGoal{}
	}

	return nil
}

func (r *PerformanceReview) BeforeUpdate(_ context.Context) error {
	if r.Ratings == nil {
		r.Ratings = []ReviewRating{}
	}
	if r.Goals == nil {
		r.Goals = []ReviewGoal{}
	}

	return nil
}

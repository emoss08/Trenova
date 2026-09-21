// Package insight holds operational findings surfaced on the home screen.
//
// The central rule of this package is the line between what was computed and
// what was written. A detector runs a query and produces exact numbers; a model
// is then asked to explain them in a sentence. Metrics, Links, Severity and
// Category come from the detector and are facts. Headline, Narrative and
// Recommendation are prose a model wrote about those facts, and an insight whose
// prose could not be produced or could not be trusted still has every number it
// started with. Nothing here lets generated text become a number a person acts
// on.
package insight

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Insight)(nil)
	_ validationframework.TenantedEntity = (*Insight)(nil)
	_ pagination.CursorEntity            = (*Insight)(nil)
	_ domaintypes.PostgresSearchable     = (*Insight)(nil)
)

const (
	// MaxHeadlineLength and MaxNarrativeLength bound generated prose. A model
	// asked for a sentence can return an essay, and an insight card has a fixed
	// amount of room on a home screen.
	MaxHeadlineLength       = 160
	MaxNarrativeLength      = 1200
	MaxRecommendationLength = 400
	// MaxMetrics bounds what one insight can carry. A card a person scans in two
	// seconds cannot hold twenty numbers, and a detector wanting more than this is
	// really two detectors.
	MaxMetrics = 8
	MaxLinks   = 4
)

// Metric is one number a detector computed. The value is exact and is never
// read back out of model output.
type Metric struct {
	Key   string          `json:"key"`
	Label string          `json:"label"`
	Value decimal.Decimal `json:"value"`
	Unit  Unit            `json:"unit"`
	// Direction lets a client colour a change without knowing what the metric
	// means. Rising revenue and rising dwell are not the same news.
	Direction Direction `json:"direction"`
	// Baseline is the comparison value where the detector computed one — the
	// prior period, the fleet average, the contracted target. Nil means the
	// metric stands alone and no change should be drawn.
	Baseline *decimal.Decimal `json:"baseline,omitempty"`
	// BaselineLabel names what Baseline is, because "vs 94%" is meaningless
	// without knowing whether that is last month or the target.
	BaselineLabel string `json:"baselineLabel,omitempty"`
}

// Link points at the records a finding was computed from.
//
// The path is always built by the detector. Nothing a model returns ever
// reaches it: a generated URL on a card a person is invited to click is an open
// redirect waiting to happen, and there is no reason to accept one when the
// detector already knows exactly which records it counted.
type Link struct {
	Label string `json:"label"`
	// Path is an in-application route, always relative and rooted.
	Path string `json:"path"`
	// Count is how many records are behind the link, so the label can say so
	// without the client fetching them.
	Count int `json:"count"`
}

// IsSafe reports whether a link points inside this application.
//
// A path must be rooted and must not begin a scheme or a protocol-relative URL.
// This is belt-and-braces over the rule that detectors build every path, and it
// is enforced at validation so a bad path cannot be stored rather than merely
// not rendered.
func (l Link) IsSafe() bool {
	return stringutils.IsSafeAppPath(l.Path)
}

type Insight struct {
	bun.BaseModel `bun:"table:insights,alias:ins" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// DetectorKey names the rule that produced this, and is the join back to the
	// code that computed the numbers. It is stable across releases so a detector
	// can be turned off without orphaning what it already found.
	DetectorKey string   `json:"detectorKey" bun:"detector_key,type:VARCHAR(100),notnull"`
	Category    Category `json:"category"    bun:"category,type:VARCHAR(50),notnull"`
	Severity    Severity `json:"severity"    bun:"severity,type:VARCHAR(50),notnull"`
	Status      Status   `json:"status"      bun:"status,type:VARCHAR(50),notnull,default:'Active'"`

	// DedupeKey identifies the same finding across refreshes. Detention at one
	// location is the same finding this week as last week with different numbers,
	// and a person who dismissed it should not see it again tomorrow.
	DedupeKey string `json:"dedupeKey" bun:"dedupe_key,type:VARCHAR(255),notnull"`
	// Subject is what the finding is about in the reader's language — a customer
	// name, a location, a lane.
	Subject string `json:"subject" bun:"subject,type:VARCHAR(255),nullzero"`

	// Headline, Narrative and Recommendation are generated prose. Narrated says
	// whether a model actually wrote them: when it is false these hold the
	// detector's own deterministic wording, which is plainer but never wrong.
	Headline       string `json:"headline"       bun:"headline,type:TEXT,notnull"`
	Narrative      string `json:"narrative"      bun:"narrative,type:TEXT,nullzero"`
	Recommendation string `json:"recommendation" bun:"recommendation,type:TEXT,nullzero"`
	Narrated       bool   `json:"narrated"       bun:"narrated,type:BOOLEAN,notnull,default:false"`

	// Metrics and Links are the detector's output and are authoritative.
	Metrics []Metric `json:"metrics" bun:"metrics,type:JSONB,notnull,default:'[]'"`
	Links   []Link   `json:"links"   bun:"links,type:JSONB,notnull,default:'[]'"`

	// WindowStart and WindowEnd are the period the numbers describe. Showing a
	// figure without its window invites someone to read a month's total as a day's.
	WindowStart int64 `json:"windowStart" bun:"window_start,type:BIGINT,notnull,default:0"`
	WindowEnd   int64 `json:"windowEnd"   bun:"window_end,type:BIGINT,notnull"`
	DetectedAt  int64 `json:"detectedAt"  bun:"detected_at,type:BIGINT,notnull"`
	// StaleAt is when these numbers stop being worth trusting. The client says so
	// rather than hiding the insight, because a stale finding is still evidence
	// that something was wrong.
	StaleAt int64 `json:"staleAt" bun:"stale_at,type:BIGINT,notnull"`

	DismissedAt   *int64   `json:"dismissedAt"   bun:"dismissed_at,type:BIGINT,nullzero"`
	DismissedByID pulid.ID `json:"dismissedById" bun:"dismissed_by_id,type:VARCHAR(100),nullzero"`
	DismissReason string   `json:"dismissReason" bun:"dismiss_reason,type:TEXT,nullzero"`

	// ModelIdentifier and ProviderID attribute the prose. They are empty on an
	// insight that was never narrated, which is exactly the signal that its
	// wording came from the detector.
	ModelIdentifier string   `json:"modelIdentifier" bun:"model_identifier,type:VARCHAR(255),nullzero"`
	ProviderID      pulid.ID `json:"providerId"      bun:"provider_id,type:VARCHAR(100),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (i *Insight) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		i,
		validation.Field(&i.DetectorKey, validation.Required.Error("Detector key is required")),
		validation.Field(&i.DedupeKey, validation.Required.Error("Dedupe key is required")),
		validation.Field(&i.Category,
			validation.Required.Error("Category is required"),
			domainvalidation.ValidEnum[Category]("Invalid category"),
		),
		validation.Field(&i.Severity,
			validation.Required.Error("Severity is required"),
			domainvalidation.ValidEnum[Severity]("Invalid severity"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Invalid status"),
		),
		validation.Field(&i.Headline,
			validation.Required.Error("Headline is required"),
			validation.Length(0, MaxHeadlineLength).Error(
				"Headline cannot be longer than 160 characters",
			),
		),
		validation.Field(&i.Narrative,
			validation.Length(0, MaxNarrativeLength).Error(
				"Narrative cannot be longer than 1200 characters",
			),
		),
		validation.Field(&i.Recommendation,
			validation.Length(0, MaxRecommendationLength).Error(
				"Recommendation cannot be longer than 400 characters",
			),
		),
		validation.Field(&i.WindowEnd,
			validation.Required.Error("Window end is required"),
		),
	))

	i.validateWindow(multiErr)
	i.validateMetrics(multiErr)
	i.validateLinks(multiErr)
}

func (i *Insight) validateWindow(multiErr *errortypes.MultiError) {
	if i.WindowStart > 0 && i.WindowEnd > 0 && i.WindowStart > i.WindowEnd {
		multiErr.Add(
			"windowStart",
			errortypes.ErrInvalid,
			"Window start must not be after window end",
		)
	}
}

func (i *Insight) validateMetrics(multiErr *errortypes.MultiError) {
	if len(i.Metrics) > MaxMetrics {
		multiErr.Add(
			"metrics",
			errortypes.ErrInvalid,
			"An insight cannot carry more than 8 metrics",
		)

		return
	}

	for index, metric := range i.Metrics {
		scoped := multiErr.WithIndex("metrics", index)

		if strings.TrimSpace(metric.Key) == "" {
			scoped.Add("key", errortypes.ErrRequired, "Metric key is required")
		}
		if strings.TrimSpace(metric.Label) == "" {
			scoped.Add("label", errortypes.ErrRequired, "Metric label is required")
		}
		if !metric.Unit.IsValid() {
			scoped.Add("unit", errortypes.ErrInvalid, "Invalid metric unit")
		}
		if !metric.Direction.IsValid() {
			scoped.Add("direction", errortypes.ErrInvalid, "Invalid metric direction")
		}
	}
}

func (i *Insight) validateLinks(multiErr *errortypes.MultiError) {
	if len(i.Links) > MaxLinks {
		multiErr.Add("links", errortypes.ErrInvalid, "An insight cannot carry more than 4 links")

		return
	}

	for index, link := range i.Links {
		scoped := multiErr.WithIndex("links", index)

		if strings.TrimSpace(link.Label) == "" {
			scoped.Add("label", errortypes.ErrRequired, "Link label is required")
		}
		if !link.IsSafe() {
			scoped.Add(
				"path",
				errortypes.ErrInvalid,
				"Link must be a path inside this application",
			)
		}
	}
}

// IsStale reports whether the numbers have aged past their refresh window.
func (i *Insight) IsStale(now int64) bool {
	return i.StaleAt > 0 && now > i.StaleAt
}

// MetricByKey finds a computed metric. It exists so callers read numbers from
// the detector's output by name rather than by position, which changes.
func (i *Insight) MetricByKey(key string) (Metric, bool) {
	for _, metric := range i.Metrics {
		if metric.Key == key {
			return metric, true
		}
	}

	return Metric{}, false
}

func (i *Insight) GetID() pulid.ID {
	return i.ID
}

func (i *Insight) GetCreatedAt() int64 {
	return i.CreatedAt
}

func (i *Insight) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *Insight) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *Insight) GetTableName() string {
	return "insights"
}

func (i *Insight) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ins",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "headline", Type: domaintypes.FieldTypeText},
			{Name: "subject", Type: domaintypes.FieldTypeText},
			{Name: "detector_key", Type: domaintypes.FieldTypeText},
			{Name: "category", Type: domaintypes.FieldTypeEnum},
			{Name: "severity", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (i *Insight) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("inst_")
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

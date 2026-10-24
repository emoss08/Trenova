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

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
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
	keel.Table `keel:"insights"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:inst_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	// DetectorKey names the rule that produced this, and is the join back to the
	// code that computed the numbers. It is stable across releases so a detector
	// can be turned off without orphaning what it already found.
	DetectorKey string   `json:"detectorKey" keel:"detector_key"`
	Category    Category `json:"category" keel:"category"`
	Severity    Severity `json:"severity" keel:"severity"`
	Status      Status   `json:"status" keel:"status"`

	// DedupeKey identifies the same finding across refreshes. Detention at one
	// location is the same finding this week as last week with different numbers,
	// and a person who dismissed it should not see it again tomorrow.
	DedupeKey string `json:"dedupeKey" keel:"dedupe_key"`
	// Subject is what the finding is about in the reader's language — a customer
	// name, a location, a lane.
	Subject string `json:"subject" keel:"subject,nullzero"`

	// Headline, Narrative and Recommendation are generated prose. Narrated says
	// whether a model actually wrote them: when it is false these hold the
	// detector's own deterministic wording, which is plainer but never wrong.
	Headline       string `json:"headline" keel:"headline"`
	Narrative      string `json:"narrative" keel:"narrative,nullzero"`
	Recommendation string `json:"recommendation" keel:"recommendation,nullzero"`
	Narrated       bool   `json:"narrated" keel:"narrated"`

	// Metrics and Links are the detector's output and are authoritative.
	Metrics []Metric `json:"metrics" keel:"metrics,jsonb"`
	Links   []Link   `json:"links" keel:"links,jsonb"`

	// WindowStart and WindowEnd are the period the numbers describe. Showing a
	// figure without its window invites someone to read a month's total as a day's.
	WindowStart int64 `json:"windowStart" keel:"window_start"`
	WindowEnd   int64 `json:"windowEnd" keel:"window_end"`
	DetectedAt  int64 `json:"detectedAt" keel:"detected_at"`
	// StaleAt is when these numbers stop being worth trusting. The client says so
	// rather than hiding the insight, because a stale finding is still evidence
	// that something was wrong.
	StaleAt int64 `json:"staleAt" keel:"stale_at"`

	DismissedAt   *int64   `json:"dismissedAt" keel:"dismissed_at"`
	DismissedByID pulid.ID `json:"dismissedById" keel:"dismissed_by_id,nullzero"`
	DismissReason string   `json:"dismissReason" keel:"dismiss_reason,nullzero"`

	// ModelIdentifier and ProviderID attribute the prose. They are empty on an
	// insight that was never narrated, which is exactly the signal that its
	// wording came from the detector.
	ModelIdentifier string   `json:"modelIdentifier" keel:"model_identifier,nullzero"`
	ProviderID      pulid.ID `json:"providerId" keel:"provider_id,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (i *Insight) BeforeCreate(ctx context.Context) error {
	if i.Status == "" {
		i.Status = StatusActive
	}

	return nil
}

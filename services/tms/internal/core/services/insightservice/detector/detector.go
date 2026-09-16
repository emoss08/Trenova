// Package detector holds the deterministic half of insight generation.
//
// Every number a person reads on an insight card originates here, from a query,
// and a detector runs with no knowledge that a model exists. That separation is
// the reason the feature is safe to put on a home screen: narration can fail, be
// disabled, be served by a small local model, or be rejected for citing a figure
// nobody computed, and the finding underneath is unchanged.
//
// A detector answers one question about one window and returns zero or more
// findings. It never decides how loudly to speak relative to other detectors,
// and it never writes a paragraph.
package detector

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

// Params is what every detector is given: which tenant, and over what period.
type Params struct {
	TenantInfo pagination.TenantInfo
	// WindowStart and WindowEnd bound the period under examination, as Unix
	// seconds. A detector comparing against a prior period derives that itself,
	// because only it knows whether the comparison should be the previous window,
	// the same window last year, or a contracted target.
	WindowStart int64
	WindowEnd   int64
	// Timezone is the organization's reporting timezone. Day boundaries decide
	// whether a 23:50 delivery was late on Tuesday or early on Wednesday, so a
	// detector that buckets by day must use this rather than UTC.
	Timezone string
}

// WindowDays is the length of the examined period in whole days, for wording
// like "over the last 30 days". It is computed rather than passed so it can
// never disagree with the window the numbers came from.
func (p Params) WindowDays() int {
	if p.WindowEnd <= p.WindowStart {
		return 0
	}

	const secondsPerDay = 86400

	return int((p.WindowEnd - p.WindowStart) / secondsPerDay)
}

// Finding is one thing a detector noticed. It is a complete, honest record on
// its own: the headline is plain but true, and narration only ever replaces the
// wording.
type Finding struct {
	// DedupeKey identifies this finding across refreshes. It must be stable for
	// the same real-world condition and distinct between conditions, because it is
	// what makes numbers update in place and a dismissal stick.
	DedupeKey string
	// Subject names what the finding is about in the reader's language.
	Subject string
	// Headline is the detector's own wording, used as-is when no model narrates
	// and as the fallback when narration is rejected.
	Headline string
	Severity insight.Severity
	Metrics  []insight.Metric
	Links    []insight.Link
}

// Explanation says what a rule looks for, in the reader's language.
//
// It exists so a person can audit why they are being shown something. A card
// that asserts a customer's service has slipped is only worth acting on if the
// reader can find out what "slipped" meant and what the rule declined to report
// — and that last part is the one that builds trust, because it is where a
// detector admits what it cannot see.
type Explanation struct {
	// Measures is what the detector computes, in one sentence.
	Measures string
	// Threshold is what has to be true before it says anything at all.
	Threshold string
	// Excludes is what it deliberately passes over, and why. A reader who knows
	// a rule ignores low-volume customers can tell "no finding" apart from "no
	// problem", which are very different things.
	Excludes string
}

// Detector is one rule. Implementations live beside this file and are registered
// with a Registry.
type Detector interface {
	// Key is the stable identifier stored on every insight this produces.
	Key() string
	Category() insight.Category
	// Resource and Operation are the permission a reader needs before this
	// detector's findings may be shown to them. An insight about customer
	// profitability must not reach someone who cannot read customers.
	Resource() permission.Resource
	Operation() permission.Operation
	// Surfaces names the working pages this detector's findings belong on. A
	// finding may matter to more than one desk; an empty list means it is only
	// ever shown on the home screen and the insights page.
	Surfaces() []insight.Surface
	// Explain describes the rule for a person reading one of its findings.
	Explain() Explanation
	// Detect runs the queries. Returning no findings is the normal, healthy case
	// and is not an error.
	Detect(ctx context.Context, params Params) ([]Finding, error)
}

// Registry holds the detectors a deployment runs.
//
// Order is preserved rather than map-random so a refresh produces findings in a
// predictable sequence, which makes a run reproducible and a failure easy to
// attribute.
type Registry struct {
	detectors []Detector
	byKey     map[string]Detector
}

func NewRegistry(detectors ...Detector) *Registry {
	registry := &Registry{
		detectors: make([]Detector, 0, len(detectors)),
		byKey:     make(map[string]Detector, len(detectors)),
	}

	for _, d := range detectors {
		if d == nil {
			continue
		}
		// A duplicate key would make two detectors fight over the same dedupe
		// space, silently superseding each other's findings on every refresh.
		if _, exists := registry.byKey[d.Key()]; exists {
			continue
		}
		registry.byKey[d.Key()] = d
		registry.detectors = append(registry.detectors, d)
	}

	return registry
}

func (r *Registry) All() []Detector {
	return r.detectors
}

func (r *Registry) Get(key string) (Detector, bool) {
	d, ok := r.byKey[key]

	return d, ok
}

// KeysForSurface lists the detectors whose findings belong on one page, in
// registration order. It is the page's half of the read: the permission filter
// is the reader's half, and the query takes the intersection.
func (r *Registry) KeysForSurface(surface insight.Surface) []string {
	keys := make([]string, 0, len(r.detectors))

	for _, d := range r.detectors {
		for _, candidate := range d.Surfaces() {
			if candidate == surface {
				keys = append(keys, d.Key())

				break
			}
		}
	}

	return keys
}

// Validate reports what is wrong with a finding.
//
// A detector that returns a malformed finding is a bug, and the run drops that
// one finding rather than failing the refresh or storing something unreadable.
func (f Finding) Validate() error {
	if strings.TrimSpace(f.DedupeKey) == "" {
		return fmt.Errorf("%w: dedupe key is empty", ErrMalformedFinding)
	}

	if strings.TrimSpace(f.Headline) == "" {
		return fmt.Errorf("%w: %s has no headline", ErrMalformedFinding, f.DedupeKey)
	}

	if !f.Severity.IsValid() {
		return fmt.Errorf(
			"%w: %s has severity %q",
			ErrMalformedFinding,
			f.DedupeKey,
			f.Severity,
		)
	}

	if len(f.Metrics) == 0 {
		// A finding with no numbers is an opinion. There is nothing for a reader
		// to check and nothing to stop narration inventing a figure, so it is
		// refused rather than shown.
		return fmt.Errorf("%w: %s carries no metrics", ErrMalformedFinding, f.DedupeKey)
	}

	return f.validateParts()
}

func (f Finding) validateParts() error {
	for _, metric := range f.Metrics {
		if strings.TrimSpace(metric.Key) == "" || strings.TrimSpace(metric.Label) == "" {
			return fmt.Errorf("%w: %s has an unlabelled metric", ErrMalformedFinding, f.DedupeKey)
		}
		if !metric.Unit.IsValid() || !metric.Direction.IsValid() {
			return fmt.Errorf(
				"%w: %s metric %q has an invalid unit or direction",
				ErrMalformedFinding,
				f.DedupeKey,
				metric.Key,
			)
		}
	}

	for _, link := range f.Links {
		if !link.IsSafe() {
			return fmt.Errorf(
				"%w: %s links outside the application",
				ErrMalformedFinding,
				f.DedupeKey,
			)
		}
	}

	return nil
}

// MetricValues maps each metric key to its value, which is what a narrator needs
// to describe the finding without being handed the entity.
func (f Finding) MetricValues() map[string]decimal.Decimal {
	values := make(map[string]decimal.Decimal, len(f.Metrics))
	for _, metric := range f.Metrics {
		values[metric.Key] = metric.Value
	}

	return values
}

package sim

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type statKind uint8

const (
	statContinuous statKind = iota
	statEvent
)

const (
	statEventLookback = 30 * 24 * time.Hour
	fieldDecorations  = "decorations"
	fieldTime         = "time"
)

type statSpec struct {
	Type                string
	SnapshotKey         string
	FeedKey             string
	Kind                statKind
	Group               string
	Decoration          bool
	DecorationKeepsTime bool
	Sample              func(ctx *statContext, at time.Time) map[string]any
	EventTimes          func(ctx *statContext, from, to time.Time) []time.Time
}

type statRegistry struct {
	name           string
	specs          map[string]*statSpec
	order          []string
	decorations    []string
	maxTypes       int
	maxDecorations int
}

type statContext struct {
	view      *fleetView
	asset     Record
	assetID   string
	track     *positionTrack
	template  Record
	positions map[int64]routeState
	hasFix    map[int64]bool
	scratch   map[string]any
}

func newStatRegistry(
	name string,
	maxTypes int,
	maxDecorations int,
	specs []*statSpec,
	decorations []string,
) *statRegistry {
	registry := &statRegistry{
		name:           name,
		specs:          make(map[string]*statSpec, len(specs)),
		order:          make([]string, 0, len(specs)),
		decorations:    decorations,
		maxTypes:       maxTypes,
		maxDecorations: maxDecorations,
	}
	for _, spec := range specs {
		if spec.Group == "" {
			spec.Group = spec.Type
		}
		registry.specs[spec.Type] = spec
		registry.order = append(registry.order, spec.Type)
	}
	return registry
}

func (r *statRegistry) parseTypes(raw []string) ([]*statSpec, error) {
	if len(raw) == 0 {
		return nil, ErrStatTypesRequired
	}
	out := make([]*statSpec, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	groups := make(map[string]struct{}, len(raw))
	for _, name := range raw {
		spec, ok := r.specs[name]
		if !ok {
			return nil, fmt.Errorf(
				"%w: %q is not a valid %s stat type",
				ErrStatTypeInvalid,
				name,
				r.name,
			)
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		groups[spec.Group] = struct{}{}
		out = append(out, spec)
	}
	if len(groups) > r.maxTypes {
		return nil, invalidParameter(
			"types",
			fmt.Sprintf("you may list up to %d types", r.maxTypes),
		)
	}
	return out, nil
}

func (r *statRegistry) parseDecorations(raw []string) ([]*statSpec, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]*statSpec, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, name := range raw {
		spec, ok := r.specs[name]
		if !ok || !spec.Decoration {
			return nil, invalidParameter(
				"decorations",
				fmt.Sprintf("%q is not a valid %s decoration", name, r.name),
			)
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, spec)
	}
	if len(out) > r.maxDecorations {
		return nil, invalidParameter(
			"decorations",
			fmt.Sprintf("you may list up to %d decorations", r.maxDecorations),
		)
	}
	return out, nil
}

func (v *fleetView) newStatContext(asset Record, windowStart, windowEnd time.Time) *statContext {
	assetID := recordID(asset)
	return &statContext{
		view:      v,
		asset:     asset,
		assetID:   assetID,
		track:     v.trackFor(assetID, windowStart, windowEnd),
		template:  v.snap.templates[assetID],
		positions: map[int64]routeState{},
		hasFix:    map[int64]bool{},
		scratch:   map[string]any{},
	}
}

func (c *statContext) position(at time.Time) (routeState, bool) {
	key := at.UnixNano()
	if known, ok := c.hasFix[key]; ok {
		return c.positions[key], known
	}
	state, ok := c.view.live.trackStateAt(c.track, at)
	c.positions[key] = state
	c.hasFix[key] = ok
	return state, ok
}

func (c *statContext) hash(parts ...string) float64 {
	return c.view.live.hashFraction(append([]string{c.assetID}, parts...)...)
}

func anyValue[T any](value T, ok bool) (any, bool) {
	if !ok {
		return nil, false
	}
	return value, true
}

func stampSample(sample map[string]any, at time.Time) map[string]any {
	sample[fieldTime] = formatSampleTime(at)
	return sample
}

func lastEventAtOrBefore(ctx *statContext, spec *statSpec, at time.Time) (time.Time, bool) {
	events := spec.EventTimes(ctx, at.Add(-statEventLookback), at)
	if len(events) == 0 {
		return time.Time{}, false
	}
	return events[len(events)-1], true
}

func snapshotStats(ctx *statContext, specs []*statSpec, at time.Time, row Record) {
	for _, spec := range specs {
		sampleAt := at
		if spec.Kind == statEvent {
			eventAt, ok := lastEventAtOrBefore(ctx, spec, at)
			if !ok {
				continue
			}
			sampleAt = eventAt
		}
		sample := spec.Sample(ctx, sampleAt)
		if sample == nil {
			continue
		}
		row[spec.SnapshotKey] = stampSample(sample, sampleAt)
	}
}

type feedWindow struct {
	Times      []time.Time
	EventFrom  time.Time
	EventTo    time.Time
	LatestOnly bool
}

func feedStats(
	ctx *statContext,
	specs []*statSpec,
	decorations []*statSpec,
	window *feedWindow,
	row Record,
) {
	for _, spec := range specs {
		samples := make([]any, 0, len(window.Times))
		for _, at := range sampleTimesFor(ctx, spec, window) {
			sample := spec.Sample(ctx, at)
			if sample == nil {
				continue
			}
			stampSample(sample, at)
			if decorated := decorationValues(ctx, decorations, at); len(decorated) > 0 {
				sample[fieldDecorations] = decorated
			}
			samples = append(samples, sample)
		}
		row[spec.FeedKey] = samples
	}
}

func sampleTimesFor(ctx *statContext, spec *statSpec, window *feedWindow) []time.Time {
	if spec.Kind != statEvent {
		return window.Times
	}
	if window.LatestOnly {
		if at, ok := lastEventAtOrBefore(ctx, spec, window.EventTo); ok {
			return []time.Time{at}
		}
		return nil
	}
	events := spec.EventTimes(ctx, window.EventFrom, window.EventTo)
	out := make([]time.Time, 0, len(events))
	for _, at := range events {
		if at.After(window.EventFrom) && !at.After(window.EventTo) {
			out = append(out, at)
		}
	}
	return out
}

func decorationValues(ctx *statContext, decorations []*statSpec, at time.Time) map[string]any {
	if len(decorations) == 0 {
		return nil
	}
	out := make(map[string]any, len(decorations))
	for _, spec := range decorations {
		sampleAt := at
		if spec.Kind == statEvent {
			eventAt, ok := lastEventAtOrBefore(ctx, spec, at)
			if !ok {
				continue
			}
			sampleAt = eventAt
		}
		value := spec.Sample(ctx, sampleAt)
		if value == nil {
			continue
		}
		if spec.DecorationKeepsTime {
			stampSample(value, sampleAt)
		}
		out[spec.Type] = value
	}
	return out
}

func statTypesFromQuery(values map[string][]string, name string) []string {
	raw := values[name]
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		for _, item := range strings.Split(entry, ",") {
			if clean := strings.TrimSpace(item); clean != "" {
				out = append(out, clean)
			}
		}
	}
	return out
}

func dailyEventTimes(
	from time.Time,
	to time.Time,
	offset func(day time.Time) (time.Duration, bool),
) []time.Time {
	out := make([]time.Time, 0, 4)
	for day := from.UTC().Truncate(24 * time.Hour); !day.After(to); day = day.Add(24 * time.Hour) {
		delta, ok := offset(day)
		if !ok {
			continue
		}
		at := day.Add(delta)
		if !at.Before(from) && !at.After(to) {
			out = append(out, at)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

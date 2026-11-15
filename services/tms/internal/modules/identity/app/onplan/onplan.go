// Package onplan answers whether the organization's plan includes a resource,
// or a field of one: the resource's feature, and the feature a field declares
// when it belongs to another. The permission engine refuses through it, and
// every surface that shows a record inside another withholds through it.
// Self-hosted, every feature is on and nothing is withheld.
package onplan

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
)

// Plan is what a decision asks of the plan. A gate error is an error: each
// caller decides what it means, and none reads it as included.
type Plan interface {
	Resource(ctx context.Context, resource string) (permissionports.PlanDenial, bool, error)
	Field(ctx context.Context, resource, field string) (permissionports.PlanDenial, bool, error)
}

var (
	_ Plan                         = (*Lookup)(nil)
	_ Plan                         = (*Memo)(nil)
	_ permissionports.PlanFeatures = (*Lookup)(nil)
)

// Lookup is the process's answer, shared by every request. Each call asks
// the gate afresh, from its own short cache; a caller that asks many times
// for one decision takes a Memo.
type Lookup struct {
	registry *permissioncatalog.Registry
	features platformcatalogports.FeatureGate
}

func New(
	registry *permissioncatalog.Registry,
	features platformcatalogports.FeatureGate,
) *Lookup {
	return &Lookup{registry: registry, features: features}
}

// Memo is one call's or one request's answers: each feature is asked of the
// gate at most once, so every decision it makes sees the same plan.
func (l *Lookup) Memo() *Memo {
	return &Memo{lookup: l}
}

func (l *Lookup) Resource(
	ctx context.Context,
	resource string,
) (permissionports.PlanDenial, bool, error) {
	return l.Memo().Resource(ctx, resource)
}

func (l *Lookup) Field(
	ctx context.Context,
	resource, field string,
) (permissionports.PlanDenial, bool, error) {
	return l.Memo().Field(ctx, resource, field)
}

func (l *Lookup) Unavailable(ctx context.Context) ([]permissionports.PlanDenial, error) {
	return l.Memo().Unavailable(ctx)
}

// Feature reports whether the plan leaves out one feature, and the denial
// that names it.
func (l *Lookup) Feature(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (permissionports.PlanDenial, bool, error) {
	return l.Memo().denial(ctx, feature)
}

type Memo struct {
	lookup *Lookup

	mu  sync.Mutex
	off map[platformcatalog.FeatureKey]bool
}

// Resource reports whether the plan leaves out a resource's feature, and the
// denial that names it. A resource the registry does not know is left to the
// role logic.
func (m *Memo) Resource(
	ctx context.Context,
	resource string,
) (permissionports.PlanDenial, bool, error) {
	feature, known := m.lookup.registry.FeatureOf(resource)
	if !known {
		return permissionports.PlanDenial{}, false, nil
	}

	return m.denial(ctx, feature)
}

// Field reports whether the plan leaves out a field of a resource: the
// resource's own feature is off, or the field belongs to a feature of its
// own that is off. The resource's own denial comes first.
func (m *Memo) Field(
	ctx context.Context,
	resource, field string,
) (permissionports.PlanDenial, bool, error) {
	denial, off, err := m.Resource(ctx, resource)
	if err != nil || off {
		return denial, off, err
	}
	feature, declared := m.lookup.registry.FieldFeatureOf(resource, field)
	if !declared {
		return permissionports.PlanDenial{}, false, nil
	}

	return m.denial(ctx, feature)
}

// Unavailable is every feature the plan leaves out, in catalog order.
func (m *Memo) Unavailable(ctx context.Context) ([]permissionports.PlanDenial, error) {
	unavailable := make([]permissionports.PlanDenial, 0)
	for _, feature := range platformcatalog.Features() {
		off, err := m.FeatureOff(ctx, feature.Key)
		if err != nil {
			return nil, err
		}
		if off {
			unavailable = append(unavailable, permissionports.NewPlanDenial(feature.Key))
		}
	}

	return unavailable, nil
}

// FeatureOff reports whether the plan leaves out one feature.
func (m *Memo) FeatureOff(ctx context.Context, feature platformcatalog.FeatureKey) (bool, error) {
	m.mu.Lock()
	off, asked := m.off[feature]
	m.mu.Unlock()
	if asked {
		return off, nil
	}

	enabled, err := m.lookup.features.Enabled(ctx, feature)
	if err != nil {
		return false, fmt.Errorf("read whether the plan includes %s: %w", feature, err)
	}

	m.mu.Lock()
	if m.off == nil {
		m.off = make(map[platformcatalog.FeatureKey]bool, 4)
	}
	m.off[feature] = !enabled
	m.mu.Unlock()

	return !enabled, nil
}

func (m *Memo) denial(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (permissionports.PlanDenial, bool, error) {
	off, err := m.FeatureOff(ctx, feature)
	if err != nil || !off {
		return permissionports.PlanDenial{}, false, err
	}

	return permissionports.NewPlanDenial(feature), true, nil
}

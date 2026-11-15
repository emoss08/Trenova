package gqlctx

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/vektah/gqlparser/v2/ast"
)

type fieldCeilingKey struct{}

type ResourceCeilings interface {
	For(ctx context.Context, resource permission.Resource) permission.FieldSensitivity
	Reads(ctx context.Context, resource permission.Resource) bool
}

// FieldCeiling is an agent read's data access ceiling and the record of what
// it withheld. Only the agent read path sets one; an HTTP request never
// carries it, so the screen is gated by @protected groups alone.
type FieldCeiling struct {
	ceilings ResourceCeilings

	mu       sync.Mutex
	withheld map[string]struct{}
	byPlan   map[WithheldByPlan]struct{}
	strip    []ast.Path
}

// WithheldByPlan is one field the organization's plan withheld, by its path
// without list indices, and the feature it belongs to.
type WithheldByPlan struct {
	Path    string                     `json:"path"`
	Feature platformcatalog.FeatureKey `json:"feature"`
}

func NewFieldCeiling(ceilings ResourceCeilings) *FieldCeiling {
	return &FieldCeiling{
		ceilings: ceilings,
		withheld: make(map[string]struct{}, 4),
		byPlan:   make(map[WithheldByPlan]struct{}),
	}
}

func WithFieldCeiling(ctx context.Context, ceiling *FieldCeiling) context.Context {
	return context.WithValue(ctx, fieldCeilingKey{}, ceiling)
}

func FieldCeilingFrom(ctx context.Context) (*FieldCeiling, bool) {
	ceiling, ok := ctx.Value(fieldCeilingKey{}).(*FieldCeiling)
	return ceiling, ok && ceiling != nil
}

func (c *FieldCeiling) For(
	ctx context.Context,
	resource permission.Resource,
) permission.FieldSensitivity {
	if c.ceilings == nil {
		return permission.SensitivityInternal
	}

	return c.ceilings.For(ctx, resource)
}

// Reads reports whether the reader may read a resource at all.
func (c *FieldCeiling) Reads(ctx context.Context, resource permission.Resource) bool {
	if c.ceilings == nil {
		return true
	}

	return c.ceilings.Reads(ctx, resource)
}

// Withhold records a field the read may not show. A field answered null
// needs nothing more; strip marks a non-null one whose value the reader must
// remove from the result, because nulling it would null its parent.
func (c *FieldCeiling) Withhold(path ast.Path, strip bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.withheld[WithheldName(path)] = struct{}{}
	if strip {
		c.strip = append(c.strip, slices.Clone(path))
	}
}

// WithholdByPlan records a field the organization's plan does not include,
// apart from what the reader's access withheld; strip is as for Withhold.
func (c *FieldCeiling) WithholdByPlan(
	path ast.Path,
	feature platformcatalog.FeatureKey,
	strip bool,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byPlan[WithheldByPlan{Path: WithheldName(path), Feature: feature}] = struct{}{}
	if strip {
		c.strip = append(c.strip, slices.Clone(path))
	}
}

// WithheldByPlan is every field the plan withheld, sorted by path and then
// feature.
func (c *FieldCeiling) WithheldByPlan() []WithheldByPlan {
	c.mu.Lock()
	defer c.mu.Unlock()

	return SortedByPlan(c.byPlan)
}

// SortedByPlan lists plan withholdings by path and then feature.
func SortedByPlan(set map[WithheldByPlan]struct{}) []WithheldByPlan {
	out := make([]WithheldByPlan, 0, len(set))
	for entry := range set {
		out = append(out, entry)
	}
	slices.SortFunc(out, func(a, b WithheldByPlan) int {
		if byPath := strings.Compare(a.Path, b.Path); byPath != 0 {
			return byPath
		}
		return strings.Compare(string(a.Feature), string(b.Feature))
	})

	return out
}

// Withheld is every withheld field, by its path without list indices, sorted.
func (c *FieldCeiling) Withheld() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]string, 0, len(c.withheld))
	for name := range c.withheld {
		out = append(out, name)
	}
	slices.Sort(out)

	return out
}

// Stripped is every non-null withheld value's full path, for the reader to
// remove from the result.
func (c *FieldCeiling) Stripped() []ast.Path {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.strip)
}

// WithheldName is a path by its field names alone, without list indices.
func WithheldName(path ast.Path) string {
	var b strings.Builder
	for _, element := range path {
		name, ok := element.(ast.PathName)
		if !ok {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(string(name))
	}

	return b.String()
}
